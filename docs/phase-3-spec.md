# Phase 3: Business Master Data, Rich Menu, Plans, Connections and CSV/XLSX

## Problem Statement

PlaiFlow already has Organization-scoped identity, Membership and fixed roles, LINE identity/group linking, Task workflow, Notification, Work Dashboard, Assistant Summary, Task CSV Export and a server-side Entitlement seam. It does not yet have a trusted commercial Plan Catalog, enforced resource limits, business counterparties, reusable reference data, safe bulk import, XLSX export, customer-owned Google Drive export or a production-managed LINE Rich Menu.

Phase 3 must add those capabilities without allowing price or Plan values from the browser to grant access, exposing tenant data through bulk operations, reading a customer's whole Google Drive, evaluating spreadsheet content or deleting/locking away existing data after Trial expiry or downgrade.

## Solution

Add a versioned, server-owned Plan Catalog for Free, Starter and Business; Organization Plan state; a one-time Business Trial; atomic Usage Reservation; and explicit Plan Over Limit behavior. Authorization remains separate from commercial Entitlement checks.

Add Organization-owned Business Contacts that can be Customers, Vendors or both, plus flat Expense Categories and Payment Channels. Provide safe CSV/XLSX Business Contact import with mapping and Validation Preview, fixed CSV/XLSX export templates, auditable job history and bounded background processing.

Add a Connections page containing existing LINE Group Connections and one explicit Organization Google Drive Connection to the customer's own My Drive. Google Drive consent is separate from Google Login and grants only access to files PlaiFlow creates. Add one static six-area LINE Rich Menu that navigates to real responsive Web surfaces. Extend the existing Work Dashboard and Settings rather than creating parallel applications.

## User Stories

1. As an Owner, I can compare Free, Starter and Business using honest monthly and prepaid totals.
2. As an Owner, I can start one 14-day Trial without a card and understand its limits and end date.
3. As a User, I retain safe read and recovery access to existing data after Trial expiry or downgrade.
4. As an Owner/Admin, I can see current Plan, Trial or Plan Over Limit status and relevant Usage against server-resolved limits.
5. As a security owner, I know no browser-supplied Plan, price, capability or Usage value can grant access.
6. As an Owner/Admin, I can create one Business Contact and mark it Customer, Vendor or both.
7. As a Member, I can find and select an active Business Contact without gaining bulk extraction authority.
8. As an Owner/Admin, I can Archive and Restore Business Contacts and reference data without breaking history.
9. As an Owner/Admin, I can maintain flat Expense Categories and Payment Channels for my Organization.
10. As an Owner/Admin, I can preview a CSV/XLSX Business Contact import before any records are created.
11. As an Owner/Admin, I can see mapping, validation errors, warnings and duplicate skips before committing an import.
12. As an Owner/Admin, I can Undo a recent import without hard-deleting records.
13. As an Owner/Admin, I can export authorized data using fixed, documented CSV/XLSX templates.
14. As a Free Owner/Admin, I can export authorized Organization data in CSV so my data is not commercially locked in.
15. As a Business Owner/Admin, I can explicitly connect one customer-owned Google Drive without granting whole-Drive access.
16. As a Drive owner, I retain ownership of folders and files after PlaiFlow disconnects.
17. As an Owner/Admin, I can reconnect or recover when Google consent, credentials or the destination folder stop working.
18. As a LINE user, I can navigate to real Phase 3 features from one stable Rich Menu.
19. As a multi-Organization User, I must choose Organization Context rather than inherit authority from a Rich Menu link.
20. As a User, I can use equivalent Web navigation when Rich Menu is unavailable.
21. As an Owner/Admin, I can see team, LINE group, job, Drive and Plan summaries on the existing Work Dashboard.
22. As an operator, I can trace Plan grants, imports, exports, Drive lifecycle and Upgrade Requests without seeing file contents or credentials.
23. As an operator, I can process large files without unbounded Drive scans, database reads, memory growth or provider retry loops.

