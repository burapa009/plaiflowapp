# Phase 13 — Advanced Company Secretary

Status: **specification only, synthesized on 2026-09-28; not implemented, deployed, or activated**. This spec follows the [Phase 13 decisions](./phase-13-decisions.md), [domain glossary](../CONTEXT.md), [ADR 0038](./adr/0038-separate-secretary-briefing-from-assistant-summary.md), and the [Phase 10 firm access contract](./phase-10-spec.md). The company pilot comes first. The firm view depends on implemented and proven Phase 10 sources and delegated authorization.

## Problem Statement

People can see Tasks and an aggregate Assistant Summary, but still need to decide which authorized work needs attention today and why. Follow-ups, expected documents, and meeting actions can be scattered across separate work views. A summary without source links is hard to verify, while a cached or generated answer can expose work after a Membership, Firm Access Grant, or Client Assignment changes. The product needs a timely, explainable daily briefing without creating work or making decisions for people.

## Solution

Add a read-only **Secretary Briefing** for the current user and Organization-local day. On first open, gather currently authorized facts, rank actionable work deterministically, and generate a per-user **Briefing Snapshot** asynchronously. Show up to five top actions with reasons and source links, a remaining count, the generation time, a possibly-stale indicator when appropriate, and a link to the full authorized work list. A small menu of predefined **Secretary Intents** answers questions from the same authorized sources. A user can refresh the snapshot and adjust their own category visibility and pins.

Company Organizations are the first pilot. A later Accounting Firm Organization view may include only clients for which Phase 10 source data and the user's current delegated authority are proven. The initial usable briefing is deterministic. Design a small optional wording input for an LLM, but keep transmission to an external provider disabled until provider, privacy, retention, and cost decisions are approved. The briefing provides operational work guidance only; it gives no financial advice.

## User Stories

