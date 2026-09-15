Status: ready-for-agent

# Phase 1: Auth, Multi-Tenancy, RBAC และ LINE Identity/Group Linking

## Problem Statement

ผู้ใช้ PlaiFlow ยังไม่สามารถเข้าสู่ระบบด้วย LINE หรือ Google, รักษา session อย่างปลอดภัย, แยกตัวตนภายในออกจาก provider identity, สร้างหรือเข้าร่วม Organization, สลับ tenant, เชิญสมาชิก หรือควบคุมสิทธิ์ตาม role ได้ ระบบจึงยังไม่สามารถเปิด Dashboard ให้ผู้ใช้จริงโดยรับประกันว่า User จะไม่เห็นหรือแก้ข้อมูลข้าม Organization

ในเวลาเดียวกัน LINE Login identity ยังไม่ได้เชื่อมกับ LINE user ที่ส่งข้อความผ่าน Messaging API และยังไม่มีกระบวนการที่พิสูจน์ได้สำหรับเชื่อม LINE group เข้ากับ Organization การ merge identity จาก email, การเชื่อ provider callback หรือ organization identifier จาก browser, และ link code ที่ใช้ซ้ำได้ล้วนเปิดทางให้เกิด account takeover หรือ cross-tenant disclosure

## Solution

เพิ่ม LINE Login และ Google Login ด้วย OIDC Authorization Code Flow บน same-origin boundary โดย Go เป็นเจ้าของ OAuth transaction, provider validation, AuthIdentity, session, tenant resolution, RBAC และ security audit ส่วน Next.js แสดง UI ที่ mobile-first และไม่รับ provider token

PlaiFlow จะมี User ภายในที่คงที่และเชื่อมกับ External Identity ผ่าน record ชื่อ `AuthIdentity` โดยใช้ `(issuer, subject)` เป็นกุญแจ provider ห้ามรวม User จาก email อัตโนมัติ การ link หรือ unlink provider ต้องเริ่มจาก session ที่ยืนยันแล้วและใช้ recent authentication

User เข้าร่วม Organization ผ่าน Membership ที่มี role Owner, Admin หรือ Member ทุก tenant request สร้าง Organization Context ฝั่ง server จาก session และ Membership แล้วบังคับซ้ำด้วย PostgreSQL Row-Level Security การเชิญใช้ capability link ที่สุ่ม, เก็บเฉพาะ hash, ใช้ครั้งเดียว และให้สิทธิ์เริ่มต้นเป็น Member เท่านั้น

LINE Group Connection ใช้ one-time link code ที่ผูกกับ Organization, ผู้สร้าง และ LINE subject ผู้สร้างต้องส่ง code จาก LINE group ด้วย User เดียวกับ LINE AuthIdentity ที่เริ่มรายการ ระบบตรวจ webhook signature, source group/user, role, expiry และ replay ก่อนสร้าง connection แบบ atomic โดยไม่ถือว่าผู้ส่งเป็น LINE group admin

## User Stories