## Implementation Decisions

### 1. Plan Catalog and Pricing

The Plan Catalog is versioned server code and is the only source that maps a Plan to display metadata, Entitlements and limits. There is no customer-facing Plan editor. One Plan applies to one Organization.

| Plan | Monthly | Six-month total / effective monthly | Yearly total / effective monthly |
|---|---:|---:|---:|
| Free | ฿0 | ฿0 / ฿0 | ฿0 / ฿0 |
| Starter | ฿199 | ฿1,098.48 / ฿183.08 | ฿1,990 / ฿165.83 |
| Business | ฿499 | ฿2,754.48 / ฿459.08 | ฿4,990 / ฿415.83 |

- Pricing defaults to Monthly and offers Six-month (`ประหยัด 8%`) and Yearly (`ประหยัด 17%`) intervals.
- A prepaid interval always shows the effective monthly amount and exact amount charged together.
- Yearly pricing represents paying for ten months and receiving twelve months.
- The six-month interval is a confirmed Phase 3 presentation option; like Yearly, it does not activate billing or change Entitlements by itself.
- Prices are Early Access hypotheses and may change for future periods. Do not promise lifetime pricing.
- The confirmed paid price remains valid through its granted or paid period; later renewal changes require advance notice.
- The billing entity's VAT registration status is not yet confirmed. Production UI must not claim `รวม VAT` until it is verified.
- Public Pricing uses three cards only. Business is marked Recommended, followed by a detailed comparison.
- Free uses `เริ่มใช้ฟรี`. Paid cards use `ทดลองใช้ฟรี 14 วัน` and identify the Trial-specific limits.
- Phase 3 has no checkout, recurring billing, payment method, automatic renewal, invoice or refund flow.
- `ขออัปเกรด` creates an Upgrade Request only and never changes effective Entitlements.
- Only a trusted operator may create a Manual Plan Grant. Each grant records Plan, reason, approving operator, start and mandatory end time. Hidden or lifetime grants are forbidden.

### 2. Entitlement Definitions and Usage Limits

| Capability or limit per Organization | Free | Starter | Business |
|---|---:|---:|---:|
| Active Memberships plus reserved Invitations, including Owner | 1 | 3 | 10 |
| Active/reserved LINE Group Connections | 1 | 2 | 5 |
| Documents per calendar month | 30 | 300 | 1,000 |
| Existing Task, Notification and Assistant Summary behavior | Yes | Yes | Yes |
| Business Contact and reference-data UI | Yes | Yes | Yes |
| CSV export | Yes | Yes | Yes |
| CSV/XLSX Business Contact import | No | Yes | Yes |
| XLSX export | No | Yes | Yes |
| Customer-owned Google Drive export | No | No | Yes |
| Background export above synchronous threshold | No | No | Yes |

- Core Task, Notification and Assistant Summary behavior remains available on every initial Plan.
- Document limits exist in the catalog, but Phase 3 has no Document Intake or OCR flow; no Document Usage is recorded or enforced yet.
- OCR calls, AI tokens and provider retries are not customer-visible Usage. A later document phase may charge at most one Document unit for each successfully accepted document while recording provider cost internally.
- CSV exports have no monthly commercial count. Technical row, byte and memory limits still apply.
- Pending Invitations reserve Member slots. Pending LINE Link Codes reserve LINE group slots. Expiry/revocation releases the reservation; acceptance/redemption consumes the same reservation atomically.
- Non-metered capabilities use a server Check. Quantity-limited operations atomically Reserve before work, Complete after accepted success and Release after terminal failure using one idempotency key.
- Usage belongs to one Organization and resets on the first day at 00:00 in the Organization Timezone.
- Usage responses contain used, limit, remaining and server-calculated reset time.
- Warn Owner/Admin at 80%, send Web and opted-in LINE warnings at 90%, and deny the next metered operation at 100%. Phase 3 never auto-charges overage.

### 3. Free, Trial, Paid and Over-Limit Status