1. As a company Member, I want one daily view of my authorized work requiring attention, so that I can start the day without searching several screens.
2. As a company Owner or Admin, I want my actions and authorized team blockers separated, so that I can see what needs my action and what needs coordination.
3. As a user, I want overdue work before work due today and Task Priority within each group, so that the order follows clear source facts.
4. As a user, I want a short reason beside each ranked action, so that I understand its position without trusting an unexplained score.
5. As a user, I want a source link for every action, so that I can inspect the current record before acting.
6. As a user, I want at most five top actions, a remaining count, and a link to my full authorized work list, so that a short briefing never hides other work.
7. As a user, I want the Organization Timezone to define the briefing day and Task due dates, so that “today” follows my Organization's calendar.
8. As a user, I want to see when a briefing was generated and when it may be stale, so that I know when to refresh.
9. As a user, I want repeated opens on the same day to show the same job or snapshot, so that I do not trigger duplicate generation.
10. As a user, I want to refresh the briefing manually, so that recent nonsecurity Task changes can appear.
11. As a user, I want a deterministic answer if optional wording or a worker fails after facts were gathered, so that the briefing remains usable.
12. As a Member, I want only my currently authorized assigned or watched work included, so that another person's private work does not appear in my briefing.
13. As an Owner or Admin, I want team blockers limited to work I can currently access, so that my role does not turn the briefing into a blanket disclosure.
14. As a user removed from a Membership, I want a cached briefing and its source links to stop revealing that Organization's records immediately, so that an old browser tab does not extend access.
15. As a user, I want a revoked item omitted or the unsafe snapshot discarded before display, so that mixed prose cannot leak inaccessible source details.
16. As a user, I want source links to check my current record access when opened, so that a stale link is never authority.
17. As a user, I want to choose “what should I do today?”, so that I get the ranked operational actions for my current context.
18. As a user, I want to choose “why is this first?”, so that I can see the due group, Task Priority, and source reason for an authorized item.
19. As a user, I want to choose “whom should I follow up with?”, so that I can find authorized contact follow-up Tasks and responsible people without sending a message automatically.
20. As a user, I want to choose “which documents are missing?”, so that I can see overdue human-defined Expected Document Items when that source is available and authorized.
21. As a user, I want to choose “which meeting actions remain open?”, so that I can see human-recorded Tasks with meeting name and date as their source.
22. As a user, I want an unconfigured source called out explicitly, so that “no data source” is not mistaken for “no outstanding work.”
23. As a user, I want my Secretary Intents to read information only, so that asking a question never creates or changes a Task.
24. As an Owner or Admin, I want to define a Routine Template with a responsible person and weekly, monthly, or quarterly cadence, so that recurring work can be proposed on time.
25. As the authorized responsible person, I want a due routine to appear as one Routine Suggestion, so that I can confirm whether a Task should be created.
26. As an authorized human, I want to confirm a Routine Suggestion before it becomes a Task, so that the system never creates recurring Tasks on its own.
27. As an Owner or Admin, I want a previous unfinished routine occurrence to block a duplicate suggestion, so that missed periods do not build a backlog.
28. As a user, I want routine timing based on my Organization Timezone, so that weekly, monthly, and quarterly boundaries are consistent.
29. As an authorized checklist user, I want a document called missing only after its human-set due date passes in the Client Organization timezone and no authorized Document is linked by a human, so that OCR guesses do not trigger false follow-ups.
30. As a user, I want completing a contact follow-up Task to leave the Expected Document Item open until a human links a Document, so that communication is not mistaken for receipt.
31. As a user, I want meeting actions to remain ordinary Tasks with human-entered meeting source details, so that Task permissions, due dates, and completion behavior remain familiar.
32. As a user, I want to show or hide briefing categories for my own Membership, so that I can focus my view without altering another person's briefing.
33. As a user, I want to pin an authorized action within its due group, so that I can adjust my personal view without changing Task Priority or moving overdue work behind later work.
34. As a user, I want an inaccessible pin to disappear, so that a preference cannot preserve access to a revoked item.
35. As a firm staff member in a later firm view, I want only clients allowed by my current firm Membership, Client Assignment, Firm Access Grant, scope, and record rights, so that the portfolio briefing respects each client's consent.
36. As a firm user in a later firm view, I want the firm timezone to define the briefing day while each client timezone defines that client's document due date, so that deadlines retain their local meaning.
37. As a user, I want the briefing available inside the app without unsolicited LINE or email delivery, so that I control when I view it.
38. As a user, I want operational explanations without financial, tax, investment, or payment recommendations, so that I do not confuse task triage with professional advice.
39. As a product operator, I want generation latency, queue time, candidate count, and cost measured without recording source text, so that the pilot can be evaluated safely.
40. As a product operator, I want the capability off by default and enabled only for named pilot Organizations after its gates pass, so that an unproven feature is not sold or exposed broadly.

## Implementation Decisions

### Product and source boundaries

- Keep Secretary Briefing separate from the synchronous, uncached Phase 2 Assistant Summary. Reuse its authorized Work Dashboard facts where useful without changing that endpoint's response, latency, or storage contract.
- The company pilot uses currently authorized Tasks, including ordinary contact follow-up Tasks and human-recorded meeting action Tasks. It may use Routine Suggestions once their human-confirmation flow exists. Do not present Phase 10 Expected Document Items or the firm daily briefing as live sources until those capabilities are implemented and authorized.
- Use only predefined read-only Secretary Intents. An answer must identify its source and distinguish an unavailable or unconfigured source from a source with zero matching records. No free-form command execution or data mutation is attached to an intent.
- A meeting action is an ordinary Task with human-entered meeting name and meeting date as provenance. Do not introduce a Meeting entity for the first version.
- An Expected Document Item is missing only after its due day has passed in the Client Organization timezone and it has no human-linked, currently authorized Document. Contact follow-up remains an ordinary Task; completing it does not complete the checklist item.
- Owner/Admin manage Organization-scoped Routine Templates with a responsible person and weekly, monthly, or quarterly cadence. A due template yields one Routine Suggestion that an authorized human must confirm before Task creation. Do not backfill many missed periods, overlap an unfinished occurrence, or shift dates for holidays in the first version.