1. As a new mobile user, I want to sign in with LINE, so that I can start from the provider I already use.
2. As a new user, I want to sign in with Google, so that I can use an existing Google identity without creating a password.
3. As a user arriving from LINE, I want LINE Login to be the primary action, so that the most natural path is immediately visible.
4. As a user inside an embedded LINE browser, I want clear guidance when Google requires the system browser, so that I can recover without assuming the login is broken.
5. As a security-conscious user, I want provider authorization to occur on the provider's own page, so that PlaiFlow never asks for my provider password.
6. As a returning user, I want the same provider identity to return me to the same User, so that repeated login does not duplicate my profile.
7. As a user who denied provider consent, I want a safe explanation and retry action, so that I can recover without exposing protocol details.
8. As a user whose login transaction expired, I want to restart safely, so that an old callback cannot authenticate me.
9. As a user, I want login to return me only to an allowed PlaiFlow destination, so that provider callbacks cannot become open redirects.
10. As a LINE-first user, I want to link Google from my authenticated account, so that I have another login method.
11. As a Google-first user, I want to link LINE from my authenticated account, so that I can connect LINE groups later.
12. As a user, I want linking to require recent authentication, so that a stolen unattended session cannot add an attacker's provider identity.
13. As a user, I want a clear conflict state when a provider identity belongs to another User, so that PlaiFlow never merges accounts silently.
14. As a user, I want matching provider emails treated as profile data rather than proof of identity, so that recycled or differently trusted email addresses cannot take over my User.
15. As a user, I want to unlink a provider only when another login method remains, so that I cannot lock myself out accidentally.
16. As a user, I want unlinking to require authentication with the identity that will remain, so that control of the identity being removed is insufficient.
17. As a user, I want an accidentally unlinked identity recoverable for a short period, so that it does not immediately create a duplicate User.
18. As a user, I want all other sessions revoked after unlinking, so that a removed provider cannot leave unknown sessions active.
19. As a new self-signup user, I want to create my first Organization, so that my business data has an explicit tenant boundary.
20. As an invited user, I want to join the inviting Organization without creating an unwanted personal Organization, so that onboarding stays short.
21. As a multi-Organization user, I want to see the Organizations I belong to in one query-backed chooser, so that switching remains fast.
22. As a multi-Organization user, I want Organization identity represented in the URL, so that navigation and deep links are predictable.
23. As a user, I want switching Organization to navigate without rewriting my session, so that tenant state cannot become stale.
24. As a security owner, I want every requested Organization checked against current Membership server-side, so that changing a URL cannot grant access.
25. As a removed member, I want access to stop on the next request, so that a long-lived session does not retain Organization rights.
26. As a user without access, I want a non-enumerating response, so that private Organization existence is not disclosed.
27. As a developer, I want every tenant-owned row constrained to one Organization, so that accidental cross-tenant relationships fail at the database boundary.
28. As an Organization Owner, I want exactly one Owner at a time, so that final responsibility is unambiguous.
29. As an Owner, I want to transfer ownership atomically, so that an Organization never has zero or multiple Owners.
30. As an Owner, I want to promote or demote Admins, so that privileged access remains under owner control.
31. As an Owner, I want to invite and remove Members, so that I can control Organization access.
32. As an Admin, I want to invite and remove Members, so that routine member administration does not require the Owner.
33. As an Admin, I want to manage LINE Group Connections, so that operational setup can be delegated safely.
34. As an Admin, I do not want the ability to manage Owner or other Admin privileges, so that delegated administration has a bounded blast radius.
35. As a Member, I want to use Organization features without member-management powers, so that least privilege is the default.
36. As an Owner or Admin, I want every invitation to grant Member only, so that a forwarded invite cannot grant administrative power.
37. As an inviter, I want to revoke an unused invitation, so that a link sent to the wrong person can be invalidated.
38. As an invitee, I want to inspect the Organization and inviter before accepting, so that I understand which tenant I am joining.
39. As an invitee, I want an invitation to survive the OAuth round trip without leaving its secret in callback URLs, so that LINE-first onboarding is safe.
40. As an invitee, I want expired, revoked, consumed and malformed invitations handled safely, so that I receive a recovery path without token leakage.
41. As an existing member, I want accepting the same Organization invite to be idempotent, so that retries do not create duplicate Memberships.
42. As an Owner, I want my leave action blocked until ownership is transferred, so that an Organization cannot become ownerless.
43. As an Admin or Member, I want to leave an Organization, so that I can end access I no longer need.
44. As a security owner, I want pending privileged actions re-checking the actor's current role at execution time, so that a downgrade invalidates stale authority.
45. As a User, I want a secure server-side session, so that browser-visible tokens do not contain identity or role authority.
46. As a User, I want to sign out from the current device, so that I can end one session.
47. As a User, I want to sign out from all devices, so that I can recover from suspected session theft.
48. As a User, I want inactive and over-age sessions to expire, so that old cookies do not remain valid indefinitely.
49. As a User, I want successful login and re-authentication to rotate the session identifier, so that session fixation cannot survive authentication.
50. As a User, I want mutation requests protected from CSRF, so that another website cannot act through my session.
51. As a LINE-linked Owner or Admin, I want to request a group link code, so that I can connect a LINE group to the current Organization.
52. As a group linker, I want the code bound to my LINE AuthIdentity, so that another group member cannot reuse my code.
53. As a group linker, I want to paste the code from mobile LINE, so that linking does not require precise typing.
54. As a security owner, I want only a signed Messaging API event from a group accepted, so that direct HTTP requests cannot link a group.
55. As a security owner, I want the message sender's LINE user ID to match the expected LINE subject, so that possession of the code alone is insufficient.
56. As a security owner, I want a missing group ID or sender user ID to fail closed, so that incomplete webhook identity never falls back to bearer-code authorization.
57. As an Organization administrator, I want one LINE group connected to at most one Organization while allowing one Organization to connect multiple groups, so that group ownership is deterministic.
58. As an Organization administrator, I want used, expired, revoked and replayed link codes rejected, so that an old message cannot change a connection.
59. As a privacy owner, I want plaintext link codes removed before durable event storage and logs, so that only a one-way code hash remains.
60. As an Organization administrator, I want to disconnect a LINE group in PlaiFlow, so that later events from that group no longer enter tenant processing.
61. As an Organization administrator, I want a LINE `leave` webhook to mark the connection disconnected, so that Dashboard state reflects the bot leaving.
62. As a user, I want PlaiFlow to state that group linking does not prove LINE group-admin status, so that the authorization claim is not overstated.
63. As a system operator, I want unconnected-group messages discarded after signature verification unless they are valid link attempts, so that PlaiFlow does not retain unrelated conversations.
64. As a security operator, I want authentication, identity, membership, invitation and LINE-link changes audited, so that incidents can be reconstructed.
65. As a privacy owner, I want audit events free of tokens, codes and raw provider responses, so that the audit trail is not a credential store.
66. As a mobile user, I want every primary control large enough to tap and all screens free of horizontal scrolling, so that onboarding works inside LINE on a phone.
67. As a keyboard or screen-reader user, I want labelled provider controls, visible focus and announced errors, so that authentication is accessible.
68. As a user, I want each screen to present one clear primary action, so that login, Organization setup and linking do not feel overwhelming.
69. As a user, I want loading controls disabled against duplicate submission, so that repeated taps cannot create multiple transactions.
70. As a user, I want inline errors with a clear recovery path, so that auth failures are understandable without technical details.
71. As a security owner, I want LINE and Google configured separately for each environment, so that staging identities and secrets cannot cross into production.
72. As a privacy owner, I want Google Login to request no Drive permission, so that login consent is limited to identity.
73. As a Google user, I want Login to avoid offline access and refresh tokens, so that PlaiFlow stores no long-lived Google authorization credential.
74. As a performance owner, I want Membership resolution to use an index and a bounded query count, so that authorization cost does not grow with Organization size.
75. As a performance owner, I want Organization switching and member listing to avoid N+1 queries, so that large memberships remain usable.
76. As a database operator, I want cleanup indexes for expired sessions and one-time transactions, so that security-retention jobs do not scan unrelated rows.
77. As a tester, I want OAuth behavior tested against controlled provider doubles at the HTTP boundary, so that security behavior is deterministic without real credentials.
78. As a tester, I want cross-tenant attacks attempted against the real PostgreSQL policies, so that both Go and RLS boundaries are proven.
79. As a tester, I want concurrent invite and link-code redemption tested, so that one-time behavior survives races.
80. As a maintainer, I want the highest integration seam to cover callback, session, RBAC and tenant data, so that implementation refactors do not invalidate behavior tests.