- Free is a permanent Plan. Trial is a temporary Organization state, not a separate Plan.
- An Owner may activate one Trial per Organization without a card. It lasts 14 days and exposes Business capabilities with limits of 100 Documents, 3 Memberships and 2 LINE groups.
- Notify Owner/Admin 7, 3 and 1 day before Trial expiry.
- Trial expiry returns the Organization to Free.
- If existing structural resources exceed the new limits, enter Plan Over Limit. Never choose members/groups, delete records or disconnect providers automatically.
- In Plan Over Limit, existing Users may sign in, read authorized data and use security/recovery controls. Owner/Admin may still perform authorized CSV export.
- Ordinary mutations, import, Document Intake and provider delivery are denied until resources are reduced or a trusted Plan becomes effective.
- Ownership transfer, Membership removal, LINE/Drive disconnect, identity unlink, session revocation, Notification opt-out, tenant isolation, audit and required data access paths are never commercially gated.
- Pricing and Plan & Usage surfaces show one clear state: Free, Trial with remaining days, Paid/Granted with end date when applicable, or Plan Over Limit with the exact resources requiring action.

### 4. Business Master Data

#### Business Contacts

- A Business Contact belongs to exactly one Organization and represents one person or one tax branch. It may have Customer, Vendor or both roles.
- Required fields are Display Name and at least one role.
- Optional fields are Contact Code, Organization/Individual type, Thai/English Legal Name, country code defaulting to `TH`, Tax ID, Head Office/Branch selection, Branch Code, phone, email, one free-text address, postal code and notes.
- Phase 3 does not add an address registry or normalize Thai administrative areas.
- Tax ID and Branch Code are strings. Thai Tax ID input removes allowed visual separators, then requires 13 digits and a valid checksum. No external Revenue Department lookup is performed.
- Thai Head Office uses branch code `00000`; another Thai branch uses exactly five digits. Tax ID presence never implies VAT registration.
- For non-TH contacts, Tax ID is optional text and Thai 13/5-digit validation does not apply.
- A Tax ID uses a non-null canonical branch identity. Strong duplicate keys are Organization/country/normalized Tax ID/canonical branch and Organization/normalized Contact Code. Uniqueness includes Archived records.
- A strong duplicate is blocked and points Owner/Admin to the existing or Archived record. Normalized name, email or phone matches produce warnings only.
- Adding Customer or Vendor behavior to the same party changes its role set rather than creating a second record.
- Archive removes a record from default lists and new selections while preserving history and eligible export. Restore reactivates it. There is no hard-delete UI.

#### Expense Categories and Payment Channels

- Both are flat Organization-owned lists with name, optional description, display order and Archived state.
- Normalized names are unique per Organization, including Archived rows; duplicates direct Owner/Admin to Restore or rename.
- Seed a small editable Thai Expense Category list.
- Seed Payment Channels for Cash, Bank Transfer, PromptPay/QR, Credit/Debit Card, E-wallet, Cheque and Other.
- Owner/Admin may add, rename, reorder, Archive and Restore. Member may list/select active entries.
- Do not add hierarchy, chart of accounts, ledger codes, accounting entries, bank credentials, full account numbers, QR payloads or bank connections.

#### Team Settings and Roles

- Reuse Owner, Admin and Member. Do not add departments, job titles, accountant roles or custom permissions.
- Owner/Admin may mutate, import and export master data.
- Member may list, search and select active Business Contacts/reference data but cannot mutate or bulk import/export.
- Full Tax ID is visible only to Owner/Admin. Member responses mask it when the field must be shown.
- Team Settings shows active/reserved Membership Usage, pending Invitations and existing Membership management.
- Invitation acceptance atomically rechecks tenant, role, invitation state and Member-slot reservation.

### 5. Settings, Connections and Google Drive OAuth

#### Settings and Connections Page

