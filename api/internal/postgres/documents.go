package postgres

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"plaiflow/api/internal/document"
	"plaiflow/api/internal/tenant"
)

func (s *Store) CommitPrepared(ctx context.Context, input document.CommitInput) (document.CommitResult, error) {
	if (input.Channel != "Web" && input.Channel != "LINE" && input.Channel != "Drive") || (input.LINEGroup && input.Channel != "LINE") || input.Now.IsZero() || input.Size < 1 || input.Size > document.MaxFileBytes || (input.Channel == "Drive" && (input.DriveConnectionID == "" || input.DriveFileID == "" || input.DriveRevision == "" || input.DriveProviderMIME == "" || input.DriveSelectedAt.IsZero())) {
		return document.CommitResult{}, errors.New("invalid prepared document")
	}
	digest, err := hex.DecodeString(input.SHA256)
	if err != nil || len(digest) != 32 {
		return document.CommitResult{}, errors.New("invalid document digest")
	}
	tx, err := s.organizationTx(ctx, input.ActorUserID, input.OrganizationID)
	if err != nil {
		return document.CommitResult{}, err
	}
	defer tx.Rollback(ctx)
	if _, err := membershipFor(ctx, tx, input.ActorUserID, input.OrganizationID); err != nil {
		return document.CommitResult{}, tenant.ErrNotFound
	}
	// A short transaction lock serializes unique-content and hard-quota decisions per Organization.
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1,0))`, input.OrganizationID); err != nil {
		return document.CommitResult{}, err
	}
	if input.LINEGroup {
		var sourceDocumentID string
		err := tx.QueryRow(ctx, `SELECT document_id FROM document_sources
		    WHERE organization_id=$1 AND channel='LINE' AND origin_key=$2 AND submitted_by_user_id=$3`,
			input.OrganizationID, input.OriginKey, input.ActorUserID).Scan(&sourceDocumentID)
		if err == nil {
			var contentDocumentID, status string
			lookupErr := tx.QueryRow(ctx, `SELECT document_id,document_status FROM lookup_document_for_intake($1::uuid,$2::bytea)`, input.OrganizationID, digest).Scan(&contentDocumentID, &status)
			if lookupErr != nil || contentDocumentID != sourceDocumentID {
				return document.CommitResult{}, document.ErrSourceConflict
			}
			if err := tx.Commit(ctx); err != nil {
				return document.CommitResult{}, err
			}
			return document.CommitResult{Document: document.Document{ID: sourceDocumentID, OrganizationID: input.OrganizationID, Status: status}, Duplicate: true}, nil
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return document.CommitResult{}, err
		}
	}
	var previous document.Document
	var previousHash []byte
	err = tx.QueryRow(ctx, `SELECT d.id,d.organization_id,d.display_filename,d.detected_mime,d.byte_size,d.status,d.storage_key,d.accepted_at,d.content_sha256
	    FROM document_sources ds JOIN documents d ON d.organization_id=ds.organization_id AND d.id=ds.document_id
	    WHERE ds.organization_id=$1 AND ds.channel=$2 AND ds.origin_key=$3 AND ds.submitted_by_user_id=$4`,
		input.OrganizationID, input.Channel, input.OriginKey, input.ActorUserID).Scan(&previous.ID, &previous.OrganizationID, &previous.Filename, &previous.MIME, &previous.Size, &previous.Status, &previous.StorageKey, &previous.AcceptedAt, &previousHash)
	if err == nil {
		if !bytes.Equal(previousHash, digest) {
			return document.CommitResult{}, document.ErrSourceConflict
		}
		if err := tx.Commit(ctx); err != nil {
			return document.CommitResult{}, err
		}
		return document.CommitResult{Document: previous, Duplicate: true}, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return document.CommitResult{}, err
	}
	var existingID, existingStatus string
	err = tx.QueryRow(ctx, `SELECT document_id,document_status FROM lookup_document_for_intake($1::uuid,$2::bytea)`, input.OrganizationID, digest).Scan(&existingID, &existingStatus)
	if err == nil {
		if existingStatus == "Trash" {
			return document.CommitResult{}, document.ErrTrashed
		}
		if err := insertDocumentSource(ctx, tx, input, existingID); err != nil {
			return document.CommitResult{}, sourceConflict(err)
		}
		if _, err := tx.Exec(ctx, `INSERT INTO document_intake_attempts
		    (id,organization_id,actor_user_id,channel,origin_key,status,document_id,created_at,updated_at,expires_at)
		    VALUES ($1,$2,$3,$4,$5,'Accepted',$6,$7,$7,$7+interval '90 days')
		    ON CONFLICT (organization_id,channel,origin_key) DO UPDATE SET status='Accepted',document_id=excluded.document_id,rejection_code=NULL,updated_at=excluded.updated_at`,
			input.AttemptID, input.OrganizationID, input.ActorUserID, input.Channel, input.OriginKey, existingID, input.Now); err != nil {
			return document.CommitResult{}, err
		}
		if input.LINEGroup {
			if err := tx.Commit(ctx); err != nil {
				return document.CommitResult{}, err
			}
			return document.CommitResult{Document: document.Document{ID: existingID, OrganizationID: input.OrganizationID, Status: existingStatus}, Duplicate: true}, nil
		}
		if err := tx.QueryRow(ctx, `SELECT id,organization_id,display_filename,detected_mime,byte_size,status,storage_key,accepted_at
		    FROM documents WHERE organization_id=$1 AND id=$2`, input.OrganizationID, existingID).Scan(
			&previous.ID, &previous.OrganizationID, &previous.Filename, &previous.MIME, &previous.Size, &previous.Status, &previous.StorageKey, &previous.AcceptedAt); err != nil {
			return document.CommitResult{}, err
		}
		if err := tx.Commit(ctx); err != nil {
			return document.CommitResult{}, err
		}
		return document.CommitResult{Document: previous, Duplicate: true}, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return document.CommitResult{}, err
	}
	var timezone string
	if err := tx.QueryRow(ctx, `SELECT timezone FROM organizations WHERE id=$1`, input.OrganizationID).Scan(&timezone); err != nil {
		return document.CommitResult{}, err
	}
	periodStart, _, err := currentDocumentPeriod(ctx, tx, input.OrganizationID, timezone, input.Now)
	if err != nil {
		return document.CommitResult{}, err
	}
	limit, trialStart, trialEnd, err := documentLimit(ctx, tx, input.OrganizationID, input.Now)
	if err != nil {
		return document.CommitResult{}, err
	}
	var used int
	if trialEnd.After(trialStart) {
		err = tx.QueryRow(ctx, `SELECT count(*) FROM document_usage_charges WHERE organization_id=$1 AND accepted_at >= $2 AND accepted_at < $3`,
			input.OrganizationID, trialStart, trialEnd).Scan(&used)
	} else {
		err = tx.QueryRow(ctx, `SELECT count(*) FROM document_usage_charges WHERE organization_id=$1 AND period_start=$2`, input.OrganizationID, periodStart).Scan(&used)
	}
	if err != nil {
		return document.CommitResult{}, err
	}
	if used >= limit {
		if _, err := tx.Exec(ctx, `INSERT INTO document_intake_attempts
		    (id,organization_id,actor_user_id,channel,origin_key,status,rejection_code,created_at,updated_at,expires_at)
		    VALUES ($1,$2,$3,$4,$5,'Rejected','quota_exhausted',$6,$6,$6+interval '90 days')
		    ON CONFLICT (organization_id,channel,origin_key) DO NOTHING`,
			input.AttemptID, input.OrganizationID, input.ActorUserID, input.Channel, input.OriginKey, input.Now); err != nil {
			return document.CommitResult{}, err
		}
		if err := tx.Commit(ctx); err != nil {
			return document.CommitResult{}, err
		}
		return document.CommitResult{}, document.ErrQuota
	}
	id := postgresUUID()
	if _, err := tx.Exec(ctx, `INSERT INTO documents
	    (id,organization_id,content_sha256,storage_key,display_filename,detected_mime,byte_size,status,submitted_by_user_id,group_restricted,accepted_at,updated_at,usage_period_start)
	    VALUES ($1,$2,$3,$4,$5,$6,$7,'Available',$8,$11,$9,$9,$10)`,
		id, input.OrganizationID, digest, input.TemporaryKey, input.Filename, input.MIME, input.Size, input.ActorUserID, input.Now, periodStart, input.LINEGroup); err != nil {
		return document.CommitResult{}, err
	}
	if err := insertDocumentSource(ctx, tx, input, id); err != nil {
		return document.CommitResult{}, sourceConflict(err)
	}
	if _, err := tx.Exec(ctx, `INSERT INTO document_usage_charges (organization_id,document_id,period_start,accepted_at)
	    VALUES ($1,$2,$3,$4)`, input.OrganizationID, id, periodStart, input.Now); err != nil {
		return document.CommitResult{}, err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO document_intake_attempts
	    (id,organization_id,actor_user_id,channel,origin_key,status,document_id,created_at,updated_at,expires_at)
	    VALUES ($1,$2,$3,$4,$5,'Accepted',$6,$7,$7,$7+interval '90 days')
	    ON CONFLICT (organization_id,channel,origin_key) DO UPDATE SET status='Accepted',document_id=excluded.document_id,
	        rejection_code=NULL,updated_at=excluded.updated_at`,
		input.AttemptID, input.OrganizationID, input.ActorUserID, input.Channel, input.OriginKey, id, input.Now); err != nil {
		return document.CommitResult{}, err
	}
	if err := auditTenant(ctx, tx, input.OrganizationID, input.ActorUserID, "document.accept", "document", id, input.Now); err != nil {
		return document.CommitResult{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return document.CommitResult{}, err
	}
	return document.CommitResult{Document: document.Document{ID: id, OrganizationID: input.OrganizationID, Filename: input.Filename, MIME: input.MIME,
		Size: input.Size, Status: "Available", StorageKey: input.TemporaryKey, AcceptedAt: input.Now}, Accepted: true}, nil
}

