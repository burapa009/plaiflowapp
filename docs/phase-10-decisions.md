# Phase 10 — Accounting Firm Workspace + Multi-Client Export decisions

Status: Agreed after final shared-understanding confirmation on 2026-09-25. These design decisions do not themselves activate a feature or prove release gates.

## Decision summary

- Reuse Organization for the accounting firm; the Client Organization keeps ownership. A client-approved, time-limited Firm Access Grant plus current firm Membership, Client Assignment, action scope, and record authorization are required for delegated access. Client Owner/Admin manages grants; Firm Owner/Admin manages staff assignments. Multiple firms may serve one client without seeing each other.
- The Accounting Firm Plan has initial ceilings of 20 reserved/active client relationships and 5 reserved/active staff seats including Owner. Client Plans still govern their own document intake, OCR, and Drive. Firm over-limit status blocks additions; Firm Plan expiry stops new workspace work but preserves security/recovery and eligible ready-file downloads.
- Portfolio, workload, Expected Document checklists, a combined Review Queue, and an on-demand daily briefing show only currently authorized clients. Owner/Admin without assignment sees relationship administration data but no client work counters. Counters may lag five minutes; authorization and revocation are immediate.
- Multi-client export is an explicitly selected, versioned, async XLSX of current Approved rows, with Summary and one sheet per client. Only an assigned firm Owner/Admin with each client's `export.approved` scope may request it; only that same currently authorized requester may download it. Limits are 20 clients, 50,000 rows, 128 MiB, 24-hour artifact retention, and a 15-minute download link. Any access or revision change invalidates the whole workbook.
- Release remains behind default-off gates and Phase 9 dependencies. Acceptance requires tenant/RBAC and revocation proof, content-free audit, set-based paginated reads without client-by-client N+1, bounded streaming export, and representative P95/CPU/RAM measurements.

## Agreed in round 1

- An accounting firm is an existing Organization with its own Memberships. Each client remains an independent Organization and owner of its data. The firm-client relationship requires client authorization; a firm Plan alone grants no client access. [ADR 0034](./adr/0034-model-accounting-firms-as-organizations.md)
- Multi-client export is a versioned generic workbook for oversight and onward handling of approved data. It is not presented as import-ready for a named accounting product without that product's official template and a verified trial import.
- The initial performance workload is 20 client Organizations, 5 firm staff, and 50,000 export rows per request. These are validation targets, not commercial entitlements or confirmed hard limits.

## Agreed in round 2

- Accounting Firm Plan is a distinct Plan on the firm's Organization. Each Client Organization retains its own Plan. The Firm Plan may enable firm workspace capabilities but does not authorize access to client data.
- A Client Organization issues a Firm Access Grant to the firm's Organization, and the firm assigns its staff to specific granted clients. Neither the grant nor assignment alone is sufficient for a staff member to access client records; every operation must pass current client authorization and record-level checks. This replaces a requirement that every firm staff member hold a Client Organization Membership. [ADR 0035](./adr/0035-require-client-grant-and-firm-assignment.md)
- A Google Drive Connection is optional for each Client Organization. Only its Owner/Admin, after recent authentication, may initiate or reauthorize the connection; assigned firm staff may request that the client connect it but cannot initiate OAuth. This aligns the desired Phase 10 rule with the Phase 3 specification; current code is Owner-only and needs an explicit change if implemented.
- The Phase 10 multi-client workbook contains only current Approved Export Rows. It does not include Raw Document metadata or merely Confirmed values.
- Multi-client Export Artifacts expire after 24 hours and signed download links after 15 minutes, as in Phase 9. Each access rechecks current authorization; revoking a Firm Access Grant makes the affected export unavailable immediately.

## Agreed in round 3