- Organization Settings contains Organization, Team, Connections, Business Data and Plan & Usage. Personal Account and identity connections remain separate.
- Connections contains cards for LINE Group Connections and Google Drive. It shows Organization-scoped status, last safe successful activity, permitted actions and recovery guidance.
- All roles may see non-sensitive Plan and capability status needed to explain an unavailable action. Only Owner/Admin sees commercial controls, Drive account details, import/export history and Upgrade Request.
- Page load reads persisted database state only; it never calls or scans a provider.
- Role denial hides administrative actions. Entitlement denial shows the unavailable capability and upgrade path. Provider failure shows recovery, not an upgrade prompt. Usage denial shows used, limit and reset time.

#### Google Drive Consent and Folder Strategy

- Google Drive consent is a separate user-initiated flow from Google Login. Drive access is never requested during login.
- Use a separate Google Cloud project and OAuth client for Login and Drive in each environment so revoking Drive does not share the Login grant boundary.
- Request only `openid`, `email` and `drive.file`, with offline access for background export. Do not request whole-Drive, broad metadata, Sheets or unrelated scopes.
- One active Drive Connection is allowed per Organization. Owner/Admin must pass recent reauthentication and confirm the Organization before connecting or reconnecting.
- The connection belongs to the Organization and records the authorizing Membership and Google account identity. It writes only to the customer's account; PlaiFlow provides no Drive account or storage quota.
- On connect, create `PlaiFlow - {Organization Name}` at the root of the customer's My Drive and persist its folder ID as canonical identity. Rename or move does not break the connection.
- Customer folder selection, Google Picker and Shared Drive are out of scope.
- Google OAuth opens in the external system browser, never an embedded LINE WebView.

#### Connection Status and Lifecycle

- Supported persisted statuses are Not Connected, Connected, Reauthorization Required, Folder Action Required and Paused.
- Reconnect replaces invalid/old credentials only after fresh explicit consent and preserves the canonical folder when it is still accessible.
- Credential invalidation or external revocation moves the connection to Reauthorization Required and stops credential retries.
- A missing or inaccessible folder moves the connection to Folder Action Required. Never create a replacement silently; Owner/Admin must confirm the action.
- If the authorizing Membership ends or drops below Admin, pause the connection and require a current Owner/Admin to reconnect.
- Disconnect immediately stops new Drive writes, deletes stored token material, attempts provider revocation, audits the safe outcome and leaves customer-owned folders/files untouched.
- Rechecking provider state occurs only by explicit action on Connections or during an export.

#### Token Storage and Upload Reliability

- Refresh tokens are server-only and encrypted at rest with AES-256-GCM using an environment-specific secret not stored in PostgreSQL.
- Access/refresh tokens never enter browser storage, logs, audit or analytics. Short-lived access tokens are refreshed server-side.
- Allocate and persist one Drive file ID per Export Job before upload. Retries target the same identity and do not create duplicates.
- Use resumable upload for large files. Retry only transient network failures, rate limits and server errors with bounded exponential backoff, jitter and provider retry timing.
- Record safe metrics for operation, latency, status class, retry count, rate-limit occurrence and bytes. Do not record account email, filenames, folder names, tokens, response bodies or file contents.
- Drive files are permanent only in customer storage and are never removed by PlaiFlow cleanup.

### 6. CSV/XLSX Import

#### Accepted Files and Mapping

- Import is included for Business Contacts only and is available on Starter and Business.
- Accept CSV as UTF-8 with optional BOM and XLSX with one data sheet.
- Reject legacy XLS, macro-enabled XLSM, encrypted/password-protected workbooks, multiple data tables and archive uploads.
- One data row represents one Business Contact. Required targets are Display Name and Roles.
- Limits are 10 MiB, 10,000 data rows, 50 columns and 10,000 characters per cell. Enforce compressed and expanded-size limits during XLSX parsing.
- Auto-map allowlisted Thai/English header aliases and let Owner/Admin correct mappings.
- One source column maps to at most one target. Unknown columns are visibly ignored and never guessed.
- Reusable saved mappings are not included.

