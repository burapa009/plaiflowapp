package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"plaiflow/api/internal/secretary"
	"plaiflow/api/internal/tenant"
)

func (s *Store) ListRoutineTemplates(ctx context.Context, userID, organizationID string) ([]secretary.RoutineTemplate, error) {
	tx, err := s.organizationTx(ctx, userID, organizationID)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	role, err := currentRole(ctx, tx, userID, organizationID)
	if err != nil {
		return nil, tenant.ErrNotFound
	}
	rows, err := tx.Query(ctx, `SELECT id,title,responsible_user_id,cadence,first_due_on::text,active
		FROM routine_templates WHERE organization_id=$1 AND ($2 IN ('Owner','Admin') OR responsible_user_id=$3)
		ORDER BY created_at,id`, organizationID, role, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []secretary.RoutineTemplate{}
	for rows.Next() {
		var item secretary.RoutineTemplate
		if err = rows.Scan(&item.ID, &item.Title, &item.ResponsibleUserID, &item.Cadence, &item.FirstDueOn, &item.Active); err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return result, tx.Commit(ctx)
}

func (s *Store) SaveRoutineTemplate(ctx context.Context, command secretary.TemplateCommand) (secretary.RoutineTemplate, error) {
	tx, err := s.organizationTx(ctx, command.UserID, command.OrganizationID)
	if err != nil {
		return secretary.RoutineTemplate{}, err
	}
	defer tx.Rollback(ctx)
	role, err := currentRole(ctx, tx, command.UserID, command.OrganizationID)
	if err != nil {
		return secretary.RoutineTemplate{}, tenant.ErrNotFound
	}
	if role != tenant.Owner && role != tenant.Admin {
		return secretary.RoutineTemplate{}, tenant.ErrForbidden
	}
	if !memberExists(ctx, tx, command.OrganizationID, command.ResponsibleUserID) {
		return secretary.RoutineTemplate{}, tenant.ErrNotFound
	}
	if command.ID == "" {
		command.ID, err = randomUUID()
		if err != nil {
			return secretary.RoutineTemplate{}, err
		}
		_, err = tx.Exec(ctx, `INSERT INTO routine_templates
			(id,organization_id,title,responsible_user_id,cadence,first_due_on,active,created_at,updated_at)
			VALUES($1,$2,$3,$4,$5,$6::date,$7,$8,$8)`, command.ID, command.OrganizationID,
			command.Title, command.ResponsibleUserID, command.Cadence, command.FirstDueOn, command.Active, command.Now.UTC())
	} else {
		var pending bool
		err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM routine_suggestions WHERE organization_id=$1
			AND template_id=$2 AND status='Pending')`, command.OrganizationID, command.ID).Scan(&pending)
		if err != nil {
			return secretary.RoutineTemplate{}, err
		}
		if pending && command.Active {
			return secretary.RoutineTemplate{}, tenant.ErrConflict
		}
		result, updateErr := tx.Exec(ctx, `UPDATE routine_templates SET title=$3,responsible_user_id=$4,cadence=$5,
			first_due_on=$6::date,active=$7,updated_at=$8 WHERE organization_id=$1 AND id=$2`,
			command.OrganizationID, command.ID, command.Title, command.ResponsibleUserID, command.Cadence,
			command.FirstDueOn, command.Active, command.Now.UTC())
		err = updateErr
		if err == nil && result.RowsAffected() != 1 {
			return secretary.RoutineTemplate{}, tenant.ErrNotFound
		}
	}
	if err != nil {
		return secretary.RoutineTemplate{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return secretary.RoutineTemplate{}, err
	}
	return command.RoutineTemplate, nil
}

func (s *Store) ListRoutineSuggestions(ctx context.Context, userID, organizationID string, now time.Time) ([]secretary.RoutineSuggestion, error) {
	tx, err := s.organizationTx(ctx, userID, organizationID)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	role, err := currentRole(ctx, tx, userID, organizationID)
	if err != nil {
		return nil, tenant.ErrNotFound
	}
	rows, err := tx.Query(ctx, `SELECT s.id,s.template_id,t.title,t.responsible_user_id,s.due_on::text,s.status,
		coalesce(s.task_id::text,'') FROM routine_suggestions s JOIN routine_templates t
		ON t.organization_id=s.organization_id AND t.id=s.template_id
		WHERE s.organization_id=$1 AND s.status='Pending' AND t.active
		AND ($2 IN ('Owner','Admin') OR t.responsible_user_id=$3)
		ORDER BY s.due_on,s.id`, organizationID, role, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []secretary.RoutineSuggestion{}
	for rows.Next() {
		var item secretary.RoutineSuggestion
		if err = rows.Scan(&item.ID, &item.TemplateID, &item.Title, &item.ResponsibleUserID, &item.DueOn, &item.Status, &item.TaskID); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return items, tx.Commit(ctx)
}

func (s *Store) ResolveRoutineSuggestion(ctx context.Context, userID, organizationID, suggestionID, action string, now time.Time) (secretary.RoutineSuggestion, error) {
	tx, err := s.organizationTx(ctx, userID, organizationID)
	if err != nil {
		return secretary.RoutineSuggestion{}, err
	}
	defer tx.Rollback(ctx)
	role, err := currentRole(ctx, tx, userID, organizationID)
	if err != nil {
		return secretary.RoutineSuggestion{}, tenant.ErrNotFound
	}
	var item secretary.RoutineSuggestion
	err = tx.QueryRow(ctx, `SELECT s.id,s.template_id,t.title,t.responsible_user_id,s.due_on::text,s.status,
		coalesce(s.task_id::text,'') FROM routine_suggestions s JOIN routine_templates t
		ON t.organization_id=s.organization_id AND t.id=s.template_id
		WHERE s.organization_id=$1 AND s.id=$2 FOR UPDATE OF s`, organizationID, suggestionID).Scan(
		&item.ID, &item.TemplateID, &item.Title, &item.ResponsibleUserID, &item.DueOn, &item.Status, &item.TaskID)
	if errors.Is(err, pgx.ErrNoRows) {
		return secretary.RoutineSuggestion{}, tenant.ErrNotFound
	}
	if err != nil {
		return secretary.RoutineSuggestion{}, err
	}
	if role != tenant.Owner && role != tenant.Admin && item.ResponsibleUserID != userID {
		return secretary.RoutineSuggestion{}, tenant.ErrNotFound
	}
	if item.Status != "Pending" {
		if item.Status == "Confirmed" && action == "confirm" || item.Status == "Skipped" && action == "skip" {
			return item, tx.Commit(ctx)
		}
		return secretary.RoutineSuggestion{}, tenant.ErrConflict
	}
	if action == "confirm" {
		item.TaskID, err = randomUUID()
		if err != nil {
			return secretary.RoutineSuggestion{}, err
		}
		_, err = tx.Exec(ctx, `INSERT INTO tasks(id,organization_id,title,description,creator_user_id,assignee_user_id,
			status,priority,due_on,created_at,updated_at,status_changed_at)
			VALUES($1,$2,$3,'',$4,$5,'Open','Normal',$6::date,$7,$7,$7)`,
			item.TaskID, organizationID, item.Title, userID, item.ResponsibleUserID, item.DueOn, now.UTC())
		if err != nil {
			return secretary.RoutineSuggestion{}, err
		}
		if err = insertDomainEvent(ctx, tx, organizationID, item.TaskID, userID, "task.created",
			map[string]any{"task_id": item.TaskID, "assignee_user_id": item.ResponsibleUserID}, now.UTC()); err != nil {
			return secretary.RoutineSuggestion{}, err
		}
		item.Status = "Confirmed"
	} else {
		item.Status = "Skipped"
	}
	_, err = tx.Exec(ctx, `UPDATE routine_suggestions SET status=$3,task_id=nullif($4,'')::uuid,resolved_at=$5
		WHERE organization_id=$1 AND id=$2`, organizationID, item.ID, item.Status, item.TaskID, now.UTC())
	if err != nil {
		return secretary.RoutineSuggestion{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return secretary.RoutineSuggestion{}, err
	}
	return item, nil
}

type routineState struct {
	template                                    secretary.RoutineTemplate
	lastID, lastDue, lastStatus, lastTaskStatus string
}

func syncRoutineSuggestions(ctx context.Context, tx pgx.Tx, userID, organizationID string, role tenant.Role, day string, now time.Time) error {
	rows, err := tx.Query(ctx, `SELECT t.id,t.title,t.responsible_user_id,t.cadence,t.first_due_on::text,t.active,
		coalesce(s.id::text,''),coalesce(s.due_on::text,''),coalesce(s.status,''),coalesce(task.status,'')
		FROM routine_templates t LEFT JOIN LATERAL(SELECT id,due_on,status,task_id FROM routine_suggestions
			WHERE organization_id=t.organization_id AND template_id=t.id ORDER BY due_on DESC LIMIT 1)s ON true
		LEFT JOIN tasks task ON task.organization_id=t.organization_id AND task.id=s.task_id
		WHERE t.organization_id=$1 AND t.active AND ($3 IN ('Owner','Admin') OR t.responsible_user_id=$2)
		ORDER BY t.id`, organizationID, userID, role)
	if err != nil {
		return err
	}
	states := []routineState{}
	for rows.Next() {
		var x routineState
		if err = rows.Scan(&x.template.ID, &x.template.Title, &x.template.ResponsibleUserID, &x.template.Cadence,
			&x.template.FirstDueOn, &x.template.Active, &x.lastID, &x.lastDue, &x.lastStatus, &x.lastTaskStatus); err != nil {
			rows.Close()
			return err
		}
		states = append(states, x)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	for _, x := range states {
		due := latestRoutineDue(x.template.FirstDueOn, x.template.Cadence, day)
		if due == "" || x.lastDue >= due || x.lastStatus == "Pending" ||
			x.lastStatus == "Confirmed" && (x.lastTaskStatus == "Open" || x.lastTaskStatus == "InProgress") {
			continue
		}
		id, e := randomUUID()
		if e != nil {
			return e
		}
		_, e = tx.Exec(ctx, `INSERT INTO routine_suggestions(id,organization_id,template_id,due_on,status,created_at)
			VALUES($1,$2,$3,$4::date,'Pending',$5) ON CONFLICT(organization_id,template_id,due_on) DO NOTHING`,
			id, organizationID, x.template.ID, due, now.UTC())
		if e != nil {
			return e
		}
	}
	return nil
}

func latestRoutineDue(first, cadence, day string) string {
	start, e1 := time.Parse("2006-01-02", first)
	today, e2 := time.Parse("2006-01-02", day)
	if e1 != nil || e2 != nil || today.Before(start) {
		return ""
	}
	if cadence == "Weekly" {
		n := int(today.Sub(start).Hours()/24) / 7
		return start.AddDate(0, 0, n*7).Format("2006-01-02")
	}
	step := 1
	if cadence == "Quarterly" {
		step = 3
	} else if cadence != "Monthly" {
		return ""
	}
	months := (today.Year()-start.Year())*12 + int(today.Month()-start.Month())
	n := months / step
	due := routineMonth(start, n*step)
	if due.After(today) {
		due = routineMonth(start, (n-1)*step)
	}
	if due.Before(start) {
		return ""
	}
	return due.Format("2006-01-02")
}

func routineMonth(start time.Time, months int) time.Time {
	monthStart := time.Date(start.Year(), start.Month()+time.Month(months), 1, 0, 0, 0, 0, time.UTC)
	last := monthStart.AddDate(0, 1, -1).Day()
	day := start.Day()
	if day > last {
		day = last
	}
	return monthStart.AddDate(0, 0, day-1)
}
