# 04: ผู้ใช้ LINE เดิมเชื่อม Google เป็น login provider ที่สอง

**What to build:** ผู้ใช้ที่ยืนยันตัวตนด้วย LINE อยู่แล้วสามารถ re-authenticate และเชื่อม Google เข้ากับ User เดิมโดยไม่เปิดทางให้ browser handoff หรือ callback เปลี่ยนเจ้าของ account

**Blocked by:** 02 — ผู้ใช้ใหม่เข้าสู่ระบบด้วย Google และสร้าง Organization แรก; 03 — ผู้ใช้จัดการ session และออกจากระบบอย่างปลอดภัย

**Status:** ready-for-agent

- [ ] เปิด fresh Codex context ก่อนเริ่มงาน ticket นี้
- [ ] linking เริ่มได้เฉพาะ authenticated session ที่ recent-auth ไม่เกิน 10 นาที และ transaction ระบุ purpose เป็น link
- [ ] transaction ผูก initiating User และ session; callback จาก session อื่น, anonymous browser หรือ transaction ที่หมด recent-auth ต้องถูกปฏิเสธ
- [ ] เมื่อ Google ต้องเปิด system browser ผู้ใช้ต้อง login เป็น PlaiFlow User เดิมใน browser นั้นก่อนเริ่ม link; handoff token เพียงอย่างเดียวไม่มีสิทธิ์ link
- [ ] successful callback เพิ่ม Google AuthIdentity ให้ User เดิมโดยไม่สร้าง User หรือ Membership ใหม่ และ rotate session ID
- [ ] database บังคับ global uniqueness ของ `(issuer, subject)` และมี active identity ได้ไม่เกินหนึ่งรายการต่อ provider ต่อ User รวมถึงกรณี concurrent callbacks
- [ ] provider access/ID token ถูกทิ้งหลัง validation/profile extraction และไม่ปรากฏใน browser storage, URL, frontend logs หรือ audit metadata
- [ ] account settings แสดง linked, available, re-auth required และ recoverable provider errors โดยไม่แสดง issuer/subject
- [ ] audit แยก link success/failure category และมี request ID โดยไม่เปิดเผยว่า email ใดมี account อยู่แล้ว
- [ ] acceptance test พิสูจน์ LINE-first User เชื่อม Google แล้ว login ด้วย provider ใดก็กลับเข้า User และ Membership ชุดเดิม

