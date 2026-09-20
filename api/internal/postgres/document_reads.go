package postgres

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"regexp"
	"time"

	"github.com/jackc/pgx/v5"

	"plaiflow/api/internal/document"
	"plaiflow/api/internal/tenant"
)

const documentExportScope = ` FROM documents d
    LEFT JOIN users submitter ON submitter.id=d.submitted_by_user_id
    LEFT JOIN users assignee ON assignee.id=d.assignee_user_id
    WHERE d.organization_id=$1 AND d.status<>'Purged'
      AND ($2<>'' OR d.status<>'Trash') AND ($2='' OR d.status=$2)
      AND ($3='' OR EXISTS (SELECT 1 FROM document_sources ds2 WHERE ds2.organization_id=d.organization_id AND ds2.document_id=d.id AND ds2.channel=$3))
      AND ($4='' OR d.display_filename ILIKE '%'||$4||'%')
      AND ($5='' OR coalesce(d.submitted_by_user_id::text,'')=$5)
      AND ($6='' OR coalesce(d.assignee_user_id::text,'')=$6)
      AND ($7::timestamptz IS NULL OR d.accepted_at >= $7)
      AND ($8::timestamptz IS NULL OR d.accepted_at < $8)`

func (s *Store) CountDocumentsForExport(ctx context.Context, userID, organizationID string, filter document.Filter) (int64, error) {
	tx, err := s.organizationTx(ctx, userID, organizationID)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx)
	role, err := currentRole(ctx, tx, userID, organizationID)
	if err != nil || (role != tenant.Owner && role != tenant.Admin) {
		return 0, tenant.ErrForbidden
	}
	var count int64
	err = tx.QueryRow(ctx, `SELECT count(*)`+documentExportScope, organizationID, filter.Status, filter.Channel, filter.Filename,
		filter.Submitter, filter.Assignee, filter.From, filter.To).Scan(&count)
	if err != nil {
		return 0, err
	}
	if err := auditTenant(ctx, tx, organizationID, userID, "document.export.preview", "organization", organizationID, time.Now().UTC()); err != nil {
		return 0, err
	}
	return count, tx.Commit(ctx)
}

func (s *Store) ExportDocuments(ctx context.Context, userID, organizationID string, filter document.Filter, format string, output io.Writer) (int64, error) {
	if format != "csv" && format != "xlsx" {
		return 0, errors.New("unsupported document export format")
	}
	tx, err := s.organizationTx(ctx, userID, organizationID)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx)
	role, err := currentRole(ctx, tx, userID, organizationID)
	if err != nil || (role != tenant.Owner && role != tenant.Admin) {
		return 0, tenant.ErrForbidden
	}
	rows, err := tx.Query(ctx, `SELECT d.id,d.display_filename,d.detected_mime,d.byte_size,d.status,
	    coalesce((SELECT ds.channel FROM document_sources ds WHERE ds.organization_id=d.organization_id AND ds.document_id=d.id ORDER BY ds.created_at,ds.id LIMIT 1),''),
	    (SELECT count(*) FROM document_sources ds WHERE ds.organization_id=d.organization_id AND ds.document_id=d.id),
	    coalesce(submitter.display_name,''),coalesce(assignee.display_name,''),d.accepted_at,d.updated_at
	    `+documentExportScope+` ORDER BY d.accepted_at DESC,d.id DESC`, organizationID, filter.Status, filter.Channel, filter.Filename, filter.Submitter, filter.Assignee, filter.From, filter.To)
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	count := int64(0)
	var values []document.ExportRow
	values = make([]document.ExportRow, 0, 128)
	for rows.Next() {
		var row document.ExportRow
		if err := rows.Scan(&row.ID, &row.Filename, &row.MIME, &row.Size, &row.Status, &row.SourceChannel, &row.SourceCount, &row.SubmitterName, &row.AssigneeName, &row.AcceptedAt, &row.UpdatedAt); err != nil {
			return count, err
		}
		count++
		if count > 5000 {
			return count, errors.New("document export requires asynchronous processing")
		}
		values = append(values, row)
	}
	if err := rows.Err(); err != nil {
		return count, err
	}
	if err := document.WriteExport(output, format, values); err != nil {
		return count, err
	}
	if err := auditTenant(ctx, tx, organizationID, userID, "document.export", "organization", organizationID, time.Now().UTC()); err != nil {
		return count, err
	}
	return count, tx.Commit(ctx)
}

