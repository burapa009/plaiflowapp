package postgres

import (
	"errors"
	"testing"
	"time"

	"plaiflow/api/internal/tenant"
)

func TestFirmGrantAssignmentAndImmediateRevoke(t *testing.T) {
	store, ctx := isolatedTestStore(t, 13)
	var err error
	firmOwner, staff, clientOwner, outsider := postgresUUID(), postgresUUID(), postgresUUID(), postgresUUID()
	firmID, clientID, otherID := postgresUUID(), postgresUUID(), postgresUUID()
	grantID, docID, periodID := postgresUUID(), postgresUUID(), postgresUUID()
	now := time.Now().UTC().Truncate(time.Second)
	if _, err = store.pool.Exec(ctx, `INSERT INTO users(id) VALUES($1),($2),($3),($4)`, firmOwner, staff, clientOwner, outsider); err != nil {
		t.Fatal(err)
	}
	if _, err = store.pool.Exec(ctx, `INSERT INTO organizations(id,name) VALUES($1,'Accounting firm'),($2,'Client A'),($3,'Other')`, firmID, clientID, otherID); err != nil {
		t.Fatal(err)
	}
	if _, err = store.pool.Exec(ctx, `INSERT INTO memberships(organization_id,user_id,role) VALUES
		($1,$2,'Owner'),($1,$3,'Member'),($4,$5,'Owner'),($6,$7,'Owner')`, firmID, firmOwner, staff, clientID, clientOwner, otherID, outsider); err != nil {
		t.Fatal(err)
	}
	if _, err = store.pool.Exec(ctx, `INSERT INTO organization_plan_periods(id,organization_id,plan_key,source,starts_at,ends_at,created_at)
		VALUES($1,$2,'AccountingFirm','manual',$3,$4,$3)`, periodID, firmID, now.Add(-time.Hour), now.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	if _, err = store.pool.Exec(ctx, `INSERT INTO document_usage_periods(organization_id,starts_at,ends_at,timezone)
		VALUES($1,$2,$3,'Asia/Bangkok')`, clientID, now.Add(-time.Hour), now.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	if _, err = store.pool.Exec(ctx, `INSERT INTO documents(id,organization_id,content_sha256,storage_key,display_filename,
		detected_mime,byte_size,status,submitted_by_user_id,accepted_at,updated_at,usage_period_start)
		VALUES($1,$2,$3,'test/private','invoice.pdf','application/pdf',12,'Available',$4,$5,$5,$6)`,
		docID, clientID, make([]byte, 32), clientOwner, now, now.Add(-time.Hour)); err != nil {
		t.Fatal(err)
	}
	if _, err = store.GetFirmDocument(ctx, staff, firmID, clientID, docID, "read"); !errors.Is(err, tenant.ErrNotFound) {
		t.Fatalf("ungranted read: %v", err)
	}
	if _, err = store.RequestGrant(ctx, firmOwner, firmID, clientID, grantID, now); err != nil {
		t.Fatal(err)
	}
	if _, err = store.RequestGrant(ctx, firmOwner, firmID, clientID, postgresUUID(), now); !errors.Is(err, tenant.ErrConflict) {
		t.Fatalf("duplicate relationship: %v", err)
	}
	if grants, listErr := store.ListGrants(ctx, outsider, otherID); listErr != nil || len(grants) != 0 {
		t.Fatalf("foreign grant list=%+v err=%v", grants, listErr)
	}
	if err = store.AssignStaff(ctx, firmOwner, firmID, grantID, staff, now); !errors.Is(err, tenant.ErrNotFound) {
		t.Fatalf("pending assignment: %v", err)
	}
	if err = store.TransitionGrant(ctx, clientOwner, clientID, grantID, "approve", now); err != nil {
		t.Fatal(err)
	}
	if err = store.TransitionGrant(ctx, firmOwner, firmID, grantID, "accept", now); err != nil {
		t.Fatal(err)
	}
	page, err := store.ListPortfolio(ctx, firmOwner, firmID, "")
	if err != nil || len(page.Clients) != 1 || page.Clients[0].Assigned || page.Clients[0].ClientName != "Client A" {
		t.Fatalf("unassigned portfolio=%+v err=%v", page, err)
	}
	if _, err = store.GetFirmDocument(ctx, firmOwner, firmID, clientID, docID, "read"); !errors.Is(err, tenant.ErrNotFound) {
		t.Fatalf("unassigned owner read: %v", err)
	}
	if err = store.AssignStaff(ctx, firmOwner, firmID, grantID, staff, now); err != nil {
		t.Fatal(err)
	}
	page, err = store.ListPortfolio(ctx, staff, firmID, "")
	if err != nil || len(page.Clients) != 1 || !page.Clients[0].Assigned {
		t.Fatalf("assigned portfolio=%+v err=%v", page, err)
	}
	if _, err = store.GetFirmDocument(ctx, staff, firmID, clientID, docID, "read"); err != nil {
		t.Fatalf("assigned preview: %v", err)
	}
	if _, err = store.GetFirmDocument(ctx, staff, firmID, clientID, docID, "original.download"); !errors.Is(err, tenant.ErrNotFound) {
		t.Fatalf("download without scope: %v", err)
	}
	if _, err = store.GetFirmDocument(ctx, outsider, otherID, clientID, docID, "read"); !errors.Is(err, tenant.ErrNotFound) {
		t.Fatalf("foreign tenant: %v", err)
	}
	if err = store.TransitionGrant(ctx, clientOwner, clientID, grantID, "revoke", now.Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	page, err = store.ListPortfolio(ctx, staff, firmID, "")
	if err != nil || len(page.Clients) != 0 {
		t.Fatalf("portfolio after revoke=%+v err=%v", page, err)
	}
	if _, err = store.GetFirmDocument(ctx, staff, firmID, clientID, docID, "read"); !errors.Is(err, tenant.ErrNotFound) {
		t.Fatalf("read after revoke: %v", err)
	}
}
