package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"plaiflow/api/internal/drive"
	"plaiflow/api/internal/tenant"
)

func (s *Store) SaveAttempt(ctx context.Context, attempt drive.Attempt) error {
	tx, err := s.organizationTx(ctx, attempt.UserID, attempt.OrganizationID)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	role, err := currentRole(ctx, tx, attempt.UserID, attempt.OrganizationID)
	if err != nil {
		return err
	}
	if role != tenant.Owner {
		return drive.ErrInvalidAttempt
	}
	_, err = tx.Exec(ctx, `INSERT INTO drive_oauth_attempts
        (state_hash,browser_hash,organization_id,organization_name,user_id,session_id,code_verifier,created_at,expires_at)
        VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`, attempt.StateHash, attempt.BrowserHash, attempt.OrganizationID,
		attempt.OrganizationName, attempt.UserID, attempt.SessionID, attempt.CodeVerifier, attempt.CreatedAt, attempt.ExpiresAt)
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *Store) ConsumeAttempt(ctx context.Context, stateHash []byte, now time.Time) (drive.Attempt, error) {
	var attempt drive.Attempt
	err := s.pool.QueryRow(ctx, `SELECT state_hash,browser_hash,organization_id,organization_name,user_id,session_id,code_verifier,created_at,expires_at
        FROM consume_drive_oauth_attempt($1,$2)`,
		stateHash, now).Scan(&attempt.StateHash, &attempt.BrowserHash, &attempt.OrganizationID, &attempt.OrganizationName,
		&attempt.UserID, &attempt.SessionID, &attempt.CodeVerifier, &attempt.CreatedAt, &attempt.ExpiresAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return drive.Attempt{}, drive.ErrInvalidAttempt
	}
	return attempt, err
}

func (s *Store) SaveConnection(ctx context.Context, connection drive.Connection) error {
	tx, err := s.organizationTx(ctx, connection.AuthorizerUserID, connection.OrganizationID)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	role, err := currentRole(ctx, tx, connection.AuthorizerUserID, connection.OrganizationID)
	if err != nil {
		return err
	}
	if role != tenant.Owner {
		return drive.ErrInvalidAttempt
	}
	err = tx.QueryRow(ctx, `INSERT INTO drive_connections
        (organization_id,status,google_subject,google_email,folder_id,authorizer_user_id,encrypted_refresh_token,token_nonce,credential_generation,connected_at,updated_at)
        VALUES ($1,'Connected',$2,$3,$4,$5,$6,$7,1,$8,$8)
        ON CONFLICT (organization_id) DO UPDATE SET status='Connected',google_subject=excluded.google_subject,
          google_email=excluded.google_email,folder_id=coalesce(drive_connections.folder_id,excluded.folder_id),
          authorizer_user_id=excluded.authorizer_user_id,encrypted_refresh_token=excluded.encrypted_refresh_token,
          token_nonce=excluded.token_nonce,credential_generation=drive_connections.credential_generation+1,
          connected_at=excluded.connected_at,updated_at=excluded.updated_at,disconnected_at=NULL
        RETURNING credential_generation,folder_id`, connection.OrganizationID, connection.GoogleSubject, connection.GoogleEmail,
		connection.FolderID, connection.AuthorizerUserID, connection.EncryptedRefreshToken, connection.TokenNonce, connection.ConnectedAt).Scan(
		&connection.CredentialGeneration, &connection.FolderID)
	if err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `UPDATE tasks SET status='Done',updated_at=$3,status_changed_at=$3
        WHERE organization_id=$1 AND status NOT IN ('Done','Cancelled') AND id IN (
          SELECT task_id FROM drive_reconnect_tasks WHERE organization_id=$1 AND resolved_at IS NULL AND credential_generation<$2
        )`, connection.OrganizationID, connection.CredentialGeneration, connection.ConnectedAt); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `UPDATE drive_reconnect_tasks SET resolved_at=$3
        WHERE organization_id=$1 AND resolved_at IS NULL AND credential_generation<$2`,
		connection.OrganizationID, connection.CredentialGeneration, connection.ConnectedAt); err != nil {
		return err
	}
	if err := auditTenant(ctx, tx, connection.OrganizationID, connection.AuthorizerUserID, "drive.connect", "drive_connection", connection.OrganizationID, connection.ConnectedAt); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *Store) GetConnection(ctx context.Context, actorUserID, organizationID string) (drive.Connection, error) {
	tx, err := s.organizationTx(ctx, actorUserID, organizationID)
	if err != nil {
		return drive.Connection{}, err
	}
	defer tx.Rollback(ctx)
	role, err := currentRole(ctx, tx, actorUserID, organizationID)
	if err != nil {
		return drive.Connection{}, err
	}
	var connection drive.Connection
	err = tx.QueryRow(ctx, `SELECT organization_id,status,
		CASE WHEN $2 IN ('Owner','Admin') THEN coalesce(google_email,'') ELSE '' END,coalesce(google_subject,''),
		CASE WHEN $2 IN ('Owner','Admin') THEN coalesce(folder_id,'') ELSE '' END,
	    CASE WHEN $2 IN ('Owner','Admin') THEN coalesce(authorizer_user_id::text,'') ELSE '' END,
		CASE WHEN $2 IN ('Owner','Admin') THEN coalesce(encrypted_refresh_token,''::bytea) ELSE ''::bytea END,
		CASE WHEN $2 IN ('Owner','Admin') THEN coalesce(token_nonce,''::bytea) ELSE ''::bytea END,
        credential_generation,coalesce(connected_at,'epoch'::timestamptz),updated_at
		FROM drive_connections WHERE organization_id=$1`, organizationID, role).Scan(&connection.OrganizationID, &connection.Status,
		&connection.GoogleEmail, &connection.GoogleSubject, &connection.FolderID, &connection.AuthorizerUserID,
		&connection.EncryptedRefreshToken, &connection.TokenNonce, &connection.CredentialGeneration, &connection.ConnectedAt, &connection.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return drive.Connection{OrganizationID: organizationID, Status: drive.StatusNotConnected}, tx.Commit(ctx)
	}
	if err != nil {
		return drive.Connection{}, err
	}
	return connection, tx.Commit(ctx)
}

