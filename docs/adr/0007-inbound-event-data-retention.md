# จำกัดอายุข้อมูล Inbound Event

PlaiFlow จะไม่เก็บ raw Webhook Delivery หรือ signature แต่จะเก็บ JSON ของ Inbound Event แต่ละรายการไว้ 30 วันเพื่อ replay และวิเคราะห์ปัญหา หลังจากนั้นลบ payload และคง metadata, idempotency key, timestamps และผลประมวลผลไว้ 90 วัน เพื่อลดข้อมูลส่วนบุคคลที่สะสมโดยยังรักษาช่วงตรวจเหตุการณ์ซ้ำที่เพียงพอ
