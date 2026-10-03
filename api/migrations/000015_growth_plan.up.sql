ALTER TABLE organization_plan_periods DROP CONSTRAINT organization_plan_periods_plan_key_check;
ALTER TABLE organization_plan_periods ADD CONSTRAINT organization_plan_periods_plan_key_check
    CHECK (plan_key IN ('Starter','Business','Growth','AccountingFirm'));

ALTER TABLE billing_intents DROP CONSTRAINT billing_intents_plan_key_check;
ALTER TABLE billing_intents ADD CONSTRAINT billing_intents_plan_key_check
    CHECK (plan_key IN ('Starter','Business','Growth'));

ALTER TABLE billing_subscriptions DROP CONSTRAINT billing_subscriptions_plan_key_check;
ALTER TABLE billing_subscriptions ADD CONSTRAINT billing_subscriptions_plan_key_check
    CHECK (plan_key IN ('Starter','Business','Growth'));
