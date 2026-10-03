package postgres

import (
	"os"
	"strings"
	"testing"
	"time"

	"plaiflow/api/internal/document"
	"plaiflow/api/internal/extraction"
	"plaiflow/api/internal/job"
	"plaiflow/api/internal/matching"
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
	claimTime := time.Now().UTC() // The insert trigger schedules the job at database now().
	claims, err := store.ClaimJobs(ctx, job.ClaimCommand{WorkerID: "ocr-test", Environment: "test",
		Kinds: []job.Kind{job.OCR}, Limit: 1, Lease: 2 * time.Minute, Now: claimTime, OCRProvider: "railway", OCRDefaultProvider: "railway"})
	if err != nil || len(claims) != 1 {
		t.Fatalf("claim before migration 19: count=%d err=%v", len(claims), err)
	}
	claim := claims[0]
	lease := job.LeaseCommand{JobID: claim.Job.ID, AttemptID: claim.Job.AttemptID, LeaseToken: claim.LeaseToken, Now: claimTime}
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
	source := matching.Facts{DocumentID: postgresUUID(), IssueDate: "2026-09-28", TotalAmount: "100.00"}
	candidates, err := store.FindMatchCandidates(ctx, user, org, source)
	if err != nil || len(candidates) != 1 || candidates[0].DocumentID != accepted.Document.ID {
		t.Fatalf("own matches=%v err=%v", candidates, err)
	}
	member := postgresUUID()
	if _, err := store.pool.Exec(ctx, `INSERT INTO users(id) VALUES($1)`, member); err != nil {
		t.Fatal(err)
	}
	if _, err := store.pool.Exec(ctx, `INSERT INTO memberships(organization_id,user_id,role) VALUES($1,$2,'Member')`, org, member); err != nil {
		t.Fatal(err)
	}
	candidates, err = store.FindMatchCandidates(ctx, member, org, source)
	if err != nil || len(candidates) != 0 {
		t.Fatalf("private same-organization matches=%v err=%v", candidates, err)
	}
	otherUser, otherOrg := postgresUUID(), postgresUUID()
	if _, err := store.pool.Exec(ctx, `INSERT INTO users(id) VALUES($1)`, otherUser); err != nil {
		t.Fatal(err)
	}
	if _, err := store.CreateOrganization(ctx, otherUser, otherOrg, "Other", now); err != nil {
		t.Fatal(err)
	}
	candidates, err = store.FindMatchCandidates(ctx, otherUser, otherOrg, source)
	if err != nil || len(candidates) != 0 {
		t.Fatalf("cross-organization matches=%v err=%v", candidates, err)
	}
	if _, err := store.FindMatchCandidates(ctx, otherUser, org, source); err == nil {
		t.Fatal("nonmember queried another organization")
	}
	checkRLS := func(actor, scope string, want int) {
		t.Helper()
		tx, err := store.pool.Begin(ctx)
		if err != nil {
			t.Fatal(err)
		}
		defer tx.Rollback(ctx)
		if _, err := tx.Exec(ctx, `SET LOCAL ROLE pg_read_all_data`); err != nil {
			t.Fatal(err)
		}
		if _, err := tx.Exec(ctx, `SELECT set_config('app.user_id',$1,true),set_config('app.organization_id',$2,true)`, actor, scope); err != nil {
			t.Fatal(err)
		}
		var bypass bool
		if err := tx.QueryRow(ctx, `SELECT rolbypassrls OR rolsuper FROM pg_roles WHERE rolname=current_user`).Scan(&bypass); err != nil || bypass {
			t.Fatalf("test role bypasses RLS: %v err=%v", bypass, err)
		}
		var count int
		if err := tx.QueryRow(ctx, `SELECT count(*) FROM document_match_index`).Scan(&count); err != nil || count != want {
			t.Fatalf("match index RLS count=%d want=%d err=%v", count, want, err)
		}
	}
	checkRLS(user, org, 1)
	checkRLS(member, org, 0)
	checkRLS(otherUser, otherOrg, 0)
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
