# Accounting document forms: local validation 2026-10-05

Candidate: `.scratch/inbox-release-20261004`, branch `feat/accounting-document-forms`, HEAD `4f506401948f1ceb9b5711253ba28935b14d4172`. All changes remain uncommitted. No deployment or production migration took place.

## Routes and evidence

`R` in the table means `/o/[organization]/documents/[document]` and the embedded review in `/o/[organization]/documents`. Both render `extraction-panel.tsx` → `document-form-editor.tsx`, using `/v1/o/{organization}/documents/{document}/form`. `P` means the saved-data summary at `R/print`, using the same canonical form record.

`DB` means the real disposable PostgreSQL test saved, reopened, confirmed, switched away and back, and verified stored values for the type. `UI` means the live authenticated page changed to that type at 375px; all 22 had `scrollWidth=innerWidth=375`. These are local checks with synthetic data. Browser persistence was exercised on invoice and delivery note; it was not repeated separately for every type. The table distinguishes this from DB coverage.

The shared print renderer supports all types. Direct print inspection covered delivery note, saved revision/status, Thai dates, conditional price visibility, and 36 persisted rows. Chrome generated a 3-page PDF from that record. Each PDF page was not separately visually inspected, and official issuance remains unsupported.

| ประเภทเอกสาร | route/component | ฟอร์มที่เปลี่ยน | บันทึกและเปิดใหม่ | แบบพิมพ์/PDF | ผลตรวจจริง | ข้อจำกัด |
|---|---|---|---|---|---|---|
| ใบเสร็จรับเงิน | R/editor | ผู้รับ/ผู้จ่าย, ยอดรับครั้งนี้, วิธีชำระ, อ้างอิง | DB | P shared | DB + UI + cash rule | ไม่อนุมานว่ารับครบ; ไม่มีบันทึกเงินเข้าบัญชี |
| ใบกำกับภาษีเต็มรูป | R/editor | คู่ค้า, VAT context, รายการ, ราคา/ภาษี | DB | P shared | DB + UI + VAT parity | เงื่อนไข VAT ต้องตรวจ; ไม่มีสิทธิเครดิตภาษีอัตโนมัติ |
| ใบเสร็จ/ใบกำกับภาษี | R/editor | หน้าที่ใบกำกับและรับชำระทั้งสองชุด | DB | P shared | DB + UI + combined-rule regression | ไม่สร้าง journal ตามจำนวนหน้าที่เอกสาร |
| ใบแจ้งหนี้ | R/editor | ผู้เรียกเก็บ, เครดิต, รายการ, ยอด | DB + browser save/reload | P shared | DB + UI + Server Action | ยอดคงเหลือจากอ้างอิงที่ทราบ; ไม่ใช่ ledger balance |
| ใบวางบิล | R/editor | หลายเอกสารอ้างอิงและยอดจัดสรร | DB | P shared | DB + UI + allocation parity | เลือก canonical confirmed ล่าสุด 100; เอกสารภายนอกกรอกหลักฐานได้ |
| ใบส่งของ | R/editor | หน้าที่ตาม 105 จัตวา/ภายใน, จำนวนส่ง, ราคาเฉพาะกรณี | DB + desktop/mobile browser | P inspected; 3-page PDF generated | DB + UI + confirm incomplete + 36 rows | ยังไม่ตรวจภาพ PDF ทุกหน้า |
| ใบสำคัญรับเงิน | R/editor | ผู้จ่าย/รับ, เหตุผล, หักภาษี, ยอดสุทธิ | DB | P shared | DB + UI | เก็บข้อมูลลงนาม; ไม่สร้างลายเซ็น/อนุมัติ |
| ใบแทนใบเสร็จ | R/editor | ผู้จัดทำ, เหตุผลไม่มีใบเสร็จ, หลักฐาน | DB | P shared | DB + UI | ไม่รับรองรายจ่ายทางภาษีหรือแทนใบกำกับ |
| ใบสำคัญจ่าย | R/editor | ผู้รับ, เหตุผล, ยอดก่อนหัก/สุทธิ, วิธีจ่าย | DB | P shared | DB + UI | ไม่มี journal หรือการโอนเงินจริง |
| ใบกำกับภาษีอย่างย่อ | R/editor | ข้อความราคารวม VAT; ผู้ซื้อไม่บังคับทั่วไป | DB | P shared | DB + UI + abbreviated parity | ไม่มีการตรวจสิทธิผู้ออก/เปิดสิทธิภาษีซื้อ |
| ใบลดหนี้ | R/editor | ทั่วไป/VAT, มูลค่าเดิม/ใหม่, อ้างอิง, เหตุผล | DB | P shared | DB + UI + arithmetic | ค่าปรับบวกและใช้ประเภทกำหนดทิศทาง |
| ใบเพิ่มหนี้ | R/editor | ทั่วไป/VAT, มูลค่าเดิม/ใหม่, อ้างอิง, เหตุผล | DB | P shared | DB + UI + arithmetic | ไม่แก้ทับเอกสารต้นทาง |
| หนังสือรับรองหัก ณ ที่จ่าย | R/editor | หลายเงินได้, วันที่จ่าย, ภาษีหัก, ลงนามตามหลักฐาน | DB | P shared | DB + UI + variable WHT parity | ไม่มีแบบทางการ/ลายเซ็น; ไม่ตั้ง 3% ทั่วไป |
| Slip | R/editor | ธุรกรรม, ธนาคาร/บัญชี, ค่าธรรมเนียม, อ้างอิง | DB | P shared | DB + UI + no VAT/items parity | ไม่มีบริการตรวจ Slip กับธนาคาร |
| เอกสารธนาคารอื่น | R/editor | ประเภทย่อย, รายละเอียดและยอดตามเอกสาร | DB | P shared | DB + UI | ไม่มีเชื่อมธนาคาร/ลงบัญชี |
| Statement | R/editor | ช่วงบัญชีและธุรกรรมหลายแถว | DB | P shared | DB + UI + balance/duplicate diagnostics | แสดงข้อมูลการจับคู่; ยังไม่ดำเนินการกระทบยอดบัญชี |
| เอกสารอื่น | R/editor | ประเภทย่อย, รายละเอียด, หลักฐาน | DB | P shared | DB + UI | ผู้ตรวจต้องระบุหน้าที่เอกสาร |
| ใบเสร็จก่อนรับเงิน | R/editor | เครดิต/ติดตามหนี้, รายการและยอด | DB | P shared | DB + UI | ไม่เป็นหลักฐานว่ารับชำระแล้ว |
| ใบเสนอราคา | R/editor | คู่ค้า/รายการ/ยอดและอ้างอิง | DB | P shared | DB + UI | แบบข้อมูล; ไม่มี workflow ออกเอกสาร |
| ใบสั่งซื้อ | R/editor | คู่ค้า/รายการ/ยอดและอ้างอิง | DB | P shared | DB + UI | ไม่มี workflow จัดซื้อ/อนุมัติ |
| ใบเบิกค่าใช้จ่าย | R/editor | ค่าใช้จ่าย/รายการ/ยอดและอ้างอิง | DB | P shared | DB + UI | ไม่มีการอนุมัติหรือจ่ายเงิน |
| เอกสารเงินสดย่อย | R/editor | ค่าใช้จ่าย/รายการ/ยอดและอ้างอิง | DB | P shared | DB + UI | ไม่ปรับบัญชีเงินสดย่อย |

