package postgres

import (
	"strings"
	"testing"
	"time"

	"plaiflow/api/internal/document"
	"plaiflow/api/internal/job"
	"plaiflow/api/internal/ocr"
)

func TestOCRModelPilotSelection(t *testing.T) {
	s, ctx := isolatedTestStore(t, 20)
	user, pilot, other := postgresUUID(), postgresUUID(), postgresUUID()
	for _, statement := range []struct {
		sql  string
		args []any
	}{
		{`INSERT INTO users(id) VALUES($1)`, []any{user}},
		{`INSERT INTO organizations(id,name) VALUES($1,'Pilot'),($2,'Control')`, []any{pilot, other}},
		{`INSERT INTO memberships(organization_id,user_id,role) VALUES($1,$3,'Owner'),($2,$3,'Owner')`, []any{pilot, other, user}},
		{`UPDATE ocr_settings SET enabled=true,model_version=$1,preprocessing_version=$2`, []any{ocr.ModelVersion, ocr.PreprocessingVersion}},
		{`INSERT INTO organization_ocr_models VALUES($1,$2,$3)`, []any{pilot, ocr.GenerativeModelVersion, ocr.GenerativePreprocessingVersion}},
	} {
		if _, err := s.pool.Exec(ctx, statement.sql, statement.args...); err != nil {
			t.Fatal(err)
		}
	}
	for _, scope := range []struct{ org, model string }{{pilot, ocr.GenerativeModelVersion}, {other, ocr.ModelVersion}} {
		tx, err := s.organizationTx(ctx, user, scope.org)
		if err != nil {
			t.Fatal(err)
		}
		var model string
		err = tx.QueryRow(ctx, `SELECT model_version FROM document_ocr_model($1)`, scope.org).Scan(&model)
		tx.Rollback(ctx)
		if err != nil || model != scope.model {
			t.Fatalf("model=%s err=%v", model, err)
		}
		accepted, err := s.CommitPrepared(ctx, document.CommitInput{AcceptInput: document.AcceptInput{
			OrganizationID: scope.org, ActorUserID: user, AttemptID: postgresUUID(), OriginKey: "model-pilot", Filename: "test.png", Channel: "Web", Now: time.Now().UTC()},
			TemporaryKey: "test/original/" + scope.org, SHA256: strings.Repeat("a", 64), MIME: "image/png", Size: 10})
		if err != nil {
			t.Fatal(err)
		}
		state, err := s.RetryOCR(ctx, user, scope.org, accepted.Document.ID, time.Now().UTC())
		if err != nil {
			t.Fatal(err)
		}
		if err = s.pool.QueryRow(ctx, `SELECT payload->>'model_version' FROM durable_jobs WHERE id=$1`, state.JobID).Scan(&model); err != nil || model != scope.model {
			t.Fatalf("job model=%s err=%v", model, err)
		}
	}
	// During cutover, new-model jobs must never be claimed by the CPU worker.
	for _, scope := range []struct{ provider, org string }{{"railway", other}, {"runpod", pilot}} {
		claimed, err := s.ClaimJobs(ctx, job.ClaimCommand{WorkerID: "model-" + scope.provider,
			Environment: "test", Kinds: []job.Kind{job.OCR}, Limit: 10, Lease: 2 * time.Minute,
			Now: time.Now().UTC(), OCRProvider: scope.provider, OCRDefaultProvider: "railway"})
		if err != nil || len(claimed) != 1 || claimed[0].Job.OrganizationID != scope.org {
			t.Fatalf("provider=%s claimed=%v err=%v", scope.provider, claimed, err)
		}
		if scope.provider == "runpod" {
			var provider string
			if err := s.pool.QueryRow(ctx, `SELECT provider FROM document_ocr_runs WHERE job_id=$1`, claimed[0].Job.ID).Scan(&provider); err != nil || provider != "runpod" {
				t.Fatalf("recorded provider=%s err=%v", provider, err)
			}
		}
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, `SET LOCAL ROLE pg_read_all_data`); err != nil {
		t.Fatal(err)
	}
	if _, err = tx.Exec(ctx, `SELECT set_config('app.user_id',$1,true),set_config('app.organization_id',$2,true)`, user, other); err != nil {
		t.Fatal(err)
	}
	var count int
	if err = tx.QueryRow(ctx, `SELECT count(*) FROM organization_ocr_models`).Scan(&count); err != nil || count != 0 {
		t.Fatalf("cross-org rows=%d err=%v", count, err)
	}
}
