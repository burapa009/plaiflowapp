# PlaiFlow

PlaiFlow เป็นผลิตภัณฑ์สำหรับจัดการขั้นตอนการทำงานทางธุรกิจ โดยแนวคิดภายในระบบไม่ผูกกับผู้ให้บริการภายนอก เช่น LINE หรือ Google

## Language

**User**:
บุคคลที่มีตัวตนหลักซึ่ง PlaiFlow เป็นเจ้าของและคงที่ โดย User หนึ่งรายอาจเชื่อมกับ External Identity ได้หลายรายการ
_Avoid_: Account, LINE user, Google user

**External Identity**:
ตัวตนจากผู้ให้บริการภายนอกที่เชื่อมกับ User เช่น ตัวตนจาก LINE Login หรือ Google Login
_Avoid_: User, social account

**Organization**:
ขอบเขต tenant ที่เป็นเจ้าของข้อมูลและการตั้งค่าของธุรกิจ โดย User เข้าถึง Organization ผ่าน Membership เท่านั้น
_Avoid_: Account, workspace, tenant

**Membership**:
ความสัมพันธ์ระหว่าง User กับ Organization ซึ่งระบุสิทธิ์ของ User ภายใน Organization นั้น
_Avoid_: User role, organization user

**Owner**:
Membership ผู้รับผิดชอบสูงสุดของ Organization ซึ่งมีได้หนึ่งรายและสามารถโอนให้ User อื่นได้
_Avoid_: Super Admin, Organization User

**Admin**:
Membership ที่จัดการ Organization ตามสิทธิ์ที่ได้รับ แต่ไม่ใช่เจ้าของและโอน ownership ไม่ได้
_Avoid_: Owner, System Admin

**Member**:
Membership มาตรฐานสำหรับใช้งานภายใน Organization โดยไม่มีสิทธิ์บริหารสมาชิกหรือ ownership
_Avoid_: User, Viewer

**Task**:
หน่วยงานที่มนุษย์ต้องดำเนินการภายใน Organization เดียว โดยมี Task Creator หนึ่งราย มี Assignee ได้ไม่เกินหนึ่งราย และมี Watcher ได้หลายราย Task ไม่ใช่ background job, Notification หรือ Event และไม่มี Task ส่วนตัวในโมเดลปัจจุบัน
_Avoid_: Job, workflow event, private task

**Task Creator**:
User ที่สร้าง Task ขณะมี Membership ใน Organization โดยตัวตนผู้สร้างยังคงอยู่ในประวัติแม้ Membership นั้นสิ้นสุดลง
_Avoid_: Owner, Assignee

**Assignee**:
Membership ที่ยังใช้งานอยู่ภายใน Organization เดียวกับ Task และรับผิดชอบ Task นั้นโดยตรง โดยหนึ่ง Task มี Assignee ได้ไม่เกินหนึ่งราย เมื่อ Membership สิ้นสุด Task ที่ยังไม่จบจะไม่มี Assignee
_Avoid_: Task Creator, Watcher, User

**Watcher**:
Membership ที่ยังใช้งานอยู่ภายใน Organization เดียวกับ Task ซึ่งติดตามการเปลี่ยนแปลงและรับ Notification โดยสถานะ Watcher ไม่ได้ให้สิทธิ์แก้ Task เพิ่มเติมและสิ้นสุดเมื่อ Membership สิ้นสุด
_Avoid_: Assignee, follower account

**Task Status**:
สถานะปัจจุบันของ Task ซึ่งมีได้เฉพาะ Open, InProgress, Done หรือ Cancelled และแยกจากสถานะของ Inbound Event
_Avoid_: Received, Processing, Processed, Overdue

**Open**:
Task ที่พร้อมให้ดำเนินการแต่ยังไม่มีผู้เริ่มทำ
_Avoid_: Received, pending event

