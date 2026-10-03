package postgres

import (
	"errors"
	"strings"
	"testing"
	"time"

	"plaiflow/api/internal/classification"
	"plaiflow/api/internal/document"
	"plaiflow/api/internal/extraction"
	"plaiflow/api/internal/tenant"
)

func TestDocumentClassificationCorrectionKeepsOCRAndPrediction(t *testing.T) {
	store, ctx := isolatedTestStore(t, 24)
	owner, member, org := postgresUUID(), postgresUUID(), postgresUUID()
	now := time.Now().UTC()
	for _, user := range []string{owner, member} {
		if _, err := store.pool.Exec(ctx, `INSERT INTO users(id) VALUES($1)`, user); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := store.CreateOrganization(ctx, owner, org, "Classification test", now); err != nil {
		t.Fatal(err)
	}
	if _, err := store.pool.Exec(ctx, `INSERT INTO memberships(organization_id,user_id,role) VALUES($1,$2,'Member')`, org, member); err != nil {
		t.Fatal(err)
	}
	if _, err := store.pool.Exec(ctx, `UPDATE ocr_settings SET enabled=true`); err != nil {
		t.Fatal(err)
	}
	accepted, err := store.CommitPrepared(ctx, document.CommitInput{AcceptInput: document.AcceptInput{
		OrganizationID: org, ActorUserID: owner, AttemptID: postgresUUID(), OriginKey: "classification-test",
		Filename: "receipt.png", Channel: "Web", Now: now}, TemporaryKey: "test/classification",
		SHA256: strings.Repeat("a", 64), MIME: "image/png", Size: 10})
	if err != nil {
		t.Fatal(err)
	}
	var jobID string
	if err := store.pool.QueryRow(ctx, `UPDATE document_ocr_runs SET published_at=$2 WHERE document_id=$1 RETURNING job_id`, accepted.Document.ID, now).Scan(&jobID); err != nil {
		t.Fatal(err)
	}
	prediction := classification.Result{DocumentType: "receipt", Confidence: .82, Method: "rules"}
	if err := store.SaveClassification(ctx, org, accepted.Document.ID, jobID, prediction); err != nil {
		t.Fatal(err)
	}
	if _, err := store.CorrectDocumentType(ctx, member, org, accepted.Document.ID, jobID, "receipt", "invoice", now); !errors.Is(err, tenant.ErrForbidden) {
		t.Fatalf("member correction: %v", err)
	}
	if _, err := store.CorrectDocumentType(ctx, owner, org, accepted.Document.ID, jobID, "receipt", "invoice", now); err != nil {
		t.Fatal(err)
	}
	for _, scope := range []struct{ job, expected string }{{jobID, "receipt"}, {postgresUUID(), "invoice"}} {
		if _, err := store.CorrectDocumentType(ctx, owner, org, accepted.Document.ID, scope.job, scope.expected, "tax_invoice", now); !errors.Is(err, extraction.ErrConflict) {
			t.Fatalf("stale correction job=%s expected=%s: %v", scope.job, scope.expected, err)
		}
	}
	record, err := store.GetClassification(ctx, owner, org, accepted.Document.ID)
	if err != nil || record.DocumentType != "receipt" || record.EffectiveType != "invoice" || record.Confidence != .82 {
		t.Fatalf("classification after correction: %+v %v", record, err)
	}
	for _, scope := range []struct {
		user string
		want int
	}{{owner, 1}, {member, 0}} {
		tx, err := store.pool.Begin(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := tx.Exec(ctx, `SET LOCAL ROLE pg_read_all_data`); err != nil {
			t.Fatal(err)
		}
		if _, err := tx.Exec(ctx, `SELECT set_config('app.user_id',$1,true),set_config('app.organization_id',$2,true)`, scope.user, org); err != nil {
			t.Fatal(err)
		}
		var visible int
		err = tx.QueryRow(ctx, `SELECT count(*) FROM document_classifications WHERE document_id=$1`, accepted.Document.ID).Scan(&visible)
		if rollbackErr := tx.Rollback(ctx); rollbackErr != nil {
			t.Fatal(rollbackErr)
		}
		if err != nil || visible != scope.want {
			t.Fatalf("classification RLS user=%s visible=%d want=%d err=%v", scope.user, visible, scope.want, err)
		}
	}
	var runs, classifications int
	if err := store.pool.QueryRow(ctx, `SELECT (SELECT count(*) FROM document_ocr_runs WHERE document_id=$1),
		(SELECT count(*) FROM document_classifications WHERE document_id=$1)`, accepted.Document.ID).Scan(&runs, &classifications); err != nil || runs != 1 || classifications != 1 {
		t.Fatalf("OCR rerun or classification duplicate: runs=%d classifications=%d err=%v", runs, classifications, err)
	}
	review := extraction.Review{ID: postgresUUID(), OrganizationID: org, DocumentID: accepted.Document.ID,
		OCRJobID: jobID, DocumentType: "invoice", Values: map[string]string{"total_amount": "100.00"},
		ConfirmedBy: owner, ConfirmedAt: now, ObjectKey: "test/classification-review"}
	if _, err := store.SaveReview(ctx, review, 0); err != nil {
		t.Fatal(err)
	}
	if _, err := store.CorrectDocumentType(ctx, owner, org, accepted.Document.ID, jobID, "invoice", "receipt", now); !errors.Is(err, extraction.ErrConflict) {
		t.Fatalf("confirmed review correction: %v", err)
	}
}
