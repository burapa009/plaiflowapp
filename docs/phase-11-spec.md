# Phase 11 — Subscription, Billing, and Payments

Status: Stripe payment design superseded by the [PromptPay amendment](./phase-11-promptpay-amendment.md), 2026-09-25. The provider-specific sections below are historical. This document does not activate payment features.

## Problem Statement

An Organization can see package information and existing server-side Entitlement seams, but it cannot buy, renew, change, or recover a commercial Subscription through a verified payment flow. Owners need clear Thai pricing, reliable payment and receipt history, and predictable limits; staff need the right access without exposure to another Organization's Billing data.

## Solution

Let an Organization Owner buy a fixed THB Plan and Billing Interval through Stripe Hosted Checkout, manage the Subscription from PlaiFlow, and buy separate Scan Credits when OCR is ready. Verified provider payment updates local Subscription state and Entitlements. Billing shows the full renewal charge, dates, Document Allowance, Scan Credits, payment problems, and seller receipts. Existing Documents remain available through downgrades and payment failures according to the agreed Plan Over Limit rules.

## User Stories

1. As an Organization Owner, I want to compare monthly, six-month, and annual full charges on mobile, so that I know what will be charged and when.
2. As an Organization Owner, I want to see effective monthly prices and discounts separately from the full charge, so that comparisons do not obscure the renewal amount.
3. As a Free user, I want Free and the one-time Business Trial visible, so that I can choose without pressure to pay.
4. As an Organization Owner, I want to review my Plan, interval, buyer details, and renewal terms before Stripe Checkout, so that I can catch mistakes.
5. As an Organization Owner, I want a hosted card checkout, so that PlaiFlow never receives my card number or CVV.
6. As an Organization Owner, I want paid features only after payment is verified, so that a pending checkout is represented honestly.
7. As an Organization Owner, I want to see my current Plan, full next charge, renewal date, and billing status, so that I can manage the Subscription.
8. As an Admin, I want to see Plan status and renewal date without payment-sensitive history, so that I can plan work safely.
9. As an Organization Owner, I want to update my payment method and see Stripe invoice/payment history, so that I can recover a failed charge.
10. As an Organization Owner, I want an ordinary seller receipt for each confirmed payment, so that I have a record distinct from Stripe payment evidence.
11. As an Organization Owner, I want to upgrade after seeing unused-time credit and the amount due, so that I know the immediate charge.
12. As an Organization Owner, I want downgrades and shorter intervals scheduled for the paid period's end, so that my paid rights remain until then.
13. As an Organization Owner, I want to cancel renewal with a clear paid-through date and undo it before that date, so that I control the Subscription.
14. As an Organization Owner, I want a seven-day grace period and clear reminders after a failed renewal, so that I have time to fix payment.
15. As an Organization Owner, I want to pay the outstanding invoice during recovery, so that paid rights can return without a duplicate Subscription.
16. As an Organization member, I want Documents preserved when payment fails or a Plan ends, so that commercial status does not erase business records.
17. As an Organization Owner, I want to see new, carried, spent, suspended, and expiring Document Allowance, so that I can predict intake capacity.
18. As an Organization Owner, I want purchased Scan Credits shown separately from Document Allowance, so that I understand which balance controls OCR.
19. As an Organization Owner, I want to buy a fixed one-time Scan Credit pack when OCR is ready, so that I can scan more pages without metered surprises.
20. As a reviewer, I want a multi-page file to reserve its full page count and charge only after successful OCR, so that failed retries do not waste credits.
21. As a firm Owner, I want firm seats, client relationships, LINE groups, and my own Document Allowance displayed separately from client limits, so that I do not mistake firm payment for client payment.
22. As a client Owner, I want my Organization to retain its own Plan and Scan Credits when a firm assists us, so that our billing remains under our control.
23. As an unauthorized staff member or another tenant, I want billing actions and sensitive records denied, so that commercial data stays private.
24. As an operator, I want missed or duplicated Stripe events reconciled safely, so that paid rights match verified payment state.
25. As an Organization Owner, I want a truthful pending or provider-unavailable state, so that the UI never claims a charge succeeded without verification.

