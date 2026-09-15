# Phase 0: Repository Foundation, LINE Bootstrap และ Event Skeleton

## Problem Statement

PlaiFlow ยังไม่มี application repository ที่สามารถพัฒนา ทดสอบ และ deploy ได้ จึงยังไม่มีฐานร่วมสำหรับ Web Dashboard, Go API, Python service ในอนาคต, PostgreSQL migrations, LINE webhook หรือ operational controls เช่น logging, request IDs, health checks และ performance measurements หากเริ่มสร้าง business features ตอนนี้ แต่ละส่วนจะกำหนด boundary, configuration และ deployment convention ของตนเอง ทำให้แก้ย้อนหลังยากและเสี่ยงต่อข้อมูลซ้ำ secret รั่ว หรือ webhook ทำงานช้า

## Solution

สร้าง monorepo ขั้นต่ำที่มี Next.js Dashboard, Go API และ worker, Python export-service shell และ PostgreSQL สำหรับ local development วาง LINE webhook ที่ตรวจ signature จาก raw request ก่อนบันทึก Inbound Event แบบ idempotent แล้วตอบกลับเร็ว งานหลัง webhook ถูกประมวลผลโดย worker แยกต่างหาก ระบบมี migrations, structured logs, request IDs, health/readiness, Docker Compose, CI, staging configuration และ tests ที่พิสูจน์ behavior จาก boundary ภายนอก

Phase นี้สร้างเฉพาะ shell สำหรับ Domain Event, export service และ auth provider เพื่อกำหนดทิศทางโดยไม่สร้าง business export, event bus, authentication จริง หรือ provider implementation

## User Stories

1. As a PlaiFlow developer, I want one repository containing every Phase 0 runtime, so that related changes can be reviewed and tested together.
2. As a PlaiFlow developer, I want each runtime to have a clear ownership boundary, so that frontend, API, worker and future Python work do not become coupled accidentally.
3. As a frontend developer, I want a bootstrapped Next.js application, so that Dashboard work starts from an agreed stack and design system.
4. As a frontend developer, I want shadcn/ui primitives owned by the application, so that components can be adapted without introducing a wrapper layer for every primitive.
5. As a Thai operator, I want Thai-first typography and readable responsive layouts, so that the Dashboard is comfortable to use on desktop and mobile.
6. As a keyboard or screen-reader user, I want visible focus, semantic navigation and accessible state feedback, so that the Dashboard is operable without relying on color or pointer input.
7. As an operator, I want to see counts of recently received, processed, ignored, retryable and failed Inbound Events, so that I can assess the pipeline quickly.
8. As an operator, I want loading, empty, error and retry states, so that the Dashboard never presents an unexplained blank screen.
9. As an operator, I want to see API, database and worker status, so that I can distinguish service failure from an empty event stream.
10. As a backend developer, I want a Go API based on standard library boundaries, so that Phase 0 has few dependencies and remains easy to trace.
11. As a backend developer, I want API and worker executables to be separate, so that background work cannot increase webhook response latency.
12. As a future export-service developer, I want a minimal Python project shell and versioned interface contract, so that a later implementation has an explicit boundary without business logic being invented now.
13. As a database developer, I want PostgreSQL available locally through Docker Compose, so that migrations and integration tests run against the production database engine.
14. As a database operator, I want ordered up/down migrations, so that schema changes are repeatable and rollback can be tested.
15. As a deployer, I want migrations run once before new API and worker versions, so that application instances do not race to mutate schema at startup.
16. As a LINE platform integrator, I want a valid empty verification delivery to receive HTTP 200, so that the webhook URL can be verified without creating work.
17. As a security owner, I want every LINE Webhook Delivery verified using its unmodified body and channel secret, so that spoofed or modified requests are rejected.
18. As a security owner, I want malformed and oversized requests rejected safely, so that they cannot consume unbounded resources or expose internal errors.
19. As a system operator, I want the webhook to acknowledge only after durable storage, so that an accepted event is not lost before asynchronous processing.
20. As a system operator, I want webhook processing to remain short, so that slow business work never runs in the inbound HTTP request.
21. As a system operator, I want duplicate LINE events suppressed by a database constraint, so that redelivery does not create duplicate work.
22. As a system operator, I want all events from one Webhook Delivery committed atomically, so that a partially persisted delivery cannot occur.
23. As a worker, I want leased database-backed work, so that work can be reclaimed after a process crash.
24. As a worker, I want bounded retry intervals and a terminal Failed state, so that permanent errors do not retry forever.
25. As an operator, I want unsupported event types marked Ignored, so that forward-compatible inputs do not look like failures.
26. As a privacy owner, I want event payloads and metadata retained only for agreed periods, so that staging does not accumulate unnecessary personal data.
27. As a future domain developer, I want a documented Domain Event envelope, so that the first real business event follows a stable vocabulary.
28. As a future domain developer, I do not want an unused event bus or publisher, so that Phase 0 does not carry speculative infrastructure.
29. As a future authentication developer, I want provider-neutral callback and redirect seams for LINE and Google, so that external identity can be added without redefining route ownership.
30. As a future authentication developer, I want PlaiFlow User identity kept separate from External Identity, so that one User can later connect multiple providers.
31. As a security owner, I want provider shells without login, sessions or token storage, so that an incomplete authentication system cannot be mistaken for production security.
32. As a developer, I want configuration loaded from environment variables and validated at startup, so that missing configuration fails clearly.
33. As a security owner, I want committed environment examples without values, so that secret names are documented without committing secrets.
34. As a security owner, I want server-only service credentials excluded from browser bundles, so that Dashboard-to-API trust remains confidential.
35. As an operator, I want structured JSON logs with request IDs and timings, so that one request can be traced across Next.js and Go.
36. As a privacy owner, I want logs to redact secrets, signatures, payloads, message text and full external identifiers, so that diagnostics do not leak sensitive data.
37. As an infrastructure operator, I want liveness and readiness endpoints with different meanings, so that process failure is not confused with dependency unavailability.
38. As a performance owner, I want webhook, API, query and queue-age baselines, so that regressions are visible before business features arrive.
39. As a developer, I want CI to verify every runtime and migration, so that broken foundation changes cannot merge unnoticed.
40. As a deployer, I want staging deployment ordered and environment-specific, so that staging cannot reuse future production resources or secrets.
41. As a product owner, I want Phase 0 staging protected by the hosting platform, so that operational data is not public before PlaiFlow authentication exists.
42. As a tester, I want a signed synthetic LINE message to travel from HTTP receipt to Processed state, so that the full inbound pipeline is proven without using customer data.
43. As a maintainer, I want a runbook for LINE staging and rollback, so that external setup can be repeated without relying on memory.

