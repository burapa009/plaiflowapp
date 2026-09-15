# 12: Owner/Admin เชื่อมและยกเลิกการเชื่อม LINE group

**What to build:** Owner/Admin ที่เชื่อม LINE identity แล้วสามารถสร้าง code, ส่ง code ในกลุ่ม LINE, เห็น connection สำเร็จ และ disconnect ได้โดย code และ webhook data ไม่รั่ว

**Blocked by:** 09 — Owner มอบสิทธิ์ Admin และโอน ownership; 11 — ผู้ใช้ Google เดิมเชื่อม LINE user identity

**Status:** ready-for-agent

- [ ] เปิด fresh Codex context ก่อนเริ่มงาน ticket นี้
- [ ] เฉพาะ current Owner/Admin ที่มี active LINE AuthIdentity และ recent-auth จึงสร้าง link code ได้; Member และ stale role ถูกปฏิเสธ
- [ ] link code มี randomness อย่างน้อย 128 bits, เก็บเฉพาะ hash, ผูก Organization/User/expected LINE subject/channel/purpose, หมดอายุ 10 นาที และ replacement revoke code เดิม
- [ ] webhook ตรวจ signature จาก exact raw body ก่อน parse และรับ redemption เฉพาะ message event จาก group ที่มี groupId/userId/channel ตรงกับ code และ sender ตรงกับ LINE subject
- [ ] join event ไม่สร้าง connection, personal-chat/room/missing sender/mismatched sender/wrong channel/group และ direct API redemption fail closed
- [ ] worker lock pending code และ target group, re-check current Membership/role, consume code และสร้าง connection ใน transaction เดียว
- [ ] group เดียวต่อ Messaging channel เชื่อม active ได้กับ Organization เดียว ขณะที่ Organization หนึ่งเชื่อมหลาย group ได้; concurrency มีผู้ชนะหนึ่งรายการ
- [ ] plaintext code ถูก hash/redact ก่อน durable event storage และไม่ปรากฏใน logs, audit, raw retained message หรือ error response
- [ ] dashboard แสดง waiting, connected, expired, conflict และ disconnected พร้อม retry โดย polling/refresh แบบ bounded query
- [ ] disconnect ใน PlaiFlow หยุด tenant processing ทันทีและบันทึก actor; valid LINE leave event ทำเครื่องหมาย disconnected
- [ ] unconnected/disconnected group messages ถูก acknowledge หลัง signature verification แต่ไม่ถูกเก็บ เว้นแต่เป็น eligible link attempt
- [ ] indexes รองรับ code lookup, pending code, expiry cleanup, Organization connections และ unique active group โดยไม่ index role/status เดี่ยว ๆ
- [ ] tests ครอบคลุม signature forgery/body alteration, replay, expiry, replacement, sender/channel/group mismatch, stale role, conflict, leave และ disconnect; ไม่ assert automated bot leave หรือ LINE group-admin proof

