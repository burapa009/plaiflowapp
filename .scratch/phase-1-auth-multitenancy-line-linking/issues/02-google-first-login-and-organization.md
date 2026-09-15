# 02: ผู้ใช้ใหม่เข้าสู่ระบบด้วย Google และสร้าง Organization แรก

**What to build:** ผู้ใช้ใหม่สามารถเลือก Google Login, ผ่าน Google OIDC อย่างปลอดภัย และสร้าง Organization แรกได้ด้วย auth/session/tenant seam เดียวกับ LINE

**Blocked by:** 01 — ผู้ใช้ใหม่เข้าสู่ระบบด้วย LINE และสร้าง Organization แรก

**Status:** ready-for-agent

- [ ] เปิด fresh Codex context ก่อนเริ่มงาน ticket นี้
- [ ] หน้า welcome แสดง Google เป็น official-brand secondary action พร้อม accessible name, keyboard/focus และ recovery state
- [ ] Google OIDC ใช้ Authorization Code Flow, state, nonce และ PKCE S256 ต่อ attempt พร้อม redirect URI ที่ตรงกับ environment
- [ ] authorization request ขอเฉพาะ `openid email profile`, ใช้ online access และไม่มี Google Drive/Sheets scope, offline access, refresh-token request หรือ incremental authorization
- [ ] callback ตรวจ signature จาก official metadata/JWKS, issuer, audience, expiry, nonce และ authorized party เมื่อเกี่ยวข้อง ก่อนใช้ `(issuer, subject)` เป็น AuthIdentity
- [ ] Google email และสถานะ email verification เป็น profile claim เท่านั้น ไม่ใช้หา User เดิมหรือ merge account
- [ ] Google Login ที่เริ่มจาก LINE embedded browser แสดงคำแนะนำเปิด system browser; transaction ใหม่เริ่มใน browser ปลายทางและไม่ใช้ token จาก browser เดิมเป็น login authority
- [ ] Google identity ใหม่สร้าง User, AuthIdentity, session และ Organization + Owner Membership ได้ครบ; identity เดิมไม่สร้าง User ซ้ำ
- [ ] provider timeout, denial, invalid token และ embedded-browser restriction กลับสู่ clean recovery page พร้อม request ID โดยไม่เปิดเผย provider parameters
- [ ] environment configuration ของ Google Login แยกจาก future Google Drive authorization และ frontend bundle/log ไม่มี provider token หรือ secret
- [ ] acceptance test ตรวจ authorization request และ end-to-end journey ด้วย Google provider double โดยไม่ใช้อินเทอร์เน็ตหรือ credential จริง

