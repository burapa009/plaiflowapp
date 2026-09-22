package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	"plaiflow/api/internal/auth"
)

func (s *Store) CreateAuthTransaction(ctx context.Context, transaction auth.Transaction) error {
	_, err := s.pool.Exec(ctx, `INSERT INTO auth_transactions
        (state_hash, browser_hash, provider, issuer, purpose, nonce, code_verifier, return_to,
         initiating_user_id, initiating_session_id, created_at, expires_at)
        VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`,
		transaction.StateHash, transaction.BrowserHash, transaction.Provider, transaction.Issuer, transaction.Purpose,
		transaction.Nonce, transaction.CodeVerifier, transaction.ReturnTo, nullableUUID(transaction.InitiatingUserID),
		nullableUUID(transaction.InitiatingSessionID), transaction.CreatedAt, transaction.ExpiresAt)
	return err
}

func (s *Store) ConsumeAuthTransaction(ctx context.Context, stateHash []byte, now time.Time) (auth.Transaction, error) {
	var transaction auth.Transaction
	var purpose string
	var userID, sessionID pgtype.UUID
	err := s.pool.QueryRow(ctx, `UPDATE auth_transactions SET consumed_at=$2
        WHERE state_hash=$1 AND consumed_at IS NULL AND expires_at>$2
        RETURNING state_hash,browser_hash,provider,issuer,purpose,nonce,code_verifier,return_to,
                  initiating_user_id,initiating_session_id,created_at,expires_at`, stateHash, now).Scan(
		&transaction.StateHash, &transaction.BrowserHash, &transaction.Provider, &transaction.Issuer, &purpose,
		&transaction.Nonce, &transaction.CodeVerifier, &transaction.ReturnTo, &userID, &sessionID,
		&transaction.CreatedAt, &transaction.ExpiresAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return auth.Transaction{}, auth.ErrAuthTransaction
	}
	if err != nil {
		return auth.Transaction{}, err
	}
	transaction.Purpose = auth.Purpose(purpose)
	transaction.InitiatingUserID = uuidString(userID)
	transaction.InitiatingSessionID = uuidString(sessionID)
	return transaction, nil
}

