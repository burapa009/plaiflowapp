# PromptPay payment failure and grace

Status: staging procedure. Public checkout stays off until the Phase 12 gates pass. A charge, QR, browser return, or screenshot is never proof of paid access.

## Triage

1. Record the Organization ID, intent ID, charge ID, UTC time, current deployment, and case owner in the private incident record. Keep provider keys, webhook signatures, QR contents, and customer billing details out of logs and group messages.
2. Ask the Owner to check the in-app Billing status. Check the matching Omise charge in the **same mode** as the API, then compare its ID, intent metadata, THB amount, PromptPay method, status, and paid timestamp with the local intent. Do not manually create a paid period.
3. If Omise is unavailable or the charge is still pending, leave the intent pending and paid rights at the last verified state. Retry reconciliation when the provider returns. If Omise confirms failure or expiry, ensure the intent is marked unpaid and no new period was added.
4. If Omise confirms success but rights are missing after five minutes, inspect webhook acceptance, `billing_reconciliation_unavailable`, `billing_entitlement_lag`, event processing, and the intent/period counts. Escalate at fifteen minutes. Reprocess through the existing reconciliation path; verify exactly one period and the correct paid-through instant. A log warning alone is not a delivered alert.

## Renewal timeline

| Time | Expected customer state | Operator check |
| --- | --- | --- |
| Seven days before paid-through | Manual PromptPay renewal reminder via verified Billing Contact Email and in-app notice | Confirm delivery and the new payment action. These reminder mechanisms are still a release blocker. |
| At paid-through | If no verified renewal and cancellation is not scheduled, enter grace for exactly seven days | Check the UTC instant and Organization Timezone display; preserve Documents and permitted reads/exports. |
| During grace | Owner can make a new manual payment; unverified attempts do not extend rights | Check failed/expired intents and reconciliation; never ask for a QR screenshot as payment proof. |
| At grace end | Apply Free/Plan Over Limit safeguards; keep existing Documents and security/recovery access | Check that paid allowance suspension and export rights follow the accepted transition rules. Paid carry remains unimplemented. |
| Later verified payment | Start a new paid period at the provider-confirmed payment instant | Check one period, Entitlement, allowance ledger, and Seller Receipt. Receipt and paid carry are still release blockers. |

A scheduled cancellation preserves paid rights until paid-through and receives no grace. The Owner can undo it before paid-through. A failed charge during an already paid period must not remove that period. No automatic debit or automatic PromptPay retry is promised.

## Escalation and rollback

- If charge identity or provider status disagrees with the local intent, stop new checkouts, retain the event and audit trail, and escalate to the named billing operator. Do not repair by changing an Entitlement row directly.
- In staging, close new checkout by clearing `BILLING_TEST_ORGANIZATION_ID` while keeping `BILLING_ENABLED=true`; that preserves the reconciliation loop. This also hides the test Organization's Billing API until the allowlist is restored, so the operator must verify its ledger through private database access. Do not set `BILLING_ENABLED=false` during a payment repair incident: that stops reconciliation. A separate production checkout pause that retains customer Billing reads is still a public-sale gate. Restore the preceding application deployment if needed; retain additive billing tables and paid periods. Do not run the Phase 11 down migration after payment data exists.
- Before reopening checkout, reconcile every provider-confirmed charge since the incident start, check for duplicates and missing periods/receipts, verify Owner access and cross-tenant denial, and run a test-mode payment failure and success smoke. Record the build, environment, timestamps, evidence, case owner, and remaining risk.

This procedure is not a passed alert, reminder, receipt, restore, or production payment drill. Those require a named operator and observed evidence in the Phase 12 gate record.