## Implementation Decisions

### Scope and ownership boundaries

- Phase 1 implements real LINE Login and Google Login behavior, AuthIdentity linking/unlinking, server-side sessions, Organization/Membership/RBAC, invitations, Organization switching, LINE Group Connections, audit records and their UI states.
- Use a single public web origin. Next.js proxies same-origin auth/application requests to Go; Go owns OAuth transactions, sessions, CSRF validation, User/AuthIdentity records, tenant resolution, RBAC, link-code consumption and audit writes.
- Next.js renders pages and submits same-origin actions. It never exchanges provider authorization codes, validates ID tokens, resolves roles or receives provider access/refresh tokens.
- Prefer two high-level verification seams: OAuth start through callback to an authenticated tenant API, and signed LINE webhook through link-code consumption to a visible LINE Group Connection.
- External provider configuration, secrets, callback registration and staging activation are separate release gates and require explicit approval.

### Provider and AuthIdentity model

- `External Identity` remains the domain term. `AuthIdentity` is the implementation record that represents it; do not introduce a generic Account entity.
- Support exactly two configured providers in Phase 1: `line` and `google`. Use an explicit provider registry rather than a plugin framework.
- Each provider implementation owns authorization parameters, code exchange and provider-specific token validation behind the existing provider-neutral auth seam.
- Identify an AuthIdentity only by the canonical provider issuer and subject. Enforce global uniqueness of `(issuer, subject)`.
- Enforce at most one active AuthIdentity per provider for a User in Phase 1. A User may have one LINE and one Google AuthIdentity.
- Store normalized provider, issuer, subject, provider display name, provider avatar URL, optional email, provider-reported email trust, linked timestamp, last-authenticated timestamp and optional unlink timestamp. Do not store raw claims.
- User owns editable `display_name` and `avatar_url`. Seed them from the first provider only; later logins refresh provider attributes without overwriting User-owned fields.
- Email is nullable profile metadata, never a User key, uniqueness rule or automatic linking signal. Google `email_verified` and hosted-domain claims remain provider attributes. LINE email is not treated as verified ownership.
- If an unknown identity has an email matching any existing record, continue without automatic merge and without disclosing account existence to an unauthenticated caller.

### LINE Login

- Use OIDC Authorization Code Flow with the official LINE authorization and token endpoints.
- Request only `openid profile` in Phase 1. Do not request LINE email because it is unnecessary for identity and does not provide the trust semantics needed for merging.
- Use PKCE `S256`, a separate nonce and a one-time state for every login, link and re-authentication attempt.
- Exchange the code only in Go using the environment's LINE channel credentials and the exact registered redirect URI.
- Validate the ID token with LINE's official verify endpoint against the expected client ID and nonce. Fail closed on issuer, audience, expiry, nonce or verification failure.
- Use the canonical LINE issuer plus `sub` as the AuthIdentity key. Do not treat display name, email or picture as identity.
- Keep the LINE Login channel and Messaging API channel for one environment under the same LINE Provider so LINE Login `sub` can be compared with Messaging API `source.userId`.
- Use separate LINE Providers/channels for development, staging and production. A readiness check must reject incompatible or missing channel configuration before group linking is enabled.

### Google Login

