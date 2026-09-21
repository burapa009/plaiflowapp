# 03: Heartbeat, stale recovery and bounded retries

**What to build:** A claimed Durable Job remains owned while its worker is healthy, recovers after a worker disappears, and retries only bounded transient failures without allowing stale attempts to overwrite current work.

**Blocked by:** 01: Worker claims one Durable Job exactly once.

**Status:** ready-for-agent

- [ ] A claim creates a two-minute Job Lease and the current attempt can renew it through heartbeat every 30 seconds.
- [ ] Heartbeat is idempotent for the current attempt and cannot revive a cancelled, terminal, expired or replaced attempt.
- [ ] An expired Running job can be reclaimed with a new attempt ID and lease token.
- [ ] Reclaiming a stale job consumes an attempt and increments a stale-recovery metric once.
- [ ] Completion or failure from the old lease is rejected and cannot attach a result or artifact.
- [ ] Repeating completion for the same successful attempt returns the existing terminal result without a second completion or artifact.
- [ ] The API classifies worker error codes through a server-owned allowlist; a worker cannot arbitrarily force retry.
- [ ] Transient failure uses jittered exponential backoff and provider retry hints; permanent and security failures terminate immediately.
- [ ] No job can be claimed more than five attempts; exhausted work becomes Failed with a safe terminal code.
- [ ] Empty workers long poll for at most 15 seconds or back off from 1 to 15 seconds without hot polling.
- [ ] API and PostgreSQL integration tests cover heartbeat, stale reclaim, stale completion, duplicate completion and retry exhaustion.
