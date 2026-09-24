package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"plaiflow/api/internal/extraction"
	"plaiflow/api/internal/job"
)

type jobStoreDouble struct {
	claim      job.ClaimCommand
	claimCalls int
	claimed    []job.Claimed
	page       job.ExportPage
}

func (s *jobStoreDouble) ClaimJobs(_ context.Context, command job.ClaimCommand) ([]job.Claimed, error) {
	s.claimCalls++
	s.claim = command
	return s.claimed, nil
}
func (*jobStoreDouble) HeartbeatJob(context.Context, job.LeaseCommand, time.Duration) (time.Time, error) {
	return time.Time{}, nil
}
func (*jobStoreDouble) FailJob(context.Context, job.FailureCommand) error { return nil }
func (s *jobStoreDouble) ReadExportPage(context.Context, job.LeaseCommand, string, int) (job.ExportPage, error) {
	return s.page, nil
}

func TestWorkerApprovedExportPageContainsOnlyVerifiedRow(t *testing.T) {
	now := time.Date(2026, 9, 24, 8, 0, 0, 0, time.UTC)
	approvedReview := extraction.Review{ID: "review-1", OrganizationID: "org-1", DocumentID: "doc-1", OCRJobID: "ocr-1",
		Revision: 2, DocumentType: "tax_invoice", Values: map[string]string{"seller_name": "บริษัท ตัวอย่าง จำกัด",
			"seller_tax_id": "0123456789012", "total_amount": "1070.00"}}
	content, _ := json.Marshal(approvedReview)
	store := &jobStoreDouble{page: job.ExportPage{Product: "approved_suggestions", Done: true,
		Entries: []job.DocumentExportEntry{{Review: job.ReviewExportMeta{ID: "review-1", OrganizationID: "org-1",
			DocumentID: "doc-1", OCRJobID: "ocr-1", Revision: 2, ObjectKey: "review.json", ConfirmedAt: now},
			Approval: job.ApprovalExportMeta{ID: "approval-1", DocumentID: "doc-1", ReviewID: "review-1", ReviewRevision: 2,
				CategoryID: "cat-1", CategoryName: "ค่าเดินทาง", Basis: "human_reviewed", ApprovedAt: now}}}}}
	auth, _ := job.NewWorkerAuth([]byte("01234567890123456789012345678901"), "staging", []string{"jobs:read"})
	handler := New(Config{Jobs: store, JobWorkerAuth: auth, ReviewEnabled: true,
		OCRStorage: &extractionBlobs{objects: map[string][]byte{"review.json": content}}, Now: func() time.Time { return now }}, &fakeStore{})
	token, _ := auth.Sign("worker-1", []string{"jobs:read"}, now, time.Minute)
	request := httptest.NewRequest(http.MethodGet, "/internal/v1/jobs/job-1/export-rows", nil)
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("X-Job-Attempt", "attempt-1")
	request.Header.Set("X-Job-Lease", "lease-1")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != 200 {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	var page job.ExportPage
	if err := json.Unmarshal(response.Body.Bytes(), &page); err != nil {
		t.Fatal(err)
	}
	if len(page.Header) != 21 || len(page.Values) != 1 || page.Values[0][6] != "บริษัท ตัวอย่าง จำกัด" ||
		page.Values[0][7] != "0123456789012" || page.Values[0][15] != "1070.00" ||
		!strings.Contains(response.Body.String(), "accounting_suggestions_v1") {
		t.Fatalf("wrong approved row: %+v", page)
	}
}
func (*jobStoreDouble) CompleteExport(context.Context, job.CompleteExportCommand) (job.ExportArtifact, error) {
	return job.ExportArtifact{}, nil
}
func (*jobStoreDouble) AuthorizeExportArtifact(context.Context, string, string, string) (job.ExportArtifact, error) {
	return job.ExportArtifact{}, nil
}
func (*jobStoreDouble) RecordArtifactDownload(context.Context, string, string, string, time.Time) error {
	return nil
}

func TestWorkerClaimRequiresScopedShortLivedToken(t *testing.T) {
	now := time.Date(2026, 9, 21, 4, 0, 0, 0, time.UTC)
	auth, err := job.NewWorkerAuth([]byte("01234567890123456789012345678901"), "staging", []string{"jobs:claim", "jobs:heartbeat", "jobs:fail", "jobs:artifact", "jobs:read"})
	if err != nil {
		t.Fatal(err)
	}
	store := &jobStoreDouble{claimed: []job.Claimed{{Job: job.Job{ID: "job-1", OrganizationID: "org-1", Kind: job.Export, Status: job.Running, AttemptID: "attempt-1", AttemptCount: 1}, LeaseToken: "lease-token", LeaseExpiresAt: now.Add(2 * time.Minute)}}}
	handler := New(Config{LineSecret: "secret", LineChannel: "channel", DashboardTokens: []string{"dashboard"}, Jobs: store, JobWorkerAuth: auth, Now: func() time.Time { return now }}, &fakeStore{})

	token, err := auth.Sign("worker-1", []string{"jobs:claim"}, now, 2*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	body := []byte(`{"kinds":["export"],"limit":1}`)
	request := httptest.NewRequest(http.MethodPost, "/internal/v1/jobs/claim", bytes.NewReader(body))
	request.Header.Set("Authorization", "Bearer "+token)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK || store.claimCalls != 1 || store.claim.WorkerID != "worker-1" || store.claim.Limit != 1 || len(store.claim.Kinds) != 1 || store.claim.Kinds[0] != job.Export {
		t.Fatalf("status=%d calls=%d command=%+v body=%s", response.Code, store.claimCalls, store.claim, response.Body.String())
	}
	var result struct {
		Jobs []job.Claimed `json:"jobs"`
	}
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil || len(result.Jobs) != 1 || result.Jobs[0].LeaseToken != "lease-token" {
		t.Fatalf("result=%+v err=%v", result, err)
	}

	request = httptest.NewRequest(http.MethodPost, "/internal/v1/jobs/claim", bytes.NewReader(body))
	request.Header.Set("Authorization", "Bearer invalid")
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusUnauthorized || store.claimCalls != 1 {
		t.Fatalf("unauthorized status=%d calls=%d", response.Code, store.claimCalls)
	}
}