func (s *Store) Login(ctx context.Context, identity auth.ExternalIdentity, newUserID, newIdentityID string, now time.Time) (auth.LoginResult, error) {
	if result, found, err := s.findLogin(ctx, identity); err != nil || found {
		return result, err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return auth.LoginResult{}, err
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, `INSERT INTO users (id,display_name,picture_url,created_at,updated_at) VALUES ($1,$2,$3,$4,$4)`,
		newUserID, emptyToNil(identity.DisplayName), emptyToNil(identity.PictureURL), now); err != nil {
		return auth.LoginResult{}, err
	}
	_, err = tx.Exec(ctx, `INSERT INTO auth_identities
        (id,user_id,provider,issuer,subject,email,email_verified,display_name,picture_url,linked_at)
        VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`, newIdentityID, newUserID, identity.Provider, identity.Issuer,
		identity.Subject, emptyToNil(identity.Email), identity.EmailVerified, emptyToNil(identity.DisplayName), emptyToNil(identity.PictureURL), now)
	if err != nil {
		var databaseError *pgconn.PgError
		if errors.As(err, &databaseError) && databaseError.Code == "23505" {
			_ = tx.Rollback(ctx)
			if result, found, findErr := s.findLogin(ctx, identity); findErr != nil || found {
				return result, findErr
			}
			return auth.LoginResult{}, auth.ErrIdentityConflict
		}
		return auth.LoginResult{}, err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO audit_events (actor_user_id,event_type,target_type,target_id,outcome)
        VALUES ($1,'auth.login','auth_identity',$2,'success')`, newUserID, newIdentityID); err != nil {
		return auth.LoginResult{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return auth.LoginResult{}, err
	}
	return auth.LoginResult{UserID: newUserID, Created: true}, nil
}

func (s *Store) findLogin(ctx context.Context, identity auth.ExternalIdentity) (auth.LoginResult, bool, error) {
	var userID string
	var disabledAt *time.Time
	err := s.pool.QueryRow(ctx, `SELECT user_id,disabled_at FROM auth_identities WHERE issuer=$1 AND subject=$2`, identity.Issuer, identity.Subject).Scan(&userID, &disabledAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return auth.LoginResult{}, false, nil
	}
	if err != nil {
		return auth.LoginResult{}, false, err
	}
	if disabledAt != nil {
		return auth.LoginResult{}, true, auth.ErrIdentityConflict
	}
	return auth.LoginResult{UserID: userID}, true, nil
}

func (s *Store) CreateSession(ctx context.Context, session auth.SessionRecord) error {
	_, err := s.pool.Exec(ctx, `INSERT INTO sessions
        (id,user_id,token_hash,csrf_hash,created_at,authenticated_at,authenticated_provider,last_seen_at,idle_expires_at,absolute_expires_at)
        VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`, session.ID, session.UserID, session.TokenHash, session.CSRFHash,
		session.CreatedAt, session.AuthenticatedAt, session.AuthenticatedProvider, session.LastSeenAt, session.IdleExpiresAt, session.AbsoluteExpiresAt)
	return err
}

func (s *Store) ResolveSession(ctx context.Context, tokenHash []byte, now time.Time) (auth.Session, error) {
	var session auth.Session
	var lastSeen time.Time
	err := s.pool.QueryRow(ctx, `SELECT id,user_id,csrf_hash,authenticated_at,authenticated_provider,last_seen_at,idle_expires_at,absolute_expires_at
        FROM sessions WHERE token_hash=$1 AND revoked_at IS NULL AND idle_expires_at>$2 AND absolute_expires_at>$2`, tokenHash, now).Scan(
		&session.ID, &session.UserID, &session.CSRFHash, &session.AuthenticatedAt, &session.AuthenticatedProvider,
		&lastSeen, &session.IdleExpiresAt, &session.AbsoluteExpiresAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return auth.Session{}, auth.ErrSession
	}
	if err != nil {
		return auth.Session{}, err
	}
	if !lastSeen.After(now.Add(-5 * time.Minute)) {
		idleExpiry := now.Add(14 * 24 * time.Hour)
		if idleExpiry.After(session.AbsoluteExpiresAt) {
			idleExpiry = session.AbsoluteExpiresAt
		}
		if _, err := s.pool.Exec(ctx, `UPDATE sessions SET last_seen_at=$2,idle_expires_at=$3
            WHERE id=$1 AND last_seen_at<=($2::timestamptz-interval '5 minutes')`, session.ID, now, idleExpiry); err != nil {
			return auth.Session{}, err
		}
		session.IdleExpiresAt = idleExpiry
	}
	return session, nil
}

func (s *Store) RevokeSession(ctx context.Context, sessionID string, now time.Time) error {
	_, err := s.pool.Exec(ctx, `UPDATE sessions SET revoked_at=$2 WHERE id=$1 AND revoked_at IS NULL`, sessionID, now)
	return err
}

func (s *Store) RevokeUserSessions(ctx context.Context, userID string, now time.Time) error {
	_, err := s.pool.Exec(ctx, `UPDATE sessions SET revoked_at=$2 WHERE user_id=$1 AND revoked_at IS NULL`, userID, now)
	return err
}

func (s *Store) LinkIdentityAndRotate(ctx context.Context, request auth.LinkRequest) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if err := requireRecentSession(ctx, tx, request.UserID, request.SessionID, request.Now); err != nil {
		return err
	}
	var ownerID string
	var disabledAt *time.Time
	err = tx.QueryRow(ctx, `SELECT user_id,disabled_at FROM auth_identities WHERE issuer=$1 AND subject=$2 FOR UPDATE`, request.Identity.Issuer, request.Identity.Subject).Scan(&ownerID, &disabledAt)
	switch {
	case err == nil && ownerID != request.UserID:
		return auth.ErrIdentityConflict
	case err == nil && disabledAt == nil:
		return auth.ErrIdentityConflict
	case err == nil:
		_, err = tx.Exec(ctx, `UPDATE auth_identities SET disabled_at=NULL,cooldown_until=NULL,email=$3,email_verified=$4,
            display_name=$5,picture_url=$6,linked_at=$7 WHERE issuer=$1 AND subject=$2`, request.Identity.Issuer,
			request.Identity.Subject, emptyToNil(request.Identity.Email), request.Identity.EmailVerified,
			emptyToNil(request.Identity.DisplayName), emptyToNil(request.Identity.PictureURL), request.Now)
	case errors.Is(err, pgx.ErrNoRows):
		_, err = tx.Exec(ctx, `INSERT INTO auth_identities
            (id,user_id,provider,issuer,subject,email,email_verified,display_name,picture_url,linked_at)
            VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`, request.NewSession.ID, request.UserID, request.Identity.Provider,
			request.Identity.Issuer, request.Identity.Subject, emptyToNil(request.Identity.Email), request.Identity.EmailVerified,
			emptyToNil(request.Identity.DisplayName), emptyToNil(request.Identity.PictureURL), request.Now)
	default:
		return err
	}
	if err != nil {
		return identityConstraintError(err)
	}
	if err := rotateSession(ctx, tx, request.SessionID, request.NewSession, request.Now); err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `INSERT INTO audit_events (actor_user_id,event_type,target_type,target_id,outcome)
        VALUES ($1,'auth.identity_link',$2,$3,'success')`, request.UserID, request.Identity.Provider, request.Identity.Subject)
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *Store) ReauthenticateAndRotate(ctx context.Context, request auth.ReauthRequest) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var ownerID string
	err = tx.QueryRow(ctx, `SELECT user_id FROM auth_identities
        WHERE issuer=$1 AND subject=$2 AND disabled_at IS NULL FOR UPDATE`, request.Identity.Issuer, request.Identity.Subject).Scan(&ownerID)
	if errors.Is(err, pgx.ErrNoRows) || ownerID != request.UserID {
		return auth.ErrIdentityConflict
	}
	if err != nil {
		return err
	}
	if err := rotateSession(ctx, tx, request.SessionID, request.NewSession, request.Now); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *Store) UnlinkIdentityAndRotate(ctx context.Context, request auth.UnlinkRequest) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var authenticatedProvider string
	err = tx.QueryRow(ctx, `SELECT authenticated_provider FROM sessions
        WHERE id=$1 AND user_id=$2 AND revoked_at IS NULL AND authenticated_at>$3-interval '10 minutes' FOR UPDATE`,
		request.SessionID, request.UserID, request.Now).Scan(&authenticatedProvider)
	if errors.Is(err, pgx.ErrNoRows) {
		return auth.ErrRecentAuth
	}
	if err != nil {
		return err
	}
	if authenticatedProvider == request.Provider {
		return auth.ErrRecentAuth
	}
	var count int
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM auth_identities WHERE user_id=$1 AND disabled_at IS NULL`, request.UserID).Scan(&count); err != nil {
		return err
	}
	if count < 2 {
		return auth.ErrLastIdentity
	}
	command, err := tx.Exec(ctx, `UPDATE auth_identities SET disabled_at=$3,cooldown_until=$4
        WHERE user_id=$1 AND provider=$2 AND disabled_at IS NULL`, request.UserID, request.Provider, request.Now, request.CooldownUntil)
	if err != nil {
		return err
	}
	if command.RowsAffected() != 1 {
		return auth.ErrLastIdentity
	}
	if _, err := tx.Exec(ctx, `UPDATE sessions SET revoked_at=$2 WHERE user_id=$1 AND revoked_at IS NULL`, request.UserID, request.Now); err != nil {
		return err
	}
	if err := insertSession(ctx, tx, request.NewSession); err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `INSERT INTO audit_events (actor_user_id,event_type,target_type,target_id,outcome)
        VALUES ($1,'auth.identity_unlink',$2,$2,'success')`, request.UserID, request.Provider)
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *Store) ListIdentities(ctx context.Context, userID string) ([]auth.IdentityState, error) {
	rows, err := s.pool.Query(ctx, `SELECT provider,true FROM auth_identities
        WHERE user_id=$1 AND disabled_at IS NULL ORDER BY provider`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (auth.IdentityState, error) {
		var identity auth.IdentityState
		err := row.Scan(&identity.Provider, &identity.Linked)
		return identity, err
	})
}

