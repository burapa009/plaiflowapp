# Phase 1 authentication prerequisites

Phase 0 reserves Go `/v1/auth/{provider}/callback` and web `/auth/callback`, validates an explicit web-origin allowlist, and returns `provider_disabled` because no provider is configured.

Before enabling LINE or Google, Phase 1 must add cryptographically random single-use state, PKCE where supported, short callback expiry, provider-specific issuer/audience/signature validation, safe token exchange/storage, and an explicit User-to-External-Identity linking policy. Do not treat the callback shell as authentication.