#### Validation Preview and Errors

- Preview shows totals and representative rows for Ready to Create, Duplicate to Skip, Warning Requiring Confirmation and Error Requiring Correction.
- Preview performs no Business Contact writes. Raw files and preview data expire within 24 hours.
- Any Error prevents all writes. Warnings require explicit confirmation.
- Strong duplicates against active or Archived data are skipped, never overwritten. For strong duplicates within the file, the first valid row wins and later rows are skipped.
- Name-only similarity is a warning that may be accepted.
- Import never automatically updates, merges or adds roles to an existing Business Contact.
- A formula cell is an Error. Hyperlinks contribute text only. Never execute formulas, macros, scripts, external links or embedded objects.
- Validate extension, MIME and signature together and enforce upload, expanded archive, row, column, cell, parse-time and memory ceilings.
- Parser failures return stable safe categories and request IDs without stack traces, raw library errors or contents.

#### Commit and Undo

- Up to 2,000 rows may validate synchronously. From 2,001 through 10,000 rows, create a background Import Job.
- Stream CSV, read XLSX incrementally and stage at most 500 rows per database insert.
- Permit one active Import Job per Organization.
- Final creation is one all-or-nothing transaction over validated non-duplicate rows. A failure creates no Business Contact.
- Successful rows retain their Import Job reference. Owner/Admin may Undo within 24 hours; Undo Archives only records created by that job and never hard-deletes or reverts a pre-existing record.

### 7. CSV/XLSX Export and Templates

#### Templates and Authorization

- Extend the existing Task CSV Export seam rather than create a parallel subsystem.
- Fixed, versioned templates are Business Contacts CSV/XLSX, Expense Categories CSV/XLSX, Payment Channels CSV/XLSX and existing Tasks CSV.
- Use stable bilingual headers such as `ชื่อที่แสดง (display_name)` where readability and round-trip mapping are both needed.
- No custom columns, user-created/saved templates, scheduled export, raw audit/event export or cross-Organization export is included.
- Owner/Admin may request bulk export and see history. Member cannot bulk export even if allowed to read individual records.

| Format | Synchronous | Background | Hard row limit | File limit |
|---|---:|---:|---:|---:|
| CSV | 0–5,000 | 5,001–100,000 | 100,000 | 256 MiB |
| XLSX | 0–5,000 | 5,001–50,000 | 50,000 | 128 MiB |

- Background export is a Business capability. Free/Starter requests over the synchronous threshold instruct the user to narrow filters or upgrade.
- Stream CSV through the existing export boundary. Generate XLSX with a streaming writer under a 64 MiB export-specific memory budget.
- Protect every untrusted CSV string from formula injection. Write user-controlled XLSX values explicitly as text, never formulas.
- PlaiFlow download files use private temporary object storage and expire within 24 hours. PlaiFlow provides no permanent customer file storage.

#### Audit and Retention

- Import/export history contains requester, Organization, template/version, sanitized filename, file hash where applicable, normalized mapping/filter summary, row counts, bytes, status, timestamps and safe failure category.
- Audit requested, rejected, validated, committed, failed, downloaded, Drive-uploaded, expired and Undone transitions as applicable.
- Never audit cell values, Tax IDs, addresses, file contents, raw parser/provider errors, tokens, object keys or signed URLs.
- Raw imports, Preview data and temporary exports expire within 24 hours. Import/export history and audit metadata remain for 365 days.
- Cleanup is idempotent, retryable and monitored. A missing temporary object can still transition safely to expired.

### 8. LINE Rich Menu and Web Settings

