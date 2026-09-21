# 01: Worker claims one Durable Job exactly once

**What to build:** An authorized large-export request creates one Organization-scoped Durable Job, and a Durable Job worker using the Internal Job API can claim it exactly once without PostgreSQL credentials. This establishes the narrow end-to-end protocol seam while leaving existing legacy queues operational.

**Blocked by:** None (can start immediately).

**Status:** ready-for-agent

- [ ] An authorized export above 5,000 rows creates one Queued `export` Durable Job with immutable Organization, requester, format, normalized filters, row count and snapshot boundary.
- [ ] A duplicate enqueue using the same request/idempotency identity returns the existing job rather than creating another job.
- [ ] A Durable Job worker authenticates with an environment-specific, short-lived service token and does not require a PostgreSQL credential.
- [ ] A worker can claim an allowlisted kind through the Internal Job API with a batch bound of at most 10.
- [ ] Claim returns a job ID, Organization ID, attempt ID, lease expiry and one-time lease token without exposing database or unrestricted storage credentials.
- [ ] Two concurrent claimers receive disjoint results; one queued job is returned to exactly one claimant and becomes Running once.
- [ ] Claiming uses eligible status/availability ordering with `SKIP LOCKED` or equivalent transactional behavior.
- [ ] An expired, wrong-environment or insufficient-scope worker token is rejected before job existence is disclosed.
- [ ] Tenant-facing job status shows safe lifecycle data while hiding lease tokens, worker identity and internal payload references.
- [ ] Contract and PostgreSQL integration tests prove concurrent exactly-once claim behavior at the API seam.
