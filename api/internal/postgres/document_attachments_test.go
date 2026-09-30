package postgres

import (
	"errors"
	"strings"
	"testing"
	"time"

	"plaiflow/api/internal/document"
	"plaiflow/api/internal/tenant"
)

func TestDocumentAttachmentsStayInTheirParentOrganization(t *testing.T) {
	store, ctx := isolatedTestStore(t, 21)
	owner, otherOwner, member := postgresUUID(), postgresUUID(), postgresUUID()
	org, otherOrg := postgresUUID(), postgresUUID()
	now := time.Now().UTC()
	for _, user := range []string{owner, otherOwner, member} {
		if _, err := store.pool.Exec(ctx, `INSERT INTO users(id) VALUES($1)`, user); err != nil {
			t.Fatal(err)
		}
	}
	for _, item := range []struct{ user, org string }{{owner, org}, {otherOwner, otherOrg}} {
		if _, err := store.CreateOrganization(ctx, item.user, item.org, "Attachment test", now); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := store.pool.Exec(ctx, `INSERT INTO memberships(organization_id,user_id,role) VALUES($1,$2,'Member')`, org, member); err != nil {
		t.Fatal(err)
	}
	accept := func(user, organization, target, key, hash string) (document.CommitResult, error) {
		return store.CommitPrepared(ctx, document.CommitInput{AcceptInput: document.AcceptInput{
			OrganizationID: organization, ActorUserID: user, AttemptID: postgresUUID(), OriginKey: key,
			Filename: key + ".png", Channel: "Web", Now: now, AttachToDocumentID: target,
		}, TemporaryKey: "test/" + key, SHA256: strings.Repeat(hash, 64), MIME: "image/png", Size: 10})
	}
	parent, err := accept(owner, org, "", "parent", "a")
	if err != nil {
		t.Fatal(err)
	}
	child, err := accept(owner, org, parent.Document.ID, "child", "b")
	if err != nil || !child.Accepted {
		t.Fatalf("attach: %+v %v", child, err)
	}
	again, err := accept(owner, org, parent.Document.ID, "child", "b")
	if err != nil || !again.Duplicate {
		t.Fatalf("retry: %+v %v", again, err)
	}
	detail, err := store.DocumentDetail(ctx, owner, org, parent.Document.ID)
	if err != nil || len(detail.Attachments) != 1 || detail.Attachments[0].ID != child.Document.ID {
		t.Fatalf("attachments: %+v %v", detail.Attachments, err)
	}
	if _, err := accept(owner, org, parent.Document.ID, "self", "a"); !errors.Is(err, document.ErrStatusConflict) {
		t.Fatalf("self attachment: %v", err)
	}
	if _, err := accept(otherOwner, otherOrg, parent.Document.ID, "foreign", "c"); !errors.Is(err, tenant.ErrNotFound) {
		t.Fatalf("cross-organization attachment: %v", err)
	}
	if _, err := accept(member, org, parent.Document.ID, "hidden-parent", "d"); !errors.Is(err, tenant.ErrNotFound) {
		t.Fatalf("member attachment to hidden parent: %v", err)
	}
}