- Use Google OIDC Authorization Code Flow and exact environment-specific HTTPS redirect URIs; localhost is the only HTTP exception.
- Request exactly `openid email profile`. Use online access, do not request a refresh token and omit `include_granted_scopes`.
- Use PKCE `S256`, separate nonce and one-time state for login, link and re-authentication.
- Validate Google ID tokens server-side using the official issuer metadata/JWKS, including signature, issuer, audience, expiry, nonce and authorized-party checks where applicable.
- Use Google issuer plus `sub` as the AuthIdentity key. Do not use email as the subject.
- When opened in a LINE embedded browser, the Google action explains that Google requires an external/system browser and provides a deliberate handoff. It must not attempt OAuth inside a prohibited embedded user-agent.
- A Google login opened in the system browser starts a new auth transaction in that browser. An invitation may cross this boundary only through a short-lived, single-use handoff derived from the already validated Member invitation; the handoff is exchanged immediately for a clean URL.
- Provider linking may not use a browser-handoff token as its only authority. When the browser context changes, the User must first authenticate as the existing PlaiFlow User in the destination browser and then start the link transaction there.
- Google Login configuration is isolated by environment and from any future Google Drive integration.
- Google Drive, Google Sheets and all non-identity scopes are forbidden in the Login authorization request.

### OAuth transaction and callback handling

- Generate independent 256-bit random `state` and `nonce` values plus a standards-compliant PKCE verifier for every attempt. Store only the state hash; use PKCE `S256` only.
- An auth transaction records provider, expected issuer, purpose `login|link|reauth`, nonce, PKCE verifier, safe relative return path, optional initiating User/session, creation time, ten-minute expiry and consumption state.
- Only allow relative return paths matching explicit application route prefixes. Never accept scheme-relative URLs, arbitrary origins or provider-supplied redirect destinations.
- Callback provider selection comes from the registered route and must match the provider stored in the transaction. Do not trust a query parameter to choose the issuer or token endpoint.
- Claim each transaction atomically before code exchange. Replayed, expired, missing, provider-mismatched and already-consumed states fail closed and require a new attempt.
- Authorization errors return to a clean same-origin page with a safe error code and request ID. The callback response never renders provider parameters and loads no analytics or third-party script.
- Authorization codes, state, nonce, PKCE verifier, ID tokens and access tokens are redacted from logs, error reports, browser storage and URLs after callback processing.
- Provider access and ID tokens are discarded after validation/profile extraction. Phase 1 stores no provider refresh token.

### Login, linking, duplicate identities and unlinking

- A successful login for an active AuthIdentity creates a fresh PlaiFlow session for its User.
- A successful login for a previously unseen `(issuer, subject)` creates a new User and AuthIdentity; it never attaches to an email-matching User.
- Provider linking can start only from an authenticated session whose recent-auth window is valid. The transaction binds the initiating User and session and cannot change target User at callback time.
- If the returned `(issuer, subject)` already belongs to another User, abort with an identity-conflict recovery state. Never reassign, merge or choose a User from email.
- Phase 1 does not implement an automatic User merge engine. Duplicate Users keep separate identities and Memberships; recovery guidance requires proving both accounts and consolidating access through normal invitation/ownership flows or an audited operator process.
- Unlinking requires another active AuthIdentity to remain and a re-authentication performed with the identity that will remain within ten minutes.
- Unlinking revokes every other PlaiFlow session, rotates the current session and disables the removed identity immediately.
- Retain a non-authenticating unlink tombstone for 30 days. During this window the identity cannot create another User and may only be relinked to its previous User after authentication through the remaining identity. Delete the provider identifier after the cooldown according to retention policy.
- Users with one AuthIdentity cannot unlink it. Owners with one provider receive a recovery warning but are not forced to link another provider.
- Phase 1 has no password, magic-link or recovery-question fallback. Loss of the only provider requires an explicitly authorized, independently verified and fully audited operator recovery process outside normal application endpoints.

### Session lifecycle and CSRF

- Generate a 256-bit opaque session token. Store only its cryptographic hash and deliver the token in a `__Host-` prefixed, host-only, `HttpOnly`, `Secure`, `SameSite=Lax`, path-root cookie.
- Session records contain User, creation/authentication/last-seen timestamps, idle and absolute expiry, revocation state and token hash. They contain no Organization or cached role authority.
- Idle expiry is 14 days and absolute lifetime is 30 days. Update `last_seen` at most once per five minutes.
- Sensitive actions require authentication within the previous ten minutes. Re-authentication rotates the session ID and preserves only an allowlisted pending action.
- Rotate the session identifier after login, re-authentication and AuthIdentity linking. Never upgrade a pre-authentication session in place.
- Logout revokes the current record and clears cookies. Logout-all revokes all User sessions before issuing no replacement session.
- Membership removal and role changes apply on the next request because authorization is resolved from current data rather than session claims.
- Protect every state-changing endpoint with SameSite cookies, exact same-origin validation and a session-bound CSRF token. OAuth state does not replace application CSRF protection.
- Reject malformed, missing or cross-session CSRF tokens with a generic response and no mutation.
- Expired/revoked session lookup must be indistinguishable from an unknown session to the client.

### Organization, Membership and RBAC

