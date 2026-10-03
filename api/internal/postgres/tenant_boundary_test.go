package postgres

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"plaiflow/api/internal/tenant"
)

// The temporary database owner seeds fixtures; the assertions run as a separate
// non-owner role, since table owners bypass RLS unless FORCE is configured.
func TestTenantBoundaryUnderNonBypassRole(t *testing.T) {
	runtimeRole := os.Getenv("OCR_TEST_RUNTIME_ROLE")
	if runtimeRole == "" {
		t.Skip("OCR_TEST_RUNTIME_ROLE not set; actual non-owner RLS proof unavailable")
	}
	store, ctx := isolatedTestStore(t, 16)
	owner, admin, member, invited := postgresUUID(), postgresUUID(), postgresUUID(), postgresUUID()
	first, second := postgresUUID(), postgresUUID()
	for _, user := range []string{owner, admin, member, invited} {
		if _, err := store.pool.Exec(ctx, `INSERT INTO users(id) VALUES($1)`, user); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := store.CreateOrganization(ctx, owner, first, "First", time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	if _, err := store.CreateOrganization(ctx, owner, second, "Second", time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	if _, err := store.pool.Exec(ctx, `INSERT INTO memberships(organization_id,user_id,role) VALUES($1,$2,'Admin'),($1,$3,'Member')`, first, admin, member); err != nil {
		t.Fatal(err)
	}
	role := pgx.Identifier{runtimeRole}.Sanitize()
	t.Cleanup(func() {
		if _, err := store.pool.Exec(context.Background(), `REVOKE ALL ON organizations, memberships FROM `+role); err != nil {
			t.Error(err)
		}
	})
	if _, err := store.pool.Exec(ctx, `GRANT SELECT, UPDATE ON organizations TO `+role); err != nil {
		t.Fatal(err)
	}
	if _, err := store.pool.Exec(ctx, `GRANT SELECT ON memberships TO `+role); err != nil {
		t.Fatal(err)
	}
	check := func(user, scope, target string, wantRead, wantWrite bool) {
		t.Helper()
		tx, err := store.pool.Begin(ctx)
		if err != nil {
			t.Fatal(err)
		}
		defer tx.Rollback(ctx)
		if _, err := tx.Exec(ctx, `SET LOCAL ROLE `+role); err != nil {
			t.Fatal(err)
		}
		if _, err := tx.Exec(ctx, `SELECT set_config('app.user_id',$1,true),set_config('app.organization_id',$2,true)`, user, scope); err != nil {
			t.Fatal(err)
		}
		var bypass bool
		if err := tx.QueryRow(ctx, `SELECT rolbypassrls OR rolsuper FROM pg_roles WHERE rolname=current_user`).Scan(&bypass); err != nil || bypass {
			t.Fatalf("runtime role bypass=%v err=%v", bypass, err)
		}
		var count int
		if err := tx.QueryRow(ctx, `SELECT count(*) FROM organizations WHERE id=$1`, target).Scan(&count); err != nil || (count == 1) != wantRead {
			t.Fatalf("read user=%s scope=%s target=%s count=%d err=%v", user, scope, target, count, err)
		}
		command, err := tx.Exec(ctx, `UPDATE organizations SET name='boundary test' WHERE id=$1`, target)
		if err != nil || (command.RowsAffected() == 1) != wantWrite {
			t.Fatalf("write user=%s scope=%s target=%s affected=%d err=%v", user, scope, target, command.RowsAffected(), err)
		}
	}
	check(owner, first, first, true, true)
	check(owner, first, second, false, false)
	check(admin, first, first, true, true)
	check(member, first, first, true, false)
	check(invited, first, first, false, false)
	if err := store.RemoveMembership(ctx, first, owner, member, time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	if _, err := store.ResolveMembership(ctx, member, first); err != tenant.ErrNotFound {
		t.Fatalf("revoked membership: %v", err)
	}
	check(member, first, first, false, false)
	check(owner, first, first, true, true)
}
