package postgres

import (
	"errors"
	"os"
	"testing"
	"time"

	"plaiflow/api/internal/job"
	"plaiflow/api/internal/secretary"
	"plaiflow/api/internal/tenant"
	"plaiflow/api/internal/work"
)

func TestSecretaryMigrationRoundTripOnEmptyDatabase(t *testing.T) {
	store, ctx := isolatedTestStore(t, 17)
	up, err := os.ReadFile("../../migrations/000018_phase13_secretary.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	down, err := os.ReadFile("../../migrations/000018_phase13_secretary.down.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = store.pool.Exec(ctx, string(up)); err != nil {
		t.Fatal("up", err)
	}
	if err = store.ReadySecretary(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err = store.pool.Exec(ctx, string(down)); err != nil {
		t.Fatal("empty down", err)
	}
	if _, err = store.pool.Exec(ctx, string(up)); err != nil {
		t.Fatal("reapply", err)
	}
}

func TestSecretaryBriefingUsesCurrentRoleAndHumanRoutineConfirmation(t *testing.T) {
	store, ctx := isolatedTestStore(t, 18)
	now := time.Date(2026, 9, 28, 3, 0, 0, 0, time.UTC)
	owner, member, outsider := postgresUUID(), postgresUUID(), postgresUUID()
	org, foreign := postgresUUID(), postgresUUID()
	for _, id := range []string{owner, member, outsider} {
		if _, err := store.pool.Exec(ctx, `INSERT INTO users(id) VALUES($1)`, id); err != nil {
			t.Fatal(err)
		}
	}
	for _, pair := range [][2]string{{owner, org}, {outsider, foreign}} {
		if _, err := store.CreateOrganization(ctx, pair[0], pair[1], "Secretary test", now); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := store.pool.Exec(ctx, `INSERT INTO memberships(organization_id,user_id,role) VALUES($1,$2,'Member')`, org, member); err != nil {
		t.Fatal(err)
	}
	create := func(title, assignee, due string, priority work.Priority) string {
		id := postgresUUID()
		if _, err := store.CreateTask(ctx, work.CreateTask{ID: id, OrganizationID: org, ActorUserID: owner,
			Title: title, AssigneeUserID: assignee, DueOn: due, Priority: priority, Now: now.Add(-time.Hour)}); err != nil {
			t.Fatal(err)
		}
		return id
	}
	urgent := create("overdue urgent", member, "2026-09-27", work.Urgent)
	create("today high", member, "2026-09-28", work.High)
	private := create("private team", owner, "2026-09-27", work.Normal)
	if _, err := store.GetTaskSource(ctx, member, org, private); !errors.Is(err, tenant.ErrNotFound) {
		t.Fatalf("member read private source: %v", err)
	}
	if _, err := store.Open(ctx, member, foreign, now); !errors.Is(err, tenant.ErrNotFound) {
		t.Fatalf("foreign open: %v", err)
	}
	first, err := store.Open(ctx, member, org, now)
	if err != nil || first.Status != "Queued" {
		t.Fatalf("first=%+v err=%v", first, err)
	}
	duplicate, err := store.Open(ctx, member, org, now)
	if err != nil || duplicate.JobID != first.JobID {
		t.Fatalf("duplicate=%+v err=%v", duplicate, err)
	}
	claimed, err := store.ClaimJobs(ctx, job.ClaimCommand{WorkerID: "secretary-test", Environment: "test", Kinds: []job.Kind{job.Secretary}, Limit: 1, Lease: 2 * time.Minute, Now: now})
	if err != nil || len(claimed) != 1 {
		t.Fatalf("claimed=%d err=%v", len(claimed), err)
	}
	lease := job.LeaseCommand{JobID: claimed[0].Job.ID, AttemptID: claimed[0].Job.AttemptID, LeaseToken: claimed[0].LeaseToken, Now: now}
	if err = store.Generate(ctx, lease); err != nil {
		t.Fatal("generate", err)
	}
	ready, err := store.Open(ctx, member, org, now)
	if err != nil || ready.Status != "Ready" || len(ready.Items) != 2 || ready.Items[0].SourceID != urgent || ready.Remaining != 0 {
		t.Fatalf("ready=%+v err=%v", ready, err)
	}
	if _, err = store.SetTaskSource(ctx, secretary.SourceCommand{TaskSource: secretary.TaskSource{TaskID: urgent, Category: "FollowUp", FollowUpWith: "Supplier"}, UserID: member, OrganizationID: org, Now: now}); !errors.Is(err, tenant.ErrForbidden) {
		t.Fatalf("member edited owner's task: %v", err)
	}
	if _, err = store.SetTaskSource(ctx, secretary.SourceCommand{TaskSource: secretary.TaskSource{TaskID: urgent, Category: "FollowUp", FollowUpWith: "Supplier"}, UserID: owner, OrganizationID: org, Now: now}); err != nil {
		t.Fatal(err)
	}
	answer, err := store.Intent(ctx, member, org, "follow-up", now)
	if err != nil || len(answer.Items) != 1 || answer.Items[0].SourceID != urgent {
		t.Fatalf("follow-up=%+v err=%v", answer, err)
	}
	if answer, err = store.Intent(ctx, member, org, "missing-documents", now); err != nil || answer.State != "unconfigured" {
		t.Fatalf("missing source=%+v err=%v", answer, err)
	}
	ownerFirst, err := store.Open(ctx, owner, org, now)
	if err != nil || ownerFirst.Status != "Queued" {
		t.Fatalf("owner first=%+v err=%v", ownerFirst, err)
	}
	claimed, err = store.ClaimJobs(ctx, job.ClaimCommand{WorkerID: "secretary-test", Environment: "test", Kinds: []job.Kind{job.Secretary}, Limit: 1, Lease: 2 * time.Minute, Now: now})
	if err != nil || len(claimed) != 1 || claimed[0].Job.ID != ownerFirst.JobID {
		t.Fatalf("owner claim=%+v err=%v", claimed, err)
	}
	if err = store.Generate(ctx, job.LeaseCommand{JobID: claimed[0].Job.ID, AttemptID: claimed[0].Job.AttemptID, LeaseToken: claimed[0].LeaseToken, Now: now}); err != nil {
		t.Fatal("owner generate", err)
	}
	ownerReady, err := store.Open(ctx, owner, org, now)
	if err != nil || ownerReady.Status != "Ready" || len(ownerReady.Items) == 0 {
		t.Fatalf("owner ready=%+v err=%v", ownerReady, err)
	}
	if err = store.TransferOwnership(ctx, org, owner, member, now); err != nil {
		t.Fatal(err)
	}
	if _, err = store.pool.Exec(ctx, `UPDATE memberships SET role='Member' WHERE organization_id=$1 AND user_id=$2`, org, owner); err != nil {
		t.Fatal(err)
	}
	changed, err := store.Open(ctx, owner, org, now)
	if err != nil || changed.Status != "Queued" || len(changed.Items) != 0 {
		t.Fatalf("role change=%+v err=%v", changed, err)
	}
	// Restore manager authority to create a template for the member.
	if err = store.TransferOwnership(ctx, org, member, owner, now); err != nil {
		t.Fatal(err)
	}
	t.Log("ownership restored")
	if _, err = store.pool.Exec(ctx, `UPDATE memberships SET role='Member' WHERE organization_id=$1 AND user_id=$2`, org, member); err != nil {
		t.Fatal(err)
	}
	template, err := store.SaveRoutineTemplate(ctx, secretary.TemplateCommand{RoutineTemplate: secretary.RoutineTemplate{
		Title: "Weekly check", ResponsibleUserID: member, Cadence: "Weekly", FirstDueOn: "2026-09-28", Active: true},
		UserID: owner, OrganizationID: org, Now: now})
	if err != nil || template.ID == "" {
		t.Fatalf("template=%+v err=%v", template, err)
	}
	t.Log("member routine template saved")
	if _, err = store.SaveRoutineTemplate(ctx, secretary.TemplateCommand{RoutineTemplate: secretary.RoutineTemplate{
		Title: "Manager check", ResponsibleUserID: owner, Cadence: "Monthly", FirstDueOn: "2026-09-28", Active: true},
		UserID: owner, OrganizationID: org, Now: now}); err != nil {
		t.Fatal(err)
	}
	visibleTemplates, err := store.ListRoutineTemplates(ctx, member, org)
	if err != nil || len(visibleTemplates) != 1 || visibleTemplates[0].ID != template.ID {
		t.Fatalf("member templates=%+v err=%v", visibleTemplates, err)
	}
	t.Log("routine template visibility checked")
	extraTasks := make([]string, 0, 4)
	for i := 0; i < 4; i++ {
		extraTasks = append(extraTasks, create("extra due today", member, "2026-09-28", work.Normal))
	}
	t.Log("extra tasks saved")
	refreshed, err := store.Refresh(ctx, member, org, now.Add(16*time.Minute))
	if err != nil || refreshed.Status != "Queued" {
		t.Fatalf("refreshed=%+v err=%v", refreshed, err)
	}
	t.Log("refresh queued")
	claimed, err = store.ClaimJobs(ctx, job.ClaimCommand{WorkerID: "secretary-test", Environment: "test", Kinds: []job.Kind{job.Secretary}, Limit: 5, Lease: 2 * time.Minute, Now: now.Add(16 * time.Minute)})
	if err != nil {
		t.Fatal("routine claim", err)
	}
	for _, candidate := range claimed {
		if candidate.Job.ID == refreshed.JobID {
			lease = job.LeaseCommand{JobID: candidate.Job.ID, AttemptID: candidate.Job.AttemptID, LeaseToken: candidate.LeaseToken, Now: now.Add(16 * time.Minute)}
		}
	}
	if lease.JobID != refreshed.JobID {
		t.Fatalf("refreshed job not claimed: %v", refreshed.JobID)
	}
	t.Log("refresh claimed")
	if err = store.Generate(ctx, lease); err != nil {
		t.Fatal("routine generate", err)
	}
	withRemainder, err := store.Open(ctx, member, org, now.Add(16*time.Minute))
	if err != nil || withRemainder.Status != "Ready" || len(withRemainder.Items) != 5 || withRemainder.Remaining != 2 {
		t.Fatalf("remainder=%+v err=%v", withRemainder, err)
	}
	shown := map[string]bool{}
	for _, item := range withRemainder.Items {
		shown[item.SourceID] = true
	}
	for _, taskID := range extraTasks {
		if !shown[taskID] {
			if _, err = store.pool.Exec(ctx, `UPDATE tasks SET assignee_user_id=$3,updated_at=$4 WHERE organization_id=$1 AND id=$2`, org, taskID, owner, now.Add(17*time.Minute)); err != nil {
				t.Fatal(err)
			}
			break
		}
	}
	invalidated, err := store.Open(ctx, member, org, now.Add(17*time.Minute))
	if err != nil || invalidated.Status != "Queued" || invalidated.Remaining != 0 || len(invalidated.Items) != 0 {
		t.Fatalf("revoked unseen task=%+v err=%v", invalidated, err)
	}
	suggestions, err := store.ListRoutineSuggestions(ctx, member, org, now)
	if err != nil || len(suggestions) != 1 || suggestions[0].TaskID != "" {
		t.Fatalf("suggestions=%+v err=%v", suggestions, err)
	}
	confirmed, err := store.ResolveRoutineSuggestion(ctx, member, org, suggestions[0].ID, "confirm", now.Add(17*time.Minute))
	if err != nil || confirmed.TaskID == "" {
		t.Fatalf("confirm=%+v err=%v", confirmed, err)
	}
	if _, err = store.GetTask(ctx, member, org, confirmed.TaskID); err != nil {
		t.Fatalf("confirmed task: %v", err)
	}
	if _, err = store.ResolveRoutineSuggestion(ctx, member, org, suggestions[0].ID, "confirm", now.Add(18*time.Minute)); err != nil {
		t.Fatalf("repeat confirm: %v", err)
	}
	if err = store.RemoveMembership(ctx, org, owner, member, now.Add(19*time.Minute)); err != nil {
		t.Fatalf("member with routine could not be removed: %v", err)
	}
}
