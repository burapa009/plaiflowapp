# 13: Owner/Admin ตรวจสอบ Export History และ audit lifecycle ได้

**What to build:** Owner/Admin เปิด Export History ของ Organization แล้วตรวจ requester, filters, row count, status และ lifecycle ของ synchronous/background exports ได้ โดย audit เป็น append-only, tenant-scoped และไม่เก็บข้อมูลไฟล์หรือ bearer secrets

**Blocked by:** 11 — Owner/Admin ส่งออก filtered Tasks เป็น formula-safe CSV; 12 — CSV ขนาดใหญ่เสร็จผ่าน background job และ expiring link

**Status:** ready-for-agent

- [ ] เปิด fresh Codex context ก่อนเริ่มงาน ticket นี้
- [ ] Export Job/history บันทึก Organization, requester, normalized filter summary, format/version, mode, status, lifecycle timestamps, authorized row/byte counts, data_as_of, safe failure reason และ object expiry
- [ ] history ไม่เก็บ signed URL, object key, CSV content, cell values, description text, provider credential หรือ raw authorization/provider error
- [ ] append-only audit ครอบคลุม requested, rejected, completed, failed, downloaded และ expired สำหรับ sync/background paths
- [ ] audit มี Organization, actor เมื่อมี, opaque export reference, request ID, outcome, safe reason, row count และ timestamps
- [ ] Owner/Admin ปัจจุบันเท่านั้นที่อ่าน Organization export history ได้; Member, removed Membership และ cross-tenant identifiers ให้ผลแบบไม่เปิดเผยข้อมูล
- [ ] history ใช้ cursor pagination และ Completed job download ได้เฉพาะก่อน object expiry หลัง reauthorization
- [ ] tenant-readable Export Job/audit data มี Organization RLS ตาม Organization Context ADR และ API ไม่ expose generic audit table โดยตรง
- [ ] worker/migration roles แยกจาก request runtime role และมีสิทธิ์เท่าที่จำเป็น
- [ ] file object ถูกลบหลัง 24 ชั่วโมง แต่ history/audit metadata ถูกเก็บ 365 วันและ cleanup ทำซ้ำได้อย่างปลอดภัย
- [ ] rejected authorization/entitlement attempt ไม่บันทึก sensitive requested values หรือสร้างข้อมูลใน Organization ที่ actor ไม่มีสิทธิ์
- [ ] UI แสดง loading, empty, status, expired, failure และ download states โดยไม่ render signed URL เป็น persistent data
- [ ] integration tests ครอบคลุมทุก lifecycle, sync/background correlation, download audit, expiration, retention, role matrix, RLS และ absence of forbidden metadata
