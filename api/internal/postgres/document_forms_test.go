package postgres

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"plaiflow/api/internal/document"
	"plaiflow/api/internal/extraction"
	"plaiflow/api/internal/job"
	"plaiflow/api/internal/ocr"
	"plaiflow/api/internal/tenant"
)

func TestDocumentFormsPersistenceAndIsolation(t *testing.T) {
	store, ctx := isolatedTestStore(t, 25)
	store.EnableReview()
	now := time.Now().UTC()
	owner, member, org, otherOrg := postgresUUID(), postgresUUID(), postgresUUID(), postgresUUID()
	for _, u := range []string{owner, member} {
		if _, e := store.pool.Exec(ctx, "INSERT INTO users(id) VALUES($1)", u); e != nil {
			t.Fatal(e)
		}
	}
	for _, o := range []string{org, otherOrg} {
		if _, e := store.CreateOrganization(ctx, owner, o, "Forms test", now); e != nil {
			t.Fatal(e)
		}
		// The 22 form types plus isolation/conversion/reference cases exceed the
		// default 30-document allowance. Give only these disposable fixtures a trial.
		if _, e := store.pool.Exec(ctx, `INSERT INTO organization_plan_periods(id,organization_id,plan_key,source,starts_at,ends_at,created_at)
		 VALUES($1,$2,'Business','trial',$3,$4,$3)`, postgresUUID(), o, now.Add(-time.Hour), now.Add(time.Hour)); e != nil {
			t.Fatal(e)
		}
	}
	if _, e := store.pool.Exec(ctx, "INSERT INTO memberships(organization_id,user_id,role) VALUES($1,$2,'Member')", org, member); e != nil {
		t.Fatal(e)
	}
	if _, e := store.pool.Exec(ctx, "UPDATE ocr_settings SET enabled=true"); e != nil {
		t.Fatal(e)
	}
	sequence := 0
	create := func(organization string) (string, string) {
		t.Helper()
		sequence++
		hash := fmt.Sprintf("%064x", sequence)
		accepted, e := store.CommitPrepared(ctx, document.CommitInput{AcceptInput: document.AcceptInput{OrganizationID: organization, ActorUserID: owner, AttemptID: postgresUUID(), OriginKey: hash, Filename: "form.png", Channel: "Web", Now: time.Now().UTC()}, TemporaryKey: "test/" + hash, SHA256: hash, MIME: "image/png", Size: 10})
		if e != nil {
			t.Fatal(e)
		}
		at := time.Now().UTC()
		claims, e := store.ClaimJobs(ctx, job.ClaimCommand{WorkerID: "form-test", Environment: "test", Kinds: []job.Kind{job.OCR}, Limit: 1, Lease: 2 * time.Minute, Now: at, OCRProvider: "railway", OCRDefaultProvider: "railway"})
		if e != nil || len(claims) != 1 {
			t.Fatalf("claims %d %v", len(claims), e)
		}
		c := claims[0]
		lease := job.LeaseCommand{JobID: c.Job.ID, AttemptID: c.Job.AttemptID, LeaseToken: c.LeaseToken, Now: at}
		in, e := store.OCRInput(ctx, lease, "form-test")
		if e != nil {
			t.Fatal(e)
		}
		result := ocr.Result{SchemaVersion: 1, InputSHA256: in.SHA256, ModelVersion: in.ModelVersion, PreprocessingVersion: in.PreprocessingVersion, Pages: []ocr.Page{{Number: 1, Width: 100, Height: 100}}}
		if _, e = store.CompleteOCR(ctx, lease, "form-test", result, "test/result/"+hash, "abc", 10); e != nil {
			t.Fatal(e)
		}
		return accepted.Document.ID, c.Job.ID
	}
	confirm := func(doc, ocrID string, d extraction.DocumentForm, formRevision, reviewRevision int) error {
		_, e := store.SaveReview(ctx, extraction.Review{ID: postgresUUID(), OrganizationID: org, DocumentID: doc, OCRJobID: ocrID, Canonical: &d, ExpectedFormRevision: formRevision, DocumentType: d.Type, Values: d.LegacyValues(), ConfirmedBy: owner, ConfirmedAt: time.Now().UTC(), ObjectKey: "test/review/" + postgresUUID()}, reviewRevision)
		return e
	}
	sourceDoc, sourceOCR := create(org)
	source := extraction.DocumentForm{Version: 1, Type: "invoice", Fields: map[string]string{"document_number": "INV-1", "issue_date": "2026-10-04", "currency": "THB", "buyer_name": "คู่ค้าทดสอบ", "buyer_tax_id": "0105552117718", "total_amount": "100.00", "paid_amount": "0.00"}, Tables: map[string][]map[string]string{}}
	if e := confirm(sourceDoc, sourceOCR, source, 0, 0); e != nil {
		t.Fatal(e)
	}
	sourceRow := func() map[string]string {
		return map[string]string{"document_id": sourceDoc, "document_number": "INV-1", "issue_date": "2026-10-04", "total_amount": "100.00", "paid_amount": "0.00", "allocated": "10.00"}
	}
	t.Run("legacy-conversion-revisions", func(t *testing.T) {
		for _, action := range []string{"draft", "confirm"} {
			doc, jobID := create(org)
			legacy := extraction.ReviewDraft{OrganizationID: org, DocumentID: doc, OCRJobID: jobID, Values: map[string]string{"seller_name": "human correction"}, Decisions: map[string]string{}, UpdatedBy: owner, UpdatedAt: now}
			if _, err := store.SaveDraft(ctx, legacy, 0, "legacy"); err != nil {
				t.Fatal(err)
			}
			data := extraction.DocumentForm{Version: 1, Type: "receipt", Fields: legacy.Values, Tables: map[string][]map[string]string{}}
			save := func(expectedLegacy int) error {
				if action == "draft" {
					_, err := store.SaveDocumentForm(ctx, owner, org, doc, jobID, data, 0, expectedLegacy, "convert", now)
					return err
				}
				_, err := store.SaveReview(ctx, extraction.Review{ID: postgresUUID(), OrganizationID: org, DocumentID: doc, OCRJobID: jobID, Canonical: &data, ExpectedLegacyDraftRevision: expectedLegacy, DocumentType: data.Type, Values: data.LegacyValues(), ConfirmedBy: owner, ConfirmedAt: now, ObjectKey: "test/convert"}, 0)
				return err
			}
			// A browser opened before any proposal existed cannot replace it.
			if err := save(0); !errors.Is(err, extraction.ErrConflict) {
				t.Fatalf("%s absent token: %v", action, err)
			}
			if _, err := store.SaveDraft(ctx, legacy, 1, "legacy-newer"); err != nil {
				t.Fatal(err)
			}
			if err := save(1); !errors.Is(err, extraction.ErrConflict) {
				t.Fatalf("%s stale token: %v", action, err)
			}
			if err := save(2); err != nil {
				t.Fatalf("%s current token: %v", action, err)
			}
			if _, err := store.SaveDraft(ctx, legacy, 2, "old-browser"); !errors.Is(err, extraction.ErrConflict) {
				t.Fatalf("legacy overwrite after %s: %v", action, err)
			}
			if _, err := store.SaveReview(ctx, extraction.Review{ID: postgresUUID(), OrganizationID: org, DocumentID: doc, OCRJobID: jobID, DocumentType: data.Type, Values: data.LegacyValues(), ConfirmedBy: owner, ConfirmedAt: now, ObjectKey: "test/old-browser"}, 0); !errors.Is(err, extraction.ErrConflict) {
				t.Fatalf("legacy confirmation after %s: %v", action, err)
			}
		}
	})
	for name, typ := range extraction.DocumentForms.Types {
		t.Run(name, func(t *testing.T) {
			doc, ocrID := create(org)
			d := extraction.DocumentForm{Version: 1, Type: name, Fields: map[string]string{"currency": "THB", "buyer_name": "คู่ค้าทดสอบ", "buyer_tax_id": "0105552117718", "notes": "ภาษาไทย\nบรรทัดสอง"}, Tables: map[string][]map[string]string{}}
			keys := []string{}
			for _, section := range typ.Sections {
				keys = append(keys, extraction.DocumentForms.Sections[section].Fields...)
			}
			for _, key := range keys {
				if d.Fields[key] != "" {
					continue
				}
				v := "ทดสอบ"
				switch extraction.DocumentForms.Fields[key].Kind {
				case "date":
					v = "2026-10-04"
				case "tax":
					v = "0105552117718"
				case "money", "signed_money", "decimal":
					v = "10.00"
				case "time":
					v = "12:30"
				case "branch":
					v = "00000"
				case "choice":
					v = extraction.DocumentForms.Fields[key].Options[0][0]
				}
				if key == "price_mode" {
					v = "exclusive"
				}
				d.Fields[key] = v
			}
			if name == "billing_note" || name == "credit_note" || name == "debit_note" {
				d.Tables["references"] = []map[string]string{sourceRow()}
			}
			if name == "withholding_tax_certificate" {
				d.Fields["income_total"] = "100.00"
				d.Fields["withholding_tax"] = "3.00"
				d.Tables["income"] = []map[string]string{{"description": "บริการ ก", "payment_date": "2026-10-04", "amount": "40", "tax": "1.20", "rate": "3"}, {"description": "บริการ ข", "payment_date": "2026-10-04", "amount": "60", "tax": "1.80", "rate": "3"}}
			}
			if name == "bank_statement" {
				d.Tables["transactions"] = []map[string]string{{"date": "2026-10-04", "description": "เงินเข้า", "credit": "10.00"}, {"date": "2026-10-04", "description": "เงินออก", "debit": "2.00"}}
			}
			saved, e := store.SaveDocumentForm(ctx, owner, org, doc, ocrID, d, 0, 0, "draft", now)
			if e != nil {
				t.Fatal(e)
			}
			reopened, e := store.GetDocumentForm(ctx, owner, org, doc)
			d.Assessment = extraction.AssessDocumentForm(d, "draft")
			if e != nil || !reflect.DeepEqual(reopened.Data, d) || reopened.Revision != saved.Revision {
				t.Fatalf("roundtrip %+v %v", reopened, e)
			}
			if e = confirm(doc, ocrID, d, 1, 0); e != nil {
				t.Fatal(e)
			}
			reopened, e = store.GetDocumentForm(ctx, owner, org, doc)
			d.Assessment = extraction.AssessDocumentForm(d, "confirm")
			if e != nil || reopened.Status != "Confirmed" || !reflect.DeepEqual(reopened.Data, d) {
				t.Fatalf("confirmation %+v %v", reopened, e)
			}
			if e = confirm(doc, ocrID, d, 1, 0); !errors.Is(e, extraction.ErrConflict) {
				t.Fatalf("stale confirm %v", e)
			}
			// Changing type preserves hidden tables/fields across persisted revisions.
			d.Type = "unknown"
			if _, e = store.SaveDocumentForm(ctx, owner, org, doc, ocrID, d, 2, 0, "switch", now); e != nil {
				t.Fatal(e)
			}
			d.Type = name
			if _, e = store.SaveDocumentForm(ctx, owner, org, doc, ocrID, d, 3, 0, "switch-back", now); e != nil {
				t.Fatal(e)
			}
			reopened, e = store.GetDocumentForm(ctx, owner, org, doc)
			d.Assessment = extraction.AssessDocumentForm(d, "draft")
			if e != nil || !reflect.DeepEqual(reopened.Data, d) {
				t.Fatalf("switch roundtrip %+v %v", reopened, e)
			}
			reviews, e := store.ListCurrentReviews(ctx, owner, org, 100)
			if e != nil {
				t.Fatal(e)
			}
			for _, r := range reviews {
				if r.DocumentID == doc {
					t.Fatal("edited draft exported as confirmed")
				}
			}
			if e = confirm(doc, ocrID, d, 4, 1); e != nil {
				t.Fatal(e)
			}
		})
	}
	// A fresh database connection sees committed data, not a cache.
	fresh, e := New(ctx, store.pool.Config().ConnString(), 2, nil)
	if e != nil {
		t.Fatal(e)
	}
	defer fresh.Close()
	if f, e := fresh.GetDocumentForm(ctx, owner, org, sourceDoc); e != nil || f.Status != "Confirmed" {
		t.Fatal(f, e)
	}
	if _, e := store.GetDocumentForm(ctx, member, org, sourceDoc); e == nil {
		t.Fatal("member read private document")
	}
	if _, e := store.GetDocumentForm(ctx, owner, otherOrg, sourceDoc); e == nil {
		t.Fatal("cross tenant read")
	}
	refs, e := store.ListFormReferences(ctx, member, org)
	if e != nil || len(refs) != 0 {
		t.Fatalf("private reference leak %d %v", len(refs), e)
	}
	foreignDoc, _ := create(otherOrg)
	targetDoc, targetOCR := create(org)
	bad := extraction.DocumentForm{Version: 1, Type: "billing_note", Fields: map[string]string{"currency": "THB", "buyer_name": "คู่ค้าทดสอบ", "buyer_tax_id": "0105552117718"}, Tables: map[string][]map[string]string{"references": {sourceRow()}}}
	for _, mode := range []string{"foreign", "duplicate", "currency", "party", "overallocate", "snapshot"} {
		row := sourceRow()
		bad.Fields["currency"] = "THB"
		bad.Fields["buyer_name"] = "คู่ค้าทดสอบ"
		bad.Tables["references"] = []map[string]string{row}
		switch mode {
		case "foreign":
			row["document_id"] = foreignDoc
		case "duplicate":
			bad.Tables["references"] = append(bad.Tables["references"], sourceRow())
		case "currency":
			bad.Fields["currency"] = "USD"
		case "party":
			bad.Fields["buyer_name"] = "อื่น"
		case "overallocate":
			row["allocated"] = "100.01"
		case "snapshot":
			row["total_amount"] = "99"
		}
		if _, e := store.SaveDocumentForm(ctx, owner, org, targetDoc, targetOCR, bad, 0, 0, mode, now); e == nil {
			t.Fatalf("accepted %s", mode)
		}
	}
	// Competing editors: exactly one succeeds.
	var wg sync.WaitGroup
	results := make(chan error, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, e := store.SaveDocumentForm(ctx, owner, org, sourceDoc, sourceOCR, source, 1, 0, "race", now)
			results <- e
		}()
	}
	wg.Wait()
	close(results)
	success, conflict := 0, 0
	for e := range results {
		if e == nil {
			success++
		} else if errors.Is(e, extraction.ErrConflict) {
			conflict++
		} else {
			t.Fatal(e)
		}
	}
	if success != 1 || conflict != 1 {
		t.Fatal(success, conflict)
	}
	var history, audits int
	if e := store.pool.QueryRow(ctx, "SELECT count(*) FROM document_form_history WHERE document_id=$1", sourceDoc).Scan(&history); e != nil {
		t.Fatal(e)
	}
	if e := store.pool.QueryRow(ctx, "SELECT count(*) FROM audit_events WHERE target_id=$1 AND event_type='document.form.save'", sourceDoc).Scan(&audits); e != nil {
		t.Fatal(e)
	}
	if history != 2 || audits != 2 {
		t.Fatal(history, audits)
	}
	// A permitted Member edit invalidates the prior review atomically; it never
	// grants confirmation authority. Stale OCR and failed references roll back.
	memberDoc, memberOCR := create(org)
	if _, e = store.pool.Exec(ctx, "UPDATE documents SET assignee_user_id=$2 WHERE id=$1", memberDoc, member); e != nil {
		t.Fatal(e)
	}
	if e = confirm(memberDoc, memberOCR, source, 0, 0); e != nil {
		t.Fatal(e)
	}
	if _, e = store.SaveDocumentForm(ctx, member, org, memberDoc, memberOCR, source, 1, 0, "member-edit", now); e != nil {
		t.Fatal(e)
	}
	var invalidated bool
	if e = store.pool.QueryRow(ctx, "SELECT form_invalidated_at IS NOT NULL FROM document_extraction_reviews WHERE document_id=$1 AND superseded_at IS NULL", memberDoc).Scan(&invalidated); e != nil || !invalidated {
		t.Fatal("member edit did not invalidate", e)
	}
	if _, e = store.SaveReview(ctx, extraction.Review{OrganizationID: org, DocumentID: memberDoc, ConfirmedBy: member, Canonical: &source}, 1); !errors.Is(e, tenant.ErrForbidden) {
		t.Fatal("member confirmed", e)
	}
	if _, e = store.SaveDocumentForm(ctx, owner, org, memberDoc, postgresUUID(), source, 2, 0, "stale-ocr", now); !errors.Is(e, extraction.ErrConflict) {
		t.Fatal("stale OCR accepted", e)
	}
	if e = confirm(memberDoc, memberOCR, bad, 2, 1); e == nil {
		t.Fatal("invalid reference confirmed")
	}
	current, e := store.GetDocumentForm(ctx, owner, org, memberDoc)
	if e != nil || current.Revision != 2 || current.Status != "Draft" {
		t.Fatal("failed confirmation changed form", e)
	}
	meta, e := store.CurrentReview(ctx, owner, org, memberDoc)
	if e != nil || meta.Revision != 1 {
		t.Fatal("failed confirmation superseded prior review", e)
	}
	if e = store.pool.QueryRow(ctx, "SELECT count(*) FROM document_form_history WHERE document_id=$1", memberDoc).Scan(&history); e != nil || history != 2 {
		t.Fatal("failed save wrote history", e)
	}
	// Read RLS under the built-in non-bypass role.
	for _, actor := range []string{owner, member} {
		tx, e := store.pool.Begin(ctx)
		if e != nil {
			t.Fatal(e)
		}
		if _, e = tx.Exec(ctx, "SET LOCAL ROLE pg_read_all_data"); e != nil {
			t.Fatal(e)
		}
		if _, e = tx.Exec(ctx, "SELECT set_config('app.user_id',$1,true),set_config('app.organization_id',$2,true)", actor, org); e != nil {
			t.Fatal(e)
		}
		var count int
		if e = tx.QueryRow(ctx, "SELECT count(*) FROM document_forms WHERE document_id=$1", sourceDoc).Scan(&count); e != nil {
			t.Fatal(e)
		}
		if (actor == owner && count != 1) || (actor == member && count != 0) {
			t.Fatal(actor, count)
		}
		tx.Rollback(ctx)
	}
	// History remains append-only for a role with table write privileges.
	role := "form_test_" + strings.ReplaceAll(postgresUUID(), "-", "")
	ident := pgx.Identifier{role}.Sanitize()
	if _, e = store.pool.Exec(ctx, "CREATE ROLE "+ident+" NOLOGIN"); e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { store.pool.Exec(ctx, "DROP OWNED BY "+ident); store.pool.Exec(ctx, "DROP ROLE "+ident) })
	if _, e = store.pool.Exec(ctx, "GRANT SELECT,INSERT,UPDATE,DELETE ON ALL TABLES IN SCHEMA public TO "+ident); e != nil {
		t.Fatal(e)
	}
	tx, e := store.pool.Begin(ctx)
	if e != nil {
		t.Fatal(e)
	}
	defer tx.Rollback(ctx)
	if _, e = tx.Exec(ctx, "SET LOCAL ROLE "+ident); e != nil {
		t.Fatal(e)
	}
	if _, e = tx.Exec(ctx, "SELECT set_config('app.user_id',$1,true),set_config('app.organization_id',$2,true)", owner, org); e != nil {
		t.Fatal(e)
	}
	command, e := tx.Exec(ctx, "DELETE FROM document_form_history WHERE document_id=$1", sourceDoc)
	if e != nil || command.RowsAffected() != 0 {
		t.Fatalf("history delete %v %v", command, e)
	}
	tx.Rollback(ctx)
	t.Run("calculated reference totals", func(t *testing.T) {
		for _, tc := range []struct{ entered, effective, over string }{
			{"", "107.00", "107.01"},
			{"108.00", "108.00", "108.01"},
			{"0.00", "0.00", "0.01"},
		} {
			doc, ocrID := create(org)
			d := extraction.DocumentForm{Version: 1, Type: "invoice", Fields: map[string]string{
				"document_number": "CALC-" + tc.effective, "issue_date": "2026-10-05", "currency": "THB", "buyer_name": "คู่ค้าทดสอบ",
				"price_mode": "exclusive", "discount": "0", "vat_rate": "7", "total_amount": tc.entered, "paid_amount": "0.00",
			}, Tables: map[string][]map[string]string{"items": {{"quantity": "1", "unit_price": "100", "discount": "0"}}}}
			if e := confirm(doc, ocrID, d, 0, 0); e != nil {
				t.Fatal(e)
			}
			refs, e := store.ListFormReferences(ctx, owner, org)
			if e != nil {
				t.Fatal(e)
			}
			var selected *extraction.FormReference
			for i := range refs {
				if refs[i].ID == doc {
					selected = &refs[i]
				}
			}
			if selected == nil || selected.Data.Fields["total_amount"] != tc.effective {
				t.Fatalf("reference omitted effective total %s: %#v", tc.effective, selected)
			}
			row := map[string]string{"document_id": doc, "allocated": tc.over}
			for _, key := range []string{"document_number", "issue_date", "total_amount", "paid_amount"} {
				row[key] = selected.Data.Fields[key]
			}
			billDoc, billOCR := create(org)
			bill := extraction.DocumentForm{Version: 1, Type: "billing_note", Fields: map[string]string{"currency": "THB", "buyer_name": "คู่ค้าทดสอบ"}, Tables: map[string][]map[string]string{"references": {row}}}
			if _, e := store.SaveDocumentForm(ctx, owner, org, billDoc, billOCR, bill, 0, 0, "over-calculated", now); e == nil {
				t.Fatal("allocation exceeded effective reference total")
			}
			row["allocated"] = tc.effective
			if _, e := store.SaveDocumentForm(ctx, owner, org, billDoc, billOCR, bill, 0, 0, "at-calculated", now); e != nil {
				t.Fatal("valid effective reference rejected", e)
			}
			stored, e := store.GetDocumentForm(ctx, owner, org, doc)
			if e != nil || stored.Data.Fields["total_amount"] != tc.entered {
				t.Fatal("reference lookup changed original evidence", e)
			}
		}
	})
	serveDocumentFormBrowser(t, store, owner, org, sourceDoc)
}