func requireRecentSession(ctx context.Context, tx pgx.Tx, userID, sessionID string, now time.Time) error {
	var valid bool
	err := tx.QueryRow(ctx, `SELECT true FROM sessions
        WHERE id=$1 AND user_id=$2 AND revoked_at IS NULL AND idle_expires_at>$3 AND absolute_expires_at>$3
          AND authenticated_at>$3-interval '10 minutes' FOR UPDATE`, sessionID, userID, now).Scan(&valid)
	if errors.Is(err, pgx.ErrNoRows) {
		return auth.ErrRecentAuth
	}
	return err
}

func rotateSession(ctx context.Context, tx pgx.Tx, oldSessionID string, session auth.SessionRecord, now time.Time) error {
	command, err := tx.Exec(ctx, `UPDATE sessions SET revoked_at=$2 WHERE id=$1 AND revoked_at IS NULL`, oldSessionID, now)
	if err != nil {
		return err
	}
	if command.RowsAffected() != 1 {
		return auth.ErrSession
	}
	return insertSession(ctx, tx, session)
}

func insertSession(ctx context.Context, tx pgx.Tx, session auth.SessionRecord) error {
	_, err := tx.Exec(ctx, `INSERT INTO sessions
        (id,user_id,token_hash,csrf_hash,created_at,authenticated_at,authenticated_provider,last_seen_at,idle_expires_at,absolute_expires_at)
        VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`, session.ID, session.UserID, session.TokenHash, session.CSRFHash,
		session.CreatedAt, session.AuthenticatedAt, session.AuthenticatedProvider, session.LastSeenAt, session.IdleExpiresAt, session.AbsoluteExpiresAt)
	return err
}

func identityConstraintError(err error) error {
	var databaseError *pgconn.PgError
	if errors.As(err, &databaseError) && databaseError.Code == "23505" {
		return auth.ErrIdentityConflict
	}
	return err
}

func nullableUUID(value string) any {
	if value == "" {
		return nil
	}
	return value
}

func uuidString(value pgtype.UUID) string {
	if !value.Valid {
		return ""
	}
	return value.String()
}

func emptyToNil(value string) any {
	if value == "" {
		return nil
	}
	return value
}