type documentCursor struct {
	At time.Time `json:"at"`
	ID string    `json:"id"`
}

func (s *Store) DocumentSummary(ctx context.Context, userID, organizationID string) (document.Summary, error) {
	tx, err := s.organizationTx(ctx, userID, organizationID)
	if err != nil {
		return document.Summary{}, err
	}
	defer tx.Rollback(ctx)
	if _, err := membershipFor(ctx, tx, userID, organizationID); err != nil {
		return document.Summary{}, tenant.ErrNotFound
	}
	now := time.Now().UTC()
	var timezone string
	if err := tx.QueryRow(ctx, `SELECT timezone FROM organizations WHERE id=$1`, organizationID).Scan(&timezone); err != nil {
		return document.Summary{}, err
	}
	periodStart, periodEnd, err := currentDocumentPeriod(ctx, tx, organizationID, timezone, now)
	if err != nil {
		return document.Summary{}, err
	}
	limit, trialStart, trialEnd, err := documentLimit(ctx, tx, organizationID, now)
	if err != nil {
		return document.Summary{}, err
	}
	usedFrom, usedTo := periodStart, periodEnd
	if trialEnd.After(trialStart) {
		usedFrom, usedTo = trialStart, trialEnd
	}
	var summary document.Summary
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM document_usage_charges c JOIN documents d ON d.organization_id=c.organization_id AND d.id=c.document_id
	    WHERE c.organization_id=$1 AND c.accepted_at >= $3 AND c.accepted_at < $4
	      AND (organization_role($2::uuid,$1::uuid) IN ('Owner','Admin') OR (d.submitted_by_user_id=$2 AND NOT d.group_restricted) OR d.assignee_user_id=$2
	        OR EXISTS (SELECT 1 FROM document_sources ds WHERE ds.organization_id=d.organization_id AND ds.document_id=d.id AND ds.submitted_by_user_id=$2 AND NOT ds.group_source))`, organizationID, userID, usedFrom, usedTo).Scan(&summary.Used); err != nil {
		return document.Summary{}, err
	}
	if err := tx.QueryRow(ctx, `SELECT count(*) FILTER (WHERE d.status='Available'),count(*) FILTER (WHERE d.status='Archived'),count(*) FILTER (WHERE d.status='Trash')
	    FROM documents d WHERE d.organization_id=$1
	      AND (organization_role($2::uuid,$1::uuid) IN ('Owner','Admin') OR (d.submitted_by_user_id=$2 AND NOT d.group_restricted) OR d.assignee_user_id=$2
	        OR EXISTS (SELECT 1 FROM document_sources ds WHERE ds.organization_id=d.organization_id AND ds.document_id=d.id AND ds.submitted_by_user_id=$2 AND NOT ds.group_source))`, organizationID, userID).Scan(&summary.Available, &summary.Archived, &summary.Trash); err != nil {
		return document.Summary{}, err
	}
	if err := tx.QueryRow(ctx, `SELECT count(*) FILTER (WHERE a.status='Checking'),count(*) FILTER (WHERE a.status='Rejected')
	    FROM document_intake_attempts a WHERE a.organization_id=$1 AND a.created_at >= now()-interval '90 days'
	      AND (organization_role($2::uuid,$1::uuid) IN ('Owner','Admin') OR a.actor_user_id=$2)`, organizationID, userID).Scan(&summary.Checking, &summary.Rejected); err != nil {
		return document.Summary{}, err
	}
	summary.Limit, summary.ResetAt = int64(limit), usedTo
	if summary.Limit > 0 && summary.Used*100 >= summary.Limit*90 {
		summary.Warning = "90"
	} else if summary.Limit > 0 && summary.Used*100 >= summary.Limit*80 {
		summary.Warning = "80"
	}
	return summary, tx.Commit(ctx)
}

var documentUUID = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

func decodeDocumentCursor(value string) (documentCursor, error) {
	if value == "" {
		return documentCursor{}, nil
	}
	if len(value) > 256 {
		return documentCursor{}, document.ErrInvalidCursor
	}
	data, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return documentCursor{}, document.ErrInvalidCursor
	}
	var cursor documentCursor
	if json.Unmarshal(data, &cursor) != nil || !documentUUID.MatchString(cursor.ID) || cursor.At.IsZero() {
		return documentCursor{}, document.ErrInvalidCursor
	}
	return cursor, nil
}

func encodeDocumentCursor(doc document.Document) string {
	data, _ := json.Marshal(documentCursor{At: doc.AcceptedAt, ID: doc.ID})
	return base64.RawURLEncoding.EncodeToString(data)
}

func (s *Store) ListDocuments(ctx context.Context, userID, organizationID, cursorValue string, limit int) (document.Page, error) {
	return s.ListDocumentsFiltered(ctx, userID, organizationID, cursorValue, limit, document.Filter{})
}

func (s *Store) ListDocumentsFiltered(ctx context.Context, userID, organizationID, cursorValue string, limit int, filter document.Filter) (document.Page, error) {
	if limit < 1 || limit > 50 {
		return document.Page{}, document.ErrInvalidCursor
	}
	cursor, err := decodeDocumentCursor(cursorValue)
	if err != nil {
		return document.Page{}, err
	}
	tx, err := s.organizationTx(ctx, userID, organizationID)
	if err != nil {
		return document.Page{}, err
	}
	defer tx.Rollback(ctx)
	if _, err := membershipFor(ctx, tx, userID, organizationID); err != nil {
		return document.Page{}, tenant.ErrNotFound
	}
	var cursorAt, cursorID any
	if cursorValue != "" {
		cursorAt = cursor.At
		cursorID = cursor.ID
	}
	rows, err := tx.Query(ctx, `SELECT d.id,d.organization_id,d.display_filename,d.detected_mime,d.byte_size,d.status,d.storage_key,d.accepted_at,
	    coalesce((SELECT ds0.channel FROM document_sources ds0 WHERE ds0.organization_id=d.organization_id AND ds0.document_id=d.id
	        AND (organization_role($2::uuid,$1::uuid) IN ('Owner','Admin') OR (ds0.submitted_by_user_id=$2::uuid AND NOT ds0.group_source))
	        ORDER BY ds0.created_at,ds0.id LIMIT 1),''),
	    (SELECT count(*) FROM document_sources ds WHERE ds.organization_id=d.organization_id AND ds.document_id=d.id
	        AND (organization_role($2::uuid,$1::uuid) IN ('Owner','Admin') OR (ds.submitted_by_user_id=$2::uuid AND NOT ds.group_source)))
	    FROM documents d WHERE d.organization_id=$1 AND d.status<>'Purged'
	    AND (organization_role($2::uuid,$1::uuid) IN ('Owner','Admin') OR (d.submitted_by_user_id=$2::uuid AND NOT d.group_restricted)
	        OR d.assignee_user_id=$2::uuid OR EXISTS (SELECT 1 FROM document_sources ds
	            WHERE ds.organization_id=d.organization_id AND ds.document_id=d.id AND ds.submitted_by_user_id=$2::uuid AND NOT ds.group_source))
	    AND ($7<>'' OR d.status<>'Trash') AND ($7='' OR d.status=$7) AND ($8='' OR EXISTS (SELECT 1 FROM document_sources ds2 WHERE ds2.organization_id=d.organization_id AND ds2.document_id=d.id AND ds2.channel=$8))
	    AND ($9='' OR d.display_filename ILIKE '%'||$9||'%') AND ($10='' OR coalesce(d.submitted_by_user_id::text,'')=$10) AND ($11='' OR coalesce(d.assignee_user_id::text,'')=$11)
	    AND ($12::timestamptz IS NULL OR d.accepted_at >= $12) AND ($13::timestamptz IS NULL OR d.accepted_at < $13)
	    AND ($3::boolean OR (d.accepted_at,d.id)<($4::timestamptz,$5::uuid))
	    ORDER BY d.accepted_at DESC,d.id DESC LIMIT $6`, organizationID, userID, cursorValue == "", cursorAt, cursorID, limit+1,
		filter.Status, filter.Channel, filter.Filename, filter.Submitter, filter.Assignee, filter.From, filter.To)
	if err != nil {
		return document.Page{}, err
	}
	docs, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (document.Document, error) {
		var doc document.Document
		err := row.Scan(&doc.ID, &doc.OrganizationID, &doc.Filename, &doc.MIME, &doc.Size, &doc.Status, &doc.StorageKey, &doc.AcceptedAt, &doc.SourceChannel, &doc.SourceCount)
		return doc, err
	})
	if err != nil {
		return document.Page{}, err
	}
	page := document.Page{Documents: docs}
	if len(docs) > limit {
		page.NextCursor = encodeDocumentCursor(docs[limit-1])
		page.Documents = docs[:limit]
	}
	if page.Documents == nil {
		page.Documents = []document.Document{}
	}
	return page, tx.Commit(ctx)
}

func (s *Store) GetDocument(ctx context.Context, userID, organizationID, documentID string) (document.Document, error) {
	if !documentUUID.MatchString(documentID) {
		return document.Document{}, tenant.ErrNotFound
	}
	tx, err := s.organizationTx(ctx, userID, organizationID)
	if err != nil {
		return document.Document{}, err
	}
	defer tx.Rollback(ctx)
	if _, err := membershipFor(ctx, tx, userID, organizationID); err != nil {
		return document.Document{}, tenant.ErrNotFound
	}
	var doc document.Document
	err = tx.QueryRow(ctx, `SELECT d.id,d.organization_id,d.display_filename,d.detected_mime,d.byte_size,d.status,d.storage_key,d.accepted_at,
	    coalesce((SELECT ds.channel FROM document_sources ds WHERE ds.organization_id=d.organization_id AND ds.document_id=d.id
	        AND (organization_role($3::uuid,$1::uuid) IN ('Owner','Admin') OR (ds.submitted_by_user_id=$3::uuid AND NOT ds.group_source))
	        ORDER BY ds.created_at,ds.id LIMIT 1),'')
	    FROM documents d WHERE d.organization_id=$1 AND d.id=$2 AND d.status IN ('Available','Archived')
	    AND (organization_role($3::uuid,$1::uuid) IN ('Owner','Admin') OR (d.submitted_by_user_id=$3::uuid AND NOT d.group_restricted)
	        OR d.assignee_user_id=$3::uuid OR EXISTS (SELECT 1 FROM document_sources ds
	            WHERE ds.organization_id=d.organization_id AND ds.document_id=d.id AND ds.submitted_by_user_id=$3::uuid AND NOT ds.group_source))`, organizationID, documentID, userID).Scan(
		&doc.ID, &doc.OrganizationID, &doc.Filename, &doc.MIME, &doc.Size, &doc.Status, &doc.StorageKey, &doc.AcceptedAt, &doc.SourceChannel)
	if errors.Is(err, pgx.ErrNoRows) {
		return document.Document{}, tenant.ErrNotFound
	}
	if err != nil {
		return document.Document{}, err
	}
	if err := auditTenant(ctx, tx, organizationID, userID, "document.retrieve", "document", documentID, time.Now().UTC()); err != nil {
		return document.Document{}, err
	}
	return doc, tx.Commit(ctx)
}

func (s *Store) DocumentDetail(ctx context.Context, userID, organizationID, documentID string) (document.Detail, error) {
	if !documentUUID.MatchString(documentID) {
		return document.Detail{}, tenant.ErrNotFound
	}
	tx, err := s.organizationTx(ctx, userID, organizationID)
	if err != nil {
		return document.Detail{}, err
	}
	defer tx.Rollback(ctx)
	role, err := currentRole(ctx, tx, userID, organizationID)
	if err != nil {
		return document.Detail{}, tenant.ErrNotFound
	}
	manager := role == tenant.Owner || role == tenant.Admin
	var detail document.Detail
	err = tx.QueryRow(ctx, `SELECT d.id,d.organization_id,d.display_filename,d.detected_mime,d.byte_size,d.status,d.accepted_at
	    FROM documents d WHERE d.organization_id=$1 AND d.id=$2 AND d.status<>'Purged'
	      AND (d.status<>'Trash' OR $4::boolean)
	      AND ($4::boolean OR (d.submitted_by_user_id=$3 AND NOT d.group_restricted) OR d.assignee_user_id=$3
	        OR EXISTS (SELECT 1 FROM document_sources ds WHERE ds.organization_id=d.organization_id AND ds.document_id=d.id AND ds.submitted_by_user_id=$3 AND NOT ds.group_source))`,
		organizationID, documentID, userID, manager).Scan(&detail.Document.ID, &detail.Document.OrganizationID,
		&detail.Document.Filename, &detail.Document.MIME, &detail.Document.Size, &detail.Document.Status, &detail.Document.AcceptedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return document.Detail{}, tenant.ErrNotFound
	}
	if err != nil {
		return document.Detail{}, err
	}
	rows, err := tx.Query(ctx, `SELECT ds.id,ds.channel,coalesce(u.display_name,''),ds.created_at,ds.provider_filename,ds.provider_mime,ds.provider_size,ds.selected_at,
	    CASE WHEN $4::boolean THEN ds.drive_file_id ELSE NULL END,
	    CASE WHEN $4::boolean THEN ds.drive_revision ELSE NULL END
	    FROM document_sources ds LEFT JOIN users u ON u.id=ds.submitted_by_user_id
	    WHERE ds.organization_id=$1 AND ds.document_id=$2
	      AND ($4::boolean OR (ds.submitted_by_user_id=$3 AND NOT ds.group_source))
	    ORDER BY ds.created_at,ds.id`, organizationID, documentID, userID, manager)
	if err != nil {
		return document.Detail{}, err
	}
	detail.Sources, err = pgx.CollectRows(rows, func(row pgx.CollectableRow) (document.Source, error) {
		var source document.Source
		err := row.Scan(&source.ID, &source.Channel, &source.SubmittedBy, &source.AcceptedAt, &source.ProviderFilename,
			&source.ProviderMIME, &source.ProviderSize, &source.SelectedAt, &source.DriveFileID, &source.DriveRevision)
		return source, err
	})
	if err != nil {
		return document.Detail{}, err
	}
	if detail.Sources == nil {
		detail.Sources = []document.Source{}
	}
	return detail, tx.Commit(ctx)
}

func (s *Store) ChangeDocumentStatus(ctx context.Context, userID, organizationID, documentID, action string, now time.Time) (document.Document, error) {
	if !documentUUID.MatchString(documentID) || (action != "archive" && action != "trash" && action != "restore") {
		return document.Document{}, tenant.ErrNotFound
	}
	tx, err := s.organizationTx(ctx, userID, organizationID)
	if err != nil {
		return document.Document{}, err
	}
	defer tx.Rollback(ctx)
	role, err := currentRole(ctx, tx, userID, organizationID)
	if err != nil || (role != tenant.Owner && role != tenant.Admin) {
		return document.Document{}, tenant.ErrForbidden
	}
	var doc document.Document
	err = tx.QueryRow(ctx, `UPDATE documents SET
	    status=CASE $3 WHEN 'archive' THEN 'Archived' WHEN 'trash' THEN 'Trash' ELSE 'Available' END,
	    trashed_at=CASE WHEN $3='trash' THEN $4::timestamptz ELSE NULL END,
	    updated_at=$4
	    WHERE organization_id=$1 AND id=$2 AND
	      (($3='archive' AND status='Available') OR
	       ($3='trash' AND status IN ('Available','Archived')) OR
	       ($3='restore' AND status='Trash' AND trashed_at > $4::timestamptz-interval '30 days'))
	    RETURNING id,organization_id,display_filename,detected_mime,byte_size,status,storage_key,accepted_at`,
		organizationID, documentID, action, now).Scan(&doc.ID, &doc.OrganizationID, &doc.Filename, &doc.MIME, &doc.Size, &doc.Status, &doc.StorageKey, &doc.AcceptedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		var exists bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM documents WHERE organization_id=$1 AND id=$2)`, organizationID, documentID).Scan(&exists); err != nil {
			return document.Document{}, err
		}
		if !exists {
			return document.Document{}, tenant.ErrNotFound
		}
		return document.Document{}, document.ErrStatusConflict
	}
	if err != nil {
		return document.Document{}, err
	}
	if err := auditTenant(ctx, tx, organizationID, userID, "document."+action, "document", documentID, now); err != nil {
		return document.Document{}, err
	}
	return doc, tx.Commit(ctx)
}
