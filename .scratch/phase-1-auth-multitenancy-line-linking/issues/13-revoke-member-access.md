# 13: Owner/Admin ถอนสิทธิ์ Member

**What to build:** Owner หรือ Admin สามารถลบ Member ออกจาก Organization และการเข้าถึง tenant นั้นหยุดตั้งแต่ request ถัดไปโดยไม่ทำลาย session หรือ Membership ของ Organization อื่น

**Blocked by:** 09 — Owner มอบสิทธิ์ Admin และโอน ownership

**Status:** ready-for-agent

- [ ] เปิด fresh Codex context ก่อนเริ่มงาน ticket นี้
- [ ] Owner/Admin ลบ Member ได้; Member ลบผู้อื่นไม่ได้ และ Admin ลบ Admin/Owner ไม่ได้
- [ ] removal ใช้ current Membership/role และ transaction ที่ป้องกัน stale/concurrent authorization
- [ ] หลัง removal request ถัดไปของ User ต่อ Organization เดิมถูกปฏิเสธ แม้ browser session เดิมยังไม่หมดอายุ
- [ ] session ของ User ไม่ถูก revoke ทั้งหมดและยังเข้าถึง Organization อื่นที่มี active Membership ได้
- [ ] pending invitation, group-link หรือ privileged action ที่อาศัย role/Membership เดิม re-check แล้ว fail closed
- [ ] non-member response ไม่เปิดเผย Organization name, member count หรือการมีอยู่ของ tenant
- [ ] membership lookup ใช้ index `(user_id, organization_id)` และ removal ไม่ trigger per-Organization/session scans
- [ ] denied/removal outcome ถูก audit แบบ append-only โดยไม่มี session หรือ invitation/link secret
- [ ] tests ครอบคลุม Owner/Admin success, Member denial, Admin boundary, next-request enforcement, stale pending action และ unaffected other-Organization access

