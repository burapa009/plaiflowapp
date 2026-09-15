# PlaiFlow

PlaiFlow เป็นผลิตภัณฑ์สำหรับจัดการขั้นตอนการทำงานทางธุรกิจ โดยแนวคิดภายในระบบไม่ผูกกับผู้ให้บริการภายนอก เช่น LINE หรือ Google

## Language

**User**:
บุคคลที่มีตัวตนหลักซึ่ง PlaiFlow เป็นเจ้าของและคงที่ โดย User หนึ่งรายอาจเชื่อมกับ External Identity ได้หลายรายการ
_Avoid_: Account, LINE user, Google user

**External Identity**:
ตัวตนจากผู้ให้บริการภายนอกที่เชื่อมกับ User เช่น ตัวตนจาก LINE Login หรือ Google Login
_Avoid_: User, social account

**Webhook Delivery**:
คำขอ HTTP หนึ่งครั้งที่ลงลายเซ็นและส่งมาจากผู้ให้บริการภายนอก โดยหนึ่ง Webhook Delivery อาจมี Inbound Event หลายรายการ
_Avoid_: Webhook event, Domain Event

**Inbound Event**:
เหตุการณ์หนึ่งรายการจากผู้ให้บริการซึ่งอยู่ภายใน Webhook Delivery และถูกจัดเก็บแยกเพื่อการตรวจเหตุการณ์ซ้ำและการประมวลผล
_Avoid_: Webhook Delivery, Domain Event

**Domain Event**:
ข้อเท็จจริงทางธุรกิจที่ PlaiFlow สร้างขึ้นหลังจากตีความ Inbound Event แล้ว
_Avoid_: Raw event, webhook payload

**Received**:
สถานะของ Inbound Event หลังจากถูกจัดเก็บในพื้นที่ถาวรสำเร็จ
_Avoid_: Processed, completed

**Processing**:
สถานะของ Inbound Event ที่ worker กำลังประมวลผลภายใต้ lease ที่ยังไม่หมดอายุ
_Avoid_: Received, Processed

**Processed**:
สถานะของ Inbound Event หลังจาก handler ที่รับผิดชอบทำงานสำเร็จ
_Avoid_: Received, delivered

**Retryable**:
สถานะของ Inbound Event ที่ประมวลผลไม่สำเร็จชั่วคราวและยังมีสิทธิ์ถูกลองใหม่ตามกำหนด
_Avoid_: Failed, queued

**Failed**:
สถานะของ Inbound Event ที่ลองประมวลผลครบจำนวนแล้วและต้องรอการตัดสินใจจาก operator
_Avoid_: Retryable, ignored

**Ignored**:
สถานะของ Inbound Event ที่จัดเก็บสำเร็จแต่ไม่มี handler สำหรับชนิดเหตุการณ์นั้น และไม่จำเป็นต้องลองใหม่
_Avoid_: Failed, Processed
