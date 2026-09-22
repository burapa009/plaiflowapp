package postgres

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"plaiflow/api/internal/document"
	"plaiflow/api/internal/job"
	"plaiflow/api/internal/ocr"
)

// This test creates and drops only its own isolated database, never test tables in staging.
func TestOCRIsolatedDatabaseLifecycle(t *testing.T) {
	url := os.Getenv("OCR_TEST_ADMIN_URL")
	if url == "" {
		t.Skip("OCR_TEST_ADMIN_URL not set")
	}
	ctx := context.Background()
	admin, err := pgx.Connect(ctx, url)
	if err != nil {
		t.Fatal("test database connection unavailable")
	}
	defer admin.Close(ctx)
	name := "plaiflow_ocr_test_" + strings.ReplaceAll(postgresUUID(), "-", "")
	identifier := pgx.Identifier{name}.Sanitize()
	if _, err = admin.Exec(ctx, "CREATE DATABASE "+identifier); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if _, e := admin.Exec(ctx, "DROP DATABASE "+identifier+" WITH (FORCE)"); e != nil {
			t.Error(e)
		}
	}()
	config, err := pgx.ParseConfig(url)
	if err != nil {
		t.Fatal(err)
	}
	config.Database = name
	store, err := New(ctx, config.ConnString(), 6, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	paths, err := filepath.Glob("../../migrations/*.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range paths {
		sql, e := os.ReadFile(p)
		if e != nil {
			t.Fatal(e)
		}
		if _, e = store.pool.Exec(ctx, string(sql)); e != nil {
			t.Fatalf("migration %s: %v", filepath.Base(p), e)
		}
	}
	down, err := os.ReadFile("../../migrations/000008_phase6_ocr.down.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = store.pool.Exec(ctx, string(down)); err != nil {
		t.Fatal("empty rollback", err)
	}
	up, _ := os.ReadFile("../../migrations/000008_phase6_ocr.up.sql")
	if _, err = store.pool.Exec(ctx, string(up)); err != nil {
		t.Fatal(err)
	}
	if _, err = store.pool.Exec(ctx, `UPDATE ocr_settings SET enabled=true`); err != nil {
		t.Fatal(err)
	}
	user, org := postgresUUID(), postgresUUID()
	now := time.Now().UTC()
	if _, err = store.pool.Exec(ctx, `INSERT INTO users(id) VALUES($1)`, user); err != nil {
		t.Fatal(err)
	}
	if _, err = store.pool.Exec(ctx, `INSERT INTO organizations(id,name) VALUES($1,'OCR integration')`, org); err != nil {
		t.Fatal(err)
	}
	if _, err = store.pool.Exec(ctx, `INSERT INTO memberships(organization_id,user_id,role) VALUES($1,$2,'Owner')`, org, user); err != nil {
		t.Fatal(err)
	}
	accepted, err := store.CommitPrepared(ctx, document.CommitInput{AcceptInput: document.AcceptInput{OrganizationID: org, ActorUserID: user, AttemptID: postgresUUID(), OriginKey: "ocr-test", Filename: "test.png", Channel: "Web", Now: now}, TemporaryKey: "test/original", SHA256: strings.Repeat("a", 64), MIME: "image/png", Size: 10})
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	out := make(chan job.Claimed, 2)
	errs := make(chan error, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			jobs, e := store.ClaimJobs(ctx, job.ClaimCommand{WorkerID: "ocr-test", Environment: "test", Kinds: []job.Kind{job.OCR}, Limit: 1, Lease: 2 * time.Minute, Now: time.Now()})
			if e != nil {
				errs <- e
			}
			for _, j := range jobs {
				out <- j
			}
		}()
	}
	wg.Wait()
	close(out)
	close(errs)
	for e := range errs {
		t.Fatal(e)
	}
	var claimed []job.Claimed
	for j := range out {
		claimed = append(claimed, j)
	}
	if len(claimed) != 1 {
		t.Fatalf("claims %d", len(claimed))
	}
	claim := claimed[0]
	lease := job.LeaseCommand{JobID: claim.Job.ID, AttemptID: claim.Job.AttemptID, LeaseToken: claim.LeaseToken, Now: time.Now()}
	in, err := store.OCRInput(ctx, lease, "ocr-test")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = store.OCRInput(ctx, lease, "other-worker"); !errors.Is(err, job.ErrLeaseLost) {
		t.Fatalf("wrong worker: %v", err)
	}
	result := ocr.Result{SchemaVersion: 1, InputSHA256: in.SHA256, ModelVersion: in.ModelVersion, PreprocessingVersion: in.PreprocessingVersion, Pages: []ocr.Page{{Number: 1, Width: 100, Height: 100, Lines: []ocr.Line{}}}}
	key, err := store.CompleteOCR(ctx, lease, "ocr-test", result, "test/result", "abc", 10)
	if err != nil {
		t.Fatal(err)
	}
	duplicate, err := store.CompleteOCR(ctx, lease, "ocr-test", result, "test/duplicate", "abc", 10)
	if err != nil || duplicate != key {
		t.Fatalf("duplicate=%s err=%v", duplicate, err)
	}
	state, err := store.OCRState(ctx, user, org, accepted.Document.ID)
	if err != nil || state.Status != "Completed" {
		t.Fatalf("state=%+v err=%v", state, err)
	}
	if _, err = store.OCRState(ctx, postgresUUID(), org, accepted.Document.ID); err == nil {
		t.Fatal("unauthorized retrieval")
	}
	if _, err = store.ChangeDocumentStatus(ctx, user, org, accepted.Document.ID, "trash", time.Now()); err != nil {
		t.Fatal(err)
	}
	if _, err = store.CompleteOCR(ctx, lease, "ocr-test", result, "test/late", "abc", 10); !errors.Is(err, job.ErrLeaseLost) {
		t.Fatalf("late completion %v", err)
	}
	if _, err = store.pool.Exec(ctx, string(down)); err == nil {
		t.Fatal("destructive rollback accepted with history")
	}
}
