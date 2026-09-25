package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"plaiflow/api/internal/billing"
	"plaiflow/api/internal/plan"
	"plaiflow/api/internal/tenant"
)

func scanBillingIntent(row pgx.Row) (billing.Intent, error) {
	var intent billing.Intent
	err := row.Scan(&intent.ID, &intent.OrganizationID, &intent.ActorUserID, &intent.Plan, &intent.Interval,
		&intent.AmountSatang, &intent.Status, &intent.ChargeID, &intent.QRImageURL, &intent.ExpiresAt)
	return intent, err
}

const billingIntentColumns = `id,organization_id,actor_user_id,plan_key,billing_interval,amount_satang,status,
    coalesce(omise_charge_id,''),coalesce(qr_image_url,''),expires_at`

func (s *Store) CreateIntent(ctx context.Context, intent billing.Intent, now time.Time) (billing.Intent, error) {
	tx, err := s.organizationTx(ctx, intent.ActorUserID, intent.OrganizationID)
	if err != nil {
		return billing.Intent{}, err
	}
	defer tx.Rollback(ctx)
	role, err := currentRole(ctx, tx, intent.ActorUserID, intent.OrganizationID)
	if err != nil || role != tenant.Owner {
		return billing.Intent{}, billing.ErrForbidden
	}
	// A row lock serializes checkouts for this Organization, including different Owner sessions.
	var organizationID string
	if err := tx.QueryRow(ctx, `SELECT id FROM organizations WHERE id=$1 FOR UPDATE`, intent.OrganizationID).Scan(&organizationID); err != nil {
		return billing.Intent{}, err
	}
	if _, err := tx.Exec(ctx, `UPDATE billing_intents SET status='expired'
        WHERE organization_id=$1 AND status IN ('creating','awaiting_payment','creation_unknown')
          AND expires_at < $2::timestamptz - interval '5 minutes'`, intent.OrganizationID, now); err != nil {
		return billing.Intent{}, err
	}
	current, err := scanBillingIntent(tx.QueryRow(ctx, `SELECT `+billingIntentColumns+` FROM billing_intents
        WHERE organization_id=$1 AND status IN ('creating','awaiting_payment','creation_unknown') FOR UPDATE`, intent.OrganizationID))
	if err == nil {
		if current.Plan != intent.Plan || current.Interval != intent.Interval {
			return billing.Intent{}, billing.ErrConflict
		}
		if current.Status != "awaiting_payment" {
			return billing.Intent{}, billing.ErrConflict
		}
		return current, tx.Commit(ctx)
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return billing.Intent{}, err
	}
	var paidThrough time.Time
	var currentPlan plan.Key
	var currentInterval plan.Interval
	err = tx.QueryRow(ctx, `SELECT plan_key,billing_interval,paid_through FROM billing_subscriptions WHERE organization_id=$1`, intent.OrganizationID).Scan(&currentPlan, &currentInterval, &paidThrough)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return billing.Intent{}, err
	}
	// A different paid Plan needs an explicit change preview and is not sold through first-checkout.
	if err == nil && paidThrough.After(now) && (currentPlan != intent.Plan || currentInterval != intent.Interval) {
		return billing.Intent{}, billing.ErrConflict
	}
	var manualActive bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM organization_plan_periods
        WHERE organization_id=$1 AND source='manual' AND starts_at<=$2 AND ends_at>$2)`, intent.OrganizationID, now).Scan(&manualActive); err != nil {
		return billing.Intent{}, err
	}
	if manualActive {
		return billing.Intent{}, billing.ErrConflict
	}
	created, err := scanBillingIntent(tx.QueryRow(ctx, `INSERT INTO billing_intents
        (id,organization_id,actor_user_id,plan_key,billing_interval,amount_satang,status,created_at,expires_at)
        VALUES ($1,$2,$3,$4,$5,$6,'creating',$7,$8) RETURNING `+billingIntentColumns,
		intent.ID, intent.OrganizationID, intent.ActorUserID, intent.Plan, intent.Interval, intent.AmountSatang, now, intent.ExpiresAt))
	if err != nil {
		return billing.Intent{}, err
	}
	if err := auditTenant(ctx, tx, intent.OrganizationID, intent.ActorUserID, "billing.checkout_requested", "billing_intent", intent.ID, now); err != nil {
		return billing.Intent{}, err
	}
	return created, tx.Commit(ctx)
}

func (s *Store) AttachCharge(ctx context.Context, intent billing.Intent, charge billing.Charge) (billing.Intent, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return billing.Intent{}, err
	}
	defer tx.Rollback(ctx)
	updated, err := scanBillingIntent(tx.QueryRow(ctx, `UPDATE billing_intents SET omise_charge_id=$3,qr_image_url=$4,status='awaiting_payment'
        WHERE organization_id=$1 AND id=$2 AND status='creating' AND omise_charge_id IS NULL
        RETURNING `+billingIntentColumns, intent.OrganizationID, intent.ID, charge.ID, charge.Source.ScannableCode.Image.DownloadURI))
	if errors.Is(err, pgx.ErrNoRows) {
		return billing.Intent{}, billing.ErrConflict
	}
	if err != nil {
		return billing.Intent{}, err
	}
	if err := auditTenant(ctx, tx, intent.OrganizationID, intent.ActorUserID, "billing.qr_created", "billing_intent", intent.ID, time.Now().UTC()); err != nil {
		return billing.Intent{}, err
	}
	return updated, tx.Commit(ctx)
}

func (s *Store) IntentForCharge(ctx context.Context, chargeID string) (billing.Intent, error) {
	intent, err := scanBillingIntent(s.pool.QueryRow(ctx, `SELECT `+billingIntentColumns+` FROM billing_intents WHERE omise_charge_id=$1`, chargeID))
	if errors.Is(err, pgx.ErrNoRows) {
		return billing.Intent{}, billing.ErrUnknownCharge
	}
	return intent, err
}

func (s *Store) Activate(ctx context.Context, intent billing.Intent, charge billing.Charge, now time.Time) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var id, organizationID, status, planKey, interval string
	var amount int64
	err = tx.QueryRow(ctx, `SELECT id,organization_id,status,plan_key,billing_interval,amount_satang FROM billing_intents
        WHERE id=$1 AND omise_charge_id=$2 FOR UPDATE`, intent.ID, charge.ID).Scan(&id, &organizationID, &status, &planKey, &interval, &amount)
	if err != nil {
		return err
	}
	if organizationID != intent.OrganizationID || planKey != string(intent.Plan) || interval != string(intent.Interval) || amount != charge.Amount {
		return billing.ErrProvider
	}
	if status == "paid" {
		return tx.Commit(ctx)
	}
	if status != "awaiting_payment" && status != "expired" {
		return billing.ErrConflict
	}
	var lockedOrganization string
	if err := tx.QueryRow(ctx, `SELECT id FROM organizations WHERE id=$1 FOR UPDATE`, organizationID).Scan(&lockedOrganization); err != nil {
		return err
	}
	var currentPlan plan.Key
	var currentInterval plan.Interval
	var paidThrough time.Time
	var anchorDay int
	err = tx.QueryRow(ctx, `SELECT plan_key,billing_interval,paid_through,anchor_day FROM billing_subscriptions WHERE organization_id=$1 FOR UPDATE`, organizationID).Scan(&currentPlan, &currentInterval, &paidThrough, &anchorDay)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return err
	}
	paidAt := charge.PaidAt.UTC()
	start := paidAt
	if !paidThrough.After(start) {
		anchorDay = start.Day()
	}
	if err == nil && paidThrough.After(start) {
		if currentPlan != intent.Plan || currentInterval != intent.Interval {
			return billing.ErrConflict
		}
		start = paidThrough
	}
	months := 1
	if intent.Interval == plan.SixMonths {
		months = 6
	} else if intent.Interval == plan.Yearly {
		months = 12
	}
	end := billing.AddMonthsClamped(start, months, anchorDay)
	var periodID string
	err = tx.QueryRow(ctx, `INSERT INTO organization_plan_periods
        (id,organization_id,plan_key,source,external_reference,starts_at,ends_at,created_at)
        VALUES (gen_random_uuid(),$1,$2,'billing',$3,$4,$5,$6)
        ON CONFLICT (organization_id,source,external_reference) DO UPDATE SET external_reference=excluded.external_reference
        RETURNING id`, organizationID, intent.Plan, charge.ID, start, end, now).Scan(&periodID)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `INSERT INTO billing_subscriptions
		(organization_id,plan_key,billing_interval,anchor_day,paid_through,grace_until,last_intent_id,updated_at)
		VALUES ($1,$2,$3,$4,$5,$5::timestamptz+interval '7 days',$6,$7)
		ON CONFLICT (organization_id) DO UPDATE SET plan_key=excluded.plan_key,billing_interval=excluded.billing_interval,
		anchor_day=excluded.anchor_day,paid_through=excluded.paid_through,grace_until=excluded.grace_until,last_intent_id=excluded.last_intent_id,
		cancel_at_period_end=false,updated_at=excluded.updated_at`, organizationID, intent.Plan, intent.Interval, anchorDay, end, intent.ID, now)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `UPDATE billing_intents SET status='paid',paid_at=$2,starts_at=$3,ends_at=$4 WHERE id=$1`, intent.ID, paidAt, start, end)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `INSERT INTO audit_events (organization_id,event_type,target_type,target_id,outcome,occurred_at)
        VALUES ($1,'billing.payment_verified','billing_intent',$2,'success',$3)`, organizationID, intent.ID, now)
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *Store) MarkUnpaid(ctx context.Context, intent billing.Intent, status string, now time.Time) error {
	if status != "failed" && status != "expired" {
		return billing.ErrProvider
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	command, err := tx.Exec(ctx, `UPDATE billing_intents SET status=$3
        WHERE id=$1 AND omise_charge_id=$2 AND status='awaiting_payment'`, intent.ID, intent.ChargeID, status)
	if err != nil {
		return err
	}
	if command.RowsAffected() > 0 {
		if err := auditTenant(ctx, tx, intent.OrganizationID, intent.ActorUserID,
			"billing.payment_"+status, "billing_intent", intent.ID, now); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (s *Store) QueueEvent(ctx context.Context, event billing.Event) error {
	_, err := s.pool.Exec(ctx, `INSERT INTO billing_provider_events(event_id,charge_id,event_type) VALUES ($1,$2,$3)
        ON CONFLICT (event_id) DO NOTHING`, event.ID, event.ChargeID, event.Type)
	return err
}

func (s *Store) ClaimEvents(ctx context.Context, limit int) ([]billing.Event, error) {
	rows, err := s.pool.Query(ctx, `WITH due AS (
        SELECT event_id FROM billing_provider_events WHERE
          (status IN ('pending','retry') AND next_attempt_at<=now()) OR
          (status='processing' AND next_attempt_at<now()-interval '5 minutes')
        ORDER BY next_attempt_at,event_id FOR UPDATE SKIP LOCKED LIMIT $1
    ) UPDATE billing_provider_events e SET status='processing',attempts=attempts+1
    FROM due WHERE e.event_id=due.event_id RETURNING e.event_id,e.charge_id,e.event_type`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var events []billing.Event
	for rows.Next() {
		var e billing.Event
		if err := rows.Scan(&e.ID, &e.ChargeID, &e.Type); err != nil {
			return nil, err
		}
		events = append(events, e)
	}
	return events, rows.Err()
}

func (s *Store) FinishEvent(ctx context.Context, id string, ok bool) error {
	status := "retry"
	if ok {
		status = "processed"
	}
	_, err := s.pool.Exec(ctx, `UPDATE billing_provider_events SET status=$2,
        next_attempt_at=CASE WHEN $2='retry' THEN now()+interval '5 minutes' ELSE next_attempt_at END
        WHERE event_id=$1 AND status='processing'`, id, status)
	return err
}

func (s *Store) PendingCharges(ctx context.Context, limit int) ([]string, error) {
	rows, err := s.pool.Query(ctx, `SELECT omise_charge_id FROM billing_intents WHERE status IN ('awaiting_payment','expired')
        AND omise_charge_id IS NOT NULL AND created_at>now()-interval '2 days'
        ORDER BY created_at,id LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func (s *Store) Summary(ctx context.Context, actorUserID, organizationID string, now time.Time) (billing.Summary, error) {
	tx, err := s.organizationTx(ctx, actorUserID, organizationID)
	if err != nil {
		return billing.Summary{}, err
	}
	defer tx.Rollback(ctx)
	role, err := currentRole(ctx, tx, actorUserID, organizationID)
	if err != nil || role != tenant.Owner {
		return billing.Summary{}, billing.ErrForbidden
	}
	summary := billing.Summary{Plan: plan.Free, Status: "free"}
	var paidThrough, graceUntil time.Time
	err = tx.QueryRow(ctx, `SELECT plan_key,billing_interval,paid_through,grace_until,cancel_at_period_end
        FROM billing_subscriptions WHERE organization_id=$1`, organizationID).Scan(
		&summary.Plan, &summary.Interval, &paidThrough, &graceUntil, &summary.CancelAtPeriodEnd)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return billing.Summary{}, err
	}
	if err == nil {
		summary.PaidThrough = &paidThrough
		summary.GraceUntil = &graceUntil
		switch {
		case now.Before(paidThrough):
			if summary.CancelAtPeriodEnd {
				summary.Status = "canceling"
			} else {
				summary.Status = "active"
			}
		case !summary.CancelAtPeriodEnd && now.Before(graceUntil):
			summary.Status = "grace"
		default:
			summary.Status = "expired"
		}
	}
	if summary.Status == "free" || summary.Status == "expired" {
		var source string
		var accessEnd time.Time
		err = tx.QueryRow(ctx, `SELECT plan_key,source,ends_at FROM organization_plan_periods
            WHERE organization_id=$1 AND source IN ('manual','trial') AND starts_at<=$2 AND ends_at>$2
            ORDER BY starts_at DESC,created_at DESC LIMIT 1`, organizationID, now).Scan(&summary.Plan, &source, &accessEnd)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return billing.Summary{}, err
		}
		if err == nil {
			summary.Status = source
			summary.PaidThrough = &accessEnd
		}
	}
	pending, err := scanBillingIntent(tx.QueryRow(ctx, `SELECT `+billingIntentColumns+` FROM billing_intents
        WHERE organization_id=$1 AND status IN ('creating','awaiting_payment','creation_unknown') ORDER BY created_at DESC LIMIT 1`, organizationID))
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return billing.Summary{}, err
	}
	if err == nil {
		summary.Pending = &pending
		summary.PendingExpired = !now.Before(pending.ExpiresAt)
	}
	if err := tx.QueryRow(ctx, `SELECT status FROM billing_intents WHERE organization_id=$1
        ORDER BY created_at DESC,id DESC LIMIT 1`, organizationID).Scan(&summary.LatestPaymentStatus); err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return billing.Summary{}, err
	}
	return summary, tx.Commit(ctx)
}

func (s *Store) SetCancellation(ctx context.Context, actorUserID, organizationID string, cancel bool, now time.Time) error {
	tx, err := s.organizationTx(ctx, actorUserID, organizationID)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	role, err := currentRole(ctx, tx, actorUserID, organizationID)
	if err != nil || role != tenant.Owner {
		return billing.ErrForbidden
	}
	command, err := tx.Exec(ctx, `UPDATE billing_subscriptions SET cancel_at_period_end=$3,updated_at=$4
        WHERE organization_id=$1 AND paid_through>$2 AND cancel_at_period_end<>$3`, organizationID, now, cancel, now)
	if err != nil {
		return err
	}
	if command.RowsAffected() > 0 {
		event := "billing.cancel_scheduled"
		if !cancel {
			event = "billing.cancel_undone"
		}
		if err := auditTenant(ctx, tx, organizationID, actorUserID, event, "billing_subscription", organizationID, now); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}
