package httpapi

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"plaiflow/api/internal/job"
	"plaiflow/api/internal/ocr"
)

type ocrProtocolStore = ocr.Store
type ocrJobStore = job.Store
type ocrStoreDouble struct {
	ocrProtocolStore
	ocrJobStore
	claims    int
	lastClaim job.ClaimCommand
}

func (s *ocrStoreDouble) ClaimJobs(_ context.Context, c job.ClaimCommand) ([]job.Claimed, error) {
	s.claims++
	s.lastClaim = c
	return []job.Claimed{}, nil
}

func TestOCRProviderClaimIsPilotScoped(t *testing.T) {
	now := time.Now().UTC()
	auth, _ := job.NewWorkerAuth(bytes.Repeat([]byte{1}, 32), "staging", []string{"ocr:claim"})
	store := &ocrStoreDouble{}
	handler := New(Config{OCR: store, OCRJobs: store, OCRAuth: auth, OCRDefaultProvider: "railway",
		OCRRunPodOrganizations: map[string]bool{"org-pilot": true}, Now: func() time.Time { return now }}, nil)
	token, _ := auth.Sign("ocr-remote", []string{"ocr:claim"}, now, time.Minute)
	req := httptest.NewRequest("POST", "/internal/v1/ocr/jobs/claim", bytes.NewBufferString(`{"provider":"runpod"}`))
	req.Header.Set("Authorization", "Bearer "+token)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, req)
	if response.Code != 200 || store.lastClaim.OCRProvider != "runpod" || store.lastClaim.OCRDefaultProvider != "railway" || len(store.lastClaim.OCRRunPodOrganizations) != 1 || store.lastClaim.OCRRunPodOrganizations[0] != "org-pilot" {
		t.Fatalf("provider claim %+v status %d", store.lastClaim, response.Code)
	}
}

func TestOCRWorkerCannotClaimExportOrUseExportCredential(t *testing.T) {
	now := time.Now().UTC()
	auth, _ := job.NewWorkerAuth(bytes.Repeat([]byte{1}, 32), "staging", []string{"ocr:claim", "ocr:heartbeat", "ocr:input", "ocr:submit", "ocr:fail"})
	store := &ocrStoreDouble{}
	handler := New(Config{OCR: store, OCRJobs: store, OCRAuth: auth, Now: func() time.Time { return now }}, nil)
	token, _ := auth.Sign("ocr-1", []string{"ocr:claim"}, now, time.Minute)
	req := httptest.NewRequest("POST", "/internal/v1/ocr/jobs/claim", bytes.NewBufferString(`{"kinds":["export"]}`))
	req.Header.Set("Authorization", "Bearer "+token)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, req)
	if response.Code != http.StatusBadRequest || store.claims != 0 {
		t.Fatalf("export claim status %d", response.Code)
	}
	token, _ = auth.Sign("ocr-1", []string{"ocr:claim"}, now, time.Minute)
	req = httptest.NewRequest("POST", "/internal/v1/ocr/jobs/claim", bytes.NewBufferString(`{}`))
	req.Header.Set("Authorization", "Bearer "+token)
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, req)
	if response.Code != http.StatusOK || store.claims != 1 {
		t.Fatalf("OCR claim status %d", response.Code)
	}
	other, _ := job.NewWorkerAuth(bytes.Repeat([]byte{2}, 32), "staging", []string{"ocr:claim"})
	token, _ = other.Sign("ocr-1", []string{"ocr:claim"}, now, time.Minute)
	req = httptest.NewRequest("POST", "/internal/v1/ocr/jobs/claim", bytes.NewBufferString(`{}`))
	req.Header.Set("Authorization", "Bearer "+token)
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, req)
	if response.Code != http.StatusUnauthorized || store.claims != 1 {
		t.Fatalf("wrong key status %d", response.Code)
	}
}
