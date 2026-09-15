# 14: ระบบปฏิเสธการเข้าถึงข้าม tenant

**What to build:** ผู้โจมตีที่เปลี่ยน Organization UUID หรือ object identifiers ไม่สามารถอ่าน เขียน ลบ หรืออนุมานข้อมูลของ Organization ที่ตนไม่มี Membership ทั้งใน Go authorization และ PostgreSQL RLS

**Blocked by:** 07 — ผู้ใช้สร้าง Organization เพิ่ม; 08 — Owner/Admin เชิญ Member และผู้รับตอบรับคำเชิญ; 12 — Owner/Admin เชื่อมและยกเลิกการเชื่อม LINE group

**Status:** ready-for-agent

- [ ] เปิด fresh Codex context ก่อนเริ่มงาน ticket นี้
- [ ] ทุก tenant request resolve User จาก session และ current Membership จาก requested Organization UUID ฝั่ง server; ไม่รับ Organization authority จาก header, cookie, form หรือ client token
- [ ] UUID/object-ID tampering ถูกปฏิเสธสำหรับ Organization, Membership, invitation และ LINE group connection ทั้ง read, write และ delete paths
- [ ] non-member และ unknown Organization ให้ generic not-found-style response เดียวโดยไม่เปิดเผยชื่อ จำนวนสมาชิก สถานะ invite/group หรือความมีอยู่
- [ ] tenant-owned tables มี non-null Organization identifier และ composite foreign keys/uniqueness ปฏิเสธ relationship ที่ Organization ไม่ตรงกัน
- [ ] API runtime และ worker runtime ไม่มี BYPASSRLS; transaction ตั้ง local User/Organization context หลัง session + Membership validation เท่านั้น
- [ ] direct SQL tests ภายใต้ runtime roles พิสูจน์ว่า RLS ปฏิเสธ cross-tenant access แม้ application check ถูกข้าม ขณะที่ migration-owner ใช้ได้เฉพาะ migration proof
- [ ] denied request ไม่เปลี่ยน target data; audit อนุญาตเฉพาะ safe denial category ที่ไม่ทำให้ enumerate tenant ได้
- [ ] representative Membership/session/invite/code/Organization-list query plans ใช้ indexes ที่กำหนด และ query-count assertions ป้องกัน N+1 โดยไม่ pin exact planner output
- [ ] authenticated tenant-read smoke target วัด p95 ต่ำกว่า 300ms ภายใต้ local/staging load ที่ระบุ โดยไม่รวม provider redirects
- [ ] migration up/down/up และ catalog tests ยืนยัน RLS policies, runtime roles, constraints และ indexes ที่ journey ทั้งหมดต้องใช้
- [ ] HTTP/PostgreSQL security suite ใช้ Users และ Organizations หลายชุดเพื่อพิสูจน์ IDOR, stale role/removal และ cross-tenant read/write/delete rejection