**InProgress**:
Task ที่มีผู้เริ่มดำเนินการแล้วแต่ยังไม่เสร็จ
_Avoid_: Processing, worker lease

**Done**:
Task ที่ดำเนินการเสร็จแล้ว โดยเก็บผู้ทำให้เสร็จและเวลาปัจจุบัน ส่วนการ reopen จะกลับเป็น Open และเก็บประวัติเดิมไว้ใน Domain Event
_Avoid_: Processed, archived

**Cancelled**:
Task ที่ไม่ต้องดำเนินการต่อและถูกซ่อนจากรายการปกติ แต่ยังค้นหาและส่งออกได้ Task ไม่มีการลบออกจากประวัติในโมเดลปัจจุบัน
_Avoid_: Deleted, Failed, Ignored

**Due Date**:
วันที่ครบกำหนดแบบไม่บังคับของ Task ตาม Organization Timezone โดยไม่มีเวลาระดับชั่วโมงหรือนาที
_Avoid_: Due time, reminder time

**Overdue**:
เงื่อนไขที่คำนวณเมื่อ Task ซึ่งยังไม่ Done หรือ Cancelled ผ่านสิ้นสุด Due Date แล้ว โดยไม่ใช่ Task Status และไม่เปลี่ยน Priority อัตโนมัติ
_Avoid_: Task Status, Failed

**Priority**:
ระดับการจัดลำดับ Task ซึ่งมี Normal, High หรือ Urgent โดยค่าเริ่มต้นคือ Normal และไม่มีผลต่อ authorization หรือ Reminder Schedule
_Avoid_: Severity, entitlement tier

**Organization Timezone**:
เขตเวลา IANA ของ Organization สำหรับตีความ Due Date, Reminder Schedule และ Digest โดยค่าเริ่มต้นคือ Asia/Bangkok
_Avoid_: Browser timezone, UTC offset

**Organization Context**:
คู่ของ User และ Organization ที่ระบบยืนยัน Membership แล้วเพื่อใช้เป็นขอบเขต authorization และข้อมูลของคำขอหนึ่งรายการ
_Avoid_: Active Organization, client organization

**Operations Dashboard**:
มุมมองสุขภาพทางเทคนิคของ PlaiFlow สำหรับ operator เช่น API, database, worker และการประมวลผล Inbound Event
_Avoid_: Work Dashboard, Organization Dashboard

**Work Dashboard**:
มุมมองงานภายใน Organization สำหรับ User โดยแสดงเฉพาะข้อมูลที่ User มีสิทธิ์เห็นภายใต้ Organization Context
_Avoid_: Operations Dashboard, system status

**Assistant Summary**:
บทสรุปแบบอ่านอย่างเดียวจากข้อมูลที่ User มีสิทธิ์เห็นใน Organization Context โดยไม่สร้างหรือแก้ Task และไม่ส่ง Notification เอง
_Avoid_: Autonomous agent, Task command

**CSV Export**:
การส่งออก Task ที่ผ่าน authorization แล้วจาก Organization เดียวตามตัวกรองของคำขอ โดยไม่รวม raw event, Notification history, LINE identifier หรือข้อมูล Membership
_Avoid_: Database dump, audit export

**Entitlement**:
สิทธิ์เชิงพาณิชย์ของ Organization ในการใช้ capability หนึ่ง ซึ่งแยกจาก Membership และไม่สามารถให้สิทธิ์เข้าถึงข้อมูลแทน authorization ได้
_Avoid_: Role, permission, frontend plan state

**Usage**:
ปริมาณการใช้ capability ที่วัดเพื่อเทียบกับ Entitlement โดยค่า Usage ไม่สามารถให้สิทธิ์เข้าถึงข้อมูลได้
_Avoid_: Permission, authorization

**LINE Group Connection**:
ความสัมพันธ์ที่ PlaiFlow บันทึกไว้ระหว่าง LINE group จาก Messaging API กับ Organization
_Avoid_: LINE Login, account link, group membership