- A Client Organization Owner/Admin may issue or revoke a Firm Access Grant only after recent reauthentication. Record the actor, scope, and time. Either the client or the firm may end the relationship; client access stops immediately on revocation.
- Firm Access Grants carry separate action scopes and start read-only. The client must explicitly enable review and export scopes; a firm's Client Assignment cannot widen them.
- The initial Accounting Firm Plan ceiling is 20 active Client Organization grants and 5 active firm Memberships, including the Owner. These commercial limits are separate from the 20/5/50,000 performance workload.
- Document quota remains with each Client Organization under its own Plan. Phase 10 adds no commercial OCR quota or shared firm pool; a Firm Plan cannot increase a client's document allowance.
- The first versioned multi-client workbook has one Summary sheet and one sheet per selected Client Organization. Client sheets contain current Approved Export Rows only, with an unambiguous client identifier.
- The daily briefing appears only inside the firm workspace when an authorized user opens it. Phase 10 sends no automatic LINE or email briefing; it uses only currently granted and assigned clients.

## Agreed in round 4

- At the client or staff ceiling, or after a downgrade that leaves existing resources over the ceiling while the Firm Plan remains active, block only new client grants and staff invitations. Existing authorized client work, reads, review, and exports continue; revoking grants, removing staff, transferring ownership, and other recovery/security actions remain available. Firm over-limit status does not revoke client authorization. Firm Plan expiry has a separate rule in round 8.
- Pending client grants and staff invitations reserve their respective slots until accepted, rejected, revoked, or expired. Expiry and rejection release the slot; concurrent requests must not exceed the ceiling.
- Firm Owner/Admin may see the list of currently granted Client Organizations to manage assignments. Every firm user, including Owner/Admin, requires a current Client Assignment before opening client records or performing client work. A firm Member sees only assigned clients.
- An assigned firm user may confirm extraction or approve an Accounting Suggestion only when the Client Organization explicitly granted the corresponding `confirm` or `approve` scope. These are separate permissions; closing a Task or bulk action never confirms or approves a record.
- If a client grant or other required access is revoked while a multi-client export is running or before download, fail the whole export, make any artifact unavailable, and require a new request. Never silently omit that client or deliver a partial workbook. A file already downloaded cannot be recalled.
- “Missing documents” means Expected Document Items explicitly set for a Client Organization and accounting period that have no matching Document. Do not infer absence from OCR output or last month's upload count.
- The Summary sheet contains each selected client's stable identifier/name, requested date range, sheet name, Approved row count, and workbook creation time. It excludes monetary totals across clients. Per-client sheets reuse the versioned Approved export columns.

## Agreed in round 5

- The Accounting Firm Plan governs firm portfolio, delegated review, and multi-client export capabilities. Each Client Organization Plan continues to govern that client's document intake, OCR availability, and Google Drive connection. A Free client may receive firm service when its current grant, staff assignment, and record-level authorization permit it; neither Plan replaces authorization. The Phase 10 delegated multi-client route uses the firm's export entitlement, while the existing Phase 9 single-client export route still follows that client's CSV/XLSX Plan capability and role rules. [ADR 0036](./adr/0036-split-firm-capabilities-from-client-resource-plans.md)
- Relationship creation is Firm request → Client Owner/Admin verifies the firm and approves scopes after recent reauthentication → Firm Owner/Admin accepts. A pending relationship reserves a client slot but grants no data access; either side may cancel it.
- Active Firm Access Grants expire after 12 months unless renewed, with advance notice. Expiry removes delegated access just like revocation.
- Firm Owner/Admin may assign multiple active staff to one client, and a staff member may be assigned to multiple clients within the seat ceiling. New grants start with no staff assigned; removing an assignment ends that staff member's client access immediately.
- Portfolio rows show currently granted clients, grant/Drive state, missing Expected Document Item count, actionable review count, assigned staff, and content-free recent activity. They omit filenames, Tax IDs, amounts, OCR text, and notes. Firm Members see only assigned clients.
- A Client Owner/Admin or an assigned firm staff member with an explicit checklist-management scope may create Expected Document Items and manually link an authorized Document to satisfy one. Phase 10 does not auto-match them with OCR.
- Only a firm Owner/Admin who is currently assigned to every selected client and whose clients each granted export scope may request or download a multi-client export. Firm Members cannot bulk export merely because they may review records.
- The initial workbook hard ceilings are 20 selected clients, 50,000 Approved rows total, and 128 MiB XLSX output. All requests run as asynchronous jobs with bounded streaming generation. Check client/row counts before queuing; enforce the byte ceiling during generation and fail safely if exceeded. Never silently truncate rows.

## Agreed in round 6

