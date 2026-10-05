package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"plaiflow/api/internal/document"
	"plaiflow/api/internal/extraction"
	"plaiflow/api/internal/ocr"
	"plaiflow/api/internal/tenant"
	"testing"
	"time"
)

type canonicalStore struct {
	extractionStore
	form extraction.FormRecord
}

func (s *canonicalStore) ListFormReferences(context.Context, string, string) ([]extraction.FormReference, error) {
	return []extraction.FormReference{}, nil
}
func (s *canonicalStore) GetDocumentForm(context.Context, string, string, string) (extraction.FormRecord, error) {
	return s.form, nil
}
func (s *canonicalStore) SaveDocumentForm(_ context.Context, _, _, _, ocrID string, data extraction.DocumentForm, expected, expectedLegacy int, _ string, now time.Time) (extraction.FormRecord, error) {
	if expected != s.form.Revision || expected == 0 && s.draft.OCRJobID == ocrID && s.draft.Revision != expectedLegacy {
		return extraction.FormRecord{}, extraction.ErrConflict
	}
	s.form = extraction.FormRecord{Data: data, Revision: expected + 1, OCRJobID: ocrID, Status: "Draft", UpdatedAt: now}
	return s.form, nil
}
func (s *canonicalStore) SaveReview(ctx context.Context, r extraction.Review, expected int) (extraction.Review, error) {
	if r.ExpectedFormRevision != s.form.Revision || r.ExpectedFormRevision == 0 && s.draft.OCRJobID == r.OCRJobID && s.draft.Revision != r.ExpectedLegacyDraftRevision {
		return extraction.Review{}, extraction.ErrConflict
	}
	saved, e := s.extractionStore.SaveReview(ctx, r, expected)
	if e != nil {
		return saved, e
	}
	s.form = extraction.FormRecord{Data: *r.Canonical, Revision: r.ExpectedFormRevision + 1, OCRJobID: r.OCRJobID, Status: "Confirmed", UpdatedAt: r.ConfirmedAt}
	return saved, nil
}
func TestDocumentFormHTTPContract(t *testing.T) {
	now := time.Date(2026, 10, 4, 12, 0, 0, 0, time.UTC)
	for _, mode := range []string{"roundtrip", "csrf", "auth", "tenant", "member", "ack", "ocr", "revision", "missing-required", "unknown-key"} {
		t.Run(mode, func(t *testing.T) {
			authService, _ := newTestAuth(now)
			sample, _ := json.Marshal(ocr.Result{SchemaVersion: 1, Pages: []ocr.Page{{Number: 1, Lines: []ocr.Line{{Text: "Receipt", Confidence: .99}}}}})
			blobs := &extractionBlobs{objects: map[string][]byte{"ocr/result.json": sample, "original": []byte("original")}}
			store := &canonicalStore{}
			tenants := &tenantStore{allowed: "org-1"}
			if mode == "member" {
				tenants.role = tenant.Member
			}
			h := New(Config{Auth: authService, Tenants: tenants, OCR: extractionOCR{}, OCRStorage: blobs, Documents: &document.Service{Reader: fieldReviewDocuments{}, Intake: document.Intake{Temporary: blobs}}, Extraction: store, ExtractionEnabled: true, OCRPilotOrganizations: map[string]bool{"*": true}, Now: func() time.Time { return now }}, &fakeStore{})
			data := extraction.DocumentForm{Version: 1, Type: "receipt", Fields: map[string]string{"issue_date": "2026-10-04", "total_amount": "100.00", "paid_amount": "10.00", "notes": "เก็บไว้"}, Tables: map[string][]map[string]string{"items": {{"description": "รายการเฉพาะ"}}}}
			input := map[string]any{"data": data, "ocr_job_id": "ocr-1", "expected_revision": 0, "expected_review_revision": 0, "confirm": true, "acknowledged": true}
			org := "org-1"
			want := 200
			switch mode {
			case "csrf", "member":
				want = 403
			case "auth":
				want = 401
			case "tenant":
				org = "org-2"
				want = 404
			case "ack":
				input["acknowledged"] = false
				want = 422
			case "ocr":
				input["ocr_job_id"] = "old"
				want = 409
			case "revision":
				input["expected_revision"] = 1
				want = 409
			case "missing-required":
				delete(data.Fields, "paid_amount")
				want = 200
			case "unknown-key":
				data.Fields["unsupported"] = "x"
				want = 422
			}
			send := func(method string, payload any) *httptest.ResponseRecorder {
				raw, _ := json.Marshal(payload)
				req := httptest.NewRequest(method, "https://app.example/v1/o/"+org+"/documents/doc-1/form", bytes.NewReader(raw))
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("Origin", "https://app.example")
				if mode != "csrf" {
					req.Header.Set("X-CSRF-Token", "csrf")
				}
				if mode != "auth" {
					req.AddCookie(&http.Cookie{Name: "__Host-plaiflow-session", Value: "session"})
				}
				out := httptest.NewRecorder()
				h.ServeHTTP(out, req)
				return out
			}
			out := send(http.MethodPost, input)
			if out.Code != want {
				t.Fatalf("status=%d want=%d body=%s", out.Code, want, out.Body.String())
			}
			if want != 200 {
				if store.form.Revision != 0 {
					t.Fatal("failed write persisted")
				}
				return
			}
			if mode == "missing-required" {
				if store.form.Data.Assessment == nil || store.form.Data.Assessment.EvidenceStatus != "incomplete" || store.form.Data.Assessment.TaxStatus != "not_assessed" {
					t.Fatal("incomplete evidence lost")
				}
				return
			}
			out = send(http.MethodGet, nil)
			var response struct {
				Form extraction.FormRecord `json:"form"`
			}
			if e := json.Unmarshal(out.Body.Bytes(), &response); e != nil || response.Form.Data.Fields["paid_amount"] != "10.00" || len(response.Form.Data.Tables["items"]) != 1 {
				t.Fatal(out.Body.String(), e)
			}
			// Partial payload preserves unmentioned fields/tables and explicitly clears notes.
			input["confirm"] = false
			input["expected_revision"] = 1
			input["data"] = extraction.DocumentForm{Version: 1, Type: "delivery_note", Fields: map[string]string{"notes": ""}}
			out = send(http.MethodPost, input)
			if out.Code != 200 || store.form.Data.Fields["notes"] != "" || store.form.Data.Fields["paid_amount"] != "10.00" || len(store.form.Data.Tables["items"]) != 1 {
				t.Fatal(out.Code, out.Body.String())
			}
			if store.review.Canonical == nil || store.review.Decisions != nil {
				t.Fatal("document acknowledgement replaced by fake field decisions")
			}
		})
	}
}