### Ranking and presentation

- Gather all eligible candidates under current authority, then rank deterministically: overdue before due today, Task Priority within each due group, and a stable tie-breaker. The “today” top actions use those two groups; other authorized work remains reachable through the full work list. The ranking and remaining count do not depend on LLM wording or its context limit.
- Show no more than five top actions, with a concise source-grounded reason and current deep link. Separate personal actions from authorized team blockers for Owner/Admin. Do not score or rank employees.
- Keep category visibility and pins per Membership. A pin changes that user's display order only inside the same due group; it does not change source Priority, another user's view, or access. Hide a pin when its source is no longer authorized.
- The company day uses the Organization Timezone. For a later firm view, the day uses the Accounting Firm Organization timezone while Expected Document due dates use each Client Organization timezone.

### Snapshot lifecycle and authorization

- Generate a Briefing Snapshot asynchronously on the first open for a user, Organization context, and local day. Keep at most one active generation for that key; duplicate opens return the current job or snapshot. The snapshot is not a permanent report or an authorization token.
- Recheck current authority before gathering, before publishing, and before every display. Each item keeps enough source identity and scope to reauthorize it. If authority changes during generation, discard the result. Filter revoked items before display only when the remaining content and prose can be proved safe; otherwise discard the snapshot and regenerate. Every deep link performs its own current record authorization.
- A nonsecurity Task change may leave a time-stamped snapshot marked possibly stale until refresh. An authorization change never permits stale protected content to display. Refresh is user-initiated; limit normal manual refresh to once per 15 minutes, while regenerating an invalidated snapshot is exempt.
- Retain a snapshot only for its local day, provide no history view, and delete it within 24 hours after that day ends. Do not add briefing Notifications or outbound LINE/email delivery in the first version.
- Use the existing Durable Job pattern for bounded claim, lease, retry, and stale-job recovery. The worker uses the internal API and does not gain database credentials. Publish the valid deterministic result before attempting optional wording, so a failed wording stage cannot erase the briefing; retry wording only within a bounded policy.

### Optional wording and privacy

- Deterministic gathering produces the initial usable briefing. Design an optional compact structured wording contract containing currently authorized facts and only the minimum short Task titles needed for readable prose. A future LLM may explain the fixed ranking but may not choose, reorder, create, approve, or act on items.
- Send at most the first 20 ranked candidates and 2,000 tokens to a future LLM. Do not send original files, full OCR text, payment data, secrets, or unrelated client content. The top five, remaining count, and links still come from the complete deterministic candidate set.
- External LLM transmission stays disabled until a named provider, privacy and retention terms, and a numeric spend ceiling are approved. Treat source titles and future model output as untrusted display text. Exclude titles, source text, prompts, and briefing prose from logs and metric labels.
- Briefing content may direct a user to an existing operational source record. It must not evaluate financial position, recommend spending, investment, lending, tax treatment, payment decisions, or accounting actions. Financial and payment values are not briefing inputs in this phase.

### Performance and activation

- Provisional pilot targets are cached-read p95 at most 500 ms and fresh-briefing p95 at most 30 seconds under a recorded representative workload. Measure gathering, queue, and optional wording time, candidate count, retries, token use, and cost per briefing. These are acceptance targets, not claims about current performance; provider cost is zero while external calls are disabled.
- Keep the Phase 13 gate off by default. Pilot only with named test Organizations after current-rights, privacy, failure, and performance evidence. Do not present this feature as an active paid-plan entitlement until release and commercial decisions are made.
- A later firm view requires implemented Phase 10 checklist/review sources plus a separate delegated-access acceptance matrix. Firm Plan entitlement alone never grants client data access; grant, Client Assignment, firm Membership, scope, and record rights all apply. Aggregate gathering must avoid one request or query per client.

## Testing Decisions

