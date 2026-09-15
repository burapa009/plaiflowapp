# Phase 2: Task, Reminder, Notification, Work Dashboard, Assistant Summary และ CSV Export v1

## Problem Statement

สมาชิกของ Organization ยังไม่มีพื้นที่กลางสำหรับสร้าง มอบหมาย ติดตาม และปิดงาน จึงต้องอาศัยข้อความหรือความจำส่วนบุคคลและไม่สามารถเห็นงานที่ใกล้ครบกำหนด งานที่ไม่มีผู้รับผิดชอบ หรือสิ่งที่ต้องทำต่อได้อย่างเชื่อถือได้ การแจ้งเตือนที่ไม่มี recipient, preference, deduplication และ tenant boundary ที่ชัดเจนยังเสี่ยงทั้งข้อความซ้ำและการเปิดเผยข้อมูลข้าม Organization

Owner และ Admin ยังไม่มี Work Dashboard ที่สรุปภาระงานโดยไม่ปะปนกับ Operations Dashboard เดิม ผู้ใช้ยังไม่มี Assistant Summary ที่ตรวจสอบย้อนกลับได้ และ Organization ยังส่งออก Task เป็น CSV พร้อมประวัติและมาตรการป้องกัน spreadsheet formula injection ไม่ได้ ขณะเดียวกันระบบต้องเตรียม seam สำหรับ Plan, Entitlement และ Usage ในอนาคตโดยไม่ให้ frontend plan state หรือ commercial policy กลายเป็นสิทธิ์เข้าถึงข้อมูล

## Solution

เพิ่ม Task ที่เป็นข้อมูลของ Organization มี Task Creator หนึ่งราย มี Assignee ได้ไม่เกินหนึ่งราย มี Watcher ได้หลายราย และใช้ lifecycle ขนาดเล็กที่ตรวจสอบได้ ระบบสร้าง Domain Event แบบเฉพาะเจาะจงใน transaction เดียวกับการเปลี่ยน Task แล้วให้ worker สร้าง Reminder, Notification, Digest และ LINE direct-message Delivery Attempt แบบ at-least-once พร้อม logical deduplication

เพิ่ม Work Dashboard แบบ query-on-read และ Assistant Summary ภาษาไทยแบบ deterministic ซึ่งใช้ read model และ authorization เดียวกัน เพิ่ม CSV Export v1 สำหรับ Task โดยใช้ fixed columns, server-side filters, role checks, formula-injection protection, export history และ audit Small export ถูก stream โดยตรง ส่วน large export ทำเป็น background job พร้อม private file และ expiring download link

เพิ่ม interface shell สำหรับ Plan, Entitlement, Usage และ server-side feature gate โดย Phase 2 ยังไม่บังคับ commercial limit จริง ทุกเส้นทางยังต้องผ่าน session, Organization Context, Membership, RBAC และ PostgreSQL tenant controls ก่อนพิจารณา Entitlement

## User Stories