## Implementation Decisions

### Repository layout and toolchains

- Use one repository with four top-level runtime/documentation areas: `web`, `api`, `python` and `docs`.
- The web area owns the Next.js application and UI components. The API area owns the Go API, worker, PostgreSQL access and migrations. The Python area is a buildable service shell for the future export boundary. The docs area owns ADRs, runbooks and architecture documentation.
- Do not add a monorepo orchestration framework. Each runtime retains its native commands and the root documentation lists the small set of commands needed to run and verify the repository.
- Select stable toolchain versions at implementation time and pin Go in the module declaration, Node in the repository version file, pnpm in package metadata and Python in its project metadata. Lock dependency versions.

### Next.js bootstrap and Dashboard shell

- Bootstrap Next.js with App Router, TypeScript, Tailwind and shadcn/ui using pnpm.
- Reuse the accepted PlaiFlow Design System: semantic OKLCH tokens, teal primary, warm-orange accent, `Noto Sans Thai` self-hosted for Thai and Latin, light/dark values selected by system preference and reduced-motion support.
- Keep shadcn primitives in the application-owned UI component area and modify them directly. Product components remain near the feature that owns them; do not introduce generic wrappers or a separate component package.
- Use a permanent sidebar at large breakpoints and a top bar with Sheet navigation below that breakpoint. Only Overview and System Status appear in Phase 0 navigation.
- Overview displays real 24-hour counts for Received, Processed, Ignored, Retryable and Failed Inbound Events. System Status displays API, database and worker heartbeat status.
- Provide stable skeleton loading, actionable empty states, local error recovery and route-level error handling. Toasts communicate completed actions only.
- Browser traffic remains same-origin with Next.js. Server-side Next.js code calls the Go API using a server-only service token. The browser never receives this token and does not call Railway directly.
- The authentication-result page may exist as a visual shell, but it must not imply that login is available.

