# 08: Owner/Admin เชิญ Member และผู้รับตอบรับคำเชิญ

**What to build:** Owner หรือ Admin สามารถสร้าง shareable invitation ให้ผู้ใช้ใหม่หรือเดิมยืนยันตัวตนและตอบรับเป็น Member ได้ โดย token ไม่ค้างใน URL หรือใช้ซ้ำ

**Blocked by:** 02 — ผู้ใช้ใหม่เข้าสู่ระบบด้วย Google และสร้าง Organization แรก

**Status:** ready-for-agent

- [ ] เปิด fresh Codex context ก่อนเริ่มงาน ticket นี้
- [ ] Owner/Admin สร้าง ดู และ revoke invitation ได้; Member ถูกปฏิเสธ และ invitation ให้ role Member เท่านั้น
- [ ] invitation ใช้ 256-bit random token, เก็บเฉพาะ hash, แสดง plaintext ครั้งเดียว และหมดอายุเริ่มต้นใน 24 ชั่วโมง
- [ ] การเปิดลิงก์ valid แลก URL secret เป็น server-bound invite intent ทันทีแล้ว redirect ไป clean URL ที่ไม่รั่วผ่าน referrer/third-party scripts
- [ ] invite onboarding แสดง Organization, inviter, Member role และแยกขั้น authenticate ออกจากปุ่มยืนยันตอบรับอย่างชัดเจน
- [ ] Google external-browser flow ใช้ hash-only one-time handoff อายุ 5 นาทีที่ให้สิทธิ์เฉพาะ invite intent และใช้ link provider หรือยกระดับ role ไม่ได้
- [ ] redemption lock invitation, re-check inviter authority และสร้าง Membership + consume invitation ใน transaction เดียว
- [ ] first-time invited User ไม่ถูกส่งไปสร้าง personal Organization ก่อน และ existing-member redemption เป็น idempotent
- [ ] expired, revoked, malformed, unknown และ replayed invitation ใช้ non-enumerating error เดียวพร้อม restart/help action
- [ ] concurrent redemption จากคนละ User มีผู้ชนะหนึ่งคน; inviter role loss ก่อน accept ต้อง fail closed
- [ ] indexes รองรับ token lookup, pending invitations และ expiry cleanup โดยไม่มี per-invite lookup ใน list
- [ ] audit บันทึก create/revoke/redeem และ tests ครอบคลุม LINE/Google authentication, clean URL, handoff, expiry, replay และ concurrency