func insertDocumentSource(ctx context.Context, tx pgx.Tx, input document.CommitInput, documentID string) error {
	var providerFilename, providerMIME, providerSize, selectedAt any
	if input.Channel == "Drive" {
		providerFilename, providerMIME, providerSize, selectedAt = input.Filename, input.DriveProviderMIME, input.DriveProviderSize, input.DriveSelectedAt
	}
	_, err := tx.Exec(ctx, `INSERT INTO document_sources
	    (id,organization_id,document_id,channel,origin_key,submitted_by_user_id,drive_connection_id,drive_file_id,drive_revision,
	     provider_filename,provider_mime,provider_size,selected_at,created_at,group_source)
	    VALUES ($1,$2,$3,$4,$5,$6,nullif($8,'')::uuid,nullif($9,''),nullif($10,''),$11,$12,$13,$14,$7,$15)`,
		postgresUUID(), input.OrganizationID, documentID, input.Channel, input.OriginKey, input.ActorUserID, input.Now,
		input.DriveConnectionID, input.DriveFileID, input.DriveRevision, providerFilename, providerMIME, providerSize, selectedAt, input.LINEGroup)
	return err
}

func sourceConflict(err error) error {
	var databaseError *pgconn.PgError
	if errors.As(err, &databaseError) && databaseError.Code == "23505" {
		return document.ErrSourceConflict
	}
	return err
}

