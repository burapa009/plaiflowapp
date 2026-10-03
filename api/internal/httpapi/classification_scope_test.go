package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"plaiflow/api/internal/classification"
	"plaiflow/api/internal/ocr"
	"plaiflow/api/internal/tenant"
)

type scopedClassifierStore struct {
	classification.Store
	saved     int
	corrected int
	record    classification.Record
}

func (s *scopedClassifierStore) CorrectDocumentType(_ context.Context, _, _, _, _, _, value string, _ time.Time) (classification.Record, error) {
	s.corrected++
	s.record.EffectiveType = value
	return s.record, nil
}

func TestClassificationPilotCorrectionStillRequiresOwnerAndSession(t *testing.T) {
	now := time.Now().UTC()
	auth, err := newTestAuth(now)
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name, org, allowed string
		role               tenant.Role
		session            bool
		want               int
	}{
		{"owner", "pilot", "pilot", tenant.Owner, true, 200},
		{"member", "pilot", "pilot", tenant.Member, true, 403},
		{"anonymous", "pilot", "pilot", tenant.Owner, false, 401},
		{"cross tenant", "pilot", "control", tenant.Owner, true, 404},
		{"outside pilot", "control", "control", tenant.Owner, true, 404},
	} {
		t.Run(test.name, func(t *testing.T) {
			classes := &scopedClassifierStore{}
			handler := New(Config{Auth: auth, Tenants: &tenantStore{allowed: test.allowed, role: test.role}, OCR: extractionOCR{}, OCRStorage: &extractionBlobs{objects: map[string][]byte{}}, Extraction: &extractionStore{}, ExtractionEnabled: true, Classification: classes, ClassificationEnabled: true, ClassificationOrganizations: map[string]bool{"pilot": true}, OCRPilotOrganizations: map[string]bool{"*": true}, Now: func() time.Time { return now }}, &fakeStore{})
			req := httptest.NewRequest(http.MethodPost, "https://app.example/v1/o/"+test.org+"/documents/doc-1/document-type", strings.NewReader("document_type=pre_receipt&ocr_job_id=ocr-1&expected_type=receipt&csrf_token=csrf"))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			req.Header.Set("Origin", "https://app.example")
			if test.session {
				req.AddCookie(&http.Cookie{Name: "__Host-plaiflow-session", Value: "session"})
			}
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, req)
			if response.Code != test.want {
				t.Fatalf("status=%d want=%d body=%s", response.Code, test.want, response.Body.String())
			}
			wantWrites := 0
			if test.want == 200 {
				wantWrites = 1
			}
			if classes.corrected != wantWrites {
				t.Fatalf("writes=%d want=%d", classes.corrected, wantWrites)
			}
		})
	}
}

func (s *scopedClassifierStore) GetClassification(context.Context, string, string, string) (classification.Record, error) {
	return s.record, nil
}
func (s *scopedClassifierStore) SaveClassification(_ context.Context, _, _, _ string, result classification.Result) error {
	s.saved++
	s.record = classification.Record{Result: result, EffectiveType: result.DocumentType}
	return nil
}

func TestClassificationPilotLeavesOtherExtractionUnchanged(t *testing.T) {
	now := time.Now().UTC()
	auth, err := newTestAuth(now)
	if err != nil {
		t.Fatal(err)
	}
	sample, _ := json.Marshal(ocr.Result{SchemaVersion: 3, Pages: []ocr.Page{{Number: 1, Text: "ใบเสร็จก่อนรับเงิน\nรหัสลูกหนี้ ก-0002\nยอดคงค้าง 100"}}})
	for _, test := range []struct {
		org, want string
		saved     int
	}{{"pilot", "pre_receipt", 1}, {"control", "receipt", 0}} {
		t.Run(test.org, func(t *testing.T) {
			classes := &scopedClassifierStore{}
			handler := New(Config{Auth: auth, Tenants: &tenantStore{allowed: test.org}, OCR: extractionOCR{}, OCRStorage: &extractionBlobs{objects: map[string][]byte{"ocr/result.json": sample}}, Extraction: &extractionStore{}, ExtractionEnabled: true, Classification: classes, ClassificationEnabled: true, ClassificationOrganizations: map[string]bool{"pilot": true}, OCRPilotOrganizations: map[string]bool{"*": true}, Now: func() time.Time { return now }}, &fakeStore{})
			request := httptest.NewRequest(http.MethodGet, "https://app.example/v1/o/"+test.org+"/documents/doc-1/extraction", nil)
			request.AddCookie(&http.Cookie{Name: "__Host-plaiflow-session", Value: "session"})
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != 200 || !strings.Contains(response.Body.String(), `"document_type":"`+test.want+`"`) || classes.saved != test.saved {
				t.Fatalf("status=%d saved=%d body=%s", response.Code, classes.saved, response.Body.String())
			}
			request = httptest.NewRequest(http.MethodGet, "https://app.example/v1/o/"+test.org+"/documents/doc-1/classification", nil)
			request.AddCookie(&http.Cookie{Name: "__Host-plaiflow-session", Value: "session"})
			response = httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			want := 200
			if test.org == "control" {
				want = 404
			}
			if response.Code != want {
				t.Fatalf("classification status=%d want=%d", response.Code, want)
			}
		})
	}
}

func TestClassificationScopeFailsClosed(t *testing.T) {
	for _, test := range []struct {
		enabled bool
		orgs    map[string]bool
		want    bool
	}{
		{false, map[string]bool{"*": true}, false},
		{true, nil, false},
		{true, map[string]bool{"other": true}, false},
		{true, map[string]bool{"pilot": true}, true},
		{true, map[string]bool{"*": true}, true},
	} {
		s := server{config: Config{ClassificationEnabled: test.enabled, ClassificationOrganizations: test.orgs}}
		if got := s.classificationAllowed("pilot"); got != test.want {
			t.Fatalf("allowed=%v want=%v", got, test.want)
		}
	}
}