**Webhook Delivery**:
คำขอ HTTP หนึ่งครั้งที่ลงลายเซ็นและส่งมาจากผู้ให้บริการภายนอก โดยหนึ่ง Webhook Delivery อาจมี Inbound Event หลายรายการ
_Avoid_: Webhook event, Domain Event

**Inbound Event**:
เหตุการณ์หนึ่งรายการจากผู้ให้บริการซึ่งอยู่ภายใน Webhook Delivery และถูกจัดเก็บแยกเพื่อการตรวจเหตุการณ์ซ้ำและการประมวลผล
_Avoid_: Webhook Delivery, Domain Event

**Domain Event**:
ข้อเท็จจริงทางธุรกิจที่ PlaiFlow สร้างขึ้นหลังจากยอมรับการเปลี่ยนแปลงทางธุรกิจ โดยอาจเกิดจาก Inbound Event, คำสั่งของ User หรือเงื่อนไขตามเวลา
_Avoid_: Raw event, webhook payload

**Causation**:
การอ้างอิงต้นเหตุของ Domain Event ซึ่งอาจเป็น Inbound Event, คำสั่งของ User หรือเงื่อนไขตามเวลา โดยไม่สร้าง Inbound Event ปลอมเมื่อเหตุไม่ได้มาจาก provider
_Avoid_: Mandatory source event, synthetic inbound event

**Task Domain Event**:
Domain Event จากการเปลี่ยน Task ซึ่งใช้ชนิด task.created, task.details_changed, task.assigned, task.unassigned, task.watcher_added, task.watcher_removed, task.due_changed, task.priority_changed หรือ task.status_changed โดยไม่มีชนิด task.updated
_Avoid_: Audit event, task.updated

**Notification**:
รายการแจ้งเตือนของ Membership หนึ่งรายเกี่ยวกับ Domain Event หรือเงื่อนไขตามเวลา โดยมีสถานะอ่านแยกจากการส่งผ่าน channel และเก็บไว้ 90 วัน
_Avoid_: LINE Group message, Domain Event, delivery attempt

**Notification Category**:
กลุ่มความสนใจที่ Membership ใช้กำหนด Notification Preference ซึ่งมี Assignment, TaskChange และ DueReminder
_Avoid_: Domain Event type, LINE message type

**Notification Preference**:
การเลือก Immediate, Digest หรือ Off ของ Membership ต่อ Notification Category และ channel ภายใน Organization หนึ่ง โดย LINE เป็น Off จนกว่า User จะ opt in
_Avoid_: Global user setting, Entitlement

**Immediate**:
โหมดที่สร้างโอกาสส่ง Notification โดยไม่รอ Digest และไม่รวมรายการเดียวกันใน Digest ภายหลัง
_Avoid_: Synchronous delivery, guaranteed delivery

**Digest**:
บทสรุป Notification รายวันเวลา 09:00 ตาม Organization Timezone ซึ่งรวมเหตุการณ์ตาม Task แสดงสถานะล่าสุดและจำนวนการเปลี่ยนแปลง และไม่ส่งเมื่อไม่มีรายการ
_Avoid_: Assistant Summary, event replay, immediate notification

**Reminder Schedule**:
กำหนด Notification สำหรับ Due Date ที่ 09:00 หนึ่งวันก่อนครบกำหนด วันครบกำหนด และหนึ่งวันหลังครบกำหนด โดยคำนวณใหม่เมื่อ Due Date เปลี่ยนและหยุดเมื่อ Task เป็น Done หรือ Cancelled
_Avoid_: Due Date, custom task reminder

**Delivery Attempt**:
ความพยายามส่ง Notification หนึ่งรายการผ่าน channel หนึ่ง เช่น LINE direct message โดย retry ใช้รายการเดิมและความล้มเหลวไม่ย้อนกลับ Task หรือ Web Notification
_Avoid_: Notification, Domain Event

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