func (s *Store) RecordRejected(ctx context.Context, input document.AcceptInput, reason string) error {
	if reason == "" || input.OrganizationID == "" || input.ActorUserID == "" || input.AttemptID == "" || input.OriginKey == "" {
		return errors.New("invalid rejected document attempt")
	}
	now := input.Now
	if now.IsZero() {
		now = time.Now().UTC()
	}
	tx, err := s.organizationTx(ctx, input.ActorUserID, input.OrganizationID)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err := membershipFor(ctx, tx, input.ActorUserID, input.OrganizationID); err != nil {
		return tenant.ErrNotFound
	}
	if _, err := tx.Exec(ctx, `INSERT INTO document_intake_attempts
	    (id,organization_id,actor_user_id,channel,origin_key,status,rejection_code,created_at,updated_at,expires_at)
	    VALUES ($1,$2,$3,$4,$5,'Rejected',$6,$7,$7,$7+interval '90 days')
	    ON CONFLICT (organization_id,channel,origin_key) DO UPDATE SET
	        status=CASE WHEN document_intake_attempts.status='Accepted' THEN document_intake_attempts.status ELSE 'Rejected' END,
	        rejection_code=CASE WHEN document_intake_attempts.status='Accepted' THEN document_intake_attempts.rejection_code ELSE excluded.rejection_code END,
	        updated_at=excluded.updated_at`,
		input.AttemptID, input.OrganizationID, input.ActorUserID, input.Channel, input.OriginKey, reason, now); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func currentDocumentPeriod(ctx context.Context, tx pgx.Tx, organizationID, timezone string, now time.Time) (time.Time, time.Time, error) {
	var start, end time.Time
	err := tx.QueryRow(ctx, `SELECT starts_at,ends_at FROM document_usage_periods
	    WHERE organization_id=$1 AND starts_at<=$2 AND ends_at>$2 ORDER BY starts_at DESC LIMIT 1`, organizationID, now).Scan(&start, &end)
	if err == nil {
		return start, end, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return time.Time{}, time.Time{}, err
	}
	location, err := time.LoadLocation(timezone)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	local := now.In(location)
	start = time.Date(local.Year(), local.Month(), 1, 0, 0, 0, 0, location).UTC()
	end = time.Date(local.Year(), local.Month()+1, 1, 0, 0, 0, 0, location).UTC()
	_, err = tx.Exec(ctx, `INSERT INTO document_usage_periods (organization_id,starts_at,ends_at,timezone)
	    VALUES ($1,$2,$3,$4) ON CONFLICT (organization_id,starts_at) DO NOTHING`, organizationID, start, end, timezone)
	return start, end, err
}

func documentLimit(ctx context.Context, tx pgx.Tx, organizationID string, now time.Time) (int, time.Time, time.Time, error) {
	var key, source string
	var start, end time.Time
	err := tx.QueryRow(ctx, `SELECT plan_key,source,starts_at,ends_at FROM organization_plan_periods
	    WHERE organization_id=$1 AND starts_at<=$2 AND ends_at>$2 ORDER BY starts_at DESC,created_at DESC LIMIT 1`, organizationID, now).Scan(&key, &source, &start, &end)
	if errors.Is(err, pgx.ErrNoRows) {
		return 30, time.Time{}, time.Time{}, nil
	}
	if err != nil {
		return 0, time.Time{}, time.Time{}, err
	}
	if source == "trial" {
		return 100, start, end, nil
	}
	if key == "Starter" {
		return 300, time.Time{}, time.Time{}, nil
	}
	if key == "Business" {
		return 1000, time.Time{}, time.Time{}, nil
	}
	return 0, time.Time{}, time.Time{}, errors.New("invalid document plan")
}