- Use immutable UUIDs for User and Organization identifiers. Organization creation and its initial Owner Membership occur in one transaction.
- A User may belong to multiple Organizations; an Organization may contain multiple Users through Membership.
- Store active Membership once per `(organization_id, user_id)` with one fixed role: Owner, Admin or Member.
- Enforce at most one Owner per Organization with a database constraint. Creation, transfer and deletion guards ensure at least one Owner remains.
- Owner may edit Organization details, invite/remove Members, promote/demote Admins, manage LINE Group Connections and transfer ownership.
- Admin may edit Organization details, invite/remove Members and manage LINE Group Connections. Admin cannot manage Admin/Owner roles or transfer ownership.
- Member may use tenant features but cannot manage Membership, invitations, roles, ownership or LINE Group Connections.
- Organization deletion, custom roles and permission editors are not implemented in Phase 1.
- Admin and Member may leave an Organization. Owner must transfer ownership before leaving.
- Ownership transfer locks the relevant Organization/Membership rows and changes old/new roles atomically. Concurrent transfers must produce one winner.
- Pending invitation, group-link and privileged actions re-check current Membership and role at execution time; authority is never frozen when a token is created.

### Tenant resolution and RLS

- Tenant UI routes use `/o/{organization_id}`. The identifier expresses requested context, not authorization.
- For every tenant request, Go resolves User from the session, loads one current Membership by `(user_id, organization_id)`, checks the required permission and creates Organization Context server-side.
- Non-members receive a generic not-found-style response that does not reveal Organization existence, member count or name.
- Organization switching navigates to another authorized UUID and does not update session state. When there is one Organization, login may enter it directly; when there are multiple and no safe return path, show an Organization chooser; when there are none, show first-Organization setup.
- Every tenant-owned table has a non-null `organization_id`. Foreign keys and uniqueness constraints include it whenever the referenced relationship must remain inside one tenant.
- Go authorization and PostgreSQL RLS are both mandatory. API runtime transactions set local User/Organization context after session and Membership validation; runtime roles have no `BYPASSRLS`.
- Use separate migration-owner, API-runtime and worker-runtime database roles. Global auth tables are not presented as tenant-owned, but remain reachable only through narrow authenticated operations.
- Never accept Organization Context from a header, cookie, form field or token without resolving it against current Membership.

### Invitations and member onboarding

- Phase 1 uses a shareable capability invitation because LINE-first Users may have no trusted email. It does not send invitation email.
- Owner and Admin may create, inspect and revoke invitations. Every invitation grants Member only; Owner alone may promote the accepted Member to Admin later.
- Generate a 256-bit invitation token, store only its hash and show the plaintext once. Default expiry is 24 hours.
- An invitation records Organization, inviter, Member role, creation/expiry, optional revocation and accepting User/time. It is single-use.
- Opening a valid invite immediately exchanges the URL secret for a server-bound invite intent and redirects to a clean URL. The OAuth transaction stores the invitation identifier, not the plaintext token.
- If Google Login requires an external browser, issue a five-minute one-time handoff bound only to that invitation intent and safe return path. Store its hash, consume it before starting OAuth and never let it authorize an Admin role or provider link.
- The invite screen identifies the Organization and inviter, explains that acceptance creates Membership, and asks the authenticated User to confirm.
- Redemption locks the invite, re-checks inviter authority and Organization state, creates Membership and consumes the invite in one transaction.
- Redemption by an existing member is idempotent and consumes no additional authority. Concurrent redemption by different Users permits only one successful claimant.
- Expired, revoked, malformed, unknown and replayed tokens share a non-enumerating error with restart/help guidance.
- Invite pages use no third-party scripts and set a referrer policy that prevents token leakage.

### LINE user and group linking

- LINE user linking is accomplished by attaching a LINE Login AuthIdentity through the authenticated provider-link flow. Do not add Messaging API native account linking in Phase 1.
- Compare LINE Login `sub` with Messaging API `source.userId` only when both channels are confirmed under the same environment-specific LINE Provider.
- One Organization may have multiple active LINE Group Connections. One `(messaging_channel, group_id)` may have at most one active Organization connection.
- Only Owner or Admin with an active LINE AuthIdentity and recent authentication may create a group link code.
- Generate at least 128 bits of randomness for the copy/paste code. Store only its hash and bind it to Organization, initiating User, expected LINE subject, Messaging API channel, purpose and a ten-minute expiry.
- Issuing a replacement code revokes the previous pending code for the same User and Organization. Codes are one-time and never accepted after expiry, revocation or consumption.
- The User pastes a command containing the code into the intended LINE group. The webhook verifies the exact raw body signature before parsing.
- A link attempt is eligible only for a message event with `source.type=group`, non-empty `source.groupId`, non-empty `source.userId`, the expected channel and a sender user ID equal to the expected LINE subject.
- A LINE `join` event never creates a connection because it does not prove who invited the bot. PlaiFlow does not claim to verify LINE group-admin status.
- Hash and redact a recognized candidate code before durable event storage. Logs and Inbound Event payloads must not contain its plaintext.
- Worker processing locks the pending code and target group, re-checks current Owner/Admin Membership, confirms the group is not actively connected elsewhere, consumes the code and creates the connection in one transaction.
- Missing sender identity, mismatched user, wrong group/channel, invalid code, expired code, stale role and active connection conflicts fail closed. Attempts are rate-limited and audited without storing the code.
- Dashboard polls or refreshes connection state and shows waiting, connected, expired, conflict and disconnected states; no outbound LINE reply is required.
- Disconnecting in PlaiFlow immediately disables tenant processing for that group and records who disconnected it. Phase 1 instructs the administrator to remove the Official Account from LINE separately; it does not call the Messaging API leave endpoint.
- A valid LINE `leave` webhook marks an active connection disconnected. Subsequent unconnected-group messages are acknowledged after signature verification but discarded unless they are eligible link attempts.

