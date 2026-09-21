package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"plaiflow/api/internal/job"
)

type jobStoreDouble struct {
	claim      job.ClaimCommand
	claimCalls int
	claimed    []job.Claimed
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
func (*jobStoreDouble) ReadExportPage(context.Context, job.LeaseCommand, string, int) (job.ExportPage, error) {
	return job.ExportPage{}, nil
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
