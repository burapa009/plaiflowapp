-- Rotate bounded reconciliation across all pending/expired charges, including older ones.
ALTER TABLE billing_intents ADD COLUMN IF NOT EXISTS next_reconcile_at timestamptz NOT NULL DEFAULT now();
CREATE INDEX IF NOT EXISTS billing_intents_reconcile_due ON billing_intents (next_reconcile_at,id)
    WHERE status IN ('awaiting_payment','expired') AND omise_charge_id IS NOT NULL;