### Audit, logging and retention

- Write append-only audit events for login success/failure category, AuthIdentity link/unlink/conflict, session revocation, invitation create/revoke/redeem, Membership add/remove/role change, ownership transfer and LINE group-code/connection outcomes.
- Audit records contain event type, timestamp, optional Organization, actor User when known, target entity identifiers, request ID, safe outcome/reason code and minimal environment metadata.
- Do not store authorization code, state, nonce, PKCE verifier, session/CSRF/invite/link token, raw provider response, raw LINE message, full cookie, channel secret or provider access/ID token.
- Security logs may record rate-limit and validation categories but never distinguish whether an email or unknown identity belongs to an existing User.
- Phase 1 provides no customer-facing audit-log UI. Operational access remains least-privilege and environment-scoped.
- Clean expired auth transactions promptly, expired/revoked sessions daily, invitation secrets after terminal state, link codes after their audit-safe retention window and unlink tombstones after 30 days.

### UI/UX behavior

- Reuse the accepted PlaiFlow Design System: semantic teal/warm-orange tokens, self-hosted Noto Sans Thai, light/dark system preference, rounded but restrained surfaces and minimal transform/opacity motion.
- Welcome/login is mobile-first with LINE as the single primary CTA and Google as a visually secondary official-brand button. Do not auto-start OAuth.
- Preserve official LINE and Google logo/button proportions and labels. Do not recolor provider marks or replace them with emoji.
- First Organization setup asks only for Organization name. It does not ask for a slug, role or optional business profile.
- Invite onboarding shows Organization, inviter, Member role and one accept action; authentication and acceptance remain distinct, understandable steps.
- Account settings show each provider as linked, available to link, re-authentication required, unlink blocked, cooldown/recovery or conflict. Never expose issuer/subject values in normal UI.
- Organization chooser and switcher show only Membership-backed Organizations and remain usable with long Thai names.
- LINE group setup uses a short step sequence: require/link LINE identity, generate code, copy it, paste into the group, wait for verification, then show connection state and a retry path.
- Auth errors use safe categories: access denied, expired attempt, provider unavailable, identity conflict, session expired, invalid invite and external-browser required. Each includes one recovery action and a request ID for support.
- Errors are announced with semantic alert behavior, form errors remain near fields, focus moves predictably, touch targets are at least 44px and layouts are checked at 375/768/1024/1440px with no horizontal scroll.
- Keep pages as Server Components by default and push client behavior to the smallest interactive controls. Do not add an animation or auth UI dependency unless existing platform features cannot provide the behavior.

### HTTP behavior

- Same-origin auth routes cover provider start/callback for `login`, `link` and `reauth`, current-session inspection, logout, logout-all and AuthIdentity unlink.
- Tenant routes use the Organization UUID and resolve Organization Context before handlers receive tenant data.
- Organization routes cover list, create, current details, Membership list/removal/role change, ownership transfer and leave.
- Invitation routes cover create, inspect, revoke, clean landing/intention exchange and authenticated acceptance.
- LINE connection routes cover connection list, code creation/revocation, status and disconnect; signed Messaging API webhook remains the only group-code redemption entry point.
- Every mutation requires method validation, content-type validation where applicable, body limits, authenticated session, CSRF validation, permission check and safe idempotency/concurrency handling.
- JSON errors expose a stable code, safe Thai-ready message and request ID. Internal provider/SQL errors are logged only as redacted categories.
- Apply endpoint-specific rate limits to auth start/callback, invite inspection/acceptance, re-authentication, provider linking and LINE link attempts. Rate-limit keys must not require logging full external identifiers.

### Schema constraints and performance indexes

