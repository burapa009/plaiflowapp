package postgres

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"plaiflow/api/internal/tenant"
)

func TestLINEGroupDocumentIsHiddenFromSubmittingMember(t *testing.T) {
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
	ownerID, memberID, organizationID, documentID, sourceID := postgresUUID(), postgresUUID(), postgresUUID(), postgresUUID(), postgresUUID()
	now := time.Now().UTC().Truncate(time.Second)
	periodStart := now.Add(-time.Hour)
	periodEnd := now.Add(time.Hour)
	defer func() {
		_, _ = store.pool.Exec(ctx, `DELETE FROM document_sources WHERE id=$1`, sourceID)
		_, _ = store.pool.Exec(ctx, `DELETE FROM documents WHERE id=$1`, documentID)
		_, _ = store.pool.Exec(ctx, `DELETE FROM document_usage_periods WHERE organization_id=$1`, organizationID)
		_, _ = store.pool.Exec(ctx, `DELETE FROM memberships WHERE organization_id=$1`, organizationID)
		_, _ = store.pool.Exec(ctx, `DELETE FROM organizations WHERE id=$1`, organizationID)
		_, _ = store.pool.Exec(ctx, `DELETE FROM users WHERE id=$1 OR id=$2`, ownerID, memberID)
	}()
	if _, err := store.pool.Exec(ctx, `INSERT INTO users (id) VALUES ($1),($2)`, ownerID, memberID); err != nil {
		t.Fatal(err)
	}
	if _, err := store.pool.Exec(ctx, `INSERT INTO organizations (id,name) VALUES ($1,'Group privacy test')`, organizationID); err != nil {
		t.Fatal(err)
	}
	if _, err := store.pool.Exec(ctx, `INSERT INTO memberships (organization_id,user_id,role) VALUES ($1,$2,'Owner'),($1,$3,'Member')`, organizationID, ownerID, memberID); err != nil {
		t.Fatal(err)
	}
	if _, err := store.pool.Exec(ctx, `INSERT INTO document_usage_periods (organization_id,starts_at,ends_at,timezone) VALUES ($1,$2,$3,'UTC')`, organizationID, periodStart, periodEnd); err != nil {
		t.Fatal(err)
	}
	if _, err := store.pool.Exec(ctx, `INSERT INTO documents
	    (id,organization_id,content_sha256,storage_key,display_filename,detected_mime,byte_size,status,submitted_by_user_id,group_restricted,accepted_at,updated_at,usage_period_start)
	    VALUES ($1,$2,$3,'test/group-original','group.pdf','application/pdf',8,'Available',$4,true,$5,$5,$6)`,
		documentID, organizationID, make([]byte, 32), memberID, now, periodStart); err != nil {
		t.Fatal(err)
	}
	if _, err := store.pool.Exec(ctx, `INSERT INTO document_sources
	    (id,organization_id,document_id,channel,origin_key,submitted_by_user_id,group_source,created_at)
	    VALUES ($1,$2,$3,'LINE','test-group-origin',$4,true,$5)`, sourceID, organizationID, documentID, memberID, now); err != nil {
		t.Fatal(err)
	}
	memberPage, err := store.ListDocuments(ctx, memberID, organizationID, "", 10)
	if err != nil || len(memberPage.Documents) != 0 {
		t.Fatalf("member page=%+v err=%v", memberPage, err)
	}
	if _, err := store.GetDocument(ctx, memberID, organizationID, documentID); !errors.Is(err, tenant.ErrNotFound) {
		t.Fatalf("member opened group original: %v", err)
	}
	ownerPage, err := store.ListDocuments(ctx, ownerID, organizationID, "", 10)
	if err != nil || len(ownerPage.Documents) != 1 || ownerPage.Documents[0].ID != documentID {
		t.Fatalf("owner page=%+v err=%v", ownerPage, err)
	}
}
