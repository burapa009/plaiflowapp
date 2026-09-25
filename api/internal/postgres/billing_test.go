package postgres

import (
	"errors"
	"testing"
	"time"

	"plaiflow/api/internal/billing"
	"plaiflow/api/internal/plan"
)

func TestPromptPayPeriodIsTenantScopedAndReplaySafe(t *testing.T) {
	store, ctx := isolatedTestStore(t, 14)
	owner, member, otherOwner := postgresUUID(), postgresUUID(), postgresUUID()
	org, other := postgresUUID(), postgresUUID()
	now := time.Now().UTC().Truncate(time.Second)
	if _, err := store.pool.Exec(ctx, `INSERT INTO users(id) VALUES($1),($2),($3)`, owner, member, otherOwner); err != nil {
		t.Fatal(err)
	}
	if _, err := store.pool.Exec(ctx, `INSERT INTO organizations(id,name) VALUES($1,'Buyer'),($2,'Other')`, org, other); err != nil {
		t.Fatal(err)
	}
	if _, err := store.pool.Exec(ctx, `INSERT INTO memberships(organization_id,user_id,role) VALUES($1,$2,'Owner'),($1,$3,'Member'),($4,$5,'Owner')`, org, owner, member, other, otherOwner); err != nil {
		t.Fatal(err)
	}
	newIntent := func(actor, organization string) billing.Intent {
		return billing.Intent{ID: postgresUUID(), OrganizationID: organization, ActorUserID: actor,
			Plan: plan.Starter, Interval: plan.SixMonths, AmountSatang: 85500, ExpiresAt: now.Add(30 * time.Minute)}
	}
	if _, err := store.CreateIntent(ctx, newIntent(member, org), now); !errors.Is(err, billing.ErrForbidden) {
		t.Fatalf("member checkout: %v", err)
	}
	if _, err := store.CreateIntent(ctx, newIntent(owner, other), now); !errors.Is(err, billing.ErrForbidden) {
		t.Fatalf("foreign checkout: %v", err)
	}
	intent, err := store.CreateIntent(ctx, newIntent(owner, org), now)
	if err != nil {
		t.Fatal(err)
	}
	charge := billing.Charge{ID: "chrg_test_unique", Status: "successful", Paid: true, Currency: "THB", Amount: 85500, Metadata: map[string]string{"intent_id": intent.ID}}
	charge.Source.Type = "promptpay"
	charge.Source.ScannableCode.Image.DownloadURI = "https://api.omise.co/charges/chrg_test_unique/documents/qr/downloads/safe"
	charge.PaidAt = &now
	intent, err = store.AttachCharge(ctx, intent, charge)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Activate(ctx, intent, charge, now); err != nil {
		t.Fatal(err)
	}
	if err := store.Activate(ctx, intent, charge, now); err != nil {
		t.Fatal("duplicate:", err)
	}
	for i := 0; i < 2; i++ {
		if err := store.QueueEvent(ctx, billing.Event{ID: "evnt_test_unique", ChargeID: charge.ID, Type: "charge.complete"}); err != nil {
			t.Fatal(err)
		}
	}
	var events int
	if err := store.pool.QueryRow(ctx, `SELECT count(*) FROM billing_provider_events WHERE event_id='evnt_test_unique'`).Scan(&events); err != nil || events != 1 {
		t.Fatalf("events=%d err=%v", events, err)
	}
	var periods int
	if err := store.pool.QueryRow(ctx, `SELECT count(*) FROM organization_plan_periods WHERE organization_id=$1 AND source='billing'`, org).Scan(&periods); err != nil || periods != 1 {
		t.Fatalf("periods=%d err=%v", periods, err)
	}
	failedIntent, err := store.CreateIntent(ctx, newIntent(owner, org), now)
	if err != nil {
		t.Fatal(err)
	}
	failedCharge := charge
	failedCharge.ID = "chrg_test_failed"
	failedIntent, err = store.AttachCharge(ctx, failedIntent, failedCharge)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		if err := store.MarkUnpaid(ctx, failedIntent, "failed", now); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := store.CreateIntent(ctx, newIntent(owner, org), now); err != nil {
		t.Fatal("failed payment blocked retry:", err)
	}
	if err := store.pool.QueryRow(ctx, `SELECT count(*) FROM organization_plan_periods WHERE organization_id=$1 AND source='billing'`, org).Scan(&periods); err != nil || periods != 1 {
		t.Fatalf("failed payment added period: %d %v", periods, err)
	}
	key, err := store.EffectivePlan(ctx, org)
	if err != nil || key != plan.Starter {
		t.Fatalf("effective plan=%s err=%v", key, err)
	}
	otherKey, err := store.EffectivePlan(ctx, other)
	if err != nil || otherKey != plan.Free {
		t.Fatalf("cross tenant plan=%s err=%v", otherKey, err)
	}
	if _, err := store.Summary(ctx, member, org, now); !errors.Is(err, billing.ErrForbidden) {
		t.Fatalf("member summary: %v", err)
	}
	if _, err := store.Summary(ctx, owner, other, now); !errors.Is(err, billing.ErrForbidden) {
		t.Fatalf("foreign summary: %v", err)
	}
	if _, err := store.pool.Exec(ctx, `UPDATE billing_subscriptions SET paid_through=$2,grace_until=$2::timestamptz+interval '7 days' WHERE organization_id=$1`, org, now.Add(-24*time.Hour)); err != nil {
		t.Fatal(err)
	}
	key, err = store.EffectivePlan(ctx, org)
	if err != nil || key != plan.Starter {
		t.Fatalf("grace plan=%s err=%v", key, err)
	}
	if _, err := store.pool.Exec(ctx, `UPDATE billing_subscriptions SET paid_through=$2,grace_until=$2::timestamptz+interval '7 days' WHERE organization_id=$1`, org, now.Add(-8*24*time.Hour)); err != nil {
		t.Fatal(err)
	}
	key, err = store.EffectivePlan(ctx, org)
	if err != nil || key != plan.Free {
		t.Fatalf("expired plan=%s err=%v", key, err)
	}
	if _, err := store.pool.Exec(ctx, `UPDATE billing_subscriptions SET paid_through=$2,grace_until=$2::timestamptz+interval '7 days',cancel_at_period_end=true WHERE organization_id=$1`, org, now.Add(-24*time.Hour)); err != nil {
		t.Fatal(err)
	}
	key, err = store.EffectivePlan(ctx, org)
	if err != nil || key != plan.Free {
		t.Fatalf("canceled plan gained grace=%s err=%v", key, err)
	}
	if _, err := store.pool.Exec(ctx, `UPDATE billing_subscriptions SET grace_until=$2 WHERE organization_id=$1`, org, now.Add(-time.Minute)); err == nil {
		t.Fatal("invalid grace constraint accepted")
	}
}
