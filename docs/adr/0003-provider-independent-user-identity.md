# แยกตัวตนผู้ใช้ของ PlaiFlow ออกจากผู้ให้บริการ Login

PlaiFlow จะเป็นเจ้าของ User identity ภายในที่คงที่และเชื่อมกับ External Identity ได้หลายรายการ เช่น LINE Login และ Google Login โดย Go API จะเป็นเจ้าของ protocol callback ที่ `/v1/auth/{provider}/callback` และ web จะแสดงผลที่ `/auth/callback` เท่านั้น Phase 0 จะกำหนดเพียงรอยต่อนี้โดยยังไม่สร้าง authentication, session, หน้า login หรือการจัดเก็บ token
