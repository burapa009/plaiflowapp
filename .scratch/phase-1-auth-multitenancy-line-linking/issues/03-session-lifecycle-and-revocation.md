# 03: ผู้ใช้จัดการ session และออกจากระบบอย่างปลอดภัย

**What to build:** ผู้ใช้ที่เข้าสู่ระบบแล้วสามารถใช้งานต่อภายในอายุ session, ออกจากอุปกรณ์ปัจจุบันหรือทุกอุปกรณ์ และได้รับ recovery ที่ชัดเจนเมื่อ session หมดอายุ

**Blocked by:** 01 — ผู้ใช้ใหม่เข้าสู่ระบบด้วย LINE และสร้าง Organization แรก

**Status:** ready-for-agent

- [ ] เปิด fresh Codex context ก่อนเริ่มงาน ticket นี้
- [ ] session เป็น opaque 256-bit token, เก็บเฉพาะ hash และไม่บรรจุ Organization หรือ role authority
- [ ] idle expiry เท่ากับ 14 วัน, absolute lifetime เท่ากับ 30 วัน และ `last_seen` update ไม่เกินหนึ่งครั้งต่อ 5 นาที
- [ ] login และ re-authentication rotate session ID; session fixation test พิสูจน์ว่า pre-auth token ไม่ถูกยกระดับใช้ต่อ
- [ ] logout revoke session ปัจจุบันและล้าง cookie; logout-all revoke ทุก session ของ User โดยไม่ออก replacement session
- [ ] expired, revoked และ unknown session ให้ผลต่อ client เหมือนกัน พร้อมหน้า session-expired ที่มี recovery action เดียว
- [ ] ทุก mutation ตรวจ exact same-origin และ session-bound CSRF token นอกเหนือจาก SameSite cookie; missing, malformed, cross-session token หรือผิด Origin ต้องไม่เกิด mutation
- [ ] sensitive action ใช้ recent-auth window 10 นาที และ re-auth เก็บต่อได้เฉพาะ pending action ที่ allowlist ไว้
- [ ] session indexes รองรับ token lookup, User-wide revocation และ expiry cleanup; query ไม่ต้อง scan session history ทั้งหมด
- [ ] audit session revocation แบบ append-only โดยไม่เก็บ token hash เต็ม, cookie หรือ CSRF secret
- [ ] HTTP/PostgreSQL tests ครอบคลุม idle/absolute expiry, logout, logout-all, rotation, CSRF และ throttled `last_seen`