1. As a Member, I want to create a Task in my Organization, so that work has an accountable record.
2. As a Task Creator, I want to provide a title and optional description, so that the expected work is understandable.
3. As a Task Creator, I want to assign at most one active Membership, so that responsibility is unambiguous.
4. As a Task Creator, I want to leave a Task unassigned, so that new work can enter a shared queue before ownership is known.
5. As a Task Creator, I want to add multiple Watchers, so that interested members can follow changes without becoming responsible for the work.
6. As an Assignee, I want assignment to imply notifications without also becoming a stored Watcher, so that I do not receive duplicate notifications.
7. As a Member, I want to see every Task in my Organization, so that work is transparent within the tenant.
8. As a Member, I want Tasks from another Organization to be inaccessible, so that tenant data never crosses boundaries.
9. As a Task Creator, I want to edit the details, Assignee, Watchers, Due Date and Priority of a Task I created, so that the work can evolve.
10. As an Assignee, I want to change the Task Status, so that progress is current.
11. As an Owner or Admin, I want to manage any Task in the Organization, so that abandoned or incorrect work can be corrected.
12. As a Watcher, I want watching alone not to grant edit rights, so that notification interest is not confused with authorization.
13. As a user, I want a Task to move among Open, InProgress, Done and Cancelled, so that its lifecycle is explicit.
14. As an Assignee, I want to reopen Done or Cancelled work as Open, so that work can resume without losing history.
15. As an auditor, I want completion actor and time retained, so that the current completion is attributable.
16. As an auditor, I want reopened completion history preserved through Domain Events, so that clearing the current completion does not erase what happened.
17. As an Organization member, I want Cancelled Tasks hidden by default but still filterable and exportable, so that obsolete work does not clutter the active queue or disappear from history.
18. As a security owner, I do not want Task hard deletion in Phase 2, so that audit and export records remain coherent.
19. As a Task Creator, I want an optional date-only Due Date, so that I can express the business deadline without unnecessary time precision.
20. As a user, I want Due Date and reminder behavior interpreted in the Organization Timezone, so that members share the same deadline.
21. As a user, I want overdue to be derived after the Due Date ends, so that Overdue is not confused with Task Status.
22. As a Task Creator, I want Normal, High and Urgent priorities, so that important work can be ordered without changing permissions or reminders.
23. As an Owner or Admin, I want removal of a Membership to unassign active work and remove watches atomically, so that departed members do not remain recipients.
24. As an auditor, I want the original Task Creator identity retained after Membership removal, so that historical attribution remains intact.
25. As a domain developer, I want every accepted Task mutation to produce a specific immutable Domain Event, so that downstream behavior does not depend on a generic task.updated diff.
26. As a domain developer, I want Domain Event Causation to support Inbound Event, User command and schedule origins, so that the system does not fabricate inbound events.
27. As a notification worker, I want Task mutation and Domain Event persistence to commit together, so that accepted state changes cannot lose their notification trigger.
28. As a user, I want a reminder one day before the Due Date, so that I can prepare.
29. As a user, I want a reminder on the Due Date, so that work is not missed.
30. As a user, I want a reminder one day after an unfinished Due Date, so that overdue work resurfaces.
31. As a user, I want missed reminder milestones collapsed into one current reminder, so that creating or editing an old Due Date does not cause a notification storm.
32. As a user, I want pending reminders recalculated when the Due Date changes, so that obsolete schedules do not fire.
33. As a user, I want pending reminders stopped when a Task becomes Done or Cancelled, so that completed work stays quiet.
34. As an Assignee, I want to receive relevant assignment, change and due notifications, so that I can act on my work.
35. As a Watcher, I want to receive relevant Task changes according to my preferences, so that watching remains useful.
36. As a Task Creator, I do not want notifications merely because I created the Task, so that recipient rules remain explicit.
37. As an Owner or Admin, I do not want notifications for every Task merely because of my role, so that administration does not create noise.
38. As an actor, I do not want a notification for my own Task mutation, so that I am not told what I just did.
39. As a former Assignee, I want to know when work is reassigned or unassigned, so that responsibility changes are clear.
40. As a newly added Watcher, I want to know that I am now watching a Task, so that the subscription is visible.
41. As a user, I want Notification separated from channel Delivery Attempt, so that read state is not confused with provider retries.
42. As a user, I want Notification Preferences scoped to my Membership, so that different Organizations can use different notification behavior.
43. As a user, I want Assignment, TaskChange and DueReminder preferences, so that I can control meaningful categories without configuring every event type.
44. As a user, I want Immediate, Digest or Off per category and channel, so that notification timing matches my needs.
45. As a new user, I want useful Web defaults and LINE disabled until opt-in, so that I receive essential information without unexpected external messages.
46. As a user, I want a daily Digest grouped by Task, so that repeated changes are concise.
47. As a user, I do not want Immediate notifications repeated in a Digest, so that one event does not become duplicate noise.
48. As a user, I do not want an empty Digest, so that PlaiFlow sends only useful messages.
49. As a user, I want LINE notifications sent only to my linked direct-message identity, so that Task information never falls back to a group or another person.
50. As a privacy-conscious user, I want LINE messages to omit Task title, description, Assignee and customer data, so that lock-screen and provider previews reveal little.
51. As a user, I want every Task deep link to recheck my current session, Membership and role, so that possession of a URL grants no access.
52. As a user, I want Web Notification to remain independent of LINE failure, so that provider outages do not hide work.
53. As an operator, I want transient LINE failures retried within bounded limits and permanent errors stopped, so that retries are useful rather than endless.
54. As a user, I want an unread Web inbox with mark-one and mark-all actions, so that I can manage attention.
55. As a privacy owner, I want user-facing Notifications retained for 90 days, so that sensitive content does not accumulate indefinitely.
56. As a Member, I want My Tasks, Watching and All Tasks views with stable cursor pagination, so that large Task sets remain navigable.
57. As a Member, I want status, Assignee, Watcher, creator, Priority, overdue and date-range filters, so that I can narrow the queue.
58. As a Member, I want My Focus, Watching and unread Notification widgets, so that the Work Dashboard answers what needs attention now.
59. As an Owner or Admin, I want a Team Queue for unassigned, overdue and Urgent work, so that operational gaps are visible.
60. As a Member, I do not want per-person performance aggregates for other members, so that Dashboard access does not become an unintended leaderboard.
61. As an Owner or Admin, I want Task counts by Assignee without ranking or performance scoring, so that workload can be balanced responsibly.
62. As an operator, I want the existing Operations Dashboard kept separate, so that tenant work data and system health retain different authorization boundaries.
63. As a user, I want a deterministic Thai Assistant Summary, so that every statement is traceable to current authorized data.
64. As a Member, I want the Assistant Summary focused on my assigned and watched Tasks, so that it stays relevant.
65. As an Owner or Admin, I want the Assistant Summary to include unassigned, overdue and Urgent team work, so that I can see Organization-level risk.
66. As a user, I want each Assistant Summary to show generation time, scope and deep links, so that I know what the summary represents.
67. As a security owner, I do not want Task data sent to an external LLM in v1, so that Assistant Summary adds no new data processor.
68. As an Owner or Admin, I want to export filtered Tasks from one Organization as CSV, so that I can analyze and archive business work.
69. As a security owner, I want Members unable to perform bulk CSV export in v1, so that normal read access does not automatically become bulk extraction authority.
70. As an exporter, I want CSV filters and ordering to match the authorized Task list, so that the downloaded result is predictable.
71. As an exporter, I want Thai text encoded for common spreadsheet tools, so that the file opens legibly.
72. As a security owner, I want every untrusted CSV cell protected from spreadsheet formula execution, so that opening an export cannot run attacker-controlled formulas.
73. As a security owner, I want CSV filenames, headers and cell quoting generated by the server, so that user input cannot alter download behavior or file structure.
74. As an exporter, I want exports of up to 5,000 rows returned synchronously, so that ordinary downloads do not require job management.
75. As an exporter, I want exports from 5,001 through 100,000 rows generated in the background, so that large files do not exhaust an HTTP request.
76. As an exporter, I want an explicit error above 100,000 rows, so that I can narrow filters instead of creating an unsafe job.
77. As an exporter, I want a completed background file available through a short-lived download link, so that persisted exports are not public.
78. As a security owner, I want authorization rechecked when a background job starts and when its file is downloaded, so that revoked users cannot retain access.
79. As an Owner or Admin, I want export history showing requester, filters, row count, outcome and timestamps, so that bulk data movement is accountable.
80. As an auditor, I want requested, completed, failed, downloaded and expired export events recorded append-only, so that the export lifecycle is reconstructable.
81. As a privacy owner, I do not want audit records to contain CSV content, signed URLs, storage keys or sensitive filter values, so that audit itself does not become a leak.
82. As a future pricing owner, I want Plan, Entitlement and Usage interface shells, so that limits can be introduced without rewriting feature code.
83. As a security owner, I want feature gates evaluated on the server after authentication and RBAC, so that frontend plan display cannot grant permission.
84. As a user, I want login, existing authorized reads, notification opt-out and security controls to remain available during entitlement failure or downgrade, so that commercial policy cannot lock me out unsafely.
85. As a developer, I want Usage recording to be idempotent and incapable of granting access, so that retries cannot inflate usage or bypass authorization.
86. As an operator, I want paginated and indexed Task and Notification queries, so that Organization growth does not create unbounded reads.
87. As an operator, I want Dashboard query and response budgets, so that a summary page cannot exhaust database capacity.
88. As an operator, I want notification batches, bounded LINE concurrency and bounded retry, so that outbound work cannot overwhelm the provider or database.
89. As an operator, I want CSV generation streamed with strict row, byte and memory ceilings, so that one export cannot exhaust the worker.
90. As a tester, I want cross-tenant, role, deduplication, threshold and formula-injection behavior proven through public boundaries, so that security properties are executable rather than assumed.

