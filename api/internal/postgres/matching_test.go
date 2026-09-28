package postgres

import (
	"os"
	"testing"
)

func TestRunPodMatchingMigrationRoundTrip(t *testing.T) {
	store, ctx := isolatedTestStore(t, 18)
	up, err := os.ReadFile("../../migrations/000019_runpod_ocr.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	down, err := os.ReadFile("../../migrations/000019_runpod_ocr.down.sql")
	if err != nil {
		t.Fatal(err)
	}
	for _, sql := range []string{string(up), string(down), string(up)} {
		if _, err := store.pool.Exec(ctx, sql); err != nil {
			t.Fatal(err)
		}
	}
	var available bool
	if err := store.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM pg_extension WHERE extname='pg_trgm')`).Scan(&available); err != nil || !available {
		t.Fatalf("pg_trgm available=%v err=%v", available, err)
	}
}
