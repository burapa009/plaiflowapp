package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"plaiflow/api/internal/secretary"
	"plaiflow/api/internal/tenant"
)

func (s *Store) GetPreferences(ctx context.Context, userID, organizationID string) (secretary.Preferences, error) {
	tx, err := s.organizationTx(ctx, userID, organizationID)
	if err != nil {
		return secretary.Preferences{}, err
	}
	defer tx.Rollback(ctx)
	role, err := currentRole(ctx, tx, userID, organizationID)
	if err != nil {
		return secretary.Preferences{}, tenant.ErrNotFound
	}
	prefs, err := readSecretaryPreferences(ctx, tx, userID, organizationID)
	if err != nil {
		return secretary.Preferences{}, err
	}
	visible := make([]string, 0, len(prefs.PinnedTaskIDs))
	for _, id := range prefs.PinnedTaskIDs {
		if s.secretarySourcesAuthorized(ctx, tx, organizationID, userID, role, []secretary.Item{{SourceID: id, Category: "Task"}}) {
			visible = append(visible, id)
		}
	}
	prefs.PinnedTaskIDs = visible
	return prefs, tx.Commit(ctx)
}

func readSecretaryPreferences(ctx context.Context, tx pgx.Tx, userID, organizationID string) (secretary.Preferences, error) {
	prefs := secretary.Preferences{HiddenCategories: []string{}, PinnedTaskIDs: []string{}}
	err := tx.QueryRow(ctx, `SELECT hidden_categories,pinned_task_ids FROM secretary_preferences
		WHERE organization_id=$1 AND user_id=$2`, organizationID, userID).Scan(&prefs.HiddenCategories, &prefs.PinnedTaskIDs)
	if errors.Is(err, pgx.ErrNoRows) {
		return prefs, nil
	}
	return prefs, err
}

func (s *Store) SetPreferences(ctx context.Context, userID, organizationID string, prefs secretary.Preferences, now time.Time) (secretary.Preferences, error) {
	if prefs.HiddenCategories == nil {
		prefs.HiddenCategories = []string{}
	}
	if prefs.PinnedTaskIDs == nil {
		prefs.PinnedTaskIDs = []string{}
	}
	tx, err := s.organizationTx(ctx, userID, organizationID)
	if err != nil {
		return secretary.Preferences{}, err
	}
	defer tx.Rollback(ctx)
	var role tenant.Role
	var timezone string
	err = tx.QueryRow(ctx, `SELECT m.role,o.timezone FROM memberships m JOIN organizations o ON o.id=m.organization_id
		WHERE m.organization_id=$1 AND m.user_id=$2 FOR UPDATE OF m`, organizationID, userID).Scan(&role, &timezone)
	if errors.Is(err, pgx.ErrNoRows) {
		return secretary.Preferences{}, tenant.ErrNotFound
	}
	if err != nil {
		return secretary.Preferences{}, err
	}
	for _, id := range prefs.PinnedTaskIDs {
		if !s.secretarySourcesAuthorized(ctx, tx, organizationID, userID, role, []secretary.Item{{SourceID: id, Category: "Task"}}) {
			return secretary.Preferences{}, tenant.ErrNotFound
		}
	}
	_, err = tx.Exec(ctx, `INSERT INTO secretary_preferences(organization_id,user_id,hidden_categories,pinned_task_ids,updated_at)
		VALUES($1,$2,$3,$4,$5) ON CONFLICT(organization_id,user_id) DO UPDATE SET
		hidden_categories=excluded.hidden_categories,pinned_task_ids=excluded.pinned_task_ids,updated_at=excluded.updated_at`,
		organizationID, userID, prefs.HiddenCategories, prefs.PinnedTaskIDs, now.UTC())
	if err != nil {
		return secretary.Preferences{}, err
	}
	location, err := time.LoadLocation(timezone)
	if err != nil {
		return secretary.Preferences{}, err
	}
	local := now.In(location)
	day := local.Format("2006-01-02")
	row, err := readSecretaryRow(ctx, tx, organizationID, userID, day)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return secretary.Preferences{}, err
	}
	if err == nil {
		expires := time.Date(local.Year(), local.Month(), local.Day()+1, 0, 0, 0, 0, location).UTC()
		if _, err = queueSecretary(ctx, tx, userID, organizationID, day, role, expires, now, row.briefing.JobID); err != nil {
			return secretary.Preferences{}, err
		}
	}
	if err = tx.Commit(ctx); err != nil {
		return secretary.Preferences{}, err
	}
	return prefs, nil
}