func TestDocumentFormPreservesLegacyDraft(t *testing.T) {
	for _, mode := range []string{"current", "review-disabled", "stale-ocr", "canonical", "changed-before-draft", "changed-before-confirm"} {
		t.Run(mode, func(t *testing.T) {
			now := time.Date(2026, 10, 5, 12, 0, 0, 0, time.UTC)
			authService, _ := newTestAuth(now)
			sample, _ := json.Marshal(ocr.Result{SchemaVersion: 1, Pages: []ocr.Page{{Number: 1, Lines: []ocr.Line{{Text: "Receipt", Confidence: .99}}}}})
			previous := extraction.Review{ID: "review-1", OCRJobID: "ocr-1", Revision: 1, DocumentType: "receipt", Values: map[string]string{"seller_name": "confirmed seller", "total_amount": "50.00"}, ObjectKey: "review.json"}
			reviewJSON, _ := json.Marshal(previous)
			blobs := &extractionBlobs{objects: map[string][]byte{"ocr/result.json": sample, "original": []byte("original"), "review.json": reviewJSON}}
			store := &canonicalStore{extractionStore: extractionStore{review: previous, draft: extraction.ReviewDraft{OCRJobID: "ocr-1", Revision: 3, Values: map[string]string{"seller_name": "human correction", "total_amount": ""}}}}
			if mode == "stale-ocr" {
				store.draft.OCRJobID = "old-ocr"
			}
			if mode == "canonical" {
				store.form = extraction.FormRecord{Revision: 2, OCRJobID: "ocr-1", Status: "Draft", Data: extraction.DocumentForm{Version: 1, Type: "receipt", Fields: map[string]string{"seller_name": "canonical seller"}, Tables: map[string][]map[string]string{}}}
			}
			h := New(Config{Auth: authService, Tenants: &tenantStore{allowed: "org-1"}, OCR: extractionOCR{}, OCRStorage: blobs, Documents: &document.Service{Reader: fieldReviewDocuments{}, Intake: document.Intake{Temporary: blobs}}, Extraction: store, ExtractionEnabled: true, ReviewEnabled: mode != "review-disabled", OCRPilotOrganizations: map[string]bool{"*": true}, Now: func() time.Time { return now }}, &fakeStore{})
			send := func(method string, payload any) *httptest.ResponseRecorder {
				raw, _ := json.Marshal(payload)
				req := httptest.NewRequest(method, "https://app.example/v1/o/org-1/documents/doc-1/form", bytes.NewReader(raw))
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("Origin", "https://app.example")
				req.Header.Set("X-CSRF-Token", "csrf")
				req.AddCookie(&http.Cookie{Name: "__Host-plaiflow-session", Value: "session"})
				out := httptest.NewRecorder()
				h.ServeHTTP(out, req)
				return out
			}
			out := send(http.MethodGet, nil)
			var response struct {
				Form           extraction.FormRecord `json:"form"`
				LegacyRevision int                   `json:"legacy_draft_revision"`
			}
			if out.Code != 200 || json.Unmarshal(out.Body.Bytes(), &response) != nil {
				t.Fatal(out.Code, out.Body.String())
			}
			switch mode {
			case "stale-ocr":
				if response.Form.Data.Fields["seller_name"] != "confirmed seller" || response.LegacyRevision != 0 {
					t.Fatal(response)
				}
			case "canonical":
				if response.Form.Data.Fields["seller_name"] != "canonical seller" || response.LegacyRevision != 0 {
					t.Fatal(response)
				}
			default:
				if response.Form.Data.Fields["seller_name"] != "human correction" || response.Form.Data.Fields["total_amount"] != "" || response.LegacyRevision != 3 {
					t.Fatal(response)
				}
				if mode != "current" && mode != "review-disabled" {
					store.draft.Revision++
				}
				out = send(http.MethodPost, map[string]any{"data": response.Form.Data, "ocr_job_id": "ocr-1", "expected_revision": 0, "expected_review_revision": 1, "expected_legacy_draft_revision": response.LegacyRevision, "confirm": mode == "changed-before-confirm", "acknowledged": true})
				want := 200
				if mode != "current" && mode != "review-disabled" {
					want = 409
				}
				if out.Code != want {
					t.Fatal(out.Code, out.Body.String())
				}
				if mode == "current" && (store.form.Data.Fields["seller_name"] != "human correction" || store.form.Data.Fields["total_amount"] != "") {
					t.Fatal(store.form)
				}
			}
		})
	}
}
