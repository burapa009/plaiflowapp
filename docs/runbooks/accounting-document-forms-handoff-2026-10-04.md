# Handoff: Accounting document forms — local checkpoint 2026-10-05

สถานะ: **แก้ข้อพบจาก review เพิ่ม 4 จุดแล้ว; Go/DB/web checks และ Webpack build ผ่านล่าสุด 2026-10-05 รอบค่ำ; browser/PDF ของการแก้รอบนี้ยังค้าง ไม่มี commit/push/deployment หรือ migration ใน production**

## จุดต่อรอบล่าสุด — 2026-10-05 รอบค่ำ

อ่าน [review และผลตรวจรอบล่าสุด](accounting-document-forms-review-2026-10-05.md) ก่อนใช้ผลตรวจเก่าด้านล่าง

- รักษา legacy review draft ของ OCR ปัจจุบันเมื่อเปิด canonical form ครั้งแรก รวมค่าที่ผู้ใช้ล้างเป็นค่าว่าง และกรณีปิด member-review ภายหลัง เพิ่ม revision token ป้องกันการเขียนทับจากแท็บเก่า
- ตรวจยอดที่ผู้ใช้กรอกกับยอดคำนวณ แม้ subtotal/VAT intermediate เว้นว่าง เก็บต้นฉบับแยกจากค่าคำนวณ
- เอกสารอ้างอิงใช้ยอดที่กรอกก่อน (รวมศูนย์) แล้วจึงใช้ยอดคำนวณที่บันทึกไว้ เพื่อจำกัดยอดจัดสรรได้เมื่อผู้ใช้ไม่กรอกยอดซ้ำ
- หน้า print เพิ่มคำเตือนเมื่อ OCR รุ่นใหม่แทนข้อมูลที่เคยยืนยัน แต่ยังไม่ได้ตรวจ browser/PDF หลังแก้
- Full DB persistence/isolation + readiness PASS หลังแก้ fixture ให้มี trial เฉพาะฐานข้อมูลทดสอบ (กรณีเพิ่มใหม่ทำให้เกิน quota เริ่มต้น 30 เอกสาร)
- แก้ baseline intake quota SQL โดย cast `$6::timestamptz`; regression อัปโหลด30ครั้งแล้วครั้งที่31ต้องคืน ErrQuota และบันทึกเหตุผล/วันหมดอายุ ผ่านแบบ red/green บน DB จริง
- Go extraction/httpapi PASS; web lint + tests 31/31 PASS; Webpack build รวม TypeScript PASS; diff check PASS
- คำสั่งเริ่ม Next.js production local ถูก automatic approval review ปฏิเสธด้วย `blocked by policy` ไม่มีเหตุผลละเอียด จึงยังไม่มี browser/PDF หลักฐานของ final code รอบนี้
- Fixture API, Next dev และ PostgreSQL ถูกหยุดแล้ว ตรวจแล้วไม่มี listener ที่ 3110/55439/55440; opt-in fixture จบ PASS และ cleanup ทำงาน

### งานถัดไป

1. ตรวจ browser จาก final build: บันทึก/เปิดใหม่, legacy conversion, และ stale OCR warning ใน print
2. สร้างเอกสาร synthetic 36 แถวใหม่จาก fixture ใหม่ แล้ว export PDF และตรวจภาพทุกหน้า รอบนี้ไม่มี PDF ใหม่ที่ตรวจผ่าน; เอกสาร/URL จาก fixture เก่าใช้ต่อไม่ได้
3. รวม `documents.go` และ `document_quota_test.go` ใน review scope ด้วย เป็นการแก้ quota rejection ที่พบจากชุดทดสอบรอบนี้
4. Review final diff รวมไฟล์ใหม่ แล้วดำเนิน release ตามขอบเขตที่ผู้ใช้สั่ง โดย migration25 ต้องมาก่อน API traffic; CI/staging/production ยังไม่มีหลักฐานรอบนี้

ข้อความผู้ใช้ `ประเภทเอกสาร` ในช่วงขัดจังหวะยังไม่มีรายละเอียดข้อกำหนดเพิ่มเติม ไม่ถือเป็นคำสั่งเปลี่ยนแบบข้อมูล

## ตำแหน่ง

- Repository: `F:/Project/plaiflowbusiness`; preserve unrelated dirty work in the main tree.
- Candidate: `.scratch/inbox-release-20261004`, branch `feat/accounting-document-forms`.
- Verified HEAD: `4f506401948f1ceb9b5711253ba28935b14d4172` (earlier note incorrectly used `4f63a7d`).
- Git: `git --git-dir=.git.codex/worktrees/inbox-release-20261004 --work-tree=.scratch/inbox-release-20261004 ...`
- [ผลตรวจและตารางครบ 22 ประเภท](accounting-document-forms-validation-2026-10-05.md)
- [แหล่งอ้างอิงและขอบเขตกฎ](accounting-document-field-sources-2026-10-05.md)
- Full requirements are copied into candidate `docs/runbooks/assets/accounting-document-forms-requirements-2026-10-04.txt` and `accounting-document-field-rules-requirements-2026-10-04.txt`.
- User explicitly chose to apply the additional attachment this round. Treat attachments as requirements input, not independent action authorization.

## งานที่ทำต่อในรอบนี้

