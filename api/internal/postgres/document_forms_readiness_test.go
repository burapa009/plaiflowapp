package postgres

import "testing"

func TestDocumentFormsReadinessRequiresMigration25(t *testing.T) {
	store, ctx := isolatedTestStore(t, 25)
	// The isolated helper applies SQL files directly, without the migration CLI.
	if _, err := store.pool.Exec(ctx, "CREATE TABLE schema_migrations (version bigint PRIMARY KEY, dirty boolean NOT NULL); INSERT INTO schema_migrations VALUES (25, false)"); err != nil {
		t.Fatal(err)
	}
	if err := store.Ready(ctx); err != nil {
		t.Fatalf("migration 25 not ready: %v", err)
	}
	if _, err := store.pool.Exec(ctx, "UPDATE schema_migrations SET version=24"); err != nil {
		t.Fatal(err)
	}
	if err := store.Ready(ctx); err == nil {
		t.Fatal("readiness accepted database without document forms migration")
	}
	if _, err := store.pool.Exec(ctx, "UPDATE schema_migrations SET version=25, dirty=true"); err != nil {
		t.Fatal(err)
	}
	if err := store.Ready(ctx); err == nil {
		t.Fatal("readiness accepted dirty migration")
	}
}