### Go API and worker bootstrap

- Use one Go module with separate server and worker commands.
- Organize internal code by capability: configuration, HTTP boundary, LINE adapter, Inbound Event lifecycle and PostgreSQL storage. Do not add a public package tree, generic repository abstraction, ORM, dependency-injection framework or web framework.
- Use `net/http`, `http.ServeMux`, `log/slog`, Go cryptography packages and `pgx/v5`.
- Apply graceful shutdown to API and worker. HTTP defaults are read-header 5 seconds, body read 5 seconds, write 10 seconds and idle 60 seconds. Webhook handling has a 3-second deadline.
- Default PostgreSQL pools are 10 connections for the API and 5 for the worker, with environment overrides retained as deployment calibration knobs.
- Application endpoints use a `/v1` prefix. LINE webhook and operational endpoints remain outside versioned application routes.
- Safe JSON errors contain a stable error code, user-safe message and request ID. Internal error text, stack traces, SQL details and signature diagnostics are never returned.

### Python shell and export service interface shell

- Create a minimal Python 3 project that imports, runs a self-check and passes standard-library unit tests without requiring a web framework.
- Define a versioned, transport-neutral export-service boundary with one asynchronous submission operation returning an opaque job reference. Keep request payload content opaque at this phase so document, OCR, accounting and rendering semantics are not designed prematurely.
- Provide contract serialization fixtures that can later be shared with a Go client, but do not expose a live export HTTP route, start a persistent Python service or deploy the Python shell to staging.
- CI verifies the Python package imports, compiles and passes its contract tests. A container build target may be verified, but no runtime resources or secrets are provisioned for it.
- This explicit shell narrowly supersedes the prior decision to create no export package; the prohibition on business export implementation, notification infrastructure and unused event-bus infrastructure remains.

### PostgreSQL local environment and migrations

- Docker Compose starts PostgreSQL with a health check, named development volume and non-production local credentials. Developers run web, API, worker and Python checks natively; Compose is not an application orchestrator.
- Use `golang-migrate` with pinned tooling and ordered up/down SQL migrations. Applications never migrate schema during startup.
- The minimum application schema contains `inbound_events` and `worker_heartbeats`; no Webhook Delivery, Domain Event, Notification, User or External Identity tables are created.
- Inbound Event rows use database-generated bigint internal IDs, provider/channel identifiers, the provider event ID, event type, nullable JSON payload, lifecycle status, attempt count, availability/lease timestamps, occurrence/receipt/processing timestamps and safe failure metadata.
- Enforce uniqueness on provider, channel and provider event ID. Store all database timestamps as `timestamptz` in UTC.
- Payload JSON is removed after 30 days. Metadata and idempotency records are removed after 90 days. The worker performs cleanup once daily; no separate scheduler is introduced.
- Worker heartbeat updates every 30 seconds. A heartbeat older than two minutes is unhealthy for Dashboard purposes.

### LINE webhook and signature verification

- Expose one public POST webhook for the single configured LINE staging channel.
- Limit the body to 1 MiB and preserve the received bytes unchanged until signature verification completes.
- Compute HMAC-SHA256 with the channel secret, Base64-encode the digest and compare it with `x-line-signature` using constant-time comparison.
- Reject missing or invalid signatures without parsing or persisting the body. Reject malformed JSON, missing provider event IDs and oversized bodies with safe non-2xx responses.
- Accept unknown JSON fields and valid future event types. A valid delivery containing no events returns HTTP 200 without creating database work.
- Persist every Inbound Event from one delivery in a single transaction and return HTTP 200 only after commit. Duplicate inserts are successful no-ops from the webhook caller's perspective.
- The HTTP boundary does signature verification and delivery validation. The LINE adapter converts each provider object into an Inbound Event representation. PostgreSQL owns the atomic idempotency guarantee.
- Do not use IP allowlisting for LINE. Do not log the signature, raw delivery, event payload, message text, channel secret or complete external identifiers.
- Use the standard library implementation rather than the LINE SDK until an outbound Messaging API capability actually requires that SDK.

### Inbound Event worker and idempotency seam

