# ยังไม่สร้าง event bus และ notification abstraction ที่ไม่มีผู้ใช้จริง

Phase 0 จะส่งมอบ Inbound Event pipeline ที่ทำงานจริงและบันทึกรูปแบบขั้นต่ำของ Domain Event สำหรับอนาคต แต่ยังไม่สร้าง publisher, event bus, domain-events table, Notifier interface หรือ export service package จนกว่าจะมี event type หรือ use case แรกที่ใช้งานจริง
