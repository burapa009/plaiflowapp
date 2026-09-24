package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"plaiflow/api/internal/job"
	"plaiflow/api/internal/tenant"
)

type documentSnapshotRow struct {
	documentID, reviewID, approvalID string
	updatedAt                        time.Time
}

func validExportDates(from, to string) bool {
	if from != "" {
		if _, err := time.Parse(time.DateOnly, from); err != nil {
			return false
		}
	}
	if to != "" {
		if _, err := time.Parse(time.DateOnly, to); err != nil {
			return false
		}
	}
	return from == "" || to == "" || from <= to
}

func documentExportDateFilter(column string, start int) string {
	from, to := "$"+strconv.Itoa(start), "$"+strconv.Itoa(start+1)
	return ` AND (` + from + `='' OR (` + column + ` AT TIME ZONE z.timezone)::date>=nullif(` + from + `,'')::date)
		AND (` + to + `='' OR (` + column + ` AT TIME ZONE z.timezone)::date<=nullif(` + to + `,'')::date)`
}

func (s *Store) CountDocumentExport(ctx context.Context, user, org, product, status, dateFrom, dateTo string) (int64, error) {
	if status != "Available" && status != "Archived" && status != "Trash" || status == "Trash" && product != "raw_documents" || !validExportDates(dateFrom, dateTo) {
		return 0, errors.New("invalid document export status")
	}
	var query, dateColumn string
	switch product {
	case "raw_documents":
		dateColumn = "d.accepted_at"
		query = `SELECT count(*) FROM documents d JOIN organizations z ON z.id=d.organization_id WHERE d.organization_id=$1 AND d.status=$2`
	case "confirmed_values":
		dateColumn = "e.confirmed_at"
		query = `SELECT count(*) FROM documents d JOIN organizations z ON z.id=d.organization_id
			JOIN document_extraction_reviews e ON e.organization_id=d.organization_id
			AND e.document_id=d.id AND e.superseded_at IS NULL JOIN document_ocr_runs o ON o.job_id=e.ocr_job_id
			AND o.organization_id=d.organization_id AND o.document_id=d.id
			WHERE d.organization_id=$1 AND d.status=$2 AND o.published_at IS NOT NULL AND o.superseded_at IS NULL AND o.deleted_at IS NULL
			AND NOT EXISTS (SELECT 1 FROM document_review_drafts rd WHERE rd.organization_id=e.organization_id
				AND rd.document_id=e.document_id AND rd.revision>e.draft_revision)`
	case "approved_suggestions":
		dateColumn = "a.approved_at"
		query = `SELECT count(*) FROM documents d JOIN organizations z ON z.id=d.organization_id
			JOIN document_extraction_reviews e ON e.organization_id=d.organization_id
			AND e.document_id=d.id AND e.superseded_at IS NULL JOIN accounting_suggestions a ON a.organization_id=d.organization_id
			AND a.document_id=d.id AND a.review_id=e.id AND a.superseded_at IS NULL
			JOIN document_ocr_runs o ON o.job_id=e.ocr_job_id AND o.organization_id=d.organization_id AND o.document_id=d.id
			WHERE d.organization_id=$1 AND d.status=$2 AND o.published_at IS NOT NULL AND o.superseded_at IS NULL AND o.deleted_at IS NULL
			AND NOT EXISTS (SELECT 1 FROM document_review_drafts rd WHERE rd.organization_id=e.organization_id
				AND rd.document_id=e.document_id AND rd.revision>e.draft_revision)`
	default:
		return 0, errors.New("invalid document export product")
	}
	query += documentExportDateFilter(dateColumn, 3)
	tx, err := s.organizationTx(ctx, user, org)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx)
	role, err := currentRole(ctx, tx, user, org)
	if err != nil || role != tenant.Owner && role != tenant.Admin {
		return 0, tenant.ErrForbidden
	}
	var count int64
	if err = tx.QueryRow(ctx, query, org, status, dateFrom, dateTo).Scan(&count); err != nil {
		return 0, err
	}
	return count, tx.Commit(ctx)
}