## Implementation Decisions

### Rollout and provider boundary

- Keep Free and a one-time 14-day Business Trial without a card. Sell Starter and Business subscriptions first using Stripe Hosted Checkout, cards, THB, and automatic renewal. Show but do not sell Accounting Firm until Phase 10 activation gates pass. Sell one-time Scan Credits only after OCR accuracy, throughput, cost, and queue-fairness gates pass.
- Billing intervals are monthly, six months, and annual. Use one trusted fixed recurring Stripe Price per paid Plan/interval, including native `month` × 6 Prices. Keep the approved full-charge catalog in the decisions document; show the full charge and renewal term more prominently than effective monthly price.
- Each Organization is one payer and one Stripe Customer. Client Organizations of a firm pay separately. No multi-currency, metered Stripe billing, coupons, marketplace payout, or bespoke enterprise invoicing in this version.
- Only Owner can buy, change, cancel, resume, view payment-sensitive history, or download seller receipts. Admin can view Plan, usage, status, and dates. Every operation resolves current Organization Membership and the stored provider IDs server-side.

### Plan, Price, BillingInterval, and Entitlements

- `BillingInterval` has the domain values `MONTHLY`, `SIX_MONTH`, and `YEARLY`. They represent recurring payment periods of 1, 6, and 12 months; Document Allowance still grants monthly. Free has no recurring payment even when it appears in the comparison selector.
- `Plan` groups trusted Entitlements and limits. `Price` is a fixed THB full charge for exactly one paid Plan and BillingInterval. The six-month price is 5% below six monthly charges; the annual price is 15% below twelve monthly charges. The approved amounts are fixed catalog entries rather than a client-side discount calculation. The server maps each paid Plan + BillingInterval to its trusted Stripe Product/Price IDs and rejects unknown or mismatched IDs and amounts.

| Plan | MONTHLY | SIX_MONTH | YEARLY |
|---|---:|---:|---:|
| Free | ฿0 | ฿0 | ฿0 |
| Starter | ฿150 | ฿855 | ฿1,530 |
| Business | ฿250 | ฿1,425 | ฿2,550 |
| Accounting Firm | ฿1,000 | ฿5,700 | ฿10,200 |

The effective monthly comparison is Starter ฿150/฿142.50/฿127.50, Business ฿250/฿237.50/฿212.50, and Accounting Firm ฿1,000/฿950/฿850 for `MONTHLY`/`SIX_MONTH`/`YEARLY`. These figures are display-only; the full charges above are payable and renew on their stated interval.

- Stripe uses a native recurring Price with `interval=month`, `interval_count=6` for `SIX_MONTH`. If the connected account cannot support it, six-month Checkout remains unavailable; a one-time charge is not treated as an automatically renewing Subscription.
- Only the verified local Plan/Subscription state determines commercial Entitlements. Current Membership and role still determine data access. Free/Starter/Business keep their current capability differences; Accounting Firm adds firm workspace capability without replacing Client Organization authorization. Existing trusted manual Plan periods remain effective until their recorded end and are not silently converted or back-charged.

| Plan | Members | LINE groups | New Documents/month | Client relationships | Scan pages included |
|---|---:|---:|---:|---:|---:|
| Free | 1 | 1 | 30 | — | 0 |
| Starter | 3 | 2 | 300 | — | 0 |
| Business | 10 | 5 | 1,000 | — | 0 |
| Accounting Firm | 5 | 5 | 1,000 for its own Organization | 20 | 0 |

### Usage, Document Allowance, and Scan Credits

