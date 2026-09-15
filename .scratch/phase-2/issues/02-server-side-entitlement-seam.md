# 02: Server-side Entitlement seam ควบคุม Task creation ได้

**What to build:** ระบบประเมิน capability ของ Organization ฝั่ง server หลัง authentication/RBAC และสามารถอนุญาตหรือปฏิเสธ Task creation ได้โดย frontend plan state ไม่เคยให้สิทธิ์ ขณะที่การอ่านข้อมูลเดิมและ security-critical actions ยังทำงานโดยไม่ถูก commercial gate

**Blocked by:** 01 — ผู้ใช้สร้าง มอบหมาย และเปิดดู Task แรกได้

**Status:** ready-for-agent

- [ ] เปิด fresh Codex context ก่อนเริ่มงาน ticket นี้
- [ ] มี interface shell ขนาดเล็กสำหรับ Plan display metadata, Entitlement decision และ idempotent Usage recording โดยยังไม่มี billing provider หรือ subscription synchronization
- [ ] feature-gate seam รับ Organization Context, allowlisted capability และ requested units แล้วคืน allowed/denied พร้อม optional limit/used/remaining/reset และ stable reason code
- [ ] Task creation เรียก gate หลัง session, Membership และ RBAC ผ่านแล้วเท่านั้น
- [ ] Phase 2 default provider อนุญาตแบบ unlimited และไม่มี commercial limit จริง
- [ ] test provider สามารถปฏิเสธ Task creation ที่ authorized ได้ และ UI แสดง safe feature-unavailable state
- [ ] hidden button, plan badge, request field หรือ browser state ไม่สามารถข้าม server decision ได้
- [ ] denied/failed operation ไม่บันทึก Usage; successful operation บันทึกด้วย idempotency key และ retry ไม่เพิ่มซ้ำ
- [ ] entitlement-provider failure ปิดเฉพาะ optional gated mutation และไม่ลดทอน RBAC
- [ ] login/session, Organization switching, authorized reads, identity/session security และ ownership/member controls ไม่เรียก commercial gate
- [ ] ไม่มี plan table, pricing UI, checkout หรือ durable commercial Usage ledger ใน ticket นี้