## Implementation Decisions

### Scope and canonical boundaries

- Phase 2 adds Task creation and editing through authenticated Web flows only. LINE is an optional outbound direct-message channel; Assistant Summary is read-only.
- Continue using User, External Identity, Organization, Membership, Owner, Admin, Member, Organization Context, Domain Event and the terminology defined in the project glossary.
- Rename the existing technical health view conceptually to Operations Dashboard. Work Dashboard is the new Organization-scoped user surface and must not reuse the service-token authorization of the Operations Dashboard.
- Task title and description must be treated as potentially containing personal or sensitive business data in logs, notifications, exports and diagnostics.
- Every tenant-owned identifier supplied by a browser, cursor, job or deep link is only a requested identifier. The server recreates Organization Context and authorizes the referenced resource for every request.

### Task model and lifecycle

- Task belongs to exactly one Organization and has a UUID, title, optional description, immutable Task Creator identity, optional Assignee, Task Status, Priority, optional Due Date, completion actor/time, status-change actor/time, created time and updated time.
- Trim title and require 1–200 Unicode characters. Description is optional and limited to 5,000 Unicode characters. Reject invalid values before persistence and never log their content.
- Task Status is exactly Open, InProgress, Done or Cancelled. New Tasks start Open. Overdue is derived and is never stored as a fifth status.
- Entering Done records the current completion actor and time. Reopening returns the Task to Open and clears the current completion fields; immutable Domain Events retain prior completion history.
- Cancelled replaces delete. Cancelled Tasks are excluded from default active lists but remain filterable, linkable, auditable and exportable. No hard-delete or soft-delete column is added.
- Priority is exactly Normal, High or Urgent and defaults to Normal. It changes ordering only; it does not alter authorization, Entitlement or Reminder Schedule.
- Due Date is nullable and date-only. Organization gains an IANA Organization Timezone defaulting to Asia/Bangkok. A nonterminal Task becomes Overdue after the local Due Date ends.
- Assignee and Watcher must be active Memberships in the same Organization as the Task. Enforce tenant consistency in both Go authorization and PostgreSQL composite constraints/RLS.
- Store Watchers in a unique Task-to-Membership relation. Assignee is an implicit recipient and is not duplicated as a Watcher; assigning an existing Watcher removes that Watcher relation. A former Assignee does not automatically become a Watcher.
- When a Membership ends, atomically clear it from active Task assignment and remove its Watcher relations. Keep Task Creator identity and Domain Event history. Membership removal must not be blocked by Task ownership.
- All active Memberships may read all Tasks in their Organization. Owner and Admin may mutate any Task. Member may create a Task, mutate Task details/assignment/watchers on Tasks they created, and change Task Status when they are the Assignee. Watcher status alone grants no mutation permission.
- All Web mutations use the existing same-origin session, Origin and session-bound CSRF protections.