- PostgreSQL is the queue. Claim at most 25 eligible events using row locking with skip-locked behavior and a 60-second lease. Start with one worker execution lane.
- Expired leases become eligible again. Do not promise global ordering; provider occurrence timestamps remain context rather than a processing lock.
- Retry transient failures after 1 minute, 5 minutes, 30 minutes, 2 hours and 12 hours. After the fifth failed attempt, move the event to Failed and require operator action for another attempt.
- In Phase 0, a valid LINE message event uses a record-only handler and becomes Processed without sending a reply or producing business data. Other valid event types become Ignored with a safe reason and are not retried.
- The primary end-to-end test seam is the public webhook boundary through PostgreSQL and worker processing. This is preferred over separate tests for every internal helper.

### Domain Event shell

- Define the Domain Event envelope contract with identifier, past-tense type, occurrence time, subject identifier, source Inbound Event identifier and opaque data.
- Provide validation and serialization tests for the envelope only.
- Do not create a publisher interface, event bus, outbox, Domain Event table, consumer registry or production event type in Phase 0.
- A raw LINE event or Webhook Delivery must never be described as a Domain Event.

### Auth provider abstraction and redirect seam

- Preserve a stable internal User concept that may later link multiple External Identities.
- Define a provider-neutral authentication boundary describing provider identity, authorization-start data and callback result without implementing LINE or Google behavior.
- Reserve Go API callback convention `/v1/auth/{provider}/callback` and web result convention `/auth/callback`.
- Centralize public web/API base URLs and allowed callback construction in validated server configuration so local, staging and future production redirects cannot be mixed accidentally.
- Do not expose a working provider route until an implementation is configured. Do not create login UI, sessions, User tables, External Identity tables, token exchange, token storage, PKCE/state flows or account linking in Phase 0.

### Configuration, secrets and environment separation

- Go configuration includes application environment, port, database URL, LINE channel identifier and secret, Dashboard service token, log level and database-pool overrides.
- Next.js server configuration includes the private API base URL and Dashboard service token. Neither value may use a public-client prefix.
- Python shell has no runtime secret in Phase 0.
- Commit environment examples containing names, safe defaults and descriptions only. Required server configuration fails fast at startup. Do not add a configuration library.
- Local and staging use different PostgreSQL databases, LINE channels, tokens, callback bases and deployment settings. Production names may be reserved, but production resources are not created.
- Secrets live only in environment-scoped platform stores. CI uses Gitleaks pinned to a commit SHA and accepts exclusions only for documented false positives.

### Logging, request IDs and performance metrics

- Emit structured JSON logs from Go using `slog`. Request completion records contain method, normalized route, status, duration and request ID.
- Public requests receive a newly generated cryptographically random request ID. A request ID is propagated from Next.js only when the server-to-server service token is valid. Return the effective request ID in every API response.
- Redact authorization values, service tokens, LINE secrets/signatures, raw bodies, message text, email addresses and complete external identifiers. Errors carry stable classifications rather than raw internal messages.
- Log database queries exceeding 200 ms without logging parameters that may contain sensitive data.
- Establish staging targets: webhook p95 below 500 ms, read API p95 below 300 ms and oldest eligible queue item below 60 seconds.
- Derive the initial backend measurements from structured timing logs and Railway metrics. Use Next.js production build output, Lighthouse and Vercel metrics for frontend baselines. Do not add Prometheus, OpenTelemetry, analytics SDKs or real-user monitoring in Phase 0.

### Health and readiness

- Liveness reports only whether the process can serve requests. It must not call PostgreSQL, LINE or other external systems.
- API readiness verifies PostgreSQL connectivity and expected migration version. It must not call LINE.
- Worker readiness includes PostgreSQL connectivity and worker startup state. Dashboard worker status comes from the persisted heartbeat and becomes unhealthy after two minutes.
- Responses are minimal, contain no configuration values and use appropriate success/non-success status codes for orchestration.

### CI and staging deployment configuration

