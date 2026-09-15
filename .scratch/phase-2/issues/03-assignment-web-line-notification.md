# 03: Assigned Task ปรากฏใน Web และ LINE

**What to build:** เมื่อสมาชิกมอบหมาย Task ให้สมาชิกอีกคน ผู้รับเห็น Web Notification และได้รับ LINE direct message เมื่อ opt in โดยการเปลี่ยน Task ไม่รอ provider, การประมวลผลซ้ำไม่สร้างข้อความซ้ำ และทุก deep link ตรวจสิทธิ์ใหม่

**Blocked by:** 01 — ผู้ใช้สร้าง มอบหมาย และเปิดดู Task แรกได้; 02 — Server-side Entitlement seam ควบคุม Task creation ได้

**Status:** ready-for-agent

- [ ] เปิด fresh Codex context ก่อนเริ่มงาน ticket นี้
- [ ] Task assignment บันทึก task.assigned Domain Event กับ Task mutation ใน transaction เดียวกัน
- [ ] worker เปลี่ยน task.created/task.assigned ให้เป็น recipient-specific Notification แบบ asynchronous และ at-least-once
- [ ] Assignment default เป็น Web Immediate; LINE เป็น Off จนกว่า Membership จะ opt in
- [ ] การ assign โดยคนหนึ่งให้อีกคนสร้าง Web Notification ที่ผู้รับเปิดดูได้จาก inbox และ link ไป Task ที่ถูกต้อง
- [ ] LINE opt-in ใช้ External Identity ของ User ผู้รับและส่ง direct message เท่านั้น ไม่มี LINE Group fallback
- [ ] LINE message มีเพียง generic notification type, Organization display name และปุ่มเปิด PlaiFlow โดยไม่มี title, description, Assignee, Watcher หรือข้อมูลลูกค้า
- [ ] deep link ไม่มี credential และปลายทางตรวจ session, Organization Context, current Membership และ role ใหม่ทุกครั้ง
- [ ] mutation actor ไม่ได้รับ notification จาก action ตนเอง; Task Creator, Owner และ Admin ไม่เป็น recipient อัตโนมัติ
- [ ] logical key ป้องกัน Notification และ Delivery Attempt ซ้ำเมื่อ worker claim หรือ provider retry ซ้ำ
- [ ] LINE failure ไม่ย้อนกลับ Task หรือ Web Notification และไม่ fallback ไป recipient อื่น
- [ ] feature gate สำหรับ LINE ถูกประเมินฝั่ง server หลัง authorization; frontend opt-in ไม่ให้ permission เอง
- [ ] integration test พิสูจน์ Web+LINE happy path, default LINE off, self-assignment suppression, duplicate worker run, provider failure, cross-tenant recipient และ revoked deep link