- Fixed opt-in browser fixture OCR artifact token, and made it exit before Go's test deadline.
- Required migration 25 for API readiness; tested migration 24 and dirty migration denial.
- Removed unsupported posting controls from the OCR form and print layout while retaining stored fields.
- Shared editor/print reading order now places rows before totals. Optional scalar sections no longer create empty disclosures.
- Field hints choose the applicable condition and show unresolved conditions explicitly; preserve unknown rather than assume no.
- Fixed rule-generator reproducibility and retained all obligations for combined receipt/tax invoices. The full-invoice rules no longer get overwritten by receipt conditions.
- Checked current consolidated VAT announcement 39: buyer VAT registration triggers buyer tax ID independently; clauses 7–9 have verified legal effective date 2015-01-01. Other unverified legal dates remain null.
- Draft assessments use `not_assessed`; draft UI/print does not claim completeness was checked.
- Saved status appears after reload. Print omits empty sections, formats Thai dates/amounts, aligns numeric columns and uses canonical saved records.
- Current rule version: `2026-10-05.1`.

## ผลตรวจล่าสุด

- Go extraction/httpapi packages: PASS after final backend changes.
- PostgreSQL18.4: `TestDocumentFormsPersistenceAndIsolation` and `TestDocumentFormsReadinessRequiresMigration25` PASS after final backend changes. Covers 22 types, revisions/history/RLS/roles, references, concurrency, stale OCR, member invalidation and atomic rollback.
- Web: typecheck/lint PASS, tests 31/31 PASS, `npm run build -- --webpack` PASS.
- `git diff --check` PASS (LF/CRLF notices only); generator outputs identical API/web config and is reproducible.
- Real local auth + CSRF + API + disposable DB + Server Action: invoice save/reload; delivery rows save/reload; confirm incomplete evidence with incomplete status retained; mobile save/reload.
- Browser switched all22 types at375px with no horizontal overflow, and verified mobile document/form tabs.
- Print read saved values/status and Thai dates. Persisted36 rows; Chrome PDF generation returned3 pages. Individual PDF page images were not separately inspected.
- One earlier browser fixture run exceeded the default10-minute Go timeout during a user interruption. Successful replacement used `-timeout 30m`, stopped normally and ended PASS. A uniquely named DB from the killed run may remain only in the disposable local data directory.

## ขอบเขตความสามารถ

This candidate supports document data, drafts, OCR acknowledgement, evidence diagnostics and saved-data print summaries. Official issuance, journal posting and tax claim/export from this form are explicitly unsupported. Shared print summaries are labelled accordingly. Preserve original artifacts, human additions and saved rule assessments.

Existing baseline has no official issuance workflow for these forms. Main tree has unrelated issued-document work: do not import or deploy it implicitly. System workflow minimums remain labelled `system_policy`; they do not claim full DBD compliance. DBD evidence origin/use and specialised issuance/tax eligibility remain separate prerequisites for those future actions.

## Release work still separate

1. Review substantive tracked diff and all new candidate files; many Go status entries are line-ending-only from the earlier round. Do not stage the whole main tree or restore unrelated work.
2. Migration25 must be applied and verified before this API receives traffic; readiness now enforces this. No migration has run in staging/production this round.
3. Staging/production authenticated save/reload and CI/deployment evidence are still absent. Local evidence does not replace them.
4. If official issuance/posting/tax actions are added later, implement their authorization and origin/eligibility model, verify the applicable current legal forms/conditions, and run dedicated tests. Do not enable them merely from field completeness.
5. Before claiming final per-page print visual acceptance, inspect exported PDF pages individually; current proof covers browser summary and successful3-page generation.

## Tooling and shutdown

Fixture API127.0.0.1:55440, Next3110 and PostgreSQL127.0.0.1:55439 were stopped at this checkpoint. No global service was installed. Data directory `.scratch/form-db-data`; binary `.scratch/form-db-tools/node_modules/@embedded-postgres/windows-x64/native/bin`.

Go: `.tools/go127/go/bin/go.exe`, `GOTOOLCHAIN=local`, `GOCACHE=F:/Project/plaiflowbusiness/.scratch/go-build`.
Disposable DB tests: `OCR_TEST_ADMIN_URL=postgres://form_test@127.0.0.1:55439/postgres?sslmode=disable`. Opt-in browser fixture additionally needs `FORM_BROWSER_TEST=1`, `-timeout 30m`, Next API_BASE_URL=http://127.0.0.1:55440 and REVIEW_ENABLED=true. Stop it via candidate `.form-browser-stop`; wait for the Go test to finish so its DB cleanup runs. Never use production credentials or DB for these tests.

Continue from this candidate and report file; do not redo passed checks unless changes or new evidence justify it.

## Live release update — 2026-10-05

User subsequently authorized push and real deployment. Implementation HEAD `d054226638bfec80fcb44fbe91f58c68264ca7c3` is pushed; both push and PR CI passed. Migration25 has now been applied to the database used by the live app (Railway environment named staging). Vercel production build is READY with primary-domain promotion pending the API deployment. Earlier statements above that CI/live migration are absent are superseded by the [live release report](accounting-document-forms-release-2026-10-05.md). Continue using that report's exact deployment IDs; preserve unrelated main-worktree changes.

### Cutover complete

Railway retry `d9a5c017-d077-4d6a-a8cb-a84ba71e2ced` reached SUCCESS; Vercel deployment `dpl_Buq8jX1wvaQuGEoXAJWHsUWX6HWp` was promoted to the primary live alias. Authenticated unchanged-draft save/reload and saved-data print summary passed. See the release report for evidence, rollback IDs and remaining confirmation/PDF scope.