func (s *Store) QueueDocumentExport(ctx context.Context, request job.DocumentExportRequest) (job.DocumentExportState, error) {
	if !documentUUID.MatchString(request.ID) || request.Format != "csv" && request.Format != "xlsx" ||
		request.Status != "Available" && request.Status != "Archived" && request.Status != "Trash" ||
		request.Status == "Trash" && request.Product != "raw_documents" || !validExportDates(request.DateFrom, request.DateTo) {
		return job.DocumentExportState{}, errors.New("invalid document export")
	}
	capRows := 5000
	var query, dateColumn, order string
	switch request.Product {
	case "raw_documents":
		capRows = 20000
		dateColumn, order = "d.accepted_at", ` ORDER BY d.accepted_at,d.id LIMIT $3`
		query = `SELECT d.id,d.updated_at,null::uuid,null::uuid FROM documents d
			JOIN organizations z ON z.id=d.organization_id WHERE d.organization_id=$1 AND d.status=$2`
	case "confirmed_values":
		dateColumn, order = "e.confirmed_at", ` ORDER BY e.confirmed_at,e.id LIMIT $3`
		query = `SELECT d.id,d.updated_at,e.id,null::uuid FROM documents d JOIN organizations z ON z.id=d.organization_id
			JOIN document_extraction_reviews e ON e.organization_id=d.organization_id AND e.document_id=d.id AND e.superseded_at IS NULL
			JOIN document_ocr_runs o ON o.job_id=e.ocr_job_id AND o.organization_id=d.organization_id AND o.document_id=d.id
			WHERE d.organization_id=$1 AND d.status=$2 AND o.published_at IS NOT NULL AND o.superseded_at IS NULL AND o.deleted_at IS NULL
			AND NOT EXISTS (SELECT 1 FROM document_review_drafts rd WHERE rd.organization_id=e.organization_id
				AND rd.document_id=e.document_id AND rd.revision>e.draft_revision)`
	case "approved_suggestions":
		dateColumn, order = "a.approved_at", ` ORDER BY a.approved_at,a.id LIMIT $3`
		query = `SELECT d.id,d.updated_at,e.id,a.id FROM documents d JOIN organizations z ON z.id=d.organization_id
			JOIN document_extraction_reviews e ON e.organization_id=d.organization_id AND e.document_id=d.id AND e.superseded_at IS NULL
			JOIN accounting_suggestions a ON a.organization_id=d.organization_id AND a.document_id=d.id
				AND a.review_id=e.id AND a.superseded_at IS NULL
			JOIN document_ocr_runs o ON o.job_id=e.ocr_job_id AND o.organization_id=d.organization_id AND o.document_id=d.id
			WHERE d.organization_id=$1 AND d.status=$2 AND o.published_at IS NOT NULL AND o.superseded_at IS NULL AND o.deleted_at IS NULL
			AND NOT EXISTS (SELECT 1 FROM document_review_drafts rd WHERE rd.organization_id=e.organization_id
				AND rd.document_id=e.document_id AND rd.revision>e.draft_revision)`
	default:
		return job.DocumentExportState{}, errors.New("invalid document export product")
	}
	query += documentExportDateFilter(dateColumn, 4) + order
	tx, err := s.organizationTx(ctx, request.RequesterUserID, request.OrganizationID)
	if err != nil {
		return job.DocumentExportState{}, err
	}
	defer tx.Rollback(ctx)
	role, err := currentRole(ctx, tx, request.RequesterUserID, request.OrganizationID)
	if err != nil || role != tenant.Owner && role != tenant.Admin {
		return job.DocumentExportState{}, tenant.ErrForbidden
	}
	rows, err := tx.Query(ctx, query, request.OrganizationID, request.Status, capRows+1, request.DateFrom, request.DateTo)
	if err != nil {
		return job.DocumentExportState{}, err
	}
	items := make([]documentSnapshotRow, 0, 128)
	for rows.Next() {
		var item documentSnapshotRow
		var reviewID, approvalID *string
		if err = rows.Scan(&item.documentID, &item.updatedAt, &reviewID, &approvalID); err != nil {
			rows.Close()
			return job.DocumentExportState{}, err
		}
		if reviewID != nil {
			item.reviewID = *reviewID
		}
		if approvalID != nil {
			item.approvalID = *approvalID
		}
		items = append(items, item)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return job.DocumentExportState{}, err
	}
	if len(items) > capRows {
		return job.DocumentExportState{}, errors.New("document export exceeds hard row limit")
	}
	payload, err := json.Marshal(map[string]any{"export_type": request.Product, "format": request.Format,
		"format_version": map[string]string{"raw_documents": "unverified_documents_v1", "confirmed_values": "structured_documents_v1",
			"approved_suggestions": "accounting_suggestions_v1"}[request.Product], "row_count": len(items),
		"date_from": request.DateFrom, "date_to": request.DateTo})
	if err != nil {
		return job.DocumentExportState{}, err
	}
	_, err = tx.Exec(ctx, `INSERT INTO durable_jobs(id,organization_id,requester_user_id,kind,status,payload,
		idempotency_key,request_id,available_at,created_at) VALUES($1,$2,$3,'export','Queued',$4,$1,$5,$6,$6)`,
		request.ID, request.OrganizationID, request.RequesterUserID, payload, request.RequestID, request.Now)
	if err != nil {
		return job.DocumentExportState{}, err
	}
	_, err = tx.Exec(ctx, `INSERT INTO document_export_snapshots(job_id,organization_id,requester_user_id,
		product,document_status,format,date_from,date_to,row_count,created_at)
		VALUES($1,$2,$3,$4,$5,$6,nullif($7,'')::date,nullif($8,'')::date,$9,$10)`,
		request.ID, request.OrganizationID, request.RequesterUserID, request.Product, request.Status, request.Format,
		request.DateFrom, request.DateTo, len(items), request.Now)
	if err != nil {
		return job.DocumentExportState{}, err
	}
	if len(items) > 0 {
		for start := 0; start < len(items); start += 500 {
			end := start + 500
			if end > len(items) {
				end = len(items)
			}
			batch := &pgx.Batch{}
			for i := start; i < end; i++ {
				item := items[i]
				batch.Queue(`INSERT INTO document_export_snapshot_rows(job_id,organization_id,ordinal,document_id,
					document_updated_at,review_id,approval_id) VALUES($1,$2,$3,$4,$5,$6,$7)`,
					request.ID, request.OrganizationID, i+1, item.documentID, item.updatedAt, nullUUID(item.reviewID), nullUUID(item.approvalID))
			}
			result := tx.SendBatch(ctx, batch)
			for i := start; i < end; i++ {
				if _, err = result.Exec(); err != nil {
					break
				}
			}
			closeErr := result.Close()
			if err != nil {
				return job.DocumentExportState{}, err
			}
			if closeErr != nil {
				return job.DocumentExportState{}, closeErr
			}
		}
	}
	_, err = tx.Exec(ctx, `INSERT INTO audit_events(organization_id,actor_user_id,event_type,target_type,target_id,
		request_id,outcome,occurred_at,metadata) VALUES($1,$2,'review.export.request','durable_job',$3,$4,'queued',$5,
		jsonb_build_object('product',$6,'status',$7,'format',$8,'row_count',$9,'date_from',$10,'date_to',$11))`, request.OrganizationID,
		request.RequesterUserID, request.ID, request.RequestID, request.Now, request.Product, request.Status, request.Format,
		len(items), request.DateFrom, request.DateTo)
	if err != nil {
		return job.DocumentExportState{}, err
	}
	state := job.DocumentExportState{ID: request.ID, Product: request.Product, Status: job.Queued, RowCount: int64(len(items)), CreatedAt: request.Now}
	return state, tx.Commit(ctx)
}