### Task list contract

- Provide Organization-scoped create, detail, update and list operations. Do not expose a Task delete operation.
- Default list scope is My Tasks with Open and InProgress statuses. Support explicit All Tasks and Watching scopes without changing the underlying authorization rule.
- Support filters for multiple statuses, Assignee including me/unassigned/specific Membership, Watcher me, Task Creator, multiple priorities, Overdue, Due Date range and created-time range.
- Due Date endpoints are inclusive local dates. Created-time filters use complete local-day boundaries converted to UTC by Organization Timezone.
- Do not add title/description full-text search in v1.
- Use opaque cursor pagination with 50 rows by default and 100 maximum. Default active ordering is Overdue first, then Due Date ascending with null last, then Urgent/High/Normal, then stable Task ID. Completed and Cancelled history uses updated time descending and stable Task ID.
- A cursor never carries authorization. Reject malformed cursors and reapply Organization Context, filters and role checks on every page.

### Domain Event and notification trigger catalog

- Persist Domain Event in the same PostgreSQL transaction as its Task mutation. The persisted event acts as the durable asynchronous handoff; do not introduce an external message broker.
- Generalize the existing envelope so source Inbound Event is optional. Record Organization, event type, occurrence time, Task subject, optional actor and typed Causation. Never create a synthetic Inbound Event for Web or schedule actions.
- The initial Task Domain Event catalog is task.created, task.details_changed, task.assigned, task.unassigned, task.watcher_added, task.watcher_removed, task.due_changed, task.priority_changed and task.status_changed. Do not add task.updated.
- Creating a Task with an initial Assignee and Watchers emits task.created only. Later reassignment uses task.assigned with old/new references; task.unassigned is used when assignment becomes empty.
- Domain Event payloads carry identifiers and changed-field names required by consumers, not title, description or copies of sensitive values.
- The notification worker processes persisted events at least once. A unique consumer result prevents duplicate user-visible Notification when a worker crashes or retries.

### Reminder behavior

- A nullable Due Date produces logical milestones at 09:00 Organization Timezone one day before, on the date, and one day after. The after-date milestone applies only while Task Status is Open or InProgress.
- Assignee and current Watchers are candidate recipients; Task Creator receives reminders only when also an Assignee or Watcher.
- Changing Due Date invalidates unsent milestones and schedules the new date. Done and Cancelled invalidate all unsent reminders.
- If a Task is created or changed after one or more milestones have passed, create at most one catch-up DueReminder representing the current condition. Do not replay every missed milestone.
- Reminder identity includes Organization, Task, recipient Membership, milestone and the Due Date value. A retry or repeated scheduler scan cannot create another Notification for the same logical occurrence.
- Scheduler execution may be late but must create each logical milestone at most once. Use Organization local calendar rules rather than fixed UTC offsets.

### Notification and Delivery Attempt

- Notification is the recipient-specific logical record. Delivery Attempt is a channel-specific send attempt. Read state, category and digest disposition belong to Notification; provider status, retry count and safe provider error classification belong to Delivery Attempt.
- Notification Category is Assignment, TaskChange or DueReminder. Security/authentication notices are not part of this feature and must not be controlled by Task notification preferences.
- Recipient behavior is:
  - task.created: initial Assignee and Watchers;
  - task.assigned: new Assignee, former Assignee and current Watchers;
  - task.unassigned: former Assignee and current Watchers;
  - details, due, priority and status changes: current Assignee and Watchers;
  - watcher added: newly added Watcher only;
  - watcher removed: no Notification.
- Suppress the mutation actor from Task-change recipients. Scheduled Reminder is not actor-suppressed. Task Creator, Owner and Admin are not implicit recipients.
- Create at most one logical Notification per Organization, recipient, Domain Event and Notification Category. Reminder uses its reminder identity instead of a Domain Event ID. Delivery has at most one logical record per Notification and channel.
- Web Immediate items appear individually in the inbox. Web Digest items appear through the daily Digest. Web Off items do not appear. LINE modes create Delivery Attempts only when their scheduled mode is due.
- If every channel is Off for a category, create no user-visible Notification or Delivery Attempt; retain only the worker idempotency result needed to prevent later retry duplication.
- Store Notification read_at, created time and safe references needed to render current Task context. Do not copy Task description or historical sensitive text into Notification rows.
- Support unread list, mark one read and mark all read. Use cursor pagination with 50 rows by default and 100 maximum. Users cannot delete individual Notifications in v1.
- Retain Notifications and Delivery Attempts for 90 days. Cleanup must preserve export/security audit records governed by their separate retention.