- Trial has 100 unique accepted Documents total over 14 days and does not transfer unused allowance. Free allowance resets monthly. A paid monthly Document grant carries until 00:00 on the first day of the corresponding month two years later, in the Organization timezone recorded at issue. Spend the earliest-expiring grant first. There is no extra balance cap or hidden daily intake cap. A timezone change affects future grants only. Issue new grants on the first day at 00:00 Organization-local time, regardless of the BillingInterval.
- Upgrading during a month adds only the difference between the old and new monthly grant. Paid-to-paid downgrade at period end clamps remaining carry to 24 times the new Plan's monthly grant; preview the loss. Voluntary move to Free forfeits paid carry. Failed-payment expiry suspends paid carry for the 30-day recovery window and restores only unexpired grants when the outstanding payment is verified. No historical carry is minted when this policy first activates.
- Distinct from Document Allowance, purchased Scan Credits are Organization-owned, never expire, and remain on Free. One image or PDF page successfully scanned uses one credit. Reserve the full page count before OCR, settle once after a complete published result, release on terminal failure, and charge no extra for system retries or duplicates. A reprocess after success needs confirmation and fresh credits. Insufficient credits leave the Document accepted but unscanned; after purchase, the user selects pending scans. Preserve the 20-page PDF OCR limit until separately validated.
- The four one-time packs are 500/800/1,800/5,000 pages for ฿219/฿339/฿699/฿1,899. No automatic top-up. No new numerical AI or Export quotas; keep existing Plan capabilities and file/job limits. No automatic overage charges.

### BillingCustomer and local billing records

- One `BillingCustomer` binds one Organization to one Stripe Customer. Store the provider Customer and Subscription IDs on trusted local records; firm and client Organizations have different BillingCustomers. The Billing Profile holds Owner-reviewed buyer name, address, tax ID, and a separately verified Billing Contact email. Snapshot buyer details on each issued receipt; later profile edits affect future receipts only.
- `Subscription` records the selected Plan, BillingInterval, provider identity, current verified state, scheduled change or cancellation, grace deadline, and recovery deadline. `SubscriptionPeriod` records verified paid-through intervals. `PaymentAttempt` and Checkout-session records track pending, failed, expired, and confirmed attempts. `ProviderEvent` tracks verified event IDs and processing; invoice/payment and Seller Receipt metadata support history. `BillingAuditEvent` records actor or provider source and action outcome. A durable Document Allowance grant/spend ledger and Scan Credit purchase/reservation/settlement ledger keep balances race-safe.

### Subscription lifecycle and state machine

| Local state | Entry and Entitlements | Exit |
|---|---|---|
| Free or Trial | No verified paid period; Free or 14-day Trial rights apply | Verified initial payment starts a paid period; Trial expires to Free |
| Checkout pending | Existing rights remain; return redirect changes no rights | Verified payment activates; failed/expired Checkout returns to prior state |
| Active | Paid Plan rights until verified paid-through end | Renewal paid, scheduled change/cancellation effective, or failed renewal enters grace |
| Active with scheduled change/cancellation | Current paid rights and dates remain visible | At period end apply the scheduled change; cancellation ends renewal; Owner may undo cancellation before end |
| Past due in grace | Retain current paid rights for exactly seven days after first renewal failure | Verified outstanding payment restores Active; deadline moves local rights to Free/Plan Over Limit |
| Unpaid recoverable | Free/Plan Over Limit rights; paid Document carry is suspended | Verified outstanding payment within 30 days restores paid rights and unexpired carry; otherwise cancel old Subscription |
| Ended | Free/Plan Over Limit rights; Documents remain | New paid Checkout creates a new paid period |

Stripe status and local commercial state are distinct. The local state changes only from verified provider facts, trusted manual grants, and deterministic time-based transitions. Payment after a voluntary cancellation is a new Checkout; an unpaid recovery within its window reuses the original Subscription. A full refund of the current paid period or finally lost dispute removes paid rights; a partial refund is reviewed manually.