- Use one static default Rich Menu for all users; do not create Plan- or role-specific variants.
- Use a large 2×3 layout: งานของฉัน, Dashboard, ลูกค้า/ผู้ขาย, นำเข้าข้อมูล, ส่งออกข้อมูล and ตั้งค่า.
- Do not include ส่งเอกสาร until Document Intake exists. Do not expose dead or coming-soon actions.
- Destinations use HTTPS URI actions and open the responsive product in LINE's in-app browser. LIFF is not included.
- Rich Menu is navigation only. It grants no session, tenant, role or Entitlement.
- A User with one Membership enters that Organization. A User with multiple Memberships enters Organization Chooser. The server rebuilds Organization Context for every destination.
- URL, Rich Menu data, local storage and last-selected Organization state are never authorization.
- Create, validate, upload and version Rich Menu through the Messaging API separately per environment. Replace by creating a new version and switching the default; do not edit API-managed menus in Official Account Manager.
- Web navigation exposes every destination because Rich Menu is unavailable on some LINE clients and desktop workflows.

### 9. Dashboard

- Extend the existing Work Dashboard. Do not create a separate Business Dashboard or alter Operations Dashboard.
- Member sees My Tasks, due/overdue work, unread Notifications, relevant Usage and explicit empty states.
- Do not show a document shortcut until a real Document Intake capability exists.
- Owner/Admin additionally sees Team Queue, future Document Usage/reset date, Member and LINE group Usage, active Customer/Vendor counts, latest Import/Export Jobs, Drive status and Trial/Plan status.
- Widget lists contain at most 5–10 items and link to paginated detail pages.
- Do not add per-person performance aggregates, ranking, leaderboards, custom widgets, real-time sockets or provider calls during render.

### 10. Security

- Every request evaluates in this order: authenticated session, Organization Context, current Membership, RBAC/resource authorization, server Entitlement, Usage Reservation, mutation.
- Use fixed server capability identifiers. Reject arbitrary client-provided capability names.
- A frontend badge, hidden button, submitted Plan/price ID, Upgrade Request, Usage value, cached response or URL never grants access.
- Every browser/file identifier is a requested identifier only; the server resolves it inside the authenticated Organization Context.
- All Phase 3 records, jobs, reservations, connections and downloads are tenant-scoped with database constraints and RLS matching the established Organization boundary.
- Unknown or stale Plan keys fail closed for optional paid mutations without weakening RBAC or recovery access.
- Bulk import/export is role-safe at request, commit/generation and download time. Role or Entitlement changes invalidate later steps where required.
- Spreadsheet files are hostile input. Use allowlisted field mappings and parameterized writes; never use filenames as storage paths.
- Apply CSV formula-injection protection and XLSX text-cell handling to every untrusted value.
- Audit security-relevant Plan, import/export and Drive transitions without credentials, business content or sensitive provider payloads.

### 11. Performance

- Business Contact list/search uses stable cursor pagination: default 50, maximum 100, ordered by normalized name and stable ID.
- Support normalized name prefix search, exact Owner/Admin Tax ID search and exact/prefix Contact Code search.
- Do not add fuzzy matching, typo correction, arbitrary substring search or a search engine.
- Add only indexes supporting accepted query shapes: tenant Tax ID/branch uniqueness, tenant Contact Code uniqueness, tenant normalized-name cursor search and active/Archived ordering.
- Import uses bounded streaming/incremental parsing and batches of at most 500 rows. One Organization has at most one active Import Job.
- Keep PostgreSQL-backed bounded worker claims. Import and large export must not starve Notification, Reminder or inbound-event work. Do not add a message broker.
- Dashboard uses one HTTP request, no N+1, no provider calls and no more than 5–8 SQL statements. Target p95 at or below 500 ms against the representative Phase 2 plus Phase 3 dataset.
- Track normalized route/job type, latency, rows/bytes, batch count, retry count, rate-limit occurrence and safe outcomes without business content.
- Never scan/list an entire Drive on page load. List or check only when explicitly needed, using provider cursors where a provider result can exceed one page.

## Testing Decisions

The highest-value test seam is the authenticated Organization-scoped HTTP boundary through real PostgreSQL, optionally followed by one bounded worker cycle and the authorized read/download or Drive-provider double. This reuses the established Phase 1/2 seam and proves tenant, role, Entitlement, transaction and retry behavior together.