### NotificationPreference, Immediate and Digest

- Store NotificationPreference per Membership, Organization, Notification Category and channel. Preference changes apply prospectively and cannot cause old events to be redelivered.
- Each category/channel mode is Immediate, Digest or Off. Defaults are Web Assignment Immediate, Web TaskChange Digest, Web DueReminder Immediate, and every LINE category Off.
- Notification opt-out and preference writes are never plan-gated. A removed Membership cannot receive notifications or modify preferences in that Organization.
- Daily Digest runs at 09:00 Organization Timezone. It groups pending Digest notifications by Task, shows the latest relevant state and number of changes, excludes items already delivered Immediate, and sends nothing when empty.
- Do not add hourly, weekly, user-selected send time, per-Task reminder configuration or arbitrary event-level preferences in v1.

### LINE direct-message delivery

- LINE delivery requires a linked External Identity for the same User and explicit per-Membership LINE opt-in. A LINE Group Connection proves no group participant Membership and cannot be used as a Task recipient.
- LINE messages contain only a generic notification type, Organization display name and an action to open PlaiFlow. They omit Task title, description, Assignee, Watchers and customer data.
- Deep links contain opaque Organization and Task identifiers only. They contain no session, access token, signed authorization claim or sensitive data. The destination reauthenticates and rebuilds Organization Context.
- Missing identity, provider blocking, unlinking or permanent provider rejection leaves Web behavior unchanged. Never fallback to a LINE Group, another External Identity or another Membership.
- Task mutation never waits for LINE. Process Delivery Attempts asynchronously with at-least-once execution and logical deduplication.
- Claim no more than 100 notification jobs per worker batch and issue no more than 10 concurrent LINE requests. Retry transient errors after 1 minute, 5 minutes, 30 minutes, 2 hours and 12 hours. Stop after the fifth failed attempt. Do not retry permanent client/provider errors.
- Store only safe provider status and error classes. Do not log message bodies, access tokens, complete provider identifiers or deep links.

### Work Dashboard read model

- Expose a single Organization-scoped Work Dashboard read model distinct from the existing Operations Dashboard.
- The read model contains generated time, effective role and:
  - My Focus counts and up to 10 items for overdue, due today, due in the next seven days and Urgent assigned Tasks;
  - Watching count and up to 10 active watched Tasks;
  - unread Notification count and up to 10 latest Notifications;
  - for Owner/Admin only, Team Queue counts and up to 10 items for unassigned, overdue and Urgent Tasks plus counts by Assignee.
- Member responses never contain per-person aggregates for other members. Owner/Admin responses contain workload counts but no leaderboard, completion rate, ranking or productivity score.
- Compute the read model on request from indexed Task, Watcher and Notification data. Do not add a persisted projection, materialized view, cross-tenant cache, WebSocket or automatic polling in v1.
- The page loads the read model once and supports manual refresh. Every item deep link reauthorizes independently.

### Assistant Summary v1

- Generate a deterministic Thai Assistant Summary from the authorized Work Dashboard read model. Do not call an LLM or external AI provider.
- Member scope includes assigned overdue work, due today/next seven days, Urgent work and active watched Tasks with changes. Owner/Admin scope additionally includes unassigned, overdue and Urgent team work.
- Output at most eight concise bullet points with exact counts, generated_at, scope and authorized deep links. Provide a clear no-action-needed result when every count is zero.
- Generate on demand and do not persist, schedule or cache summaries. Assistant Summary cannot create or change Task, Notification, Reminder, preference or export.
- The summary must never infer financial, accounting or customer advice and must not invent causes, recommendations or missing facts.

### CSV Export v1 scope and authorization

- CSV Export contains Task rows from exactly one Organization. It excludes Domain Event payloads, Notification/Delivery history, audit records, Membership lists, External Identity, LINE identifiers and raw provider data.
- Owner and Admin may request Task CSV Export. Member cannot perform bulk CSV Export in v1 even though Member may read individual Organization Tasks. Enforce this rule server-side before counting or generating rows.
- Use the same Task filter semantics and stable ordering as the Task list. Export accepts status, Assignee, creator, Priority, Overdue, Due Date range and created-time range. It does not accept arbitrary SQL, free-text search, custom columns or cross-Organization identifiers.
- If the request originates from a filtered list, submit the normalized server-understood filters rather than trusting a frontend row count or serialized result set.
- Date ranges are optional and inclusive according to Organization Timezone. No default hidden range is applied; an unfiltered request means every authorized Task subject to row ceilings.
- Use a fixed CSV v1 column order: organization_id, organization_name, task_id, title, description, status, priority, due_date, is_overdue, creator_display_name, assignee_display_name, watcher_display_names, created_at, updated_at, status_changed_at and completed_at.
- watcher_display_names is a deterministic display-name list sorted by normalized name and joined with a vertical bar inside one CSV cell. Empty nullable values are empty cells.
- Encode Due Date as YYYY-MM-DD and timestamps as RFC 3339 UTC. Encode the file as UTF-8 with BOM for Thai spreadsheet compatibility and use RFC 4180-compatible quoting and CRLF records.
- Generate the filename from server-controlled Organization slug/date fragments and a random export reference. Never place raw user input in Content-Disposition.

