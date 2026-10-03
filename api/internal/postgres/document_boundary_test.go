package postgres

import (
	"errors"
	"strings"
	"testing"
	"time"

	"plaiflow/api/internal/document"
	"plaiflow/api/internal/tenant"
)

func TestDocumentObjectIDCannotCrossOrganization(t *testing.T) {
	store, ctx := isolatedTestStore(t, 16)
	firstUser, secondUser := postgresUUID(), postgresUUID()
	firstOrg, secondOrg := postgresUUID(), postgresUUID()
	for _, user := range []string{firstUser, secondUser} {
		if _, err := store.pool.Exec(ctx, `INSERT INTO users(id) VALUES($1)`, user); err != nil {
			t.Fatal(err)
		}
	}
	now := time.Now().UTC()
	for _, item := range []struct{ user, org string }{{firstUser, firstOrg}, {secondUser, secondOrg}} {
		if _, err := store.CreateOrganization(ctx, item.user, item.org, "Document boundary", now); err != nil {
			t.Fatal(err)
		}
	}
	result, err := store.CommitPrepared(ctx, document.CommitInput{AcceptInput: document.AcceptInput{
		OrganizationID: secondOrg, ActorUserID: secondUser, AttemptID: postgresUUID(), OriginKey: "document-boundary",
		Filename: "invoice.pdf", Channel: "Web", Now: now,
	}, TemporaryKey: "quarantine/test-boundary", SHA256: strings.Repeat("a", 64), MIME: "application/pdf", Size: 12})
	if err != nil || !result.Accepted {
		t.Fatalf("seed accepted=%v err=%v", result.Accepted, err)
	}
	if _, err := store.GetDocument(ctx, secondUser, secondOrg, result.Document.ID); err != nil {
		t.Fatalf("permitted owner read: %v", err)
	}
	for _, item := range []struct{ user, org string }{{firstUser, firstOrg}, {firstUser, secondOrg}} {
		if _, err := store.GetDocument(ctx, item.user, item.org, result.Document.ID); !errors.Is(err, tenant.ErrNotFound) {
			t.Fatalf("foreign object read user=%s scope=%s err=%v", item.user, item.org, err)
		}
	}
}
