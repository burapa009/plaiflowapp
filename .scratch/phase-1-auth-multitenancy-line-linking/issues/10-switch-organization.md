# 10: ผู้ใช้สลับ Organization

**What to build:** ผู้ใช้ที่เป็นสมาชิกหลาย Organization สามารถเลือกและนำทางไปยัง Organization ที่ได้รับสิทธิ์ โดยไม่เปลี่ยน session หรือเชื่อ tenant identifier จาก client

**Blocked by:** 07 — ผู้ใช้สร้าง Organization เพิ่ม

**Status:** ready-for-agent

- [ ] เปิด fresh Codex context ก่อนเริ่มงาน ticket นี้
- [ ] หลัง login: ไม่มี Organization แสดง first-setup, มีหนึ่งรายการเข้าได้โดยตรง และมีหลายรายการแสดง chooser เมื่อไม่มี safe return path
- [ ] chooser/switcher แสดงเฉพาะ Organization จาก current Membership และรองรับชื่อไทยยาวโดยไม่มี horizontal scroll
- [ ] switch เป็น navigation ไปยัง Organization UUID ที่เลือกและไม่เขียน current Organization ลง session, cookie หรือ User record
- [ ] ทุกปลายทาง resolve User จาก session แล้ว lookup `(user_id, organization_id)` และ role ปัจจุบันก่อนสร้าง Organization Context
- [ ] safe return path ใช้ได้เฉพาะ application route ที่ allowlist และไม่เปิด open redirect หรือ tenant spoofing
- [ ] Organization list/switch ใช้หนึ่ง indexed Membership-to-Organization query พร้อม bounded query-count assertion และไม่มี N+1
- [ ] UI ผ่าน keyboard, focus, 44px touch target และ responsive checks ที่ 375/768/1024/1440px
- [ ] tests ครอบคลุม zero/one/many Organizations, stale Membership, long Thai names และ session ที่ไม่เปลี่ยนหลัง switch

