-- Expand-only PromptPay ledger. Billing routes remain disabled until provider and staging gates pass.
CREATE TABLE billing_intents (
    id uuid PRIMARY KEY,
    organization_id uuid NOT NULL REFERENCES organizations(id),
    actor_user_id uuid NOT NULL REFERENCES users(id),
    plan_key text NOT NULL CHECK (plan_key IN ('Starter','Business')),
    billing_interval text NOT NULL CHECK (billing_interval IN ('monthly','six_months','yearly')),
    amount_satang bigint NOT NULL CHECK (amount_satang > 0),
    status text NOT NULL CHECK (status IN ('creating','awaiting_payment','paid','failed','expired','creation_unknown')),
    omise_charge_id text UNIQUE,
    qr_image_url text,
    created_at timestamptz NOT NULL,
    expires_at timestamptz NOT NULL,
    paid_at timestamptz,
    starts_at timestamptz,
    ends_at timestamptz,
    CHECK (expires_at > created_at),
    UNIQUE (organization_id,id)
);
CREATE UNIQUE INDEX billing_intents_one_open_per_org ON billing_intents (organization_id)
    WHERE status IN ('creating','awaiting_payment','creation_unknown');
CREATE INDEX billing_intents_org_history ON billing_intents (organization_id,created_at DESC);
CREATE INDEX billing_intents_reconcile ON billing_intents (created_at,id)
    WHERE status IN ('creating','awaiting_payment','creation_unknown');

CREATE TABLE billing_subscriptions (
    organization_id uuid PRIMARY KEY REFERENCES organizations(id),
    plan_key text NOT NULL CHECK (plan_key IN ('Starter','Business')),
    billing_interval text NOT NULL CHECK (billing_interval IN ('monthly','six_months','yearly')),
    anchor_day integer NOT NULL CHECK (anchor_day BETWEEN 1 AND 31),
    paid_through timestamptz NOT NULL,
    grace_until timestamptz NOT NULL,
    cancel_at_period_end boolean NOT NULL DEFAULT false,
    last_intent_id uuid NOT NULL,
    updated_at timestamptz NOT NULL,
    FOREIGN KEY (organization_id,last_intent_id) REFERENCES billing_intents (organization_id,id),
    CHECK (grace_until = paid_through + interval '7 days')
);
CREATE INDEX billing_subscriptions_due ON billing_subscriptions (paid_through)
    WHERE cancel_at_period_end = false;

CREATE TABLE billing_provider_events (
    event_id text PRIMARY KEY,
    charge_id text NOT NULL,
    event_type text NOT NULL,
    status text NOT NULL DEFAULT 'pending' CHECK (status IN ('pending','processing','processed','retry')),
    received_at timestamptz NOT NULL DEFAULT now(),
    next_attempt_at timestamptz NOT NULL DEFAULT now(),
    attempts integer NOT NULL DEFAULT 0
);
CREATE INDEX billing_provider_events_due ON billing_provider_events (next_attempt_at,event_id)
    WHERE status IN ('pending','retry');

ALTER TABLE billing_intents ENABLE ROW LEVEL SECURITY;
ALTER TABLE billing_subscriptions ENABLE ROW LEVEL SECURITY;
ALTER TABLE billing_provider_events ENABLE ROW LEVEL SECURITY;
CREATE POLICY billing_intents_owner_read ON billing_intents FOR SELECT
    USING (organization_id=app_organization_id() AND organization_role(app_user_id(),organization_id)='Owner');
CREATE POLICY billing_subscriptions_owner_read ON billing_subscriptions FOR SELECT
    USING (organization_id=app_organization_id() AND organization_role(app_user_id(),organization_id)='Owner');

CREATE OR REPLACE FUNCTION effective_organization_plan(candidate_organization_id uuid)
RETURNS text
LANGUAGE sql
STABLE
SECURITY DEFINER
SET search_path = public, pg_temp
AS $$
    SELECT coalesce((
        SELECT plan_key FROM billing_subscriptions
        WHERE organization_id=candidate_organization_id
          AND (paid_through>now() OR (NOT cancel_at_period_end AND grace_until>now()))
    ),(
        SELECT plan_key FROM organization_plan_periods
        WHERE organization_id=candidate_organization_id AND source<>'billing'
          AND starts_at<=now() AND ends_at>now()
        ORDER BY starts_at DESC,created_at DESC LIMIT 1
    ),'Free')
$$;
