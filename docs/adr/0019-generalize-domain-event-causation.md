# ให้ Domain Event รองรับต้นเหตุที่ไม่ใช่ Inbound Event

Phase 2 ใช้ Domain Event envelope เดียวสำหรับข้อเท็จจริงที่เกิดจาก Inbound Event, คำสั่งของ User หรือเงื่อนไขตามเวลา โดย source Inbound Event เป็น optional causation และ actor แยกจาก source แทนการสร้าง Inbound Event ปลอมหรือแยก envelope หลายแบบ เพราะ Task และ Reminder เป็น consumer จริงกลุ่มแรกที่ไม่ได้เริ่มจาก webhook เสมอไป
