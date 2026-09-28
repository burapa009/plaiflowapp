# OCR service threat model — local implementation review

Scope: PlaiFlow's authenticated upload, existing leased OCR worker, private
FastAPI OCR service, PaddleOCR subprocess, artifact publication, and separate
Go extraction layer. Documents can contain tax IDs, names, addresses and
financial data. This review does not attest to a deployed network policy or
secret rotation.

## Data flow and trust boundaries

```text
Browser ──HTTPS/session──> Go upload/tenant check ──private object──> storage
                             │                                 │
                             └──leased job──> OCR worker <─────┘
                                                │
                                private WireGuard HTTP + bearer token
                                                ▼
                                      FastAPI → PaddleOCR subprocess
                                                │
                                      structured JSON → OCR worker
                                                │
                               lease-checked artifact publication
                                                ▼
                                      Go extraction and review
```

The browser/API, worker/service, service/model, and worker/artifact transitions
are trust boundaries. Only the Go API authorizes Organization and Document
access. The OCR service has no database or object storage credentials.

Relevant actors: a malicious uploader can submit crafted files or many jobs;
a stolen worker token can call inference; an operator with container access
can read temporary documents. A compromised model dependency can run code in
the OCR container. Nation-state targeting is outside this local review.

| Component | S | T | R | I | D | E | Overall |
| --- | --- | --- | --- | --- | --- | --- | --- |
| Go upload and artifact API | M | M | M | H | H | M | High |
| OCR worker / service link | H | M | M | H | M | M | High |
| PaddleOCR subprocess | L | M | L | M | H | H | High |
| Private OCR artifact | M | H | M | H | M | M | High |

S/T/R/I/D/E are STRIDE: spoofing, tampering, repudiation, information
disclosure, denial of service, and elevation of privilege.

| ID | STRIDE / ATT&CK | Threat | Risk | Mitigation and owner | Status |
| --- | --- | --- | --- | --- | --- |
| OCR-1 | S,I / T1078,T1530 | Stolen service token exposes uploaded documents or OCR output | High | Project operator: private endpoint, environment-specific 32+ character token, rotation and no request-body logs | Token rollout open |
| OCR-2 | D / T1499.003 | Crafted PDF or upload flood exhausts CPU, RAM or temporary disk | High | Backend/operator: 20 MiB intake, 50-page service cap, pixel cap, one inference at a time, 45-second page timeout, ingress body limit | Ingress limit open |
| OCR-3 | T,E / T1195,T1190 | Malformed decoder/model output or compromised dependency alters artifact | High | OCR/backend: pinned dependencies/models, subprocess isolation, geometry/schema validation, lease and checksum checks | Local checks pass; deployment proof open |
| OCR-4 | I,R / T1530,T1070 | Raw OCR/debug data or retained temp file leaks private data | Medium | OCR/operator: debug default off, temporary cleanup, non-root container and metadata-only logs | Local code complete; runtime drill open |

The remaining release evidence is a private-network and token-rotation drill,
ingress body-size enforcement, representative malformed-file/resource tests,
and an artifact authorization check across Organizations. The Phase 12 gate
remains open until these are recorded with the other release checks.