- A client-approved read scope covers all of that Client Organization's current and future Documents that remain eligible under their status and record-access rules. The approval screen must say that new eligible Documents will become visible to assigned firm staff without a second approval. Every record open still checks the current grant, assignment, Document status, and applicable visibility rules.
- Firm Owner/Admin manages staff assignments without asking the client to approve each person. The Client Owner/Admin can see the current assigned staff roster and its change history; a roster view does not itself grant those staff more access.
- A pending firm-client request expires after 7 days. Pending state gives no access and releases its reserved client slot when it expires. The existing 24-hour staff invitation lifetime remains a separate rule.
- Phase 10 defines Firm Plan entitlements without adding public pricing, checkout, or a new trial. During testing, trusted manual grants must have an end time; pricing and payment are separate future decisions.
- Multi-client export date filters are interpreted separately in each Client Organization's timezone. The Summary sheet identifies each client's timezone so the local date boundaries are clear.
- Multi-client Approved export includes Available Documents by default. Archived requires an explicit choice; Trash and Purged are never included. Every selected client receives a sheet with the versioned header even when it has zero eligible rows, and Summary shows zero.
- Grant, assignment, and multi-client export audit is content-free and retained for one year. Record actor, firm/client IDs, grant scope/revision, request/job correlation, row count, and request/denial/completion/download/expiry events; do not record filenames, cell values, tokens, or original/OCR content. Privacy/legal approval of retention remains a release gate.
- Develop Phase 10 behind default-off gates. Firm review and multi-client Approved export cannot activate until the dependent Phase 9 OCR/review/export staging, security, and performance gates pass. Portfolio and Expected Document checklist may activate separately after their own authorization proof.
- Portfolio aggregate counters may lag source events by at most five minutes and show their last-updated time. Grant revocation and record authorization are immediate and cannot rely on a stale portfolio read model.

## Agreed in round 7

- The firm has one cross-client actionable Review Queue built from current Client Organization review items. It uses stable keyset pagination and client labels with Mine, Unassigned, and Client filters. Listing requires current grant, client assignment, and the action scope applicable to that row (`review.propose`, `review.confirm`, or `suggestion.approve`); assigning an item does not grant access. Opening a record reauthorizes it against current client state.
- A Client Organization Google Drive Connection serves only that client's own work. The multi-client workbook is downloaded from the firm workspace; it is never written into one or every client's Drive.
- Grant scope expansion and 12-month renewal require fresh Client Owner/Admin authentication. The firm must accept an expanded scope before using it. Scope reduction and revocation take effect immediately.
- A granted read scope permits original preview for authorized review, but downloading original bytes requires a separate `original.download` scope and a fresh authorization check.
- If any included Approved row's current source or approval revision changes during generation or before download, invalidate the whole workbook and require a new request. Do not substitute, omit, or deliver stale rows.
- An Expected Document Item is Pending before its due date and Missing/Overdue afterward if a human has not linked an authorized Document. The due date is interpreted in the Client Organization timezone.
- Each Client Organization may have monthly or quarterly Expected Document Templates that create items for a period. Authorized humans may edit that period's items; Phase 10 does not auto-match uploaded Documents with OCR.
- The requester explicitly selects up to 20 clients from those they are currently assigned to and whose grants allow export. No clients are preselected and “all clients” is not the default.

## Agreed in round 8