func (s *Store) GetDocumentExport(ctx context.Context, user, org, id string) (job.DocumentExportState, error) {
	tx, err := s.organizationTx(ctx, user, org)
	if err != nil {
		return job.DocumentExportState{}, err
	}
	defer tx.Rollback(ctx)
	role, err := currentRole(ctx, tx, user, org)
	if err != nil || role != tenant.Owner && role != tenant.Admin {
		return job.DocumentExportState{}, tenant.ErrForbidden
	}
	var state job.DocumentExportState
	err = tx.QueryRow(ctx, `SELECT j.id,s.product,
		CASE WHEN j.status='Completed' AND a.expires_at<=now() THEN 'Expired'
			WHEN j.status='Completed' AND a.expires_at>now() AND a.deleted_at IS NULL THEN 'Ready'
			ELSE j.status END,s.row_count,coalesce(j.failure_code,''),j.created_at
		FROM document_export_snapshots s JOIN durable_jobs j ON j.id=s.job_id
		LEFT JOIN durable_job_artifacts a ON a.job_id=j.id
		WHERE s.organization_id=$1 AND s.job_id=$2`, org, id).Scan(
		&state.ID, &state.Product, &state.Status, &state.RowCount, &state.FailureCode, &state.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return job.DocumentExportState{}, tenant.ErrNotFound
	}
	if err != nil {
		return job.DocumentExportState{}, err
	}
	return state, tx.Commit(ctx)
}

