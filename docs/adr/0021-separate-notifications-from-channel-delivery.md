# แยก Notification ออกจาก channel delivery

Phase 2 เก็บ Notification ต่อ recipient แยกจาก Delivery Attempt ต่อ channel และประมวลผลแบบ at-least-once หลัง Task mutation กับ Domain Event ถูกบันทึกใน transaction เดียวกัน โดยใช้ unique logical keys ทำให้ผลที่ผู้ใช้เห็นไม่ซ้ำและไม่ให้ LINE latency, retry หรือ provider failure ย้อนกลับการเปลี่ยน Task
