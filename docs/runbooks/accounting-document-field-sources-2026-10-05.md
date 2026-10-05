# Field-rule sources checked 2026-10-05

The current form handles imported evidence, edits and OCR acknowledgement. Its `issue`, `account` and `tax` actions are blocked. The catalogue describes why evidence matters without authorizing those workflows.

| Source | Finding used | Implementation limit |
|---|---|---|
| [Revenue Code 86/4, 86/6, 86/9, 86/10](https://www.rd.go.th/5208.html) | Different invoice/abbreviated invoice and adjustment duties; combined receipt/invoice must retain both rule sets | Imported documents remain saveable with missing evidence |
| [Revenue Code 105, 105 bis, 105 quater](https://www.rd.go.th/5203.html) | Receipt and sale delivery duties depend on the applicable case | Unknown scope remains unresolved; internal movement does not require price from these sale rules |
| [VAT announcement 39, current consolidated text](https://www.rd.go.th/3400.html) | Clauses 7–9 use buyer VAT registration and registered establishment data; the text incorporates amendments 199 and 247 | Buyer VAT registration now independently triggers buyer tax ID. Verified clauses 7–9 carry legal effective date 2015-01-01; no guessed headquarters code |
| [WHT announcement 62, consolidated text](https://www.rd.go.th/3171.html) | Certificate particulars depend on the prescribed form and income/payment facts | No universal 3% default or fabricated signature; no official certificate generation |
| [Accounting legislation compilation hosted by DIP, PDF pages 146–147](https://www.dip.go.th/uploadcontent/boom_LAW/GNB_022.pdf) | DBD accounting evidence duties distinguish external, internally issued and internal-use evidence; additional receipt/delivery/internal evidence particulars depend on that origin/use | Existing workflow minimums remain labelled `system_policy`. This catalogue does not assert full DBD compliance: journal posting requires a separate origin/use assessment and accounting workflow |

The source check does not prove every specialised industry, foreign-currency, replacement or electronic-issuance rule. Broad case selectors and `legalEffectiveFrom=null` remain explicit for unverified cases. Rule changes are versioned and saved assessments remain attached to their revision.