func (s *Store) Disconnect(ctx context.Context, actorUserID, organizationID string, now time.Time) (drive.Connection, error) {
	tx, err := s.organizationTx(ctx, actorUserID, organizationID)
	if err != nil {
		return drive.Connection{}, err
	}
	defer tx.Rollback(ctx)
	role, err := currentRole(ctx, tx, actorUserID, organizationID)
	if err != nil {
		return drive.Connection{}, err
	}
	if role != tenant.Owner {
		return drive.Connection{}, tenant.ErrForbidden
	}
	var connection drive.Connection
	err = tx.QueryRow(ctx, `WITH previous AS (
        SELECT * FROM drive_connections WHERE organization_id=$1 AND status<>'Not Connected' FOR UPDATE
    ), updated AS (
        UPDATE drive_connections SET status='Not Connected',encrypted_refresh_token=NULL,token_nonce=NULL,
          updated_at=$3,disconnected_at=$3 FROM previous WHERE drive_connections.organization_id=previous.organization_id
        RETURNING drive_connections.organization_id
    ) SELECT previous.organization_id,'Not Connected',coalesce(previous.google_email,''),coalesce(previous.google_subject,''),
        coalesce(previous.folder_id,''),coalesce(previous.authorizer_user_id::text,''),previous.encrypted_refresh_token,
        previous.token_nonce,previous.credential_generation,coalesce(previous.connected_at,'epoch'::timestamptz),$3
      FROM previous JOIN updated ON updated.organization_id=previous.organization_id`, organizationID, actorUserID, now).Scan(
		&connection.OrganizationID, &connection.Status, &connection.GoogleEmail, &connection.GoogleSubject, &connection.FolderID,
		&connection.AuthorizerUserID, &connection.EncryptedRefreshToken, &connection.TokenNonce, &connection.CredentialGeneration,
		&connection.ConnectedAt, &connection.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return drive.Connection{}, drive.ErrNotConnected
	}
	if err != nil {
		return drive.Connection{}, err
	}
	if err := auditTenant(ctx, tx, organizationID, actorUserID, "drive.disconnect", "drive_connection", organizationID, now); err != nil {
		return drive.Connection{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return drive.Connection{}, err
	}
	return connection, nil
}

func (s *Store) RequireReconnect(ctx context.Context, organizationID string, generation int64, now time.Time) error {
	var changed bool
	return s.pool.QueryRow(ctx, `SELECT mark_drive_reconnect_required($1::uuid,$2,$3,$4::uuid)`,
		organizationID, generation, now, postgresUUID()).Scan(&changed)
}