func (s *Store) readDocumentExportPage(ctx context.Context, jobID, org, product, cursor string, limit int) (job.ExportPage, error) {
	last := 0
	if cursor != "" {
		var err error
		last, err = strconv.Atoi(cursor)
		if err != nil || last < 0 || last > 20000 {
			return job.ExportPage{}, errors.New("invalid document export cursor")
		}
	}
	page := job.ExportPage{Product: product, Entries: make([]job.DocumentExportEntry, 0, limit), Done: true}
	var query string
	switch product {
	case "raw_documents":
		query = `SELECT sr.ordinal,d.id,d.display_filename,d.detected_mime,d.byte_size,d.status,
			coalesce((SELECT ds.channel FROM document_sources ds WHERE ds.organization_id=d.organization_id AND ds.document_id=d.id
				ORDER BY ds.created_at,ds.id LIMIT 1),''),
			(SELECT count(*) FROM document_sources ds WHERE ds.organization_id=d.organization_id AND ds.document_id=d.id),
			coalesce(submitter.display_name,''),coalesce(assignee.display_name,''),d.accepted_at,d.updated_at` +
			` FROM document_export_snapshot_rows sr JOIN document_export_snapshots snap ON snap.job_id=sr.job_id AND snap.organization_id=sr.organization_id
			JOIN documents d ON d.organization_id=sr.organization_id AND d.id=sr.document_id AND d.updated_at=sr.document_updated_at AND d.status=snap.document_status
			LEFT JOIN users submitter ON submitter.id=d.submitted_by_user_id LEFT JOIN users assignee ON assignee.id=d.assignee_user_id
			WHERE sr.job_id=$1 AND sr.organization_id=$2 AND sr.ordinal>$3 ORDER BY sr.ordinal LIMIT $4`
	case "confirmed_values":
		query = `SELECT sr.ordinal,e.id,e.organization_id,e.document_id,e.ocr_job_id,e.revision,e.object_key,e.confirmed_by,e.confirmed_at
			FROM document_export_snapshot_rows sr JOIN document_export_snapshots snap ON snap.job_id=sr.job_id AND snap.organization_id=sr.organization_id
			JOIN documents d ON d.organization_id=sr.organization_id AND d.id=sr.document_id AND d.updated_at=sr.document_updated_at AND d.status=snap.document_status
			JOIN document_extraction_reviews e ON e.organization_id=sr.organization_id AND e.id=sr.review_id
				AND e.document_id=sr.document_id AND e.superseded_at IS NULL
			JOIN document_ocr_runs o ON o.job_id=e.ocr_job_id AND o.organization_id=e.organization_id AND o.document_id=e.document_id
				AND o.published_at IS NOT NULL AND o.superseded_at IS NULL AND o.deleted_at IS NULL
			WHERE sr.job_id=$1 AND sr.organization_id=$2 AND sr.ordinal>$3
			AND NOT EXISTS (SELECT 1 FROM document_review_drafts rd WHERE rd.organization_id=e.organization_id
				AND rd.document_id=e.document_id AND rd.revision>e.draft_revision)
			ORDER BY sr.ordinal LIMIT $4`
	case "approved_suggestions":
		query = `SELECT sr.ordinal,e.id,e.organization_id,e.document_id,e.ocr_job_id,e.revision,e.object_key,e.confirmed_by,e.confirmed_at,
			a.id,a.revision,a.review_revision,a.category_id,a.category_name,coalesce(a.vendor_id::text,''),a.vendor_contact_code,
			a.suggestion_basis,coalesce(a.rule_version,0),a.approved_at
			FROM document_export_snapshot_rows sr JOIN document_export_snapshots snap ON snap.job_id=sr.job_id AND snap.organization_id=sr.organization_id
			JOIN documents d ON d.organization_id=sr.organization_id AND d.id=sr.document_id AND d.updated_at=sr.document_updated_at AND d.status=snap.document_status
			JOIN document_extraction_reviews e ON e.organization_id=sr.organization_id AND e.id=sr.review_id AND e.document_id=sr.document_id AND e.superseded_at IS NULL
			JOIN accounting_suggestions a ON a.organization_id=sr.organization_id AND a.id=sr.approval_id AND a.document_id=sr.document_id
				AND a.review_id=e.id AND a.superseded_at IS NULL
			JOIN document_ocr_runs o ON o.job_id=e.ocr_job_id AND o.organization_id=e.organization_id AND o.document_id=e.document_id
				AND o.published_at IS NOT NULL AND o.superseded_at IS NULL AND o.deleted_at IS NULL
			WHERE sr.job_id=$1 AND sr.organization_id=$2 AND sr.ordinal>$3
			AND NOT EXISTS (SELECT 1 FROM document_review_drafts rd WHERE rd.organization_id=e.organization_id
				AND rd.document_id=e.document_id AND rd.revision>e.draft_revision)
			ORDER BY sr.ordinal LIMIT $4`
	default:
		return job.ExportPage{}, errors.New("invalid export product")
	}
	rows, err := s.pool.Query(ctx, query, jobID, org, last, limit+1)
	if err != nil {
		return job.ExportPage{}, err
	}
	defer rows.Close()
	cursorOrdinal := last
	for rows.Next() {
		var ordinal int
		var entry job.DocumentExportEntry
		switch product {
		case "raw_documents":
			err = rows.Scan(&ordinal, &entry.Document.ID, &entry.Document.Filename, &entry.Document.MIME,
				&entry.Document.Size, &entry.Document.Status, &entry.Document.SourceChannel, &entry.Document.SourceCount,
				&entry.Document.SubmitterName, &entry.Document.AssigneeName, &entry.Document.AcceptedAt, &entry.Document.UpdatedAt)
		case "confirmed_values":
			err = rows.Scan(&ordinal, &entry.Review.ID, &entry.Review.OrganizationID, &entry.Review.DocumentID,
				&entry.Review.OCRJobID, &entry.Review.Revision, &entry.Review.ObjectKey, &entry.Review.ConfirmedBy, &entry.Review.ConfirmedAt)
		case "approved_suggestions":
			err = rows.Scan(&ordinal, &entry.Review.ID, &entry.Review.OrganizationID, &entry.Review.DocumentID,
				&entry.Review.OCRJobID, &entry.Review.Revision, &entry.Review.ObjectKey, &entry.Review.ConfirmedBy, &entry.Review.ConfirmedAt,
				&entry.Approval.ID, &entry.Approval.Revision, &entry.Approval.ReviewRevision, &entry.Approval.CategoryID,
				&entry.Approval.CategoryName, &entry.Approval.VendorID, &entry.Approval.ContactCode, &entry.Approval.Basis,
				&entry.Approval.RuleVersion, &entry.Approval.ApprovedAt)
			entry.Approval.DocumentID, entry.Approval.ReviewID = entry.Review.DocumentID, entry.Review.ID
		}
		if err != nil {
			return job.ExportPage{}, err
		}
		page.Entries = append(page.Entries, entry)
		last = ordinal
		if len(page.Entries) <= limit {
			cursorOrdinal = ordinal
		}
	}
	if err = rows.Err(); err != nil {
		return job.ExportPage{}, err
	}
	if len(page.Entries) > limit {
		page.Entries = page.Entries[:limit]
		page.Done = false
		page.NextCursor = strconv.Itoa(cursorOrdinal)
	}
	return page, nil
}

