package postgres

import (
	"crypto/sha256"
	"testing"
	"time"

	"plaiflow/api/internal/inbound"
	"plaiflow/api/internal/tenant"
)

func TestLINEGroupLinkCodeIsSingleUseAndBoundToSender(t *testing.T) {
	store, ctx := isolatedTestStore(t, 11)
	user, organization := postgresUUID(), postgresUUID()
	if _, err := store.pool.Exec(ctx, `INSERT INTO users(id) VALUES($1)`, user); err != nil {
		t.Fatal(err)
	}
	if _, err := store.CreateOrganization(ctx, user, organization, "LINE group test", time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	if _, err := store.pool.Exec(ctx, `INSERT INTO auth_identities(id,user_id,provider,issuer,subject) VALUES($1,$2,'line','https://access.line.me','line-owner')`, postgresUUID(), user); err != nil {
		t.Fatal(err)
	}
	issue := func(value string) inbound.Event {
		t.Helper()
		hash := sha256.Sum256([]byte(value))
		now := time.Now().UTC()
		err := store.CreateLineLinkCode(ctx, tenant.LineCodeCreate{ID: postgresUUID(), OrganizationID: organization, ActorUserID: user, Channel: "channel", CodeHash: hash[:], Now: now, ExpiresAt: now.Add(10 * time.Minute)})
		if err != nil {
			t.Fatal(err)
		}
		return inbound.Event{Provider: "line", Channel: "channel", SourceType: "group", SourceGroupID: "group-1", SourceUserID: "line-owner", LinkCodeHash: hash[:]}
	}
	event := issue("first-code")
	event.SourceUserID = "other-user"
	if linked, err := store.ConsumeLINEGroupCode(ctx, event); err != nil || linked {
		t.Fatalf("wrong sender linked=%v err=%v", linked, err)
	}
	event.SourceUserID = "line-owner"
	if linked, err := store.ConsumeLINEGroupCode(ctx, event); err != nil || !linked {
		t.Fatalf("valid code linked=%v err=%v", linked, err)
	}
	if linked, err := store.ConsumeLINEGroupCode(ctx, event); err != nil || linked {
		t.Fatalf("replayed code linked=%v err=%v", linked, err)
	}
	connections, err := store.ListLineConnections(ctx, organization, user)
	if err != nil || len(connections) != 1 || connections[0].Status != "connected" {
		t.Fatalf("connections=%+v err=%v", connections, err)
	}
	if err := store.DisconnectLineConnection(ctx, organization, user, connections[0].ID, time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	event = issue("second-code")
	if linked, err := store.ConsumeLINEGroupCode(ctx, event); err != nil || !linked {
		t.Fatalf("reconnect linked=%v err=%v", linked, err)
	}
	if disconnected, err := store.DisconnectLINEGroupBySource(ctx, inbound.Event{Provider: "line", Type: "leave", Channel: "channel", SourceType: "group", SourceGroupID: "group-1"}); err != nil || !disconnected {
		t.Fatalf("bot leave disconnected=%v err=%v", disconnected, err)
	}
	connections, err = store.ListLineConnections(ctx, organization, user)
	if err != nil || len(connections) != 2 || connections[0].Status != "disconnected" || connections[1].Status != "disconnected" {
		t.Fatalf("connections after bot leave=%+v err=%v", connections, err)
	}
}
