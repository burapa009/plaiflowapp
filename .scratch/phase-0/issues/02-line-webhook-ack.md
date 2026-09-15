# 02: LINE webhook ที่ถูกต้องได้รับการตรวจสอบและตอบรับอย่างปลอดภัย

**What to build:** เมื่อ LINE ส่ง webhook ที่มีลายเซ็นถูกต้อง ระบบอ่าน raw body, ตรวจสอบ HMAC และบันทึก Inbound Event แบบ durable ก่อนตอบรับ โดย request ที่ผิดรูปแบบหรือปลอมแปลงถูกปฏิเสธอย่างปลอดภัย

**Blocked by:** 01 — นักพัฒนาเปิด PlaiFlow ในเครื่องได้จาก fresh checkout

**Status:** ready-for-agent

- [ ] เปิด fresh Codex context ก่อนเริ่มงาน ticket นี้
- [ ] ตรวจลายเซ็นด้วย HMAC-SHA256 และ constant-time comparison บน raw body
- [ ] จำกัด body size ที่ 1 MiB และมี timeout/concurrency guard สำหรับ public webhook
- [ ] malformed, missing signature และ invalid signature ไม่ทำให้ process ล่มและไม่เปิดเผย payload ใน log
- [ ] empty event delivery ตอบ `200` ได้ และ valid delivery จะตอบรับหลัง durable insert สำเร็จเท่านั้น
- [ ] ทุก request มี request ID และมี baseline API timing metric