- Pull requests run independent web, Go, Python, migration and secret-scanning checks. Required checks include formatting, type/lint checks appropriate to each runtime, tests, production builds and migration validation.
- CI uses a PostgreSQL service container and a test-only database URL. Local integration tests use the Docker Compose PostgreSQL instance.
- Frontend behavior is tested at the highest practical seam with a small browser test covering shell navigation and loading/empty/error rendering. Avoid duplicating these assertions in component-level suites.
- Vercel produces web previews for pull requests. A dedicated Vercel staging project is protected using platform deployment protection.
- Railway hosts staging PostgreSQL, Go API and worker in a staging environment. Disable Railway auto-deploy; GitHub Actions performs migration, API deployment, worker deployment and smoke testing in that order.
- The Python shell is build-verified only and is not deployed during Phase 0.
- A staging smoke test checks liveness/readiness, safe rejection, empty LINE verification delivery, signed synthetic message ingestion, duplicate suppression, worker transition to Processed, request-ID propagation and Dashboard real-state rendering.
- Creating cloud resources, storing secrets, enabling LINE webhook/redelivery or sending a synthetic staging webhook requires explicit approval at implementation time.

## Testing Decisions

- Good tests observe externally meaningful behavior at the highest stable seam: HTTP status/body/headers, committed database state, visible Dashboard state, process readiness or serialized boundary contracts. Tests must not assert private helper calls or internal struct layout unless that layout is itself a published contract.
- The principal backend integration test sends a signed Webhook Delivery through the real HTTP handler into a real PostgreSQL test database, executes one worker cycle and verifies the final Inbound Event state. Signature, idempotency and lifecycle behavior should not be split across redundant mock-heavy suites.
- Signature tests use LINE's published verification vector plus invalid, missing and body-mutated cases. The received raw bytes must be the test input.
- Malformed-request coverage includes oversized body, invalid JSON, missing signature, malformed Base64 signature, missing event ID, empty events and mixed duplicate/new events.
- PostgreSQL tests verify migration up/down, unique idempotency enforcement, atomic multi-event persistence, skip-locked claiming, expired-lease recovery, retry timing, terminal failure, heartbeat freshness and retention cleanup.
- HTTP tests verify safe error envelopes, request-ID generation/propagation, body/time limits, liveness independence and readiness dependency behavior.
- Dashboard browser tests verify responsive navigation, real aggregate/status rendering, loading, empty, error and retry behavior, keyboard focus and absence of future-feature navigation.
- Python standard-library tests verify importability, contract versioning, asynchronous submission-envelope serialization and rejection of malformed contract data. No export output is tested.
- Domain Event shell tests verify envelope validation and round-trip serialization only; they do not invent business event types.
- Auth shell tests verify provider-neutral callback naming and environment-specific redirect construction without contacting LINE, Google or storing tokens.
- CI build checks verify no server secret is present in the client bundle and Gitleaks finds no committed secret.
- Staging smoke tests use synthetic data only. Automated tests must never call a production LINE channel or perform mass/load testing through LINE.
- There is no comparable application test suite in the repository yet; Phase 0 establishes these testing conventions as the prior art for subsequent work.

## Out of Scope

- Business export generation, document rendering, file download, export storage and export status UI
- Document management, OCR, accounting workflows and AI advisor capabilities
- Real LINE Login or Google Login, session management, account linking, User persistence and External Identity persistence
- Outbound LINE messaging and notification abstractions
- Domain Event persistence, publishing, event bus, outbox and business event handlers
- Multiple LINE channels or channel-management UI
- Production environment, production domain, production LINE channel and customer data
- Message broker, distributed scheduler, ORM, web framework, DI framework and monorepo orchestration framework
- Custom permanent logo, mascot or brand identity program
- Analytics SDK, real-user monitoring, Prometheus and OpenTelemetry
- Business-level authorization and role management

## Further Notes

- This specification uses the canonical vocabulary in the project glossary: Webhook Delivery, Inbound Event, Domain Event, User, External Identity and the event lifecycle states.
- The Python, Domain Event, export and auth items are deliberately narrow shells. They establish compile/test-time boundaries but must not grow into speculative implementation during Phase 0.
- The export interface shell requested here supersedes only the “no export package” portion of the earlier deferral ADR. If implementation is approved, that ADR should be amended before code is written; the ban on business export implementation remains.
- The highest-value test seam is the end-to-end inbound boundary: signed HTTP request to durable idempotent storage to one worker cycle. New lower-level seams should be introduced only when failure cases cannot be exercised reliably through this path.
- The repository currently has no Git metadata or configured issue tracker. This spec is written locally only and is not published or labeled.