- When the Accounting Firm Plan expires, stop new portfolio, workload, and daily-briefing aggregation, delegated review, and multi-client export requests. Keep relationship/status access and security or recovery actions, including revocation. A previously completed workbook remains downloadable only while its artifact and link are live and every current client authorization check passes; expiry cannot revive revoked access.
- Per-staff workload counts actionable Review Items assigned to that staff member, overdue Expected Document Items assigned for follow-up, and open Tasks of the Accounting Firm Organization assigned to that staff member. It does not infer effort from financial amounts or all uploaded Documents.
- Firm Owner/Admin assigns Review Items only to staff who currently have the Client Assignment and review scope. Staff cannot claim an item themselves in the first version. Removing a Client Assignment makes that staff member's Review Items unassigned without completing the review or closing related Tasks.
- The daily briefing uses the firm Organization timezone for its day and shows the user's assigned review work, Expected Document Items due or overdue in each client's timezone, open work, grants nearing expiry, and failed exports. It exposes only content-free counts/status for currently authorized clients and appears in the firm workspace on demand.
- Portfolio uses cursor pagination at 10 clients per page, a stable client-name-plus-ID order, and a “needs follow-up” filter derived from counters. It does not sort pages by mutable backlog counts.
- The initial Firm Access Grant scopes are `read`, `review.propose`, `review.confirm`, `suggestion.approve`, `checklist.manage`, `export.approved`, and `original.download`. The initial grant enables only `read`. Every delegated client operation checks current firm membership and the applicable Firm Plan capability, client grant, assignment, action scope, and record authorization. Relationship revocation and security/recovery actions remain possible after Firm Plan expiry. These precise names supersede the shorthand `confirm` and `approve` in round 4.
- Performance acceptance targets are portfolio and cross-client Review Queue P95 at most 2 seconds for 20 clients and 5 staff, and a 50,000-row workbook completed within 10 minutes under representative load. Measure actual CPU, memory, and concurrency before enabling; a passing build is not evidence of these targets.
- Only the original requester may download a completed workbook. They must still be a firm Owner/Admin, have a current assignment to every included client, and satisfy each client's current `export.approved` grant and source-revision checks. Another Owner/Admin creates a new request; possession of a link or job ID is not authorization.

## Agreed in round 9

- A firm Owner/Admin without a current Client Assignment may see only the client's identity, grant state, and staff roster needed to manage the relationship. Missing-document counts, review counts, and activity for that client require the viewer's own current assignment and applicable grant scope. A firm Member still sees only assigned clients.
- A Client Owner/Admin may inspect content-free multi-client export audit entries for their own Client Organization, including the firm, actor, time, client row count, and job status. They cannot see the other selected clients, their counts, or workbook content. Firm Owner/Admin may inspect the firm's own audit under current firm authorization.
- Open Tasks in staff workload are Tasks owned by the Accounting Firm Organization and assigned to that firm staff member under the existing Task model. A firm Task may refer to an assigned client without copying sensitive client content. Phase 10 does not introduce a delegated Client Organization Task Assignee; the earlier Q56 wording about client-owned assigned Tasks is superseded.

## Agreed in round 10

- Firm Owner/Admin assigns Expected Document Items for follow-up only to current firm staff who have the relevant Client Assignment and `checklist.manage` scope. Client Owner/Admin may still create and edit the checklist but does not manage the firm's staff roster. Removing a Client Assignment makes affected follow-up items unassigned without changing whether the document is Missing/Overdue.
- A Client Organization may grant access to multiple Accounting Firm Organizations at once. Each firm-client grant, scope, expiry, staff roster, and audit view remains separate; a firm cannot discover another firm's relationship or work through its own workspace.
- Show grant-renewal reminders in both client and firm workspaces to authorized grant managers starting 30 days before expiry. Phase 10 sends no automatic LINE/email reminder. Renewal still requires the Client Owner/Admin's recent reauthentication and the firm acceptance rule for expanded scope.

## Implementation checks from the Phase 10 brief

- Treat firm entitlement, firm role, current Client Assignment, current client grant/scope, Client Organization tenant boundary, and record visibility as separate checks. A Plan limit never substitutes for RBAC or client authorization. Filter all aggregates to currently granted and visible clients; reauthorize each record open and download. Revocation must remove access immediately even if a counter projection is up to five minutes old.
- Build portfolio, workload, daily briefing, and the cross-client queue from a paginated, set-based read path over authorized client IDs. Avoid one API/SQL round trip per client; measure query count and latency at the agreed 20-client/5-staff workload. Do not use a stale read model as the authorization source.
- Generate the selected-client workbook asynchronously with bounded streaming and a private artifact. Enforce the agreed client, row, and XLSX-byte ceilings; reuse Phase 9's fixed Approved row schema and spreadsheet cell safety. At request, generation, status, and download, validate current authorization and frozen source revisions. Any lost access or stale row invalidates the whole file.
- Audit grant, assignment, export request/denial/completion/download/expiry, and file invalidation without workbook cells, original content, tokens, or other sensitive payloads. Verify cross-tenant and revocation races before enabling a feature gate.