1. **Checkout:** Owner reviews Organization, buyer Billing Profile, Plan or pack, interval, full amount, renewal terms, and verified Billing Contact email. Server maps the request to a trusted Price ID, creates or reuses the Organization's Stripe Customer, and stores a scoped Checkout intent/session. A return URL is a locator only.
2. **Activation:** Verified provider card-payment and subscription facts update local `Subscription` and paid-through `SubscriptionPeriod` records. For this card-only version, an invoice merely marked paid out of band does not grant rights without a verified successful provider payment or a separately trusted manual Plan grant. The local verified state grants Entitlements. Initial failed payment leaves current Free/Trial rights. A repeated or expired Checkout session never creates a second active Subscription.
3. **Changes:** Paid upgrade charges the disclosed amount due after unused-time credit and activates only after verified payment. This is the agreed immediate proration case. A same-Plan move from monthly to six-month/yearly, a paid-to-paid downgrade, or a move from annual/six-month to a shorter interval takes effect at the next paid renewal without an immediate charge. Show the effective date, retained rights, and any Document Allowance loss before confirmation. Cancel stops renewal at period end after explicit confirmation; Owner can undo before that time.
4. **Failed renewal:** First failure starts seven days of paid-rights grace, with Owner email immediately and on days three and six if unpaid. Up to three automatic retries occur within grace. At seven days, local rights become Free/Plan Over Limit without deleting Documents. Stripe remains `unpaid` with a 30-day same-Subscription payment recovery path; after that deadline cancel it, and a later purchase starts a new Checkout. Restored rights require verified payment of the outstanding invoice.
5. **Refund/dispute:** Support handles refunds. Verified full current-period refund or finally lost dispute removes paid rights; partial refund is manually reviewed. Full Scan Credit pack refund removes unspent credits; spent credits need manual resolution. None of these deletes source Documents.
6. **Receipt:** Each confirmed subscription or pack payment creates one sequential ordinary seller receipt with immutable seller, buyer, amount, and payment snapshot. The Owner can download it; send it to the verified Billing Contact and retry email with the same receipt ID. Keep a copy at least five years. Stripe invoices/payment receipts remain distinct from the seller receipt; show no VAT line or VAT tax-invoice label while the seller is unregistered and lacks authority to issue one. Publish the agreed support-led, case-by-case refund policy before Checkout.

### Provider adapter, Checkout, webhooks, and reconciliation

- Keep one narrow payment-provider boundary for creating hosted Subscription or one-time pack Checkout, creating a restricted payment-method/invoice Portal session, previewing an agreed change, retrieving Customer/Subscription/Invoice/payment facts, applying a change or cancellation, and verifying webhook signatures. Stripe is the only adapter in this phase; Plan policy, tenant authorization, and Entitlement calculation stay in PlaiFlow.
- Checkout accepts only a server-catalog Plan + BillingInterval or pack key, scoped to the Owner's current Organization. A trusted catalog maps it to Stripe Product/Price ID and fixed amount. Persist session identity and idempotency key before redirect. Disable repeat submission, show pending verification on return, and allow safe retry after an expired or canceled session without a duplicate Subscription or pack grant.
- The webhook verifies the raw-body Stripe signature, persists the unique ProviderEvent ID durably, and acknowledges quickly. Slow work runs asynchronously. Event processing checks stored Customer/Subscription/Checkout identity, Organization, Price, amount, and current provider payment state; the event ID and paid invoice/payment ID are both idempotency boundaries. Out-of-order or replayed events must not regress or double-apply local state.
- Reconcile pending Checkout Sessions and past-due Subscriptions every 15 minutes, and active Subscriptions daily. Detect provider/local mismatch, repair from verified provider facts, and audit the correction. If Stripe is unavailable, show the last verified local provider state and a pending-sync message while local grace and cancellation deadlines still advance; a success redirect alone never grants rights.

### Trusted data, security, and privacy

