# 11: ผู้ใช้ Google เดิมเชื่อม LINE user identity

**What to build:** ผู้ใช้ที่เข้าสู่ระบบด้วย Google สามารถยืนยันตัวตนล่าสุดและเชื่อม LINE Login AuthIdentity เพื่อเตรียมพิสูจน์ sender ใน LINE group โดยไม่ใช้ Messaging API native account linking

**Blocked by:** 04 — ผู้ใช้ LINE เดิมเชื่อม Google เป็น login provider ที่สอง

**Status:** ready-for-agent

- [ ] เปิด fresh Codex context ก่อนเริ่มงาน ticket นี้
- [ ] Google-first User เริ่ม LINE link ได้เฉพาะ authenticated recent session และ callback ผูกกับ initiating User/session
- [ ] successful link เพิ่ม LINE AuthIdentity ให้ User เดิม, rotate session และไม่สร้าง User/Organization/Membership ใหม่
- [ ] ระบบใช้ canonical LINE issuer + `sub` และไม่ใช้ display name, email หรือ picture เป็น identity
- [ ] readiness check ยืนยันว่า LINE Login และ Messaging API channels ของ environment อยู่ใต้ LINE Provider เดียวกันก่อนเปิด group linking
- [ ] development, staging และ production configuration แยกกัน และ incompatible/missing configuration fail closed
- [ ] LINE identity ที่เป็นของ User อื่นหรืออยู่ใน unlink cooldown แสดง conflict/recovery state โดยไม่ merge หรือ reassign
- [ ] ไม่มี Messaging API native account-linking flow, password fallback หรือ provider token storage เพิ่มเข้ามา
- [ ] audit/log/UI ไม่แสดง LINE subject, access/ID token หรือ raw claims
- [ ] acceptance test พิสูจน์ว่า LINE Login `sub` ที่เชื่อมแล้วเป็น expected LINE subject สำหรับ ticket group linking ถัดไป

