# 05: ผู้ดูแลเห็นสถานะ Inbound Event pipeline และสุขภาพระบบจาก Dashboard

**What to build:** ผู้ดูแลเปิด dashboard แล้วเห็น live counts ของ Inbound Event, สถานะ API/database/worker และ state ของ loading, empty และ error อย่างอ่านง่ายบน desktop และ mobile

**Blocked by:** 03 — LINE event ที่ส่งซ้ำถูกเก็บและประมวลผลเพียงครั้งเดียว; 04 — Inbound Event ที่ล้มเหลวหรือ worker ทิ้งงานสามารถฟื้นตัวได้

**Status:** ready-for-agent

- [ ] เปิด fresh Codex context ก่อนเริ่มงาน ticket นี้
- [ ] Go aggregate endpoint และ readiness data แสดงสถานะที่ dashboard ใช้ได้จริง
- [ ] Next server-to-Go call ใช้ service token ฝั่ง server เท่านั้น
- [ ] token validation มี scope จำกัด, rotation seam, constant-time comparison และ rate limit
- [ ] dashboard shell เป็นมิตร เข้าถึงได้ และมี loading/empty/error states ที่สอดคล้องกัน
- [ ] ไม่มี real user authentication ใน ticket นี้