- Keep Stripe secret keys and webhook signing secrets server-side only; store no card number or CVV.
- Persist `BillingCustomer` (Organization ↔ Stripe Customer), `Subscription` and paid-through periods, Checkout/payment attempts, `ProviderEvent` IDs and processing states, invoice/payment metadata, seller receipt metadata, and `BillingAuditEvent`. Existing `Usage` remains a measurement; add durable Document grant/spend records and Scan Credit purchase/reservation/settlement records. Keep provider payloads minimal and redact secrets and sensitive data from logs.
- Handle Checkout completion, subscription creation/update/deletion, invoice paid/failure/action-required, one-time payment success/failure, refunds, and finally resolved disputes. Verify the actual payment behind an `invoice.paid` event before granting card-paid rights because it can also represent an out-of-band mark-paid action.
- Entitlement checks and the Billing dashboard read indexed local Organization state first and never call Stripe per feature request. Document admission and OCR credit reservation use atomic, tenant-scoped transactions with idempotency keys. Avoid aggregate scans on hot authorization paths. Receipt and invoice access always rechecks Organization and role.

- Only current Organization Owner authority permits Checkout, Portal access, Plan/interval changes, cancellation, resumption, Billing Profile edits, or sensitive payment and seller-receipt reads. Admin sees status and renewal facts only; STAFF/VIEWER and firm delegates gain no billing authority. Every read, write, download, and background action is tenant-scoped. Server validation rejects tampered Plan, Product ID, Price ID, amount, and cross-tenant provider IDs.
- Record BillingAuditEvents for Checkout request/result, paid activation, renewal, interval/Plan change, cancellation/undo, failure/grace/recovery, credit purchase/spend/refund, receipt issue/correction, webhook rejection/application, and reconciliation repair. Keep actor or provider source, Organization, object references, outcome, and time without raw card data, secrets, or sensitive provider payloads. Restrict receipt files and invoice links by tenant and current role. Provider secrets remain server-side only.

### Billing and Pricing experience

- Mobile pricing cards stack, keep Free visible, mark Business recommended modestly, show actual charge and renewal terms before checkout, and show effective monthly price only as comparison. Cancellation states the paid-through date and asks for clear confirmation without a forced reason.
- The three-option interval selector updates all cards and remains selected through Checkout review. Show the full THB charge, exact renewal period, savings, and effective monthly comparison; never present the effective monthly figure as the billed amount. No preselected paid Plan, countdown pressure, obscured Free option, or blocking retention offer. Accounting Firm pricing may be visible while its purchase action is plainly unavailable pending Phase 10 activation.
- Billing shows current Plan, full next charge, interval, renewal/access-end date, payment state, member/LINE/client limits, Document monthly grant/carry/used/available/next expiry, and Scan Credit available/reserved/used. Free shows its monthly reset, not a carried balance. Distinguish seller receipts from Stripe invoice/payment history. Pending return, scheduled change, scheduled cancellation, past due/grace, grace expired, and provider outage have explicit text states. The Owner sees card/update and payment actions; Admin sees only status and dates.
- Show positive paid Document Allowance nearing expiry in-app 30 days beforehand. Email the Owner seven days before renewal; after first payment failure email immediately and, if still unpaid, on days three and six with amount and grace deadline. Email the ordinary seller receipt to the verified Billing Contact; retry failed delivery against the same receipt. LINE billing notices are deferred to an opt-in future flow.
- Show Scan Credit packs separately from recurring Plans with a one-time charge and “1 image or 1 PDF page = 1 credit.” A 20-page PDF requires 20 credits; OCR of a longer PDF remains unavailable under the current limit. If credits are insufficient, retain the Document, show missing credits, and let a user select pending scans after purchase. OCR failures release reservations; reprocessing after success needs explicit confirmation.

### Failure modes

