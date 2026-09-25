# Phase 11 PromptPay amendment

Status: Agreed payment direction, 2026-09-25. This supersedes the Stripe-specific payment and automatic-renewal portions of [Phase 11 decisions](./phase-11-decisions.md), [spec](./phase-11-spec.md), and the paused tickets. It does not enable billing or authorize live charges.

## Decisions

- Stripe is canceled. Accept THB through PromptPay QR only for the first commercial version. Each monthly, six-month, or annual period is a separate customer-approved payment. Do not describe this as automatic debit or store a reusable payment method.
- Keep the approved Plan prices, intervals, grants, credit packs, Owner-only billing authority, tenant isolation, seven-day grace, receipt rules, and data retention unless changed below. The displayed charge is the full amount for the selected period. Show the period end and when the next manual payment will be due.
- A paid period begins only after a verified provider payment. A browser return, QR display, receipt screenshot, or a client's claim of payment does not grant rights.
- Send a renewal reminder seven days before the paid-through date with a fresh payment action. If the next period is unpaid at expiry, enter the agreed seven-day grace state without deleting Documents. After grace, use Free/Plan Over Limit. A later verified payment starts a new paid period on its payment date; show that date before the customer pays. Purchased Scan Credits remain with the Organization.
- Cancel means stop renewal reminders and end paid rights at the already paid-through date. Since PromptPay does not debit automatically, cancellation must never imply that a future charge was scheduled. Resume before the end date restores reminders; after the end date the Owner starts a new payment.
- The trusted server catalog determines Plan, interval, and THB amount. One-time Scan Credit packs also use a separate verified PromptPay payment when OCR launch gates pass.

## Provider and release gate

Omise is the implementation candidate because its documentation supports PromptPay charges, webhook signature verification, and Thai individual-merchant applications. A real merchant account, test keys, webhook secret, and PromptPay channel activation are still unverified. Keep the payment feature flag off and do not deploy a live payment flow until those checks and staging smoke pass. Do not expose provider keys or raw payment payloads in logs.

The Omise PromptPay flow uses a customer-approved QR; its webhook is an asynchronous hint. Verify the signed raw webhook, then retrieve the charge server-side and match the stored charge ID, THB amount, payment method, mode, and Organization-owned intent before applying one paid period. Reconcile missed webhook events. Omise's documentation says PromptPay refunds cannot currently be initiated through its API, so the existing manual support refund policy remains.

Sources: [PromptPay flow and limits](https://docs.omise.co/th/promptpay/thailand), [webhook signature verification](https://docs.omise.co/api-webhooks), [Thai individual and company account documents](https://docs.omise.co/th/how-do-i-enable-live-account/thailand).
