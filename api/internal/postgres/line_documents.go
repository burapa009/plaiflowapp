package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"plaiflow/api/internal/inbound"
)

var ErrLINEOrganizationUnmapped = errors.New("LINE sender has no unambiguous organization")

func (s *Store) ResolveLINEDocument(ctx context.Context, event inbound.Event) (string, string, error) {
	if event.SourceUserID == "" {
		return "", "", ErrLINEOrganizationUnmapped
	}
	if event.SourceType == "group" {
		var organizationID, userID string
		err := s.pool.QueryRow(ctx, `SELECT lg.organization_id,m.user_id
		    FROM line_group_connections lg
		    JOIN memberships m ON m.organization_id=lg.organization_id
		    JOIN auth_identities ai ON ai.user_id=m.user_id AND ai.provider='line' AND ai.subject=$3 AND ai.disabled_at IS NULL
		    WHERE lg.messaging_channel=$1 AND lg.group_id=$2 AND lg.status='connected'
		    LIMIT 1`, event.Channel, event.SourceGroupID, event.SourceUserID).Scan(&organizationID, &userID)
		if errors.Is(err, pgx.ErrNoRows) {
			return "", "", ErrLINEOrganizationUnmapped
		}
		return organizationID, userID, err
	}
	rows, err := s.pool.Query(ctx, `SELECT m.organization_id,m.user_id
	    FROM auth_identities ai JOIN memberships m ON m.user_id=ai.user_id
	    WHERE ai.provider='line' AND ai.subject=$1 AND ai.disabled_at IS NULL
	    ORDER BY m.organization_id`, event.SourceUserID)
	if err != nil {
		return "", "", err
	}
	defer rows.Close()
	var organizationID, userID string
	count := 0
	for rows.Next() {
		if err := rows.Scan(&organizationID, &userID); err != nil {
			return "", "", err
		}
		count++
	}
	if err := rows.Err(); err != nil {
		return "", "", err
	}
	if count != 1 {
		return "", "", ErrLINEOrganizationUnmapped
	}
	return organizationID, userID, nil
}
