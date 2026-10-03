# Phase 11 billing threat model

Status: Paused, 2026-09-25. Stripe was canceled and no provider is selected. Provider-specific threats below require revision before implementation resumes; controls are requirements, not verified deployed controls.

## Actors, assets, and boundaries

| Actor | Likely path | Target |
|---|---|---|
| External attacker or another tenant's user | Billing API, Checkout return, receipt URL, webhook endpoint | Paid rights, seller receipts, customer metadata |
| Compromised Organization account | Owner action or stolen session | Unauthorized purchase, cancellation, payment history |
| Operator or leaked service credential | Provider dashboard, database, logs, background worker | Stripe secret, webhook secret, subscription state, audit trail |

Sensitive assets are Stripe secrets and provider IDs, Billing Profile/receipt data, payment history, paid-through state, Document Allowance, and Scan Credit balances. Card number and CVV remain entirely at Stripe.

```text
Owner browser ── authenticated HTTPS ──> PlaiFlow API ── server-secret HTTPS ──> Stripe Checkout/Portal
     │                                  │                                  │
     │ success return (untrusted)       │                                  │ signed webhook (untrusted until verified)
     └──────────────────────────────────>│<─────────────────────────────────┘
                                        │ durable event / queue
                                        v
                                  PostgreSQL billing state ──> worker / email
                                        │
                                        └── local entitlement and receipt reads
```

Trust boundaries: browser→API needs current session, Organization, and role; API→Stripe needs server secret and trusted Price mapping; Stripe→webhook needs raw-body signature verification; worker→database needs narrow service authority; receipt download needs a new tenant/Owner authorization check. Existing firm-client delegation never grants access to a Client Organization's Billing Profile or receipts.

## Component check

| Component | STRIDE focus | Required control |
|---|---|---|
| Checkout and Plan change API | S, T, E | Current Owner role; server catalog and amount; scoped idempotency |
| Stripe webhook and queue | S, T, D | Signature, event uniqueness, bounded body, fast durable acknowledgement |
| Local Subscription and entitlement reads | T, D, E | Verified state transitions, indexes, no provider call on each gate |
| Document/Scan ledgers | T, D | Atomic grant and reservation, oldest expiry first, concurrency tests |
| Receipts, Portal, and logs | R, I, E | Tenant Owner check, immutable receipt snapshot, audit, redaction |

## Threat register

| ID | STRIDE | Design threat | Surface | ATT&CK example | Risk | Mitigation and proof required | Owner | Status |
|---|---|---|---|---|---|---|---|---|
| B-01 | S | Forged webhook grants a paid Plan or Scan Credits | Public webhook | T1190 | High | Verify raw-body Stripe signature before durable insert; reject invalid/missing signatures; test forged payload | API | Open |
| B-02 | T | Browser changes Price ID, amount, Organization, or pack quantity | Checkout API | T1565 | High | Accept only Plan/interval/pack keys; resolve fixed trusted catalog and current Owner Organization server-side; test tampering | API | Open |
| B-03 | T | Duplicate or out-of-order provider events double-grant time, receipts, or credits | Event worker | T1565 | High | Unique provider event ID plus unique paid invoice/payment keys; re-fetch current provider state; transactionally apply transitions; replay tests | Billing worker | Open |
| B-04 | R | Cancellation, refund, manual repair, or receipt correction cannot be reconstructed | Audit store | T1070 | High | Append tenant-scoped BillingAuditEvent with actor/source, old/new state, provider object IDs and reason; protect against user edits; review audit coverage | API/operator | Open |
| B-05 | I | Cross-tenant receipt/Portal access or secrets in logs | Billing reads and logs | T1530, T1552 | High | Recheck current Owner and tenant for every download/session; store only needed provider metadata; redact secrets, URLs, payload and buyer data from logs; authorization tests | API/web | Open |
| B-06 | D | Webhook flood or slow side effects prevent Stripe delivery and reconciliation | Public webhook/queue | T1499 | High | Body/time limits; acknowledge after durable enqueue; asynchronous bounded workers; alert backlog and drift; load test | API/ops | Open |
| B-07 | E | Admin or firm delegate triggers Owner-only billing mutation | Role boundary | T1078 | High | Resolve current Membership on every call; Owner-only write and sensitive read; firm grants do not widen billing authority; role-revocation tests | API | Open |
| B-08 | T | Concurrent uploads or OCR retries overspend Document Allowance or Scan Credits | Usage ledgers | T1565 | High | Tenant-scoped atomic reservations and unique settlement keys; test parallel intake, retry, failure release, refund and two-year expiry | API/worker | Open |

Release blocks: prove the High-risk controls above in tests and staging, rotate any exposed secret, and reconcile provider/local drift before enabling paid Checkout. Revisit this model if payment methods, VAT status, firm payer behavior, or provider topology changes.
