# 06: Queued cancellation and artifact cleanup

**What to build:** Authorized users can stop work before processing begins, and the system removes expired or abandoned export objects safely without losing job history or audit evidence.

**Blocked by:** 04: Large CSV export completes with an expiring secure download.

**Status:** ready-for-agent

- [ ] The requester or a current Owner/Admin can cancel a Queued job.
- [ ] Cancellation races atomically with claim so exactly one transition wins.
- [ ] Running, Completed, Failed and already Cancelled jobs cannot be newly cancelled.
- [ ] A cancelled job cannot be claimed, heartbeated, completed or given an upload/download grant.
- [ ] Cleanup removes completed export artifacts after 24 hours and treats an already-missing expired object as success.
- [ ] Cleanup removes abandoned partial uploads and expired temporary grants without changing terminal job history.
- [ ] Failed cleanup retries through bounded Durable Job behavior and remains observable.
- [ ] Job metadata remains readable for 90 days and export audit remains available for 365 days.
- [ ] Cleanup and cancellation audit records contain safe identifiers and never contain object keys, signed URLs or exported values.
- [ ] Time-controlled tests prove cancellation boundaries, expiry, idempotent deletion and retention separation.
