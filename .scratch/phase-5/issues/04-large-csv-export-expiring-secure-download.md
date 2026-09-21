# 04: Large CSV export completes with an expiring secure download

**What to build:** An entitled Owner/Admin can request a CSV export above 5,000 rows, leave the page, and later download the completed artifact securely after a worker streams the authorized dataset through the Durable Job protocol.

**Blocked by:** 02: Migrate active legacy export jobs safely; 03: Heartbeat, stale recovery and bounded retries.

**Status:** ready-for-agent

- [ ] Exports through 5,000 rows retain synchronous behavior; larger entitled CSV requests return an accepted Durable Job.
- [ ] Requests above 100,000 rows or 256 MiB are rejected before enqueue with guidance to narrow filters.
- [ ] The job freezes Organization, requester, versioned template, normalized filters, authorized row count and snapshot boundary.
- [ ] The claimed worker reads opaque job-bound cursor pages of at most 500 authorized rows through the Internal Job API.
- [ ] Cursors cannot cross jobs, attempts or Organizations and cannot alter filters or the snapshot boundary.
- [ ] CSV generation streams output within the 64 MiB memory budget and protects every untrusted cell from spreadsheet-formula execution.
- [ ] The worker receives a short-lived signed upload grant for one server-selected private object key and no master storage credentials.
- [ ] Completion verifies artifact ownership, format, row count, byte count, checksum and size ceiling before recording Completed.
- [ ] Duplicate completion does not create another object, terminal transition or completion audit.
- [ ] The requester or a current Owner/Admin can obtain a signed download URL valid for 15 minutes; other users and revoked Memberships are denied.
- [ ] The artifact records a 24-hour expiry and object keys, upload grants and download URLs never appear in tenant-facing responses or logs.
- [ ] End-to-end tests prove one large job reaches Completed and downloads the expected tenant-scoped, formula-safe CSV.