| Condition | Required behavior |
|---|---|
| Checkout canceled/expired or initial card charge fails | Keep existing Free/Trial/paid rights; allow safe retry without duplicate Customer or Subscription |
| Browser returns from success URL before verified payment | Show pending verification; grant no new paid rights or Scan Credits |
| Forged, duplicate, or out-of-order webhook | Reject bad signature; process valid payment once; reconcile current provider state |
| Missed webhook or provider outage | Keep last verified local provider snapshot, advance deterministic local deadlines, show sync pending, repair drift after provider recovery |
| Renewal failure | Enter seven-day grace, then Free/Plan Over Limit, retain Documents and recovery/security access; recover same Subscription for 30 days |
| Receipt email failure | Preserve the issued receipt and retry delivery without changing its number or financial content |
| OCR terminal failure or system retry | Release or reuse the same Scan Credit reservation; never charge an extra credit for retry |
| Concurrent Document intake or credit purchase callback | Atomic, tenant-scoped settlement prevents overrun or duplicate grant |

## Testing Decisions

- Test externally visible behavior at the authenticated Organization API seam with a controlled provider adapter; assert the returned state and real Entitlement enforcement, not private helper calls. Exercise atomic Document/credit ledgers and webhook idempotency against PostgreSQL, and prove payment flows with Stripe test mode. Reuse existing API, tenant authorization, Plan gate, document-intake, durable-job, and isolated database test patterns. Map the requested STAFF/VIEWER denial cases to the actual non-Owner RBAC roles without inventing new roles. Test Pricing/Billing UI at mobile and desktop sizes with keyboard, screen reader order, visible focus, and Thai copy.
- Test local state transitions, provider events, and background reconciliation separately from the browser redirect. Verify receipt uniqueness, tenant/role restrictions, refund and dispute effects, and no deletion of customer Documents. Do not count a build, readiness endpoint, or synthetic OCR completion as end-to-end billing proof.

### Acceptance tests

| # | Scenario | Expected observable result |
|---:|---|---|
| 1 | Owner completes monthly Starter or Business Checkout | Correct THB full charge, verified active local Subscription, paid features enabled, one receipt |
| 2 | Owner completes six-month Checkout | Native six-month recurring Price, approved full charge, correct renewal date and local rights |
| 3 | Owner completes annual Checkout | Approved annual charge, correct renewal date and local rights |
| 4 | Owner switches monthly to six-month/yearly on the same Plan | New interval begins at next renewal; current period and rights remain until then |
| 5 | Owner switches six-month/yearly to a shorter interval | Change occurs at paid-period end without immediate charge |
| 6 | Forged Stripe webhook is submitted | Signature rejection; no local payment, receipt, Scan Credit, or Entitlement change |
| 7 | Same valid webhook arrives repeatedly or out of order | One effective Subscription/credit/receipt transition; provider IDs and audit remain consistent |
| 8 | Browser opens success redirect without verified payment | UI remains pending; paid feature gate denies new rights |
| 9 | Renewal charge fails | Local state enters seven-day grace; paid rights persist; Owner sees debt/deadline and receives first notice |
| 10 | Grace reaches seven days without payment | Free/Plan Over Limit rights apply; Documents remain; paid carry is suspended |
| 11 | Outstanding payment succeeds inside the 30-day recovery window | Same Subscription restores paid rights and only unexpired paid carry; no duplicate charge or Subscription |
| 12 | Owner cancels at period end and then undoes before that date | First action schedules end with confirmation; undo restores renewal; paid rights stay until period end |
| 13 | Owner upgrades paid Plan | Preview shows unused-time credit and amount due; higher rights and monthly allowance difference apply only after verified payment |
| 14 | Owner downgrades paid Plan | Current rights last to period end; destination limits and any carry clamp apply then and are previewed |
| 15 | STAFF/VIEWER tries billing mutation or sensitive read | Denied despite a valid login; Admin sees only allowed status/dates |
| 16 | Organization A requests or changes Organization B billing, Portal, or receipt | Denied without leaking provider IDs or buyer/payment data |
| 17 | Browser tampers with Plan, Price ID, Product ID, amount, or pack size | Trusted catalog rejects tampered values; no cheaper/other-tenant charge |
| 18 | Verified webhook is missed and reconciliation runs | Local state converges to verified Stripe payment/subscription state once; drift is auditable |
| 19 | Real Document, seat, LINE, export, OCR, and firm-workspace actions run under each Plan | Existing feature gates and agreed Plan limits allow/deny correctly without per-request Stripe calls |
| 20 | Free/Trial/paid Document limits and rollover cross month, timezone, downgrade, and two-year expiry | Free never carries, Trial does not transfer, paid oldest grant spends first, limits are atomic under concurrent intake |
| 21 | After Firm activation, Accounting Firm adds staff/clients and uses own or client Documents | 5 staff, 20 clients, 5 own LINE groups, 1,000 own Documents/month; each client's Plan and Scan Credits remain separate |
| 22 | One-time Scan Credit pack succeeds, fails, repeats, or is fully refunded | Correct fixed credits granted once; failures grant none; refund removes only unspent credits |
| 23 | PDF or image OCR succeeds, retries, fails, or is explicitly reprocessed | One credit per successfully scanned page, full reservation, failure release, no retry double-charge, confirmed reprocess charged anew |
| 24 | Stripe or receipt email is unavailable | Local verified rights remain coherent; UI states uncertainty; delivery retry uses one receipt; reconciliation repairs payment drift |
| 25 | Renewal charge succeeds, including after an earlier failure | Paid-through period extends once; grace notices stop; full next charge and date update |
| 26 | Trial Owner buys a paid Plan | Paid rights begin after verified payment; unused Trial Document limit does not transfer |
| 27 | Full current-period refund or finally lost dispute is verified | Paid rights fall to Free/Plan Over Limit without deleting Documents; partial refund does not automatically change rights |
| 28 | Recovery deadline passes unpaid and Owner later returns | Old Subscription ends; new Checkout is required; suspended paid carry is not restored |
| 29 | Seller receipt is issued, emailed, downloaded, and later Billing Profile changes | One immutable numbered receipt per payment, tenant Owner access only, email retry uses same receipt, old buyer snapshot stays intact |

