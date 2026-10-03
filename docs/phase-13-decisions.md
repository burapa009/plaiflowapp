# Phase 13 — Advanced Company Secretary decisions

Status: interview in progress, updated 2026-09-28. These decisions do not authorize implementation, deployment, or activation. Phase 10's on-demand firm briefing and authorization model remain the baseline until an explicit later decision changes them.

## Existing product boundary

- Phase 2 already has a synchronous, read-only Assistant Summary based on the authorized Work Dashboard. It returns aggregate bullets without source links or an LLM. Its existing contract explicitly excludes persistence, scheduling, and caching; Phase 13's async cached briefing needs an explicit separate contract or a deliberate revision of that one.
- Task, priority, due date, reminder, and per-Membership Web/LINE notification preferences exist. Recurring Tasks and Meetings do not. Phase 10's Expected Document checklist and firm daily briefing are documented but not yet implemented; Phase 13 must not treat them as live sources.

## User constraints

- Gather candidate facts deterministically before any LLM call. Build the briefing asynchronously, pass a compact context, cache the resulting snapshot, and measure generation latency and cost.
- Include only data the current user can access. A cached result must not leak across users, roles, Organizations, clients, grants, or assignments; every deep link reauthorizes. The secretary does not execute tasks, send follow-ups, approve records, or change business data automatically.

## Confirmed in round 1

- The first job of the Secretary Briefing is to prioritize work requiring action today, with an explanation and a link to each source. Follow-ups, missing documents, and meetings are possible sources of that work, not separate autonomous workflows by default.
- The secretary is a capability for existing Owner/Admin/Member roles, not a new authorization role. Start with a company briefing. A firm view may include only clients currently permitted by the firm's grant, the user's assignment, scope, and record access.
- Initial Secretary Intents are predefined read-only questions. When the available data cannot answer one, state what is missing; do not create a Task from the conversation.
- Build the briefing snapshot asynchronously and show it in the app first. Any later outbound briefing is a separate user opt-in by channel and recipient; the Phase 10 rule of no automatic LINE/email briefing still holds for the initial version.
- A meeting action item starts from a human-recorded or human-confirmed Task. Do not infer an instruction to create work from a calendar event, transcript, or uploaded document.

## Confirmed in round 2

- Secretary Briefing is a separate capability from the Phase 2 Assistant Summary. Keep the existing synchronous summary contract while introducing the async cached briefing. [ADR 0038](./adr/0038-separate-secretary-briefing-from-assistant-summary.md).
- In a company briefing, a Member sees only work they can currently access, such as assigned or watched Tasks. Owner/Admin may also see authorized team work. Opening every source link rechecks current record and role authorization.
- Deterministic ranking puts overdue work before work due today; Task Priority orders items within each group. A user's personal ordering does not change the Task's Priority or anyone else's view.
- A recurring routine produces a suggestion for a human to confirm before creating a Task. Do not create a duplicate while a previous occurrence remains unfinished.
- Missing-document follow-up starts only from a human-defined Expected Document Item whose due date has passed in the Client Organization timezone and has no human-linked authorized Document. Briefing may show who should follow up and why; it does not infer absence from OCR or send a message.
- Generate an async Briefing Snapshot on the user's first opening for that day, show its generation time, and allow manual refresh. Before every display, filter against current authorization; discard a snapshot whose content cannot be safely filtered after an access change. A deep link reauthorizes independently.

## Confirmed in round 3

- Deliver the company briefing first. The firm view follows only after its Phase 10 source data and delegated authorization checks are implemented and proven.
- A company briefing day uses the Organization Timezone. A firm briefing day uses the firm Organization timezone, while each client's document due date remains interpreted in that Client Organization's timezone.
- Show at most five top actions, the remaining count, and a link to the full authorized work list. Deterministic due-date and Priority rules choose the order; the LLM may explain but cannot reorder it.
- Owner/Admin define an Organization's Routine Template and responsible person. A due routine creates one Routine Suggestion for human confirmation; do not backfill many missed occurrences or create another while the prior occurrence remains unfinished.
- Meeting action items are human-recorded Tasks with human-entered meeting source details in the first version. No Meeting entity or calendar integration is required yet.
- The first version adds no briefing Notification or outbound delivery. Users open and refresh the briefing themselves. A later opt-in may offer at most one daily briefing notice per recipient, using that user's chosen channel, local time, and quiet hours.
- Pass only currently authorized, compact structured facts and the minimum short Task titles needed for wording to the LLM. Exclude original files, full OCR text, payment data, and secrets. On LLM failure, show a deterministic briefing from the gathered facts.

