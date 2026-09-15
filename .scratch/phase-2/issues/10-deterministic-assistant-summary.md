# 10: Assistant Summary สรุป Work Dashboard อย่างตรวจสอบได้

**What to build:** ผู้ใช้ขอ Assistant Summary ภาษาไทยจาก Work Dashboard read model แล้วได้รับข้อความสั้นที่ตัวเลขและ deep links ตรวจสอบย้อนกลับได้ตาม role โดยไม่มี LLM, data egress, persistence หรือการเปลี่ยน Task อัตโนมัติ

**Blocked by:** 02 — Server-side Entitlement seam ควบคุม Task creation ได้; 09 — Work Dashboard แสดง verified counts ตาม role

**Status:** ready-for-agent

- [ ] เปิด fresh Codex context ก่อนเริ่มงาน ticket นี้
- [ ] Assistant Summary ใช้ authorized Work Dashboard read-model seam เดียวและไม่สร้าง query family ที่ตีความ Task ต่างออกไป
- [ ] Member summary ครอบคลุม assigned overdue, due today/next seven days, Urgent และ watched Tasks ที่เกี่ยวข้อง
- [ ] Owner/Admin summary เพิ่ม unassigned, overdue และ Urgent team work โดยไม่มี ranking รายบุคคล
- [ ] output เป็น deterministic Thai template สูงสุดแปด bullet points พร้อม exact counts, generated_at, scope และ authorized deep links
- [ ] empty dataset ให้ผล no-action-needed ที่ชัดเจนแทน error หรือข้อความประดิษฐ์
- [ ] ไม่มี external LLM/network call, prompt, persisted summary, schedule, cache หรือ autonomous action
- [ ] summary ไม่สร้าง/แก้ Task, Reminder, Notification, preference หรือ Export Job และไม่ให้ OCR/accounting/financial advice
- [ ] Assistant capability ผ่าน feature gate ฝั่ง server หลัง Organization Context/RBAC; frontend plan state ข้าม gate ไม่ได้
- [ ] role/Membership ที่เปลี่ยนมีผลใน request ถัดไปและ deep link แต่ละรายการ reauthorize
- [ ] API มีเป้าหมาย p95 ไม่เกิน 500 ms ที่ reference data
- [ ] tests พิสูจน์ deterministic output, eight-bullet cap, exact counts, empty state, role-sensitive omission, cross-tenant denial และไม่มี external call

