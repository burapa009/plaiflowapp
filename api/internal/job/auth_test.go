package job

import (
	"testing"
	"time"
)

func TestWorkerTokenIsShortLivedEnvironmentAndScopeBound(t *testing.T) {
	now := time.Date(2026, 9, 21, 4, 0, 0, 0, time.UTC)
	auth, err := NewWorkerAuth([]byte("01234567890123456789012345678901"), "staging", []string{"jobs:claim", "jobs:heartbeat"})
	if err != nil {
		t.Fatal(err)
	}
	token, err := auth.Sign("worker-1", []string{"jobs:claim"}, now, 2*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	identity, err := auth.Verify(token, "jobs:claim", now.Add(time.Minute))
	if err != nil || identity.WorkerID != "worker-1" {
		t.Fatalf("identity=%+v err=%v", identity, err)
	}
	if _, err := auth.Verify(token, "jobs:heartbeat", now.Add(time.Minute)); err == nil {
		t.Fatal("token accepted an ungranted scope")
	}
	if _, err := auth.Verify(token, "jobs:claim", now.Add(3*time.Minute)); err == nil {
		t.Fatal("expired token accepted")
	}
	production, err := NewWorkerAuth([]byte("01234567890123456789012345678901"), "production", []string{"jobs:claim"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := production.Verify(token, "jobs:claim", now.Add(time.Minute)); err == nil {
		t.Fatal("staging token accepted in production")
	}
}
