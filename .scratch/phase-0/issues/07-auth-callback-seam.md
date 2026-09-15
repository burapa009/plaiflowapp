# 07: ระบบปฏิเสธ auth callback configuration ที่ไม่ปลอดภัย

**What to build:** ระบบมี provider-neutral seam สำหรับ LINE Login และ Google Login พร้อม callback/redirect configuration แยกตาม environment และยังคงปิดการ login จริงไว้

**Blocked by:** 01 — นักพัฒนาเปิด PlaiFlow ในเครื่องได้จาก fresh checkout

**Status:** ready-for-agent

- [ ] เปิด fresh Codex context ก่อนเริ่มงาน ticket นี้
- [ ] provider abstraction และ callback route seam compile/test ได้โดยไม่มี provider enabled
- [ ] redirect URI ใช้ explicit allowlist แยก local/staging และปฏิเสธ open redirect
- [ ] secrets ไม่อยู่ใน source, logs หรือ client bundle
- [ ] บันทึก prerequisites สำหรับ state, PKCE, callback expiry และ provider-specific validation
- [ ] ไม่มี account creation, session, token exchange หรือ real authentication