### CSV synchronous and background thresholds

- Run an authorized count using the normalized filters before generation.
- For 0–5,000 rows, stream CSV synchronously from Go using the standard library. Do not persist the file and do not buffer all Task rows in memory.
- For 5,001–100,000 rows, create an Export Job and return an opaque job reference. Generate CSV asynchronously in the Go worker; the existing Python export shell remains inactive because plain CSV requires no separate runtime.
- Above 100,000 rows, reject with a stable export_too_large response that instructs the user to narrow filters. Do not silently truncate.
- Generate rows through a bounded database cursor/stream. Keep export-specific in-memory buffering at or below 8 MiB, allow only one active large export per worker execution lane, and fail safely if the produced file exceeds 256 MiB.
- Background exports represent authorized data at generation start. Record data_as_of when generation begins; do not hold a long-lived database snapshot from request time.
- Recheck active session-derived actor identity, current Membership and Owner/Admin role when accepting a request. Recheck Membership/role before a background worker reads Task rows and when a download is requested. A revoked or downgraded role fails the job or download without revealing file metadata.
- Entitlement is checked when the export request is accepted. A later plan change does not invalidate an already accepted completed file before its expiry, but current Membership and RBAC still apply.

### CSV security and persisted files

- Apply formula-injection protection to every user-controlled string cell, including title, description, Organization/display names and watcher lists. If the value after leading whitespace begins with equals, plus, minus or at-sign, or begins with tab, carriage return or line feed, prefix the exported cell with a single quote before CSV encoding.
- Formula protection occurs before standard CSV quoting and applies regardless of whether a field appears numeric. Server-generated enums, booleans, UUIDs and timestamps use validated canonical serialization.
- Use parameterized SQL and allowlisted filter/column mappings only. Reject unknown filters, sort keys and enum values.
- Use response headers that prevent content sniffing and unintended inline rendering. Synchronous output is an attachment.
- Store background files only in private object storage behind an internal storage seam. Do not store CSV bytes in PostgreSQL or a public web directory.
- Delete persisted files after 24 hours. A download request first authorizes through PlaiFlow, then returns or redirects through a signed URL that expires within 5 minutes. Signed URLs are bearer secrets and must never appear in logs, audit metadata or frontend analytics.
- Do not provision a production storage account, bucket or secret as part of writing or locally verifying this specification. Implementation must keep provider configuration environment-scoped.

### Export Job, history and audit

- Export Job belongs to one Organization and records requester User, normalized filters, format/version, sync/background mode, status, requested/started/completed/failed/expired times, authorized row count, produced byte count, data_as_of, safe failure reason and object expiry. It never stores a signed URL.
- Export Job statuses are Queued, Running, Completed, Failed and Expired. They are not Task Status or Inbound Event status values.
- Expose export history to current Owner/Admin only with cursor pagination. History metadata remains visible after the file expires; download is available only for a nonexpired Completed job.
- Write append-only audit events for requested, rejected, completed, failed, downloaded and expired transitions. Include Organization, actor where present, export reference, request ID, outcome, safe reason, normalized filter summary, row count and timestamps.
- Do not store CSV content, cell values, description text, signed URLs, object keys, provider credentials or raw authorization errors in export history/audit.
- Retain export history and export audit metadata for 365 days; remove file objects after 24 hours. Cleanup is idempotent and an already missing object still transitions safely to Expired.
- Apply Organization RLS to Export Job and any tenant-readable audit data. The export-history API must not expose the generic audit table directly. Worker and migration roles remain separate from request runtime roles and receive only required access.

### Plan, Entitlement, Usage and feature-gate shell

- Define small server-side interfaces for current Plan display metadata, Entitlement decision and idempotent Usage recording. Do not add billing-provider, checkout, subscription synchronization or plan-management UI.
- The feature-gate seam accepts authenticated Organization Context, a fixed capability identifier and requested units, then returns allowed/denied, optional limit/used/remaining/reset values and a stable reason code.
- Phase 2 ships one unlimited/no-commercial-enforcement provider. It may report observed units for tests and operational diagnostics but does not deny a feature based on price plan.
- Initial gateable capability identifiers are Task creation above a future quota, LINE delivery, Assistant Summary, CSV Export and background CSV Export. Capability names are server constants, not frontend-provided arbitrary strings.
- Invoke the feature gate only after authentication, Organization Context and RBAC succeed. A frontend plan badge, hidden button, query parameter, Usage value or cached browser state never grants permission.
- Optional paid capability checks fail closed when the Entitlement provider is unavailable and return a safe stable error. Failure must not weaken or skip RBAC.
- Never entitlement-gate login/session, Organization switching, current authorized reads of existing Task data, NotificationPreference opt-out, identity unlink/session revocation, security controls, ownership transfer/member removal, tenant isolation, audit logging or legally/contractually required deletion/export pathways.
- Business CSV Export may be commercially gated in a future phase. A legally required personal-data request is a separate compliance pathway and is not implemented by this CSV feature.
- Usage recording occurs only after the measured operation succeeds, uses an idempotency key tied to the accepted operation, and can never change an authorization decision from denied to allowed.

