# 05: Large XLSX export uses the same secure job pipeline

**What to build:** An entitled large XLSX export follows the proven Durable Job, chunk, artifact and secure-download path without introducing a second lifecycle or loading the whole workbook into memory.

**Blocked by:** 04: Large CSV export completes with an expiring secure download.

**Status:** ready-for-agent

- [ ] Large XLSX requests use the same generic `export` job lifecycle, ownership and status endpoints as CSV.
- [ ] Requests above 50,000 rows or 128 MiB are rejected before enqueue.
- [ ] The worker consumes the same immutable 500-row cursor pages and stays within the 64 MiB export memory budget.
- [ ] Every untrusted workbook value is written as text and does not become a formula.
- [ ] The XLSX artifact uses the same job-bound signed upload, verified completion and 24-hour expiry rules.
- [ ] The requester or current Owner/Admin receives the same 15-minute tenant-authorized signed download behavior.
- [ ] Retry and duplicate completion cannot create a second workbook artifact.
- [ ] End-to-end tests prove the downloaded workbook contains the authorized filtered rows and no other Organization's data.
