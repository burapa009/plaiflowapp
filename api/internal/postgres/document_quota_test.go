package postgres

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"plaiflow/api/internal/document"
)

func TestDocumentQuotaRejectsWithoutSQLFailure(t *testing.T) {
	store, ctx := isolatedTestStore(t, 25)
	user, org := postgresUUID(), postgresUUID()
	now := time.Now().UTC()
	if _, err := store.pool.Exec(ctx, "INSERT INTO users(id) VALUES($1)", user); err != nil {
		t.Fatal(err)
	}
	if _, err := store.CreateOrganization(ctx, user, org, "Quota regression", now); err != nil {
		t.Fatal(err)
	}
	for i := 0; i <= 30; i++ {
		input := document.CommitInput{AcceptInput: document.AcceptInput{
			OrganizationID: org, ActorUserID: user, AttemptID: postgresUUID(), OriginKey: fmt.Sprint(i),
			Filename: "quota.png", Channel: "Web", Now: now,
		}, TemporaryKey: "test/quota", SHA256: fmt.Sprintf("%064x", i), MIME: "image/png", Size: 10}
		result, err := store.CommitPrepared(ctx, input)
		if i < 30 {
			if err != nil || !result.Accepted {
				t.Fatalf("document %d: accepted=%v err=%v", i+1, result.Accepted, err)
			}
			continue
		}
		if !errors.Is(err, document.ErrQuota) {
			t.Fatalf("expected quota rejection, got %v", err)
		}
		var status, reason string
		var expires time.Time
		if err := store.pool.QueryRow(ctx, "SELECT status,rejection_code,expires_at FROM document_intake_attempts WHERE id=$1", input.AttemptID).Scan(&status, &reason, &expires); err != nil {
			t.Fatal(err)
		}
		if status != "Rejected" || reason != "quota_exhausted" || expires.Sub(now).Round(time.Second) != 90*24*time.Hour {
			t.Fatalf("rejection not recorded: %s %s %v", status, reason, expires)
		}
	}
}
