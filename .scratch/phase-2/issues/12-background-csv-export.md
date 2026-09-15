# 12: CSV ขนาดใหญ่เสร็จผ่าน background job และ expiring link

**What to build:** Owner/Admin ที่ export 5,001–100,000 rows ได้ Export Job ซึ่งประมวลผลใน background ด้วย memory จำกัด แล้วดาวน์โหลด private file ที่หมดอายุได้ โดยงานเกินเพดานถูกปฏิเสธและ Task/Notification work ไม่ถูกอดคิว

**Blocked by:** 11 — Owner/Admin ส่งออก filtered Tasks เป็น formula-safe CSV

**Status:** ready-for-agent

- [ ] เปิด fresh Codex context ก่อนเริ่มงาน ticket นี้
- [ ] authorized count เลือก sync ที่ไม่เกิน 5,000 rows, background ที่ 5,001–100,000 และ stable export_too_large เหนือ 100,000 โดยไม่ truncate
- [ ] background request คืน opaque job reference และ UI แสดง Queued, Running, Completed, Failed หรือ Expired พร้อม safe retry guidance
- [ ] worker recheck current Membership และ Owner/Admin role ก่อนอ่าน Task rows; removed/downgraded requester ทำให้งานล้มเหลวแบบไม่เปิด metadata
- [ ] Entitlement ถูกตรวจตอนรับ request; plan change ภายหลังไม่ยกเลิกไฟล์ที่รับไว้ก่อน expiry แต่ current RBAC ยังบังคับตอน download
- [ ] CSV ใช้ encoder/security rules เดียวกับ synchronous path และ stream จาก database ไป private object storage โดยไม่เก็บ bytes ใน PostgreSQL/public directory
- [ ] data_as_of ถูกบันทึกเมื่อ generation เริ่มและ output สะท้อน authorized data ณ จุดนั้นโดยไม่ถือ long-lived transaction จากเวลา request
- [ ] export-specific buffer ไม่เกิน 8 MiB, output ไม่เกิน 256 MiB, rows ไม่เกิน 100,000 และมี large export active สูงสุดหนึ่งงานต่อ worker execution lane
- [ ] large export ไม่ขัดขวาง notification claim หรือ Reminder processing และ failure cleanup เป็น idempotent
- [ ] persisted object ถูกลบหลัง 24 ชั่วโมง; download ผ่าน PlaiFlow authorization ก่อนรับ signed URL ที่อายุไม่เกิน 5 นาที
- [ ] signed URL, object key และ provider secret ไม่อยู่ใน logs, audit metadata หรือ frontend analytics
- [ ] ใช้ fake private-object-store boundary ใน tests และไม่ provision production bucket/secret ใน ticket นี้
- [ ] threshold tests ครอบคลุม 5,000, 5,001, 100,000, 100,001 rows พร้อม memory/file ceiling, authorization recheck, expiry และ missing-object cleanup

