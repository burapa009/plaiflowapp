package httpapi

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"plaiflow/api/internal/accounting"
	"plaiflow/api/internal/extraction"
)

type accountingTestStore struct {
	accounting.Store
	approval accounting.Approval
}

func (s *accountingTestStore) ListCategories(context.Context, string, string) ([]accounting.Category, error) {
	return []accounting.Category{{ID: "cat-1", Name: "เดินทาง"}}, nil
}
func (s *accountingTestStore) Evaluate(_ context.Context, _, _ string, review extraction.Review) (accounting.Evaluation, error) {
	return accounting.Evaluation{Candidate: accounting.Candidate{Basis: "no_rule", Confidence: "unrated", Warnings: []string{"vendor_unmatched"}}, RuleSetVersion: 0}, nil
}
func (s *accountingTestStore) Approve(_ context.Context, input accounting.ApproveInput) (accounting.Approval, error) {
	s.approval = accounting.Approval{ID: input.ID, DocumentID: input.DocumentID, ReviewID: input.ReviewID,
		ReviewRevision: input.ReviewRevision, CategoryID: input.CategoryID, CategoryName: "เดินทาง", Basis: "human_reviewed",
		ApprovedAt: input.ApprovedAt, Warnings: []string{}}
	return s.approval, nil
}
func (s *accountingTestStore) ListApproved(context.Context, string, string, int, string) ([]accounting.Approval, error) {
	if s.approval.ID == "" {
		return nil, nil
	}
	return []accounting.Approval{s.approval}, nil
}

func TestConfirmedInvoiceCanBeApprovedAndExportedAsGenericAccountingRow(t *testing.T) {
	now := time.Date(2026, 9, 23, 12, 0, 0, 0, time.UTC)
	authService, err := newTestAuth(now)
	if err != nil {
		t.Fatal(err)
	}
	review := extraction.Review{ID: "review-1", OrganizationID: "org-1", DocumentID: "doc-1", OCRJobID: "ocr-1", Revision: 1,
		DocumentType: "tax_invoice", Values: map[string]string{"issue_date": "2026-09-23", "document_number": "INV-42",
			"seller_name": "บริษัท ตัวอย่าง จำกัด", "seller_tax_id": "0123456789012", "total_amount": "1070.00"},
		ObjectKey: "review.json", ConfirmedAt: now}
	data, _ := json.Marshal(review)
	blobs := &extractionBlobs{objects: map[string][]byte{"review.json": data}}
	approved := &accountingTestStore{}
	handler := New(Config{Auth: authService, Tenants: &tenantStore{allowed: "org-1"}, OCR: extractionOCR{}, OCRStorage: blobs,
		Extraction: &extractionStore{review: review}, ExtractionEnabled: true, Accounting: approved, AccountingEnabled: true,
		Now: func() time.Time { return now }}, &fakeStore{})
	base := "https://app.example/v1/o/org-1/documents/doc-1/accounting"
	request := httptest.NewRequest(http.MethodGet, base, nil)
	request.AddCookie(&http.Cookie{Name: "__Host-plaiflow-session", Value: "session"})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != 200 || !strings.Contains(response.Body.String(), `"vendor_unmatched"`) || !strings.Contains(response.Body.String(), `"unrated"`) {
		t.Fatalf("suggestion status=%d body=%s", response.Code, response.Body.String())
	}
	request = httptest.NewRequest(http.MethodGet, "https://app.example/v1/o/other-org/documents/doc-1/accounting", nil)
	request.AddCookie(&http.Cookie{Name: "__Host-plaiflow-session", Value: "session"})
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusNotFound {
		t.Fatalf("cross-organization suggestion status=%d", response.Code)
	}
	form := url.Values{"csrf_token": {"csrf"}, "review_id": {"review-1"}, "review_revision": {"1"}, "rule_set_version": {"0"},
		"category_id": {"cat-1"}, "unmatched_vendor_reason": {"vendor not in contacts"}, "expected_revision": {"0"}}
	request = httptest.NewRequest(http.MethodPost, base+"/approve", strings.NewReader(form.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Set("Origin", "https://app.example")
	request.AddCookie(&http.Cookie{Name: "__Host-plaiflow-session", Value: "session"})
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != 200 || approved.approval.ID == "" {
		t.Fatalf("approval status=%d body=%s", response.Code, response.Body.String())
	}
	request = httptest.NewRequest(http.MethodGet, "https://app.example/v1/o/org-1/documents/accounting.xlsx", nil)
	request.AddCookie(&http.Cookie{Name: "__Host-plaiflow-session", Value: "session"})
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != 200 || !bytes.HasPrefix(response.Body.Bytes(), []byte("PK")) {
		t.Fatalf("export status=%d body=%s", response.Code, response.Body.String())
	}
	archive, err := zip.NewReader(bytes.NewReader(response.Body.Bytes()), int64(response.Body.Len()))
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range archive.File {
		if file.Name != "xl/worksheets/sheet1.xml" {
			continue
		}
		body, _ := file.Open()
		sheet, _ := io.ReadAll(body)
		body.Close()
		if strings.Count(string(sheet), "<row ") != 2 || !strings.Contains(string(sheet), "1070.00") || !strings.Contains(string(sheet), "เดินทาง") {
			t.Fatalf("expected header and one reviewed row: %s", sheet)
		}
		return
	}
	t.Fatal("sheet missing")
}