### Observability and performance budgets

| Area | Agreed budget or invariant |
|---|---|
| Entitlement authorization | Indexed local Organization state; zero Stripe calls on feature requests; measure latency |
| Billing dashboard | Local state first; provider sync asynchronous where possible |
| Webhook acknowledgement | Signature check and durable enqueue before prompt acknowledgement; slow side effects asynchronous; measure ack and processing latency |
| Reconciliation | Pending Checkout/past-due every 15 minutes; active daily; measure drift and repair |
| Renewal and grace | Seven-day grace from first failed renewal; at most three automatic retries inside it; 30 further days of same-Subscription recovery |
| Usage | Atomic admission/reservation; no full aggregate scan on each request; measure contention and queue fairness |
| Commercial flow | Measure Checkout creation latency and failed-payment rate |

No numeric millisecond latency SLO was agreed during the grill. Record measured baselines and representative load results before setting pass thresholds or advertising processing time.

## Out of Scope

- Full accounting-software replacement, marketplace payouts, payroll, POS, complex enterprise contract billing, multiple currencies, and unrestricted metered billing.
- Stripe-hosted PromptPay subscriptions, coupons/promo codes, automatic refunds, automatic credit top-up, pooling firm and client Scan Credits, and buying credits on behalf of a client.
- New numerical AI/Export quotas or overage pricing; Thai VAT tax invoices while the seller is not authorized to issue them; LINE billing messages in this version.

## Further Notes

### External readiness facts to verify

- Stripe account eligibility and live-mode recurring/card configuration, including native six-month Prices, retry policy, `unpaid` recovery, and no duplicate invoice on recovery.
- Seller legal name, tax ID, VAT status, ordinary receipt format and delivery, and exact refund document handling with a Thai accountant before collecting live payments.
- Phase 10 activation evidence before selling Accounting Firm and OCR acceptance evidence before selling Scan Credits.
