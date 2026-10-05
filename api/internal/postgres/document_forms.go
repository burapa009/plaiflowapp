package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/jackc/pgx/v5"
	"plaiflow/api/internal/extraction"
	"plaiflow/api/internal/tenant"
	"time"
)

func (s *Store) ListFormReferences(ctx context.Context, user, org string) ([]extraction.FormReference, error) {
	tx, err := s.organizationTx(ctx, user, org)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	rows, err := tx.Query(ctx, `SELECT f.document_id,f.data FROM document_forms f JOIN documents d ON d.organization_id=f.organization_id AND d.id=f.document_id
 JOIN document_ocr_runs o ON o.organization_id=d.organization_id AND o.document_id=d.id AND o.job_id=f.ocr_job_id AND o.superseded_at IS NULL AND o.deleted_at IS NULL
 WHERE f.organization_id=$1 AND f.status='Confirmed' AND d.status IN ('Available','Archived')
 AND (organization_role($2::uuid,$1::uuid) IN ('Owner','Admin') OR (d.submitted_by_user_id=$2::uuid AND NOT d.group_restricted) OR d.assignee_user_id=$2::uuid
 OR EXISTS (SELECT 1 FROM document_sources ds WHERE ds.organization_id=d.organization_id AND ds.document_id=d.id AND ds.submitted_by_user_id=$2::uuid AND NOT ds.group_source))
 ORDER BY f.updated_at DESC LIMIT 100`, org, user)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []extraction.FormReference{}
	for rows.Next() {
		var ref extraction.FormReference
		var raw []byte
		if err = rows.Scan(&ref.ID, &raw); err != nil {
			return nil, err
		}
		if err = json.Unmarshal(raw, &ref.Data); err != nil {
			return nil, err
		}
		allowed := map[string]string{}
		for _, key := range []string{"document_number", "issue_date", "total_amount", "paid_amount", "currency", "buyer_name", "buyer_tax_id"} {
			allowed[key] = ref.Data.Fields[key]
		}
		allowed["total_amount"] = referenceFormTotal(ref.Data)
		ref.Data.Tables = nil
		ref.Data.Fields = allowed
		out = append(out, ref)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	rows.Close()
	return out, tx.Commit(ctx)
}

func (s *Store) GetDocumentForm(ctx context.Context, user, org, doc string) (extraction.FormRecord, error) {
	if _, err := s.GetDocument(ctx, user, org, doc); err != nil {
		return extraction.FormRecord{}, err
	}
	tx, err := s.organizationTx(ctx, user, org)
	if err != nil {
		return extraction.FormRecord{}, err
	}
	defer tx.Rollback(ctx)
	var f extraction.FormRecord
	var data []byte
	err = tx.QueryRow(ctx, "SELECT data,revision,ocr_job_id,status,updated_at FROM document_forms WHERE organization_id=$1 AND document_id=$2", org, doc).Scan(&data, &f.Revision, &f.OCRJobID, &f.Status, &f.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return f, tx.Commit(ctx)
	}
	if err != nil {
		return f, err
	}
	if err = json.Unmarshal(data, &f.Data); err != nil {
		return f, err
	}
	return f, tx.Commit(ctx)
}

func (s *Store) SaveDocumentForm(ctx context.Context, user, org, doc, ocr string, data extraction.DocumentForm, expected, expectedLegacyDraft int, requestID string, now time.Time) (extraction.FormRecord, error) {
	if _, err := s.GetDocument(ctx, user, org, doc); err != nil {
		return extraction.FormRecord{}, err
	}
	tx, err := s.organizationTx(ctx, user, org)
	if err != nil {
		return extraction.FormRecord{}, err
	}
	defer tx.Rollback(ctx)
	role, err := currentRole(ctx, tx, user, org)
	if err != nil || !s.reviewEnabled && role != tenant.Owner && role != tenant.Admin {
		return extraction.FormRecord{}, tenant.ErrForbidden
	}
	// Serialize with OCR publication and confirmation using the existing document lock.
	var current string
	err = tx.QueryRow(ctx, "SELECT id FROM documents WHERE organization_id=$1 AND id=$2 AND status IN ('Available','Archived') FOR UPDATE", org, doc).Scan(&current)
	if err != nil {
		return extraction.FormRecord{}, tenant.ErrNotFound
	}
	err = tx.QueryRow(ctx, "SELECT job_id FROM document_ocr_runs WHERE organization_id=$1 AND document_id=$2 AND published_at IS NOT NULL AND superseded_at IS NULL AND deleted_at IS NULL", org, doc).Scan(&current)
	if err != nil || current != ocr {
		return extraction.FormRecord{}, extraction.ErrConflict
	}
	record, err := saveForm(ctx, tx, user, org, doc, ocr, data, expected, expectedLegacyDraft, "Draft", requestID, now)
	if err != nil {
		return record, err
	}
	return record, tx.Commit(ctx)
}

func saveForm(ctx context.Context, tx pgx.Tx, user, org, doc, ocr string, data extraction.DocumentForm, expected, expectedLegacyDraft int, status, requestID string, now time.Time) (extraction.FormRecord, error) {
	if errs := extraction.ValidateDocumentForm(data, status == "Confirmed"); len(errs) > 0 {
		return extraction.FormRecord{}, errs
	}
	action := "draft"
	if status == "Confirmed" {
		action = "confirm"
	}
	data.Assessment = extraction.AssessDocumentForm(data, action)
	var revision int
	err := tx.QueryRow(ctx, "SELECT revision FROM document_forms WHERE organization_id=$1 AND document_id=$2 FOR UPDATE", org, doc).Scan(&revision)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return extraction.FormRecord{}, err
	}
	if revision != expected {
		return extraction.FormRecord{}, extraction.ErrConflict
	}
	// Both legacy and canonical writes hold the document lock. Check the proposal
	// being replaced before creating its first canonical revision, including zero.
	if revision == 0 {
		var legacyRevision int
		err = tx.QueryRow(ctx, `SELECT revision FROM document_review_drafts WHERE organization_id=$1 AND document_id=$2 AND ocr_job_id=$3`, org, doc, ocr).Scan(&legacyRevision)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return extraction.FormRecord{}, err
		}
		if legacyRevision != expectedLegacyDraft {
			return extraction.FormRecord{}, extraction.ErrConflict
		}
	}
	// Validate references server-side even for drafts. A UUID is not authorization.
	typ := extraction.DocumentForms.Types[data.Type]
	seen := map[string]bool{}
	for _, table := range typ.Tables {
		for _, row := range data.Tables[table] {
			id := row["document_id"]
			if id == "" {
				// External source documents are captured by number and evidence.
				// Incomplete originals may still be acknowledged, never auto-posted.
				continue
			}
			if id == doc || (seen[id] && table != "transactions") {
				return extraction.FormRecord{}, extraction.FormErrors{table: "เอกสารอ้างอิงซ้ำหรืออ้างถึงตัวเอง"}
			}
			seen[id] = true
			var raw []byte
			err = tx.QueryRow(ctx, `SELECT f.data FROM document_forms f JOIN documents d ON d.organization_id=f.organization_id AND d.id=f.document_id
   JOIN document_ocr_runs o ON o.organization_id=d.organization_id AND o.document_id=d.id AND o.job_id=f.ocr_job_id AND o.superseded_at IS NULL AND o.deleted_at IS NULL
   WHERE f.organization_id=$1 AND f.document_id=$2 AND f.status='Confirmed' AND d.status IN ('Available','Archived')
   AND (organization_role($3::uuid,$1::uuid) IN ('Owner','Admin') OR (d.submitted_by_user_id=$3::uuid AND NOT d.group_restricted) OR d.assignee_user_id=$3::uuid
   OR EXISTS (SELECT 1 FROM document_sources ds WHERE ds.organization_id=d.organization_id AND ds.document_id=d.id AND ds.submitted_by_user_id=$3::uuid AND NOT ds.group_source))
   FOR SHARE OF f`, org, id, user).Scan(&raw)
			if err != nil {
				return extraction.FormRecord{}, extraction.FormErrors{table: "เอกสารอ้างอิงไม่พร้อมหรือไม่มีสิทธิ์"}
			}
			var target extraction.DocumentForm
			if err = json.Unmarshal(raw, &target); err != nil {
				return extraction.FormRecord{}, err
			}
			if data.Fields["currency"] == "" || target.Fields["currency"] != data.Fields["currency"] {
				return extraction.FormRecord{}, extraction.FormErrors{table: "สกุลเงินเอกสารอ้างอิงไม่ตรงกัน"}
			}
			if data.Type == "billing_note" {
				if target.Type != "invoice" || data.Fields["buyer_name"] == "" || target.Fields["buyer_name"] != data.Fields["buyer_name"] || target.Fields["buyer_tax_id"] != data.Fields["buyer_tax_id"] {
					return extraction.FormRecord{}, extraction.FormErrors{table: "เลือกใบแจ้งหนี้ของคู่ค้าเดียวกัน"}
				}
			}
			if table == "references" {
				targetTotal := referenceFormTotal(target)
				for _, key := range []string{"document_number", "issue_date", "total_amount", "paid_amount"} {
					expected := target.Fields[key]
					if key == "total_amount" {
						expected = targetTotal
					}
					if row[key] != expected {
						return extraction.FormRecord{}, extraction.FormErrors{table: "ข้อมูลอ้างอิงเปลี่ยนแล้ว กรุณาเลือกใหม่"}
					}
				}
				if data.Type == "credit_note" || data.Type == "debit_note" {
					if target.Type != "invoice" && target.Type != "tax_invoice" && target.Type != "tax_invoice_receipt" {
						return extraction.FormRecord{}, extraction.FormErrors{table: "เลือกใบแจ้งหนี้หรือใบกำกับภาษีต้นทาง"}
					}
				}
				allocated, a := extraction.FormCents(row["allocated"])
				total, t := extraction.FormCents(targetTotal)
				paid, p := extraction.FormCents(target.Fields["paid_amount"])
				if a && t && (allocated > total || data.Type == "billing_note" && p && allocated > total-paid) {
					return extraction.FormRecord{}, extraction.FormErrors{table: "ยอดจัดสรรเกินยอดเอกสารหรือยอดคงเหลือที่ทราบ"}
				}
			}
		}
	}
	raw, err := json.Marshal(data)
	if err != nil {
		return extraction.FormRecord{}, err
	}
	record := extraction.FormRecord{Data: data, Revision: expected + 1, OCRJobID: ocr, Status: status, UpdatedAt: now}
	_, err = tx.Exec(ctx, `INSERT INTO document_forms(organization_id,document_id,ocr_job_id,revision,data,status,updated_by,updated_at)
 VALUES($1,$2,$3,$4,$5,$6,$7,$8) ON CONFLICT(organization_id,document_id) DO UPDATE SET
 ocr_job_id=excluded.ocr_job_id,revision=excluded.revision,data=excluded.data,status=excluded.status,updated_by=excluded.updated_by,updated_at=excluded.updated_at`, org, doc, ocr, record.Revision, raw, status, user, now)
	if err != nil {
		return record, err
	}
	_, err = tx.Exec(ctx, `INSERT INTO document_form_history SELECT * FROM document_forms WHERE organization_id=$1 AND document_id=$2`, org, doc)
	if err != nil {
		return record, err
	}
	_, err = tx.Exec(ctx, `INSERT INTO audit_events(organization_id,actor_user_id,event_type,target_type,target_id,request_id,outcome,occurred_at,metadata)
 VALUES($1,$2,'document.form.save','document',$3,$4,'success',$5,jsonb_build_object('revision',$6::integer,'status',$7::text))`, org, user, doc, requestID, now, record.Revision, status)
	return record, err
}

// Reference snapshots use the entered amount first. Derived amounts come from
// the saved assessment so newer rules do not silently change old references.
func referenceFormTotal(data extraction.DocumentForm) string {
	if value := data.Fields["total_amount"]; value != "" {
		return value
	}
	if data.Assessment != nil {
		return data.Assessment.Calculated["total_amount"]
	}
	return extraction.CalculateForm(data)["total_amount"]
}
