# 08: Metrics, audit, staging migration and rollback proof

**What to build:** Operators can observe and safely roll out the complete Durable Job protocol in staging, with tenant-isolation evidence, lifecycle metrics/audit and a tested rollback path.

**Blocked by:** 04: Large CSV export completes with an expiring secure download; 05: Large XLSX export uses the same secure job pipeline; 06: Queued cancellation and artifact cleanup; 07: Provider-independent OCR Durable Job seam.

**Status:** ready-for-agent

- [ ] Metrics cover queue wait, runtime, attempts, retries, stale reclaims, safe terminal failures, export rows/bytes, artifact downloads and worker heartbeat age.
- [ ] Metric labels exclude Organization, User, Job, Document, filename, token and cursor values.
- [ ] Alerts cover missing worker heartbeat for two minutes, excessive queue wait, stale-reclaim growth, cleanup backlog and failure spikes by kind.
- [ ] Audit covers creation, claim/reclaim, retry, completion/failure, artifact creation, signed URL issuance, download, cleanup, cancellation and authorization failure.
- [ ] Audit contains Organization, safe actor/worker identity, job ID and request ID without tokens, signed URLs, object keys, exported values or OCR content.
- [ ] Staging migration preserves active legacy exports and readable history without duplicate execution.
- [ ] Staging smoke tests prove concurrent exactly-once claim, stale recovery, large CSV/XLSX completion, secure expiry-bound retrieval and OCR protocol isolation.
- [ ] The Durable Job worker runs without a PostgreSQL credential and with environment-specific least-privilege service credentials.
- [ ] Logs and metrics show no serious error, secret leakage, hot polling or unbounded memory behavior during the smoke test.
- [ ] The rollback runbook stops new Durable Job creation, rolls back workers before destructive cleanup, preserves readable history and lets valid leases finish or expire.
- [ ] Legacy schema/read-path removal remains deferred until the 90-day metadata window and separate approval.
