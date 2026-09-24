package postgres

import (
	"testing"
	"time"

	"plaiflow/api/internal/job"
)

func TestExportDateRange(t *testing.T) {
	for _, tc := range []struct {
		from, to string
		valid    bool
	}{
		{"", "", true},
		{"2026-09-01", "2026-09-24", true},
		{"2026-09-24", "2026-09-01", false},
		{"2026-02-30", "", false},
	} {
		if got := validExportDates(tc.from, tc.to); got != tc.valid {
			t.Fatalf("range %q..%q: got %v", tc.from, tc.to, got)
		}
	}
}

func TestPhase9MigrationAndEmptyExport(t *testing.T) {
	store, ctx := isolatedTestStore(t, 12)
	store.EnableReview()
	if err := store.ReadyReview(ctx); err != nil {
		t.Fatal(err)
	}
	user, org := postgresUUID(), postgresUUID()
	if _, err := store.pool.Exec(ctx, `INSERT INTO users(id) VALUES($1)`, user); err != nil {
		t.Fatal(err)
	}
	if _, err := store.CreateOrganization(ctx, user, org, "Review export", time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	if _, err := store.ListReviewQueue(ctx, user, org, "", "all", "Available", 20); err != nil {
		t.Fatal(err)
	}
	count, err := store.CountDocumentExport(ctx, user, org, "raw_documents", "Available", "2026-09-01", "2026-09-24")
	if err != nil || count != 0 {
		t.Fatalf("count=%d err=%v", count, err)
	}
	state, err := store.QueueDocumentExport(ctx, job.DocumentExportRequest{ID: postgresUUID(), OrganizationID: org,
		RequesterUserID: user, Product: "raw_documents", Status: "Available", Format: "csv",
		DateFrom: "2026-09-01", DateTo: "2026-09-24", Now: time.Now().UTC()})
	if err != nil || state.RowCount != 0 {
		t.Fatalf("state=%+v err=%v", state, err)
	}
}