- `users`: UUID primary key, editable profile, lifecycle timestamps and closed state. No unique email column.
- `auth_identities`: UUID primary key; User foreign key; provider, issuer, subject and safe provider attributes; unique `(issuer, subject)` including cooldown tombstones; partial unique active `(user_id, provider)`; index active identities by User.
- `auth_transactions`: unique state hash; provider/purpose/initiator and expiry data; indexes for state lookup, initiating User/session and expiry cleanup.
- `sessions`: unique token hash; User and lifecycle timestamps; indexes for active token lookup, User session listing/revocation and expiry cleanup.
- `organizations`: UUID primary key and name. Organization creation must not expose sequential identifiers.
- `memberships`: primary or unique `(organization_id, user_id)`; partial unique one Owner per Organization; index `(user_id, organization_id)` including role for authorization/switching and `(organization_id, role, user_id)` for member lists.
- `invitations`: unique token hash; Organization/inviter/status/expiry fields; indexes for token lookup, Organization pending-invite listing and expiry cleanup.
- `invite_handoffs`: unique token hash, invitation intent, safe return path and five-minute expiry; indexes for token lookup and expiry cleanup. It carries no authenticated User or provider-link authority.
- `line_group_connections`: Organization, Messaging API channel, group ID, status and audit timestamps; unique active `(channel, group_id)` and index `(organization_id, status)` for Dashboard reads.
- `line_link_codes`: unique code hash and binding fields; indexes for code lookup, pending code by User/Organization and expiry cleanup.
- `audit_events`: time-ordered identifier, optional Organization/actor and safe metadata; indexes `(organization_id, occurred_at desc, id)` and `(actor_user_id, occurred_at desc, id)` for future investigation without adding an audit UI.
- Indexes exist for demonstrated queries only. Do not add indexes on low-selectivity role/status columns alone.
- Organization list/switch uses one indexed Membership-to-Organization query. Member lists and provider state load in bounded queries independent of row count; no per-row lookup is allowed.
- Session `last_seen` writes are throttled to once per five minutes. Auth and Membership resolution must not update Organization state.
- Initial performance targets under local/staging load are p95 below 300ms for authenticated tenant reads excluding provider redirects, one Membership lookup per tenant request, and bounded query counts for Organization/member lists.

## Testing Decisions

- Tests assert behavior at HTTP, PostgreSQL and rendered-page boundaries. Do not test private helper structure or mock away the authorization boundary being proven.
- Extend the existing public HTTP-to-PostgreSQL integration style. Use controlled local provider doubles for token/verify/JWKS responses; CI never requires real LINE/Google credentials or internet access.
- The primary auth acceptance test starts login, receives a simulated provider callback, creates/loads User and AuthIdentity, receives the secure cookie, then accesses one permitted tenant endpoint.
- The primary LINE-link acceptance test starts from an authenticated Owner/Admin, creates a code, sends a signed webhook event from the matching group/user, runs worker processing and observes the connected group through an authorized tenant endpoint.
- Test LINE and Google authorization requests separately for exact scopes, state, nonce, PKCE S256, registered redirect URI, expected issuer and absence of Drive/offline/incremental scopes.
- Test callback rejection for missing, malformed, expired, replayed, provider-mismatched and cross-session state; wrong nonce; wrong issuer/audience/authorized party; invalid signature; expired ID token; reused authorization code and provider timeout.
- Test OAuth mix-up attempts by returning a Google response to the LINE callback and vice versa.
- Test that callback responses redirect to clean allowlisted paths and never render or forward provider parameters.
- Test open-redirect inputs including absolute URLs, scheme-relative URLs, encoded path traversal, userinfo, fragments and unlisted origins.
- Test that first login creates one User/AuthIdentity and concurrent/repeated callbacks create no duplicate.
- Test that equal emails across LINE/Google create no automatic link or merge and do not reveal existing-account status.
- Test linking only from authenticated, recent and initiating sessions. Reject anonymous, expired-recent-auth, different-session and identity-owned-by-other-User attempts.
- Test at-most-one active identity per provider per User and global `(issuer, subject)` uniqueness under concurrency.
- Test unlink rejection when it is the last identity or re-auth used the identity being removed. Test successful unlink, session rotation, other-session revocation, cooldown login rejection, safe relink and tombstone expiry.
- Assert provider tokens, codes, state, nonce, verifier, invite/link tokens and raw provider responses are absent from logs, audit payloads, browser output and built frontend assets.
- Assert the session cookie has the `__Host-` prefix, Secure, HttpOnly, SameSite=Lax, root path and no Domain attribute.
- Test session fixation by seeding a pre-auth cookie and proving login issues a different token. Test idle expiry, absolute expiry, revocation, logout, logout-all and throttled `last_seen` behavior.
- Test CSRF rejection for absent/wrong/cross-session token and wrong/missing Origin; test valid same-origin mutation.
- Exercise the complete Owner/Admin/Member permission matrix against each privileged endpoint. Each denied request must leave database and audit target state unchanged except for a safe denial audit where specified.
- Test Organization creation plus Owner Membership atomicity and rollback.
- Test Organization URL spoofing, direct-object identifier changes and cross-tenant reads/writes/deletes against both Go checks and PostgreSQL RLS using the non-bypass runtime role.
- Test composite foreign keys reject relationships whose Organization identifiers disagree.
- Test one-Owner enforcement, concurrent ownership transfers, blocked Owner leave, Admin inability to manage Admin/Owner and immediate role/removal effect on existing sessions.
- Test Organization chooser/list with zero, one, many and long Thai-named Organizations using a bounded query count.
- Test invitation creation by Owner/Admin, Member denial, Member-only grant, token hashing, clean-URL exchange, 24-hour expiry, revocation, replay and non-enumerating invalid states.
- Test embedded-to-system-browser invitation handoff for hash-only storage, five-minute expiry, one-time consumption and inability to authorize provider linking or elevated roles.
- Test that provider linking after a browser-context change requires authentication as the existing User in the destination browser and rejects a handoff token presented as link authority.
- Test concurrent invitation redemption by two Users permits one claimant and creates one Membership. Existing-member redemption remains idempotent.
- Test pending invite acceptance after inviter role loss fails closed.
- Test link-code randomness boundary, hash-only storage, ten-minute expiry, replacement revocation, replay and atomic consumption.
- Test valid group linking only for matching channel, group, sender LINE subject and current Owner/Admin role.
- Test forged webhook signature, altered raw body, direct API redemption, personal-chat source, room source, missing sender, mismatched sender, wrong channel, wrong group, stale role and group-already-linked conflict.
- Test a valid LINE `join` event does not create a connection. Test a valid `leave` event disconnects the existing connection.
- Test plaintext link code is redacted before Inbound Event persistence and never appears in structured logs or audit metadata.
- Test messages from disconnected/unconnected groups are acknowledged but not retained unless they are eligible link attempts.
- Test one group cannot be connected to two Organizations concurrently while one Organization may connect multiple groups.
- Test disconnect authorization and verify later events cannot enter tenant processing. Do not assert automated bot leave because it is out of scope.
- Test audit creation for every specified security action and assert token/code/raw-claim fields cannot be serialized into audit metadata.
- Test UI states for login, first Organization setup, invite onboarding, Organization chooser, identity linking/unlinking, group linking and all named recovery states.
- Test provider buttons with accessible names, visible focus, keyboard activation, disabled/loading semantics and official text labels. Test errors with alert semantics and focus recovery.
- Test responsive behavior at 375, 768, 1024 and 1440px in light/dark/reduced-motion settings, including LINE in-app-browser-sized viewports and external-browser guidance for Google.
- Verify required indexes through migration/catalog tests. Use query-count assertions for N+1 protection and representative query-plan smoke checks for Membership lookup, session lookup, invitation/code lookup and Organization listing without pinning exact planner output.
- Run migration up/down/up against an isolated PostgreSQL database and prove RLS policies with API runtime, worker runtime and migration-owner roles.
- Run secret scanning and a frontend bundle scan in CI. Provider credentials and real customer identities are never test fixtures.

