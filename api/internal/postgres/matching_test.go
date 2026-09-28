package postgres

import (
	"os"
	"strings"
	"testing"
	"time"

	"plaiflow/api/internal/document"
	"plaiflow/api/internal/extraction"
	"plaiflow/api/internal/job"
	"plaiflow/api/internal/ocr"
)

func TestOCRAndReviewRemainUsableBeforeMigration19(t *testing.T) {
	store, ctx := isolatedTestStore(t, 18)
	if _, err := store.pool.Exec(ctx, `UPDATE ocr_settings SET enabled=true`); err != nil {
		t.Fatal(err)
	}
	user, org, now := postgresUUID(), postgresUUID(), time.Now().UTC()
	if _, err := store.pool.Exec(ctx, `INSERT INTO users(id) VALUES($1)`, user); err != nil {
		t.Fatal(err)
	}
	if _, err := store.pool.Exec(ctx, `INSERT INTO organizations(id,name) VALUES($1,'OCR compatibility')`, org); err != nil {
		t.Fatal(err)
	}
	if _, err := store.pool.Exec(ctx, `INSERT INTO memberships(organization_id,user_id,role) VALUES($1,$2,'Owner')`, org, user); err != nil {
		t.Fatal(err)
	}
	accepted, err := store.CommitPrepared(ctx, document.CommitInput{AcceptInput: document.AcceptInput{
		OrganizationID: org, ActorUserID: user, AttemptID: postgresUUID(), OriginKey: "ocr-compatibility",
		Filename: "receipt.png", Channel: "Web", Now: now}, TemporaryKey: "test/original",
		SHA256: strings.Repeat("a", 64), MIME: "image/png", Size: 10})
	if err != nil {
		t.Fatal(err)
	}
	claims, err := store.ClaimJobs(ctx, job.ClaimCommand{WorkerID: "ocr-test", Environment: "test",
		Kinds: []job.Kind{job.OCR}, Limit: 1, Lease: 2 * time.Minute, Now: now, OCRProvider: "railway", OCRDefaultProvider: "railway"})
	if err != nil || len(claims) != 1 {
		t.Fatalf("claim before migration 19: count=%d err=%v", len(claims), err)
	}
	claim := claims[0]
	lease := job.LeaseCommand{JobID: claim.Job.ID, AttemptID: claim.Job.AttemptID, LeaseToken: claim.LeaseToken, Now: now}
	input, err := store.OCRInput(ctx, lease, "ocr-test")
	if err != nil {
		t.Fatal(err)
	}
	result := ocr.Result{SchemaVersion: 1, InputSHA256: input.SHA256,
		ModelVersion: input.ModelVersion, PreprocessingVersion: input.PreprocessingVersion,
		Pages: []ocr.Page{{Number: 1, Width: 100, Height: 100}}}
	if _, err := store.CompleteOCR(ctx, lease, "ocr-test", result, "test/result", "abc", 10); err != nil {
		t.Fatalf("complete before migration 19: %v", err)
	}
	review := extraction.Review{ID: postgresUUID(), OrganizationID: org, DocumentID: accepted.Document.ID,
		OCRJobID: claim.Job.ID, DocumentType: "receipt", Values: map[string]string{
			"issue_date": "2026-09-28", "total_amount": "100.00", "seller_name": "ร้านทดสอบ"},
		ConfirmedBy: user, ConfirmedAt: now, ObjectKey: "test/review"}
	if _, err := store.SaveReview(ctx, review, 0); err != nil {
		t.Fatalf("confirm before migration 19: %v", err)
	}
	up, err := os.ReadFile("../../migrations/000019_runpod_ocr.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.pool.Exec(ctx, string(up)); err != nil {
		t.Fatal(err)
	}
	store.EnableMatching()
	review.ID, review.ObjectKey = postgresUUID(), "test/review-2"
	if _, err := store.SaveReview(ctx, review, 1); err != nil {
		t.Fatalf("confirm after migration 19: %v", err)
	}
	var indexed int
	if err := store.pool.QueryRow(ctx, `SELECT count(*) FROM document_match_index WHERE organization_id=$1 AND document_id=$2`, org, accepted.Document.ID).Scan(&indexed); err != nil || indexed != 1 {
		t.Fatalf("match index count=%d err=%v", indexed, err)
	}
}

func TestRunPodMatchingMigrationRoundTrip(t *testing.T) {
	store, ctx := isolatedTestStore(t, 18)
	up, err := os.ReadFile("../../migrations/000019_runpod_ocr.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	down, err := os.ReadFile("../../migrations/000019_runpod_ocr.down.sql")
	if err != nil {
		t.Fatal(err)
	}
	for _, sql := range []string{string(up), string(down), string(up)} {
		if _, err := store.pool.Exec(ctx, sql); err != nil {
			t.Fatal(err)
		}
	}
	var available bool
	if err := store.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM pg_extension WHERE extname='pg_trgm')`).Scan(&available); err != nil || !available {
		t.Fatalf("pg_trgm available=%v err=%v", available, err)
	}
}
