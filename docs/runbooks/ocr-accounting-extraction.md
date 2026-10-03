# Thai accounting OCR extraction

## Entry point

Open an uploaded document after its PaddleOCR job completes. The existing
`GET /v1/o/{organization}/documents/{document}/extraction` response includes
`draft.accounting`. The document review page displays this under
“ข้อมูลบัญชีแบบละเอียดจาก OCR” and offers a draft JSON download.

Access uses the existing session, organization and document authorization.
Responses are private and not cached. No OCR text is added to application logs.
The existing `EXTRACTION_ENABLED` gate still applies.

## Contract

- Nested document, seller, buyer, items, summary, payment, confidence, validation,
  warnings and raw_text follow the requested accounting schema.
- Unknown or ambiguous values are null; absent line items are an empty array.
- Monetary values are JSON numbers. Validation uses decimal cents, without
  filling missing amounts or correcting contradictory values.
- Currency requires evidence; THB is not inferred solely from Thai text.
- `raw_value` preserves tax identifier candidates, including invalid candidates.
- `tax_id_valid` stays null for a 13-digit candidate until an authoritative
  registry check exists; length alone is not proof that the ID is valid.
- `raw_text` retains each original OCR page text in page order, joined with a
  newline between pages. The authoritative OCR artifact retains the separate
  page texts and polygons. Older artifacts without page text fall back to their
  unchanged line texts joined with newlines.
- Confidence describes OCR evidence quality, not calibrated extraction accuracy.

## Processing and review

Extraction runs locally with deterministic rules. No external LLM receives the
document. Labels and reliably aligned table cells are required; uncertain layouts
require checking the original. This is not an accuracy guarantee for every scan.

Detailed JSON is an automatic draft. Existing human confirmation and accounting
CSV/XLSX exports retain their current reviewed fields. Downloading this JSON does
not confirm it or post an accounting transaction. The detailed draft can be
regenerated from the retained OCR artifact; no database migration is required.

## Verification

Run `go test ./...` from `api`, and `npm test`, `npm run lint`,
`npm run typecheck`, `npm run build` from `web` with the usual local API settings.
Parser tests cover nullable values and arithmetic; HTTP tests check delivery via
the authorized extraction route and rejection across organizations.

Before enabling on a deployment, upload representative real Thai documents,
compare extracted fields and item rows against each original, download JSON,
and check missing/ambiguous results and warnings. Local tests do not establish
production OCR accuracy or prove database RLS integration.

Rollback is reverting the OCR accounting implementation commit or disabling the
existing extraction feature gate. Original OCR artifacts are unchanged.
