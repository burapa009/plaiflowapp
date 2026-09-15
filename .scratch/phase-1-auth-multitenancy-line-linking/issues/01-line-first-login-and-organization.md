# 01: ผู้ใช้ใหม่เข้าสู่ระบบด้วย LINE และสร้าง Organization แรก

**What to build:** ผู้ใช้มือถือที่มาจาก LINE สามารถเริ่ม LINE Login, กลับมาพร้อม session ที่ปลอดภัย และสร้าง Organization แรกพร้อม Owner Membership ได้ใน journey เดียว

**Blocked by:** None (can start immediately)

**Status:** ready-for-agent

- [ ] เปิด fresh Codex context ก่อนเริ่มงาน ticket นี้
- [ ] หน้า welcome แบบ mobile-first แสดง LINE เป็น primary action โดยไม่เริ่ม OAuth อัตโนมัติ และมี loading, denial, expiry และ retry states ที่เข้าถึงได้
- [ ] LINE Login ใช้ Authorization Code OIDC พร้อม state, nonce และ PKCE S256 แบบสุ่ม แยกต่อ attempt อายุไม่เกิน 10 นาที และ callback claim transaction แบบ atomic ก่อนแลก code
- [ ] callback ตรวจ provider route, issuer, audience, expiry, nonce และ LINE ID token ฝั่ง server; return path จำกัดเฉพาะ relative application routes และ response กลับสู่ URL สะอาด
- [ ] identity ใหม่สร้าง User และ LINE AuthIdentity ด้วย `(issuer, subject)`; identity เดิมกลับเข้า User เดิม และ repeated/concurrent callback ไม่สร้างข้อมูลซ้ำ
- [ ] login ออก session token ใหม่ ไม่ upgrade pre-auth session เดิม และตั้ง host-only `__Host-` cookie ที่มี Secure, HttpOnly, SameSite=Lax, root path และไม่มี Domain
- [ ] first-Organization form ขอเฉพาะชื่อ และสร้าง Organization กับ Owner Membership ใน transaction เดียว; failure ต้อง rollback ทั้งคู่
- [ ] request แรกภายใต้ Organization ผ่าน server-side Membership resolution และ PostgreSQL RLS ด้วย runtime role ที่ไม่มี BYPASSRLS
- [ ] state, session และ Membership lookup มี constraints/indexes ตาม query ที่ใช้จริง และไม่มี per-row lookup
- [ ] audit บันทึก login และ Organization creation ด้วย safe reason/request ID โดยไม่เก็บ code, state, nonce, verifier, token, cookie หรือ raw provider response
- [ ] acceptance test ผ่าน rendered page, HTTP และ PostgreSQL โดยใช้ LINE provider double ใน local/CI; ไม่ต้องใช้ credential จริงหรือ deploy ภายนอก