## Out of Scope

- Google Drive authorization, Drive API, Google Sheets scopes, incremental authorization, offline access and Google refresh-token storage.
- Any non-identity Google scope or bundling future Drive consent into Google Login.
- Password authentication, email magic links, recovery questions, local MFA and passkeys.
- LIFF application/SDK integration; Phase 1 remains responsive web optimized for users arriving from LINE.
- Messaging API native account-linking flow; LINE user identity is linked through LINE Login.
- Automatic User/account merge, automatic Membership collision resolution and self-service recovery when the only provider is lost.
- Custom roles, permission editor, Organization deletion and customer-facing audit-log UI.
- Email invitation delivery or treating email possession as the only invitation authorization factor.
- Proving that a User is a LINE group owner/admin; LINE group linking proves only the authenticated PlaiFlow role and matching LINE message sender.
- Automated Messaging API bot leave, outbound LINE replies and group/member enumeration.
- Storing or processing unrelated messages from unconnected LINE groups.
- Business workflow, document/OCR, accounting, AI advisor, notifications and Google Drive connector behavior.
- Production credentials, provider-console mutation, callback activation, cloud resource creation and live deployment without explicit approval.

## Further Notes

- The spec intentionally prefers duplicate Users over email-based automatic merging because a recoverable duplicate has lower impact than an account takeover.
- Invitation and link-code secrets are capability credentials: display once, hash at rest, keep out of callback URLs/logs and consume atomically.
- LINE Login `sub` can match Messaging API `source.userId` only when the channels share the same LINE Provider. Channel/provider layout must be verified before staging activation.
- LINE group linking cannot establish LINE group-admin status because relevant webhook/group APIs do not supply that proof.
- Google OAuth may reject embedded user-agents. The external-browser handoff must be verified on real LINE iOS and Android clients before claiming staging readiness.
- Relevant primary references: [LINE Login integration](https://developers.line.biz/en/docs/line-login/integrate-line-login/), [LINE PKCE](https://developers.line.biz/en/docs/line-login/integrate-pkce/), [LINE ID-token verification](https://developers.line.biz/en/docs/line-login/verify-id-token/), [LINE group chats](https://developers.line.biz/en/docs/messaging-api/group-chats/), [Google OIDC](https://developers.google.com/identity/openid-connect/openid-connect), [Google OAuth policy](https://developers.google.com/identity/protocols/oauth2/policies), and [OAuth 2.0 Security Best Current Practice](https://www.rfc-editor.org/rfc/rfc9700.html).
- This document is specification only. It does not authorize implementation or any external activation/deployment action.
