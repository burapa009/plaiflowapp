# 06: OWNER เชื่อม Google Drive ด้วย scope ขั้นต่ำ

**What to build:** authenticated OWNER เชื่อม Google Drive ให้ Organization ผ่าน consent flow ที่แยกจาก Google Login โดย PlaiFlow ขอสิทธิ์เท่าที่ต้องใช้สำหรับไฟล์ที่ระบบสร้างและเก็บ credential ไว้ฝั่ง server เท่านั้น

**Blocked by:** 01/เปรียบเทียบ Plan จาก authoritative catalog.

**Status:** ready-for-agent

- [ ] เฉพาะ current OWNER ที่ผ่าน recent reauthentication และยืนยัน Organization จึงเริ่มหรือจบ connect flow ได้
- [ ] Authorization request ใช้ OAuth client แยกจาก Login และขอเฉพาะ `openid`, `email`, `drive.file` พร้อม offline access; ไม่มี whole-Drive, Sheets หรือ broad metadata scope
- [ ] State, callback และ Organization binding ป้องกัน CSRF, tenant swap และ account confusion; OAuth เปิดใน external system browser
- [ ] เมื่อสำเร็จ ระบบสร้างหรือผูก canonical PlaiFlow folder หนึ่งรายการและบันทึกหนึ่ง active Drive Connection ต่อ Organization
- [ ] Refresh token ถูกเข้ารหัสที่ rest, ไม่ออกสู่ browser/log/audit และการไม่มี Business Entitlement ไม่สามารถถูกข้ามด้วยค่าจาก client
- [ ] มี runnable check ผ่าน provider double สำหรับ exact scopes, OWNER-only access, tenant binding และไม่มี token exposure
