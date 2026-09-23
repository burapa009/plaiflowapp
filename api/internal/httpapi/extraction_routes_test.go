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

	"plaiflow/api/internal/extraction"
	"plaiflow/api/internal/ocr"
)

type extractionOCR struct{ ocr.Store }

func (extractionOCR) OCRState(context.Context, string, string, string) (ocr.State, error) {
	return ocr.State{JobID: "ocr-1", Status: "Completed", ObjectKey: "ocr/result.json"}, nil
}

type extractionStore struct {
	extraction.Store
	review extraction.Review
}

func (s *extractionStore) CurrentReview(context.Context, string, string, string) (extraction.Review, error) {
	return s.review, nil
}
func (s *extractionStore) SaveReview(_ context.Context, review extraction.Review, expected int) (extraction.Review, error) {
	if expected != s.review.Revision {
		return extraction.Review{}, extraction.ErrConflict
	}
	review.Revision = expected + 1
	s.review = review
	return review, nil
}
func (s *extractionStore) ListCurrentReviews(context.Context, string, string, int) ([]extraction.Review, error) {
	if s.review.ID == "" {
		return nil, nil
	}
	return []extraction.Review{s.review}, nil
}

type extractionBlobs struct{ objects map[string][]byte }

func (b *extractionBlobs) Put(_ context.Context, key string, in io.Reader) error {
	data, err := io.ReadAll(in)
	b.objects[key] = data
	return err
}
func (b *extractionBlobs) Open(_ context.Context, key string) (io.ReadCloser, error) {
	return io.NopCloser(bytes.NewReader(b.objects[key])), nil
}
func (b *extractionBlobs) Delete(_ context.Context, key string) error {
	delete(b.objects, key)
	return nil
}

func TestThaiTaxInvoiceCanBeReviewedAndExportedAsOneStructuredRow(t *testing.T) {
	now := time.Date(2026, 9, 23, 12, 0, 0, 0, time.UTC)
	authService, err := newTestAuth(now)
	if err != nil {
		t.Fatal(err)
	}
	sample, _ := json.Marshal(ocr.Result{SchemaVersion: 1, Pages: []ocr.Page{{Number: 1, Lines: []ocr.Line{
		{Text: "ใบกำกับภาษี", Confidence: .99}, {Text: "เลขที่ INV-42", Confidence: .98},
		{Text: "วันที่ 23/09/2569", Confidence: .97}, {Text: "ผู้ขาย: บริษัท ตัวอย่าง จำกัด", Confidence: .95},
		{Text: "เลขประจำตัวผู้เสียภาษี 0123456789012", Confidence: .92},
		{Text: "มูลค่าก่อนภาษี 1,000.00", Confidence: .98}, {Text: "ภาษีมูลค่าเพิ่ม 70.00", Confidence: .94},
		{Text: "ยอดรวม 1,070.00", Confidence: .99},
	}}}})
	blobs := &extractionBlobs{objects: map[string][]byte{"ocr/result.json": sample}}
	store := &extractionStore{}
	handler := New(Config{Auth: authService, Tenants: &tenantStore{allowed: "org-1"}, OCR: extractionOCR{}, OCRStorage: blobs,
		Extraction: store, ExtractionEnabled: true, Now: func() time.Time { return now }}, &fakeStore{})
	base := "https://app.example/v1/o/org-1/documents/doc-1/extraction"
	request := httptest.NewRequest(http.MethodGet, base, nil)
	request.AddCookie(&http.Cookie{Name: "__Host-plaiflow-session", Value: "session"})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"raw":"1,070.00"`) || !strings.Contains(response.Body.String(), `"normalized":"1070.00"`) {
		t.Fatalf("draft status=%d body=%s", response.Code, response.Body.String())
	}
	request = httptest.NewRequest(http.MethodGet, "https://app.example/v1/o/another-org/documents/doc-1/extraction", nil)
	request.AddCookie(&http.Cookie{Name: "__Host-plaiflow-session", Value: "session"})
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusNotFound {
		t.Fatalf("cross-organization draft status=%d", response.Code)
	}
	form := url.Values{"ocr_job_id": {"ocr-1"}, "expected_revision": {"0"}, "csrf_token": {"csrf"}, "review_ack": {"1"},
		"document_number": {"INV-42"}, "issue_date": {"2026-09-23"}, "seller_name": {"บริษัท ตัวอย่าง จำกัด"},
		"seller_tax_id": {"0123456789012"}, "subtotal": {"1000.00"}, "vat_amount": {"70.00"}, "total_amount": {"1070.00"}}
	request = httptest.NewRequest(http.MethodPost, base+"/confirm", strings.NewReader(form.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Set("Origin", "https://app.example")
	request.AddCookie(&http.Cookie{Name: "__Host-plaiflow-session", Value: "session"})
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK || store.review.Revision != 1 {
		t.Fatalf("confirmation status=%d body=%s", response.Code, response.Body.String())
	}
	request = httptest.NewRequest(http.MethodGet, "https://app.example/v1/o/org-1/documents/extraction.xlsx", nil)
	request.AddCookie(&http.Cookie{Name: "__Host-plaiflow-session", Value: "session"})
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK || !bytes.HasPrefix(response.Body.Bytes(), []byte("PK")) {
		t.Fatalf("xlsx status=%d bytes=%d body=%s", response.Code, response.Body.Len(), response.Body.String())
	}
	archive, err := zip.NewReader(bytes.NewReader(response.Body.Bytes()), int64(response.Body.Len()))
	if err != nil {
		t.Fatal(err)
	}
	var sheet string
	for _, file := range archive.File {
		if file.Name == "xl/worksheets/sheet1.xml" {
			body, err := file.Open()
			if err != nil {
				t.Fatal(err)
			}
			contents, err := io.ReadAll(body)
			body.Close()
			if err != nil {
				t.Fatal(err)
			}
			sheet = string(contents)
		}
	}
	if strings.Count(sheet, "<row ") != 2 || !strings.Contains(sheet, "1070.00") || !strings.Contains(sheet, "INV-42") {
		t.Fatalf("expected header and one tax-invoice row, got %s", sheet)
	}
	request = httptest.NewRequest(http.MethodGet, "https://app.example/v1/o/org-1/documents/extraction.csv", nil)
	request.AddCookie(&http.Cookie{Name: "__Host-plaiflow-session", Value: "session"})
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK || strings.Count(response.Body.String(), "\r\n") != 2 || !strings.Contains(response.Body.String(), "INV-42") {
		t.Fatalf("expected header and one CSV row, status=%d body=%q", response.Code, response.Body.String())
	}
}
