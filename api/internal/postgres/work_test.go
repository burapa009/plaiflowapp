package postgres

import (
	"plaiflow/api/internal/work"
	"testing"
	"time"
)

func TestUndatedTaskWorkspace(t *testing.T) {
	store, ctx := isolatedTestStore(t)
	user, org := postgresUUID(), postgresUUID()
	if _, err := store.pool.Exec(ctx, `INSERT INTO users(id) VALUES($1)`, user); err != nil {
		t.Fatal(err)
	}
	if _, err := store.CreateOrganization(ctx, user, org, "Workspace test", time.Now()); err != nil {
		t.Fatal(err)
	}
	if _, err := store.ListTasks(ctx, user, org, work.TaskFilter{Limit: 50}); err != nil {
		t.Fatal(err)
	}
	task, err := store.CreateTask(ctx, work.CreateTask{ID: postgresUUID(), OrganizationID: org, ActorUserID: user, Title: "No deadline", Priority: work.Normal, Now: time.Now()})
	if err != nil {
		t.Fatal(err)
	}
	if task.IsOverdue {
		t.Fatal("undated task is overdue")
	}
	for _, filtered := range []bool{false, true} {
		filter := work.TaskFilter{Limit: 50}
		overdue := false
		if filtered {
			filter.Overdue = &overdue
		}
		page, err := store.ListTasks(ctx, user, org, filter)
		if err != nil {
			t.Fatal(err)
		}
		if len(page.Tasks) != 1 || page.Tasks[0].IsOverdue {
			t.Fatal("undated task missing or overdue")
		}
	}
}