### Performance budgets and indexes

- Design and verify against 100,000 historical Tasks, 10,000 active Tasks, 100 Memberships and 100,000 retained Notifications per Organization. Do not add partitioning, search infrastructure or materialized views for this target.
- Task list and Notification inbox read APIs target p95 at or below 300 ms. Work Dashboard and Assistant Summary target p95 at or below 500 ms at the reference dataset, measured server-side without external LINE latency.
- Work Dashboard uses one HTTP request, no more than five SQL statements, no N+1 queries and no widget list over 10 items. Assistant Summary derives from the same authorized read-model seam rather than issuing a second family of queries.
- Every Task, Watcher, Notification, Domain Event worker, Export Job and audit query starts from Organization or a worker claim key. Tenant-owned tables include organization_id, composite same-tenant constraints and RLS consistent with the existing Organization Context ADR.
- Add only indexes justified by accepted query shapes: active Tasks by Organization/status/Due Date/stable ID; Tasks by Organization/Assignee/status/Due Date; Tasks by Organization/created time for export; Watchers by Organization/Membership/Task; Notifications by Organization/recipient/read time/created time; worker claims by processing state/availability/stable ID; Export Jobs by Organization/request time/stable ID.
- Verify representative filter combinations with query plans at the reference dataset. Do not create one index for every possible filter combination; add another index only after a measured query misses the budgets.
- Cursor endpoints return 50 rows by default and cap at 100. No read or export endpoint loads an unbounded Task or Notification collection.
- Notification worker claims at most 100 logical jobs per batch. LINE concurrency is capped at 10. Digest grouping is performed in bounded recipient batches and does not load all Organizations into memory.
- Synchronous CSV begins streaming without building the full result. Background CSV obeys the 8 MiB export-buffer, 100,000-row and 256 MiB file ceilings. A large export must not block notification claims or Reminder processing.
- Retain existing slow-query logging threshold and redact parameters. Add normalized route, duration, row count, export mode, worker batch size and safe outcome metrics without logging sensitive Task or CSV values.

### Error and empty-state behavior

- Use stable safe error codes for invalid Task input, forbidden role action, stale/missing Membership, malformed cursor/filter, unavailable LINE identity, export_too_large, export expired, feature unavailable and usage limit reached.
- Return not-found-equivalent behavior when a Task, Notification or Export Job is outside the current Organization Context; do not reveal that a cross-tenant identifier exists.
- Work Dashboard and Assistant Summary return explicit empty states rather than errors when the authorized query contains no actionable work.
- LINE and background export failures are visible through safe status where authorized but never expose provider response bodies, storage paths, SQL text or secrets.

## Testing Decisions