## Checks completed

- `go test ./internal/extraction ./internal/httpapi`: pass after final rule/assessment changes.
- `go test ./internal/postgres -run 'TestDocumentForms(PersistenceAndIsolation|ReadinessRequiresMigration25)' -count=1`: pass on PostgreSQL 18.4 after final backend changes.
- DB coverage also includes roles, private/cross-tenant access, reference snapshots/party/currency/allocation, optimistic concurrency, stale OCR, member edits invalidating old confirmation, failed confirmation rollback, history/audit and non-bypass RLS reads.
- `npm run typecheck`, `npm run lint`, `npm test`: pass, 31 tests.
- `npm run build -- --webpack`: pass. Detail and print routes are included in the build output.
- `git diff --check`: pass; Git emits LF/CRLF notices only. Many tracked Go files have line-ending-only status changes from the earlier round; do not stage them wholesale.
- Shared config generation produced identical frontend/backend JSON and reproduced the same hashes on rerun.
- Browser: invoice save/reload; delivery row save/reload; confirm with incomplete evidence kept the incomplete status; mobile save/reload; switch all 22 types at 375px; persisted print summary; 36-row save and 3-page PDF generation.

An earlier opt-in browser test hit Go's default 10-minute timeout while the browser session waited. The helper now exits before the test deadline, and the completed rerun used `-timeout 30m` and ended PASS with cleanup. That earlier killed test may have left its own uniquely named DB under the disposable local data directory. No production data was involved.

## Release requirements

Migration 25 is now required by readiness. Apply and verify it before this API candidate receives traffic. Staging/production deployment and authenticated checks there remain separate work. Official issuance, journal posting and tax claim/export from this form return unsupported-action errors. Legal effective dates still marked null are not historical legal conclusions.
