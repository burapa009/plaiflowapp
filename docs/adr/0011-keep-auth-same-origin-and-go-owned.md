# ให้ Auth อยู่บน same origin และให้ Go เป็นเจ้าของ

Phase 1 ใช้ public web origin เดียว โดย Next.js proxy เส้นทาง `/api/auth/*` ไปยัง Go ซึ่งเป็นเจ้าของ OAuth transaction, opaque server-side session, identity, RBAC และ audit ส่วน Next.js มีหน้าที่แสดง UI เท่านั้น แนวทางนี้หลีกเลี่ยง cross-origin cookie และระบบ session ซ้ำซ้อน; cookie เป็น host-only, `HttpOnly`, `Secure`, `SameSite=Lax` และ provider token ไม่ออกจาก backend
