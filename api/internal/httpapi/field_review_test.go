package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"plaiflow/api/internal/document"
	"plaiflow/api/internal/extraction"
	"plaiflow/api/internal/ocr"
	"plaiflow/api/internal/tenant"
	"strings"
	"testing"
	"time"
)

func TestFieldReviewConfirmationContract(t *testing.T) {
	now := time.Date(2026, 10, 3, 12, 0, 0, 0, time.UTC)
	for _, tc := range []struct {
		name, mutate string
		status       int
	}{
		{"persist and reload", "", 200}, {"missing decision", "missing", 422}, {"invalid decision", "invalid", 422},
		{"accepted changed value", "accepted-edited", 422}, {"unknown has value", "unknown-value", 422},
		{"empty accepted", "empty-accepted", 422}, {"empty corrected", "empty-corrected", 422},
		{"stale OCR", "ocr", 409}, {"stale revision", "revision", 409},
		{"unknown version", "version", 422}, {"legacy client", "legacy", 200},
		{"cross tenant", "tenant", 404}, {"no CSRF", "csrf", 403}, {"no auth", "auth", 401}, {"no acknowledgement", "ack", 422}, {"member cannot confirm", "member", 403}, {"stale type", "type", 409},
	} {
		t.Run(tc.name, func(t *testing.T) {
			authService, _ := newTestAuth(now)
			sample, _ := json.Marshal(ocr.Result{SchemaVersion: 1, Pages: []ocr.Page{{Number: 1, Lines: []ocr.Line{{Text: "ใบเสร็จรับเงิน", Confidence: .98}, {Text: "เลขที่ INV-42", Confidence: .98}}}}})
			blobs := &extractionBlobs{objects: map[string][]byte{"ocr/result.json": sample}}
			store := &extractionStore{}
			tenants := &tenantStore{allowed: "org-1"}
			if tc.mutate == "member" {
				tenants.role = tenant.Member
			}
			handler := New(Config{Auth: authService, Tenants: tenants, OCR: extractionOCR{}, OCRStorage: blobs, Extraction: store, ExtractionEnabled: true, OCRPilotOrganizations: map[string]bool{"*": true}, Now: func() time.Time { return now }}, &fakeStore{})
			form := url.Values{"csrf_token": {"csrf"}, "ocr_job_id": {"ocr-1"}, "expected_revision": {"0"}, "review_ack": {"1"}, "field_review_version": {"1"}, "document_number": {"INV-43"}, "document_number_decision": {"corrected"}}
			for _, key := range extraction.Keys {
				if key != "document_number" {
					form.Set(key+"_decision", "unknown")
				}
			}
			form.Set("issue_date", "2026-10-03")
			form.Set("issue_date_decision", "corrected")
			form.Set("total_amount", "100.00")
			form.Set("total_amount_decision", "corrected")
			form.Set("expected_document_type", "receipt")
			org := "org-1"
			switch tc.mutate {
			case "type":
				form.Set("expected_document_type", "invoice")
			case "missing":
				form.Del("seller_name_decision")
			case "invalid":
				form.Set("seller_name_decision", "verified")
			case "accepted-edited":
				form.Set("document_number_decision", "accepted")
			case "unknown-value":
				form.Set("document_number_decision", "unknown")
			case "empty-accepted":
				form.Set("seller_name_decision", "accepted")
			case "empty-corrected":
				form.Set("seller_name_decision", "corrected")
			case "ocr":
				form.Set("ocr_job_id", "old")
			case "revision":
				form.Set("expected_revision", "1")
			case "version":
				form.Set("field_review_version", "2")
			case "legacy":
				form.Del("field_review_version")
				for _, key := range extraction.Keys {
					form.Del(key + "_decision")
				}
			case "tenant":
				org = "org-2"
			case "csrf":
				form.Del("csrf_token")
			case "ack":
				form.Del("review_ack")
			}
			base := "https://app.example/v1/o/" + org + "/documents/doc-1/extraction"
			req := httptest.NewRequest(http.MethodPost, base+"/confirm", strings.NewReader(form.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			req.Header.Set("Origin", "https://app.example")
			if tc.mutate != "auth" {
				req.AddCookie(&http.Cookie{Name: "__Host-plaiflow-session", Value: "session"})
			}
			res := httptest.NewRecorder()
			handler.ServeHTTP(res, req)
			if res.Code != tc.status {
				t.Fatalf("status=%d want=%d body=%s", res.Code, tc.status, res.Body.String())
			}
			if tc.status != 200 {
				if store.review.ID != "" {
					t.Fatal("invalid review persisted")
				}
				return
			}
			req = httptest.NewRequest(http.MethodGet, base, nil)
			req.AddCookie(&http.Cookie{Name: "__Host-plaiflow-session", Value: "session"})
			res = httptest.NewRecorder()
			handler.ServeHTTP(res, req)
			var payload struct {
				Confirmed *extraction.Review `json:"confirmed"`
			}
			json.Unmarshal(res.Body.Bytes(), &payload)
			if res.Code != 200 || payload.Confirmed == nil || payload.Confirmed.Values["document_number"] != "INV-43" || payload.Confirmed.Revision != 1 || payload.Confirmed.ConfirmedBy == "" || !payload.Confirmed.ConfirmedAt.Equal(now) {
				t.Fatalf("reload=%s", res.Body.String())
			}
			if tc.mutate != "legacy" && (payload.Confirmed.Decisions["document_number"] != "corrected" || payload.Confirmed.OriginalValues["document_number"] != "INV-42" || payload.Confirmed.Decisions["seller_name"] != "unknown") {
				t.Fatalf("field evidence lost: %+v", payload.Confirmed)
			}
		})
	}
}

type fieldReviewDocuments struct{ document.Reader }

func (fieldReviewDocuments) GetDocument(context.Context, string, string, string) (document.Document, error) {
	return document.Document{StorageKey: "original"}, nil
}

func TestFieldReviewSavedDraftGate(t *testing.T) {
	now := time.Date(2026, 10, 3, 12, 0, 0, 0, time.UTC)
	for _, tc := range []struct {
		name   string
		mutate string
		status int
	}{
		{"saved proposal confirms", "", 200}, {"must save first", "missing", 409},
		{"stale draft", "stale", 409}, {"unsaved value", "edit", 409}, {"unsaved decision", "decision", 409},
		{"accepted changed value rejected", "accepted", 422}, {"invalid decision rejected", "invalid", 422},
		{"original unavailable", "original", 409}, {"legacy review client", "legacy", 200},
	} {
		t.Run(tc.name, func(t *testing.T) {
			authService, _ := newTestAuth(now)
			sample, _ := json.Marshal(ocr.Result{SchemaVersion: 1, Pages: []ocr.Page{{Number: 1, Lines: []ocr.Line{{Text: "ใบเสร็จรับเงิน", Confidence: .98}, {Text: "เลขที่ INV-42", Confidence: .98}}}}})
			blobs := &extractionBlobs{objects: map[string][]byte{"ocr/result.json": sample, "original": []byte("original")}}
			store := &extractionStore{draft: extraction.ReviewDraft{OCRJobID: "ocr-1", Revision: 1, Values: map[string]string{}, Decisions: map[string]string{}}}
			for _, key := range extraction.Keys {
				store.draft.Values[key] = ""
				store.draft.Decisions[key] = "unknown"
			}
			store.draft.Values["document_number"] = "INV-43"
			store.draft.Decisions["document_number"] = "corrected"
			store.draft.Values["issue_date"] = "2026-10-03"
			store.draft.Decisions["issue_date"] = "corrected"
			store.draft.Values["total_amount"] = "100.00"
			store.draft.Decisions["total_amount"] = "corrected"
			form := url.Values{"csrf_token": {"csrf"}, "ocr_job_id": {"ocr-1"}, "expected_revision": {"0"}, "expected_draft_revision": {"1"}, "review_ack": {"1"}, "field_review_version": {"1"}, "expected_document_type": {"receipt"}}
			if tc.mutate == "accepted" {
				store.draft.Decisions["document_number"] = "accepted"
			}
			if tc.mutate == "invalid" {
				store.draft.Decisions["seller_name"] = ""
			}
			for _, key := range extraction.Keys {
				form.Set(key, store.draft.Values[key])
				form.Set(key+"_decision", store.draft.Decisions[key])
			}
			switch tc.mutate {
			case "missing":
				store.draft.Revision = 0
			case "stale":
				form.Set("expected_draft_revision", "0")
			case "edit":
				form.Set("document_number", "INV-44")
			case "decision":
				form.Set("document_number_decision", "accepted")
			case "original":
				delete(blobs.objects, "original")
			case "legacy":
				form.Del("field_review_version")
			}
			handler := New(Config{Auth: authService, Tenants: &tenantStore{allowed: "org-1"}, OCR: extractionOCR{}, OCRStorage: blobs, Documents: &document.Service{Reader: fieldReviewDocuments{}, Intake: document.Intake{Temporary: blobs}}, Extraction: store, ExtractionEnabled: true, ReviewEnabled: true, OCRPilotOrganizations: map[string]bool{"*": true}, Now: func() time.Time { return now }}, &fakeStore{})
			req := httptest.NewRequest(http.MethodPost, "https://app.example/v1/o/org-1/documents/doc-1/extraction/confirm", strings.NewReader(form.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			req.Header.Set("Origin", "https://app.example")
			req.AddCookie(&http.Cookie{Name: "__Host-plaiflow-session", Value: "session"})
			res := httptest.NewRecorder()
			handler.ServeHTTP(res, req)
			if res.Code != tc.status {
				t.Fatalf("status=%d want=%d body=%s", res.Code, tc.status, res.Body.String())
			}
			if tc.status == 200 {
				var artifact extraction.Review
				if err := json.Unmarshal(blobs.objects[store.review.ObjectKey], &artifact); err != nil || artifact.Decisions["document_number"] != "corrected" || artifact.OriginalValues["document_number"] != "INV-42" || store.review.DraftRevision != 1 {
					t.Fatalf("saved evidence lost: %+v err=%v", artifact, err)
				}
			} else if store.review.ID != "" {
				t.Fatal("rejected review persisted")
			}
		})
	}
}
