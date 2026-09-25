-- Only for an unused staging migration. Disable billing instead once payment history exists.
CREATE OR REPLACE FUNCTION effective_organization_plan(candidate_organization_id uuid)
RETURNS text
LANGUAGE sql
STABLE
SECURITY DEFINER
SET search_path = public, pg_temp
AS $$
    SELECT coalesce((
        SELECT plan_key FROM organization_plan_periods
        WHERE organization_id=candidate_organization_id AND starts_at<=now() AND ends_at>now()
        ORDER BY starts_at DESC,created_at DESC LIMIT 1
    ),'Free')
$$;
DROP TABLE billing_provider_events;
DROP TABLE billing_subscriptions;
DROP TABLE billing_intents;
