# 11: Owner/Admin ส่งออก filtered Tasks เป็น formula-safe CSV

**What to build:** Owner หรือ Admin ส่งออกผล Task filters ปัจจุบันไม่เกิน 5,000 แถวเป็น CSV ภาษาไทยที่ stream โดยตรง เปิดใน spreadsheet ได้ และทุก user-controlled cell ถูกป้องกัน formula injection โดยไม่ข้าม tenant หรือเปิดเผยข้อมูลนอก scope

**Blocked by:** 02 — Server-side Entitlement seam ควบคุม Task creation ได้; 06 — ผู้ใช้กรองและแบ่งหน้ารายการ Task ได้อย่างเสถียร

**Status:** ready-for-agent

- [ ] เปิด fresh Codex context ก่อนเริ่มงาน ticket นี้
- [ ] Owner/Admin export ได้หลัง session, Organization Context, RBAC และ Entitlement ผ่าน; Member และ cross-tenant request ถูกปฏิเสธแบบไม่เปิดเผยข้อมูล
- [ ] server normalize filters เดียวกับ Task list และไม่เชื่อ frontend row count, result rows, SQL, sort หรือ column names
- [ ] unfiltered request หมายถึงทุก Task ใน Organization ภายใต้ threshold; date ranges เป็น inclusive ตาม Organization Timezone
- [ ] fixed CSV columns/order ตรง Phase 2 spec และไม่รวม Domain Event, Notification, audit, Membership list, External Identity หรือ LINE identifiers
- [ ] 0–5,000 rows ถูก stream synchronously จาก Go โดยไม่ buffer Task ทั้งชุดและไม่ activate Python export shell
- [ ] output ใช้ UTF-8 BOM, RFC 4180-compatible quoting, CRLF rows, YYYY-MM-DD Due Date และ RFC 3339 UTC timestamps
- [ ] user-controlled string ที่หลัง leading whitespace เริ่มด้วย =, +, -, @, tab, CR หรือ LF ถูก prefix single quote ก่อน CSV encoding
- [ ] formula protection ครอบคลุม title, description, Organization/display names และ watcher list แม้ค่าดูเหมือนตัวเลข
- [ ] filename และ Content-Disposition สร้างจาก safe server-controlled fragments; response เป็น attachment และป้องกัน content sniffing
- [ ] watcher names มี deterministic order; nullable values เป็น empty cells; export order เสถียรและไม่ truncate
- [ ] export-specific buffer ไม่เกิน 8 MiB และ synchronous generation ไม่ใช้ unbounded collection
- [ ] append-only audit ขั้นต่ำบันทึก requested/rejected/completed/failed โดยไม่มี CSV content, cells, signed URL หรือ storage key
- [ ] golden tests ครอบคลุม Thai/Unicode, comma, quote, CR/LF, blank, duplicate names และ injection cases ทุก prefix รวมถึง cross-tenant/role/filter tests

