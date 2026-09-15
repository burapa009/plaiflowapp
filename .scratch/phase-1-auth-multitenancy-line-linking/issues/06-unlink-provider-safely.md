# 06: ผู้ใช้ยกเลิก provider โดยไม่ล็อกตัวเองออก

**What to build:** ผู้ใช้ที่มีสอง login providers สามารถยกเลิกหนึ่ง provider หลังยืนยันด้วย provider ที่จะเก็บไว้ โดยระบบป้องกัน lockout และการนำ identity ไปสร้าง account ใหม่ทันที

**Blocked by:** 04 — ผู้ใช้ LINE เดิมเชื่อม Google เป็น login provider ที่สอง

**Status:** ready-for-agent

- [ ] เปิด fresh Codex context ก่อนเริ่มงาน ticket นี้
- [ ] unlink ถูกปฏิเสธเมื่อเหลือ active AuthIdentity เพียงหนึ่งรายการ
- [ ] unlink ต้องมี re-authentication ภายใน 10 นาทีด้วย identity ที่จะคงอยู่; re-auth ด้วย identity ที่กำลังลบไม่เพียงพอ
- [ ] successful unlink ปิด identity ทันที, revoke session อื่นทั้งหมด และ rotate current session
- [ ] เก็บ non-authenticating tombstone 30 วันภายใต้ global `(issuer, subject)` uniqueness เพื่อห้าม identity สร้าง User ใหม่ระหว่าง cooldown
- [ ] ระหว่าง cooldown identity relink ได้เฉพาะ User เดิมหลัง login ผ่าน remaining identity และ tombstone expiry ทำตาม retention policy
- [ ] Owner ที่มี provider เดียวได้รับ recovery warning แต่ไม่ถูกบังคับให้เชื่อม provider ใหม่; ไม่มี password/magic-link fallback เพิ่มใน Phase 1
- [ ] account settings แสดง unlink blocked, re-auth required, cooldown และ recovery states โดยไม่เปิดเผย provider subject
- [ ] audit บันทึก unlink, blocked attempt, relink และ session revocation โดยไม่มี token/raw provider data
- [ ] tests ครอบคลุม last-identity rejection, wrong re-auth identity, successful unlink, session rotation/revocation, cooldown login rejection, safe relink และ tombstone expiry