## Confirmed in round 4

- A Briefing Snapshot is for its Organization-local day only; there is no briefing history. Delete it within 24 hours after that day ends.
- Check current authority before gathering data, before publishing a generated snapshot, and before each display. If authority changes during generation, discard the result. A non-security Task change may leave a time-stamped snapshot marked possibly stale until manual refresh; an authorization change never permits stale content to be shown.
- Design a compact LLM input contract, but keep external LLM transmission disabled until a named provider and its retention/privacy terms are approved. The deterministic briefing is the initial usable result.
- Gather and rank all eligible candidates deterministically, then pass at most the first 20 and 2,000 tokens to a future LLM. The five displayed actions and the remaining count come from deterministic ranking; truncation of LLM context cannot hide items from the full authorized list.
- Once deterministic gathering succeeds, make that briefing available even if the worker or LLM wording fails. Retry wording only within a bounded job policy and never duplicate or cross-scope a result.
- Provisional measurement targets: cached read p95 at most 500 ms and fresh briefing p95 at most 30 seconds. Record gather, queue, and LLM time; candidate count, token usage, and cost per briefing. Set a numeric spend ceiling after choosing a provider and measuring a representative baseline; these targets are not a claim of current performance.

## Confirmed in round 5

- The first intent menu asks: what to do today, why an item ranks first, whom to follow up with, which documents are missing, and which meeting actions remain open. An intent whose source is not configured says so explicitly; it never reports zero outstanding work from absent source data.
- Member sees personal authorized work. Owner/Admin sees separate personal actions and authorized team blockers, without employee rankings or productivity scores.
- Routine Templates support weekly, monthly, and quarterly schedules in the Organization Timezone. The first version has no holiday shifting and does not backfill many missed periods.
- A contact follow-up is an ordinary Task. Completing a contact Task does not complete its Expected Document Item; the item stays missing until a human links an authorized Document. No separate Follow-up state exists in the first version.
- A meeting action is a Task with human-entered meeting name and date as its source. The first version has no Meeting entity, transcript ingestion, or calendar connection.
- User preferences are scoped to Membership: show or hide categories and pin items in that user's briefing. They cannot change source Task Priority, another user's order, or authorization.
- Keep the Phase 13 feature flag off by default and pilot only with named test Organizations. Begin with deterministic output; do not market it as an active paid-plan entitlement before release evidence and commercial decisions.

## Open branches for the next rounds

- Confirm the proposed company-pilot closeout below, including refresh limits, pin ordering, and acceptance evidence.
- Named LLM provider, retention/privacy review, numeric spend ceiling, outbound notification controls, and firm per-client behavior belong to later activation decisions.

## Proposed company-pilot closeout — awaiting shared-understanding confirmation

- Keep at most one active generation per User, Organization, and local day. Duplicate opens return that job or snapshot. Manual refresh replaces it at most once per 15 minutes; a revoked or invalid snapshot is hidden immediately and its regeneration is exempt from the cooldown. Bound retries and always retain the deterministic fallback when gathered data is valid.
- Personal pins affect order only within the same overdue/due-date group; they cannot push overdue work below future work, change a Task, or expose an inaccessible record. An inaccessible pin disappears from the user's view.
- Every snapshot item retains source identity and scope so current authorization can be checked before display. Mixed-source prose that cannot be filtered safely after revocation is discarded. Deep links perform their own current record authorization. Task titles, source text, LLM prompts, and briefing prose do not enter logs or metric labels.
- Company-pilot acceptance: test Member/Owner/Admin, two Organizations, role removal during generation and after caching, cross-tenant IDs, revoked deep links, a hostile Task title, routine duplicate/skip behavior, meeting-source visibility, timezone boundaries, and the unavailable-checklist state. Assert no automatic Task creation, follow-up message, or approval. Measure cached-read and fresh-generation p95 under a recorded representative workload, queue depth, retries, and per-briefing provider cost (zero while external LLM is disabled).
- The firm view remains a later dependency on implemented Phase 10 checklist/review data and a separate delegated-access matrix, including multiple firms, assigned versus unassigned staff, grant expiry/revocation, client-local due dates, and no client-by-client query growth.
