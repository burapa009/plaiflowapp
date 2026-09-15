# 09: Staging พิสูจน์ LINE message สังเคราะห์แบบ end-to-end

**What to build:** staging แยกจาก local และ production สามารถ deploy migration, API, worker และ dashboard ตามลำดับ แล้วพิสูจน์ valid synthetic LINE message ตั้งแต่ webhook ถึง dashboard ได้

**Blocked by:** 05 — ผู้ดูแลเห็นสถานะ Inbound Event pipeline และสุขภาพระบบจาก Dashboard; 06 — Go และ Python ตรวจสอบ export contract เดียวกันได้; 07 — ระบบปฏิเสธ auth callback configuration ที่ไม่ปลอดภัย; 08 — ข้อมูล Inbound Event หมดอายุตามนโยบายโดยไม่ทำลาย duplicate protection

**Status:** ready-for-agent

- [ ] เปิด fresh Codex context ก่อนเริ่มงาน ticket นี้
- [ ] Vercel/Railway configuration แยก environment และ Vercel dashboard protection เปิดใช้งาน
- [ ] staging database ใช้ private/TLS connection, role แยก และไม่มี production secret reuse
- [ ] deploy order และ migration failure handling ถูกตรวจสอบก่อนเปิด traffic
- [ ] smoke test ยืนยัน signature verification, idempotency, worker state, readiness และ dashboard counts
- [ ] ตรวจ baseline latency/error metrics และไม่ทำ long-running work ใน webhook
- [ ] LINE Verify/redelivery และ external deployment side effects ทำเมื่อมี explicit approval เท่านั้น
