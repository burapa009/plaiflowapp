# Phase 1 authentication baseline

Phase 0 reserves Go `/v1/auth/{provider}/callback` and web `/auth/callback`. Phase 1 keeps Go as the owner of authentication and exposes its auth routes through the same public web origin.

## OAuth transaction

- LINE and Google use Authorization Code Flow with independent 256-bit `state` and `nonce` values plus PKCE `S256`.
- Store only the `state` hash. Bind the 10-minute, single-use transaction to its provider, issuer, `login|link` purpose, safe relative return path, and initiating User/session when linking.
- The callback validates state, issuer, audience, expiry and nonce server-side, then redirects to a clean web URL without rendering provider parameters or loading third-party scripts.
- LINE ID tokens are checked through LINE's verify endpoint; Google ID tokens are checked through its JWKS.
- Authorization codes, provider tokens and raw provider responses never enter browser state or application logs. Login-only access tokens are discarded and refresh tokens are not requested.

## Session

- Use an opaque server-side session whose token is stored only as a hash and sent in a host-only `HttpOnly`, `Secure`, `SameSite=Lax` cookie.
- Idle timeout is 14 days, absolute lifetime is 30 days, and sensitive actions require authentication within the last 10 minutes.
- Rotate the session identifier after login, re-authentication and External Identity linking. Support logout for the current session and all sessions.
- Update `last_seen` at most once every five minutes. Membership and role changes are authorized on the next request rather than cached for the session lifetime.
- State and same-origin protections do not replace mutation CSRF checks; validate the request origin and a session-bound CSRF token server-side.

## Audit baseline

Store append-only security events for login results, External Identity link/unlink, session revocation, invite lifecycle, Membership/role/ownership changes and LINE Group Connection link/unlink. Do not store tokens, authorization codes, link codes or raw provider responses. Phase 1 does not add an audit-log UI.
