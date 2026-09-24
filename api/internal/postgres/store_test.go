package postgres

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"plaiflow/api/internal/business"
	"plaiflow/api/internal/inbound"
)

func TestBusinessDuplicateKeysUseTaxAndContactCode(t *testing.T) {
	keys := duplicateKeys(business.Contact{Country: "TH", TaxID: "0105552117718", BranchCode: "00000", ContactCode: "Vendor-1"})
	if len(keys) != 2 || keys[0] != "tax:TH:0105552117718:00000" || keys[1] != "code:vendor-1" {
		t.Fatalf("keys=%v", keys)
	}
}

func TestReviewOffKeepsPhase7ReadsOnMigration11(t *testing.T) {
	store, ctx := isolatedTestStore(t, 11)
	user, org := postgresUUID(), postgresUUID()
	if _, err := store.pool.Exec(ctx, `INSERT INTO users(id) VALUES($1)`, user); err != nil {
		t.Fatal(err)
	}
	if _, err := store.CreateOrganization(ctx, user, org, "Review gate", time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	if _, err := store.ListCurrentReviews(ctx, user, org, 100); err != nil {
		t.Fatalf("Phase 7 read requires Phase 9 tables while review is disabled: %v", err)
	}
}

func TestIdempotentInsertAndWorkerLifecycle(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}
	ctx := context.Background()
	store, err := New(ctx, databaseURL, 2, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	if _, err := store.pool.Exec(ctx, "TRUNCATE inbound_events, worker_heartbeats"); err != nil {
		t.Fatal(err)
	}
	event := inbound.Event{Provider: "line", Channel: "test", ProviderEventID: "duplicate", Type: "message", Payload: json.RawMessage(`{"type":"message"}`), OccurredAt: time.Now().UTC()}
	if err := store.InsertEvents(ctx, []inbound.Event{event}); err != nil {
		t.Fatal(err)
	}
	if err := store.InsertEvents(ctx, []inbound.Event{event}); err != nil {
		t.Fatal(err)
	}
	events, err := store.Claim(ctx, 25, time.Minute)
	if err != nil || len(events) != 1 {
		t.Fatalf("claim: events=%d err=%v", len(events), err)
	}
	if err := store.Complete(ctx, events[0].ID, inbound.Processed, ""); err != nil {
		t.Fatal(err)
	}
	snapshot, err := store.Snapshot(ctx)
	if err != nil || snapshot.Counts.Processed != 1 {
		t.Fatalf("snapshot=%+v err=%v", snapshot, err)
	}
}

func TestInsertEventAcceptsEmptyOptionalSourceFields(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}
	ctx := context.Background()
	store, err := New(ctx, databaseURL, 2, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	event := inbound.Event{Provider: "line", Channel: "test", ProviderEventID: "empty-source-fields", Type: "message", Payload: json.RawMessage(`{}`), OccurredAt: time.Now().UTC()}
	if err := store.InsertEvents(ctx, []inbound.Event{event}); err != nil {
		t.Fatal(err)
	}
	var exists bool
	if err := store.pool.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM inbound_events WHERE provider='line' AND channel='test' AND provider_event_id='empty-source-fields')").Scan(&exists); err != nil || !exists {
		t.Fatalf("exists=%v err=%v", exists, err)
	}
}

func TestCleanupKeepsIdempotencyAndActiveWork(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}
	ctx := context.Background()
	store, err := New(ctx, databaseURL, 2, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	if _, err := store.pool.Exec(ctx, "TRUNCATE inbound_events, worker_heartbeats"); err != nil {
		t.Fatal(err)
	}
	_, err = store.pool.Exec(ctx, `INSERT INTO inbound_events
        (provider, channel, provider_event_id, event_type, payload, status, occurred_at, received_at)
        VALUES ('line','test','payload-expired','message','{}','Processed',now()-interval '31 days',now()-interval '31 days'),
               ('line','test','metadata-expired','message','{}','Processed',now()-interval '91 days',now()-interval '91 days'),
               ('line','test','active-retry','message','{}','Retryable',now()-interval '91 days',now()-interval '91 days')`)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Cleanup(ctx); err != nil {
		t.Fatal(err)
	}
	var payloadMissing bool
	if err := store.pool.QueryRow(ctx, "SELECT payload IS NULL FROM inbound_events WHERE provider_event_id='payload-expired'").Scan(&payloadMissing); err != nil || !payloadMissing {
		t.Fatalf("payload retained: %v", err)
	}
	var activeExists bool
	if err := store.pool.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM inbound_events WHERE provider_event_id='active-retry')").Scan(&activeExists); err != nil || !activeExists {
		t.Fatalf("active retry deleted: %v", err)
	}
	var expiredExists bool
	if err := store.pool.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM inbound_events WHERE provider_event_id='metadata-expired')").Scan(&expiredExists); err != nil || expiredExists {
		t.Fatalf("expired metadata retained: %v", err)
	}
}