- Prefer one high-level authenticated API journey as the primary seam: open briefing → observe queued or ready state → inspect ranked snapshot and intent answers → refresh → change source data or authority → observe current safe result. Assert public behavior, persisted job/snapshot state, and absence of unintended Task, message, or approval effects. The existing Work Dashboard/Assistant Summary route tests and Durable Job HTTP/database tests are prior art; do not change the Assistant Summary contract to test Phase 13.
- Use database-backed integration where concurrency or tenant policy matters. Cover duplicate opens, one active generation, refresh cooldown and revocation exemption, bounded retries, stale lease, day rollover, deletion deadline, and a Membership or delegated grant change while a job is running. Verify a deterministic snapshot remains usable when optional wording fails.
- Exercise Member, Owner, and Admin across at least two Organizations. Remove a role during generation and after caching; try cross-tenant IDs and revoked deep links. Confirm team blockers, category preferences, and pins cannot reveal another user's records. Test hostile Task titles and mixed-source prose so a revoked source cannot survive in an explanation or log.
- Test overdue and due-today ranking, Priority and stable ties, top-five truncation, remaining count, full-list navigation, timezone boundaries, and the possibly-stale marker. Check that an unconfigured checklist yields an unavailable-source answer rather than “zero missing.” Verify the five predefined intents against authorized sources.
- Test weekly, monthly, and quarterly Routine Suggestions across local day boundaries, confirmation before Task creation, duplicate suppression, skipped periods, and unfinished occurrences. Test human-entered meeting provenance and current Task access. When Phase 10 checklist sources exist, test human Document linking, client-local due dates, and the independence of contact Task completion from checklist completion.
- Run a small browser journey at desktop and narrow widths for first open, queued/fallback/ready states, source navigation, refresh, and access-revoked recovery. Check that status and source labels are understandable and accessible.
- For the later firm view, use a separate authenticated matrix with two firms and assigned versus unassigned staff, grant expiry/revocation, scope changes, multiple clients and client-local due dates. Confirm no client-by-client query growth and no cross-client leakage before enabling that view.
- Before any pilot activation, record the exact build and environment, representative workload, cached and fresh p95, queue depth, retries, candidate counts, cost, authorization results, and rollback evidence. Local tests or a ready endpoint alone do not establish a passing release gate.

## Out of Scope

- Financial, tax, investment, lending, payment, accounting, or legal advice; forecasts, optimization, suitability judgments, or automated financial decisions.
- Autonomous Task creation, follow-up delivery, approval, record edits, or fulfillment. The sole Task creation path from a Routine Suggestion requires explicit human confirmation.
- Free-form agent commands, autonomous chat, employee rankings, productivity scores, and changing another user's work order.
- Calendar or transcript ingestion, a Meeting entity, OCR-based missing-document inference, automatic Document linking, and a separate Follow-up state.
- Automatic LINE/email briefing, a Notification Digest, and user opt-in delivery controls. A later outbound option needs its own recipient, channel, local time, quiet-hours, and at-most-once-per-day decision.
- Live external LLM requests, provider-specific behavior, a numeric spend ceiling, or paid-plan packaging before those decisions are approved.
- Activating the firm view before Phase 10 sources and delegated authorization are implemented and proven; changing Phase 2 Assistant Summary behavior.

## Further Notes

- The [Phase 13 decisions](./phase-13-decisions.md) mark the company-pilot closeout as proposed rather than confirmed. This spec adopts its one-active-generation, 15-minute normal refresh limit, pin ordering, source-safe filtering, and acceptance matrix as **provisional pilot defaults**. Confirm these before activation; they are not evidence that implementation or release has happened.
- The current [Phase 12 gate status](./runbooks/phase-12-gate-status.md) has no release candidate and open security, privacy, backup, and performance gates. Phase 13 does not inherit a pass from local builds or earlier staging checks.
- Named LLM provider, privacy/retention approval, numeric spend ceiling, outbound delivery, firm per-client behavior, and commercial entitlement remain later decisions. This document creates no implementation ticket or deployment authorization.