Narrow doubles are allowed only at external or file-format boundaries: trusted Plan Catalog/provider, Google OAuth/Drive, spreadsheet parser/encoder and private object storage, and LINE Rich Menu provider. Automated tests do not publish to live providers.

- Prove submitted Plan, price, capability and Usage values cannot change effective Entitlements.
- Prove every gated action checks session, Membership and RBAC before Entitlement. A foreign tenant receives not-found-equivalent behavior without Plan/Usage disclosure.
- Test concurrent Member Invitations, LINE Link Codes and future Document Reservations at the exact limit; active plus reserved units never exceed it.
- Test Trial one-time activation, 7/3/1-day warnings, expiry, Free transition, Plan Over Limit, recovery and ungated security/read/CSV paths.
- Test Business Contact Thai Tax ID checksum, Branch Code, country behavior, strong/weak duplicates, Customer+Vendor roles, Archive/Restore, masking and tenant isolation.
- Test cursor stability and representative query plans for accepted indexes.
- Test wrong extension/signature/MIME, oversized/expanded workbooks, formula cells, macros, encrypted/legacy formats, hyperlinks, long cells, row/column ceilings and malformed archives.
- Test import mapping, write-free Preview, duplicate skips, warning confirmation, any-error rejection, background validation, all-or-nothing commit, idempotent retry and 24-hour Undo/expiry.
- Test export templates with Thai text, commas, quotes, line breaks, Unicode and formula-leading characters; assert CSV protection and XLSX text cells.
- Test sync/background thresholds, hard limits, memory/file ceilings, authorization changes between request/generation/download and temporary cleanup.
- Test Google authorization uses the separate client, exact scopes, state, PKCE where applicable, offline access, external-browser handoff and no token exposure.
- Test Drive tenant/role checks, one-active constraint, authorizer removal/downgrade, refresh, external revocation, missing folder, reconnect, disconnect, audit and no deletion of customer files.
- Test upload interruption, bounded transient retry, provider retry timing, one persisted Drive file ID and no duplicate file creation.
- Test Rich Menu object validation and exact navigation actions using a provider double.
- Test Dashboard variants, bounded queries/items, no provider calls and no per-person performance output.

## Out of Scope

- Accounting Firm Plan, client portfolio, delegated accountant access and per-client Drive connections
- Checkout, payment provider, subscription lifecycle, automatic renewal, refunds, invoices and unverified production VAT claims
- Six-month billing, overage charging, member/LINE add-ons and lifetime price guarantees
- Document Intake, permanent PlaiFlow document storage, OCR/AI processing and customer-visible AI token/call quotas
- Google Drive import, Drive scan/sync, Google Picker, Shared Drive, Google Sheets and whole-Drive scopes
- Company-to-many-Branch hierarchy, external Tax ID lookup, accounting ledger, journal entries, tax calculation and bank integration
- Master-data upsert/merge, partial import, saved mappings, custom columns/templates and scheduled export
- XLS, XLSM, formula evaluation, macros, external-link processing and embedded-object processing
- Departments, custom roles, LIFF, per-user/per-Plan Rich Menus and dead Send Document actions
- New message broker, search engine, fuzzy search, real-time Dashboard and speculative infrastructure
- KMS until production scale or compliance requires it; environment-scoped AES-256-GCM is the Phase 3 token boundary
- Production cloud/OAuth provisioning, provider verification, secret entry, live Rich Menu publication and deployment without a separately approved rollout

## Further Notes

- Status: `ready-for-agent` after user approval of this specification. This document authorizes no implementation, deployment or external activation.
- The only unresolved external fact affecting public Pricing copy is the real billing entity's VAT registration status. It does not block local implementation of other Phase 3 work after approval.
- Existing Phase 1/2 Organization, Membership, Task, Notification, Work Dashboard, Entitlement and Task CSV Export seams are extended rather than replaced.
- No active issue tracker is configured in this workspace, so this specification remains local and no issue or ticket is created by this step.