type queryRower interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

func validDocumentExportSnapshot(ctx context.Context, db queryRower, jobID, org string) (bool, error) {
	var expected, found, valid int64
	err := db.QueryRow(ctx, `SELECT snap.row_count,count(sr.ordinal),
		count(sr.ordinal) FILTER (WHERE d.id IS NOT NULL AND
			(snap.product='raw_documents' OR e.id IS NOT NULL AND o.job_id IS NOT NULL
				AND (snap.product='confirmed_values' OR a.id IS NOT NULL)))
		FROM document_export_snapshots snap
		LEFT JOIN document_export_snapshot_rows sr ON sr.job_id=snap.job_id AND sr.organization_id=snap.organization_id
		LEFT JOIN documents d ON d.organization_id=sr.organization_id AND d.id=sr.document_id
			AND d.updated_at=sr.document_updated_at AND d.status=snap.document_status
		LEFT JOIN document_extraction_reviews e ON e.organization_id=sr.organization_id AND e.id=sr.review_id
			AND e.document_id=sr.document_id AND e.superseded_at IS NULL
			AND NOT EXISTS (SELECT 1 FROM document_review_drafts rd WHERE rd.organization_id=e.organization_id
				AND rd.document_id=e.document_id AND rd.revision>e.draft_revision)
		LEFT JOIN document_ocr_runs o ON o.job_id=e.ocr_job_id AND o.organization_id=e.organization_id
			AND o.document_id=e.document_id AND o.published_at IS NOT NULL AND o.superseded_at IS NULL AND o.deleted_at IS NULL
		LEFT JOIN accounting_suggestions a ON a.organization_id=sr.organization_id AND a.id=sr.approval_id
			AND a.document_id=sr.document_id AND a.review_id=e.id AND a.review_revision=e.revision AND a.superseded_at IS NULL
		WHERE snap.job_id=$1 AND snap.organization_id=$2 GROUP BY snap.row_count,snap.product`, jobID, org).Scan(&expected, &found, &valid)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	return expected == found && found == valid, err
}