- Prefer external behavior over implementation details. The primary acceptance seam is an authenticated Organization-scoped HTTP request through PostgreSQL, followed where needed by one worker cycle, then an authorized read/download request. This extends the existing patterns demonstrated by TestLineLoginCreatesOrganizationAndResolvesMembership and TestWebhookToPostgresToWorker.
- Test Task behavior through public HTTP routes with a real PostgreSQL test database: creation, validation, role matrix, same-tenant Assignee/Watcher constraints, lifecycle transitions, reopening, cancellation, Membership removal and cursor/filter behavior.
- Test tenant isolation twice: Go must reject foreign Organization/resource identifiers, and PostgreSQL RLS/composite constraints must reject or hide cross-tenant rows even when a query omits an application predicate.
- Test Domain Event behavior through Task mutations rather than testing helper calls: one immutable event per accepted transition, no event on rejected/rolled-back mutation, no task.updated event, correct optional Causation and one task.created event for initial relationships.
- Test the notification pipeline end to end from Task mutation or logical reminder through worker processing to Notification and Delivery Attempt. Repeat the same worker claim to prove user-visible deduplication.
- Test recipient behavior for actor suppression, creator/Owner/Admin nonsubscription, assignment/reassignment/unassignment, Watcher add/remove and Membership removal.
- Test Reminder boundaries in Organization Timezone: before, exactly at and after every milestone; late scheduler catch-up; Due Date changes; Done/Cancelled cancellation; repeated scans; and a non-Bangkok IANA timezone.
- Test NotificationPreference defaults and every Immediate/Digest/Off path by category and channel. Prove changes are prospective, Immediate items do not reappear in Digest and empty Digest sends nothing.
- Use a fake outbound LINE boundary to verify generic content, no sensitive fields, no group fallback, bounded retry, permanent failure, concurrency ceiling and unchanged Web behavior. Never call the real provider in automated tests.
- Test every deep link after session expiry, Membership removal, role change and cross-Organization substitution. The URL alone must never authorize access.
- Test Web inbox read/unread, mark-one, mark-all, pagination stability and 90-day cleanup through public behavior.
- Test Work Dashboard role variants through its HTTP seam. Assert Member responses omit team-person aggregates, Owner/Admin receive allowed Team Queue data, widget items are capped and every count matches the same authorized Task dataset.
- Test Assistant Summary as deterministic output from the authorized read model: Thai empty state, eight-bullet cap, exact counts, scope metadata, deep links, role-sensitive omission and no external network call.
- Test CSV through the export HTTP seam with fixed golden headers and representative Thai, commas, quotes, CR/LF, Unicode, blank values and duplicate display names.
- Treat CSV formula injection as a trust-boundary test. Include cells beginning directly or after whitespace with equals, plus, minus, at-sign, tab, carriage return and line feed. Assert protection occurs before RFC-compatible quoting and applies to every user-controlled column.
- Test server-generated filename and response headers with malicious Organization/display names. Assert synchronous output is an attachment and no user input becomes a header fragment.
- Test export authorization for Owner, Admin, Member, removed Membership and cross-tenant IDs at request, background execution and download. Frontend plan data must have no effect.
- Test exact row thresholds at 0, 5,000, 5,001, 100,000 and 100,001 rows. Assert no truncation, sync/background selection, stable ordering and safe export_too_large behavior.
- Test background job lifecycle, single logical job handling, data_as_of, private object expiry, five-minute signed-link expiry, revoked-user denial, download audit and idempotent cleanup using a fake private object-store boundary.
- Test export history and audit externally: allowed metadata, 365-day retention, rejected/completed/failed/downloaded/expired events, pagination, RLS, and absence of content, signed URL and object key.
- Test the feature-gate seam through protected HTTP actions. Assert RBAC runs independently, entitlement denial cannot reveal foreign data, frontend state cannot grant access, provider failure closes only optional capabilities, and never-gated safety/access actions remain available.
- Test Usage idempotency by replaying the same successful operation and asserting one measured unit. Assert failed and denied actions do not consume usage.
- Load representative reference data and assert cursor correctness, bounded result sizes and query plans for Task filters, Dashboard, Notification inbox and export count. Performance tests should report budgets separately from correctness tests and must not claim production latency from local results.
- Reuse the repository's standard Go tests, PostgreSQL integration pattern, worker RunOnce seam and Node test runner. Do not add a test framework, browser automation suite or live-provider dependency solely for Phase 2.

## Out of Scope

- Creating or editing Task from LINE, API clients other than the authenticated Web flow, or Assistant Summary
- Multiple Assignees, private Task, subtasks, dependencies, comments, attachments, labels, recurring Task, custom status, custom Priority and percentage progress
- Task hard deletion, bulk Task mutation and import from CSV
- Per-Task custom Reminder, hourly/weekly/custom-time Digest and automatic Priority escalation
- LINE Group Task notifications, rich Task content in LINE, SMS, email, mobile push and provider fallback
- External message broker, distributed scheduler, general event bus and speculative notification-provider framework
- Real-time Dashboard, WebSocket, automatic polling, custom widgets, drag-and-drop, charts, leaderboard and employee performance scoring
- LLM integration, autonomous assistant actions, recommendations, OCR, accounting, bookkeeping, tax calculation, financial advice and insurance advice
- Export formats other than CSV, custom columns, raw event/audit export, cross-Organization export, CSV import and scheduled exports
- Production object-storage provisioning, production secrets and external service activation
- Billing provider, checkout, paid plan enforcement, subscription synchronization, pricing UI and durable commercial Usage ledger
- Legally mandated data-subject request workflow beyond preserving an ungated seam for it
- Partitioning, search engine, full-text description search, materialized Dashboard projection and analytics warehouse

## Further Notes

- This specification follows the project glossary and ADRs for Organization Context, fixed roles, tenant enforcement, compact Task lifecycle, generalized Domain Event Causation, entitlement separation, notification delivery and private LINE behavior.
- Phase 2 is the first real consumer that justifies Domain Event persistence and notification processing; it does not justify an external broker or generic plugin architecture.
- CSV v1 uses Go streaming and the existing worker. The Python export shell remains available for a future capability that genuinely needs a separate runtime, but it is not activated for CSV, OCR, accounting or document rendering.
- The synchronous CSV path deliberately supersedes the Phase 0 export shell's async-only assumption for files of at most 5,000 rows. Background CSV keeps the opaque job-reference pattern for larger files.
- The highest-value test seam is the authenticated Organization HTTP boundary through PostgreSQL and one worker cycle. Narrow contract seams are added only for CSV encoding, private object storage, LINE delivery and Entitlement because those are trust or provider boundaries that cannot be exercised safely through a live dependency.
- Round 4 and detailed CSV mechanics were not separately interviewed because the user requested direct spec synthesis. This specification uses the smallest defaults consistent with the already accepted security and performance boundaries.
- The repository currently has no active Git metadata or configured issue tracker. This specification is stored locally only and cannot receive the ready-for-agent label until project tracker setup is completed.
