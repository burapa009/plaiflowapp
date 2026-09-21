# 07: Provider-independent OCR Durable Job seam

**What to build:** A trusted server path can enqueue an OCR Durable Job for one authorized Document, and an OCR-scoped worker can claim, heartbeat, access only that original, and complete or fail through the generic protocol without implementing OCR processing.

**Blocked by:** 03: Heartbeat, stale recovery and bounded retries.

**Status:** ready-for-agent

- [ ] The server-owned enqueue seam creates an `ocr` job bound to one Organization, Document and immutable input version/checksum.
- [ ] Client-provided object keys, Organization IDs and storage metadata are never trusted as the OCR input binding.
- [ ] Only a worker token with OCR scopes can claim the job or request its input.
- [ ] The current attempt can obtain a short-lived signed read grant for the referenced Document only.
- [ ] An OCR grant cannot access another Document, job, attempt or Organization and becomes unusable after its short lifetime.
- [ ] OCR heartbeat, stale recovery, bounded retry and stale-completion behavior are identical to other Durable Jobs.
- [ ] Completion accepts only a versioned provider-independent result reference, safe outcome metadata and checksum.
- [ ] OCR retry/completion creates no Document, Document Source or Document Intake Usage increment.
- [ ] Safe provider-independent failure codes are stored without OCR content or provider secrets.
- [ ] Contract tests prove the complete OCR protocol seam without an OCR provider, SDK, model, parser, extracted-field schema, UI or production OCR handler.
