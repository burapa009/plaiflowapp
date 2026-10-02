package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"plaiflow/api/internal/classification"
	"plaiflow/api/internal/extraction"
	"plaiflow/api/internal/tenant"
)

func (s *Store) SaveClassification(ctx context.Context, org, doc, ocrJob string, result classification.Result) error {
	if !classification.Valid(result.DocumentType) {
		return errors.New("invalid classification")
	}
	data, err := json.Marshal(result)
	if err != nil {
		return err
	}
	tag, err := s.pool.Exec(ctx, `INSERT INTO document_classifications(ocr_job_id,organization_id,document_id,document_type,result)
		SELECT o.job_id,o.organization_id,o.document_id,$4,$5 FROM document_ocr_runs o
		WHERE o.job_id=$1 AND o.organization_id=$2 AND o.document_id=$3 AND o.published_at IS NOT NULL
		AND o.superseded_at IS NULL AND o.deleted_at IS NULL ON CONFLICT (ocr_job_id) DO NOTHING`, ocrJob, org, doc, result.DocumentType, data)
	if err == nil && tag.RowsAffected() == 0 {
		var exists bool
		err = s.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM document_classifications WHERE ocr_job_id=$1 AND organization_id=$2 AND document_id=$3)`, ocrJob, org, doc).Scan(&exists)
		if err == nil && !exists {
			return extraction.ErrConflict
		}
	}
	return err
}

func (s *Store) GetClassification(ctx context.Context, user, org, doc string) (classification.Record, error) {
	if _, err := s.GetDocument(ctx, user, org, doc); err != nil {
		return classification.Record{}, err
	}
	tx, err := s.organizationTx(ctx, user, org)
	if err != nil {
		return classification.Record{}, err
	}
	defer tx.Rollback(ctx)
	var data []byte
	var corrected string
	var by string
	var at *time.Time
	err = tx.QueryRow(ctx, `SELECT c.result,coalesce(c.corrected_type,''),coalesce(c.corrected_by::text,''),c.corrected_at
		FROM document_classifications c JOIN document_ocr_runs o ON o.job_id=c.ocr_job_id
		WHERE c.organization_id=$1 AND c.document_id=$2 AND o.published_at IS NOT NULL
		AND o.superseded_at IS NULL AND o.deleted_at IS NULL`, org, doc).Scan(&data, &corrected, &by, &at)
	if errors.Is(err, pgx.ErrNoRows) {
		return classification.Record{}, tx.Commit(ctx)
	}
	if err != nil {
		return classification.Record{}, err
	}
	var record classification.Record
	if err = json.Unmarshal(data, &record.Result); err != nil {
		return classification.Record{}, err
	}
	record.EffectiveType, record.CorrectedBy, record.CorrectedAt = record.DocumentType, by, at
	if corrected != "" {
		record.EffectiveType = corrected
	}
	return record, tx.Commit(ctx)
}

func (s *Store) CorrectDocumentType(ctx context.Context, user, org, doc, ocrJob, expected, kind string, now time.Time) (classification.Record, error) {
	if !classification.Valid(kind) || !classification.Valid(expected) {
		return classification.Record{}, tenant.ErrForbidden
	}
	tx, err := s.organizationTx(ctx, user, org)
	if err != nil {
		return classification.Record{}, err
	}
	defer tx.Rollback(ctx)
	role, err := currentRole(ctx, tx, user, org)
	if err != nil || role != tenant.Owner && role != tenant.Admin {
		return classification.Record{}, tenant.ErrForbidden
	}
	var data []byte
	err = tx.QueryRow(ctx, `UPDATE document_classifications c SET corrected_type=$5,corrected_by=$1,corrected_at=$6
		FROM document_ocr_runs o JOIN documents d ON d.id=o.document_id AND d.organization_id=o.organization_id
		WHERE c.ocr_job_id=o.job_id AND c.organization_id=$2 AND c.document_id=$3 AND c.ocr_job_id=$4
		AND o.published_at IS NOT NULL AND o.superseded_at IS NULL AND o.deleted_at IS NULL
		AND d.status IN ('Available','Archived') AND coalesce(c.corrected_type,c.document_type)=$7
		AND NOT EXISTS (SELECT 1 FROM document_extraction_reviews e WHERE e.organization_id=$2 AND e.document_id=$3 AND e.superseded_at IS NULL)
		RETURNING c.result`, user, org, doc, ocrJob, kind, now, expected).Scan(&data)
	if errors.Is(err, pgx.ErrNoRows) {
		return classification.Record{}, extraction.ErrConflict
	}
	if err != nil {
		return classification.Record{}, err
	}
	_, err = tx.Exec(ctx, `INSERT INTO audit_events(organization_id,actor_user_id,event_type,target_type,target_id,outcome,occurred_at,metadata)
		VALUES($1,$2,'classification.correct','document',$3,'success',$4,jsonb_build_object('ocr_job_id',$5,'corrected_type',$6))`, org, user, doc, now, ocrJob, kind)
	if err != nil {
		return classification.Record{}, err
	}
	var record classification.Record
	if err = json.Unmarshal(data, &record.Result); err != nil {
		return classification.Record{}, err
	}
	record.EffectiveType, record.CorrectedBy, record.CorrectedAt = kind, user, &now
	return record, tx.Commit(ctx)
}
