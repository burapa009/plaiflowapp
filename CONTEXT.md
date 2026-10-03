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
ขอบเขต tenant ที่เป็นเจ้าของข้อมูลและการตั้งค่าของธุรกิจ โดย User เข้าถึงผ่าน Membership ของ Organization นั้น หรือสิทธิ์สำนักงานบัญชีที่ Client Organization อนุมัติอย่างชัดเจน
_Avoid_: Account, workspace, tenant

**Accounting Firm Organization**:
Organization ที่เป็นตัวแทนสำนักงานบัญชี มี Membership ของบุคลากรสำนักงานเอง และไม่เป็นเจ้าของข้อมูลของ Organization ลูกค้า
_Avoid_: Firm entity, shared client tenant

**Client Organization**:
Organization ที่เป็นเจ้าของข้อมูลธุรกิจของตนเองและอาจอนุญาตให้ Accounting Firm Organization ทำงานให้ โดยไม่เปลี่ยนเจ้าของข้อมูล
_Avoid_: Customer, Business Contact, firm-owned client record

**Firm Access Grant**:
สิทธิ์ที่ Client Organization อนุมัติให้ Accounting Firm Organization ทำงานกับข้อมูลลูกค้าภายในขอบเขตที่กำหนด โดยไม่ให้สิทธิ์แก่พนักงานสำนักงานทุกคนโดยอัตโนมัติ
_Avoid_: Firm Plan, client Membership, blanket tenant access

**Client Assignment**:
ความสัมพันธ์ที่ Accounting Firm Organization ระบุว่าบุคลากรคนใดรับผิดชอบ Client Organization ที่ได้รับ Firm Access Grant โดย Assignment ไม่ขยายขอบเขตสิทธิ์ของ Grant
_Avoid_: Client Membership, access grant, Task Assignee

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

**Firm Client Context**:
ขอบเขตการทำงานหนึ่งรายการที่ยืนยัน Membership ใน Accounting Firm Organization, Client Assignment และ Firm Access Grant ปัจจุบัน พร้อมสิทธิ์ต่อข้อมูลของ Client Organization ที่เกี่ยวข้อง
_Avoid_: Organization Context, firm-wide access, implicit client Membership

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
การส่งออก Task หรือ Business Master Data ที่ผ่าน authorization แล้วจาก Organization เดียวตาม Export Template และตัวกรองของคำขอ โดยไม่รวม raw event, Notification history, LINE identifier หรือข้อมูล Membership
_Avoid_: Database dump, audit export

**Multi-Client Export**:
workbook ที่มีเวอร์ชันซึ่งรวม Approved Export Rows ปัจจุบันของ Client Organization ที่ผู้ขอมีสิทธิ์เท่านั้น โดยมี Summary และ sheet แยกตามลูกค้า
_Avoid_: CSV Export, database dump, accounting-system import template

**Entitlement**:
สิทธิ์เชิงพาณิชย์ของ Organization ในการใช้ capability หนึ่ง ซึ่งแยกจาก Membership และไม่สามารถให้สิทธิ์เข้าถึงข้อมูลแทน authorization ได้
_Avoid_: Role, permission, frontend plan state

**Usage**:
ปริมาณการใช้ capability ที่วัดเพื่อเทียบกับ Entitlement โดยค่า Usage ไม่สามารถให้สิทธิ์เข้าถึงข้อมูลได้
_Avoid_: Permission, authorization

**Usage Reservation**:
การจอง Usage แบบ atomic ก่อนเริ่มงานที่มี hard quota โดย Complete หรือ Release ด้วย idempotency key เดิมเพื่อไม่ให้คำขอพร้อมกันหรือ retry ใช้เกินหรือนับซ้ำ
_Avoid_: Authorization, completed Usage, client-side counter

**Plan**:
แพ็กเกจเชิงพาณิชย์ของ Organization จาก catalog ที่ระบบฝั่ง server เชื่อถือ ซึ่งจัดกลุ่ม Entitlement และ Usage Limit โดยมี Free, Starter, Business และ Accounting Firm
_Avoid_: Role, client-selected price, Trial

**Billing Price**:
ยอดเงินที่ Organization ต้องชำระสำหรับ Plan และ Billing Interval หนึ่งตาม catalog ที่เชื่อถือได้ โดยราคาเฉลี่ยต่อเดือนเป็นเพียงข้อมูลเปรียบเทียบ ไม่ใช่ยอดที่จะเรียกเก็บ
_Avoid_: effective monthly price, client-supplied amount, Entitlement

**Billing Interval**:
ระยะเวลาระหว่างการต่ออายุและชำระเงินของ Subscription ซึ่งมีรายเดือน ทุก 6 เดือน หรือรายปี
_Avoid_: monthly usage period, Trial duration

**Billing Customer**:
ตัวตนผู้ชำระเงินของ Organization หนึ่งแห่งกับผู้ให้บริการชำระเงิน โดยไม่แทน User, Membership หรือ Client Organization ที่สำนักงานบัญชีดูแล
_Avoid_: User, Business Contact, Accounting Firm client

**Billing Contact Email**:
อีเมลที่ Owner ระบุและยืนยันสำหรับรับข้อมูลการชำระเงินของ Organization ซึ่งอาจต่างจากอีเมลของ External Identity ที่ใช้ลงชื่อเข้าใช้
_Avoid_: unverified provider email, User identity, LINE contact

**Billing Profile**:
ข้อมูลชื่อ ที่อยู่ และเลขผู้เสียภาษีของผู้ซื้อที่ Owner ตรวจทานก่อนชำระเงิน โดยข้อมูล ณ วันที่ชำระถูกบันทึกกับใบเสร็จของผู้ขาย และการแก้ไข Profile มีผลกับใบเสร็จใหม่เท่านั้น
_Avoid_: live Organization display name, Stripe Customer identity, editable issued receipt

**Seller Receipt**:
ใบรับเงินของผู้ขายที่ออกหลังยืนยันว่าชำระสำเร็จ โดยมีเลขลำดับ ข้อมูลผู้ขาย ผู้ซื้อ และยอดเงินที่บันทึกไว้ ณ วันออกใบรับ แยกจากหลักฐานการชำระเงินหรือ invoice ของ Stripe และไม่ใช่ใบกำกับภาษี VAT
_Avoid_: Stripe payment receipt, VAT tax invoice, unverified checkout redirect

**Scan Credit**:
หน่วยสิทธิ์เสริมที่ Organization ซื้อเพื่อใช้สแกนหน้าเอกสารอัตโนมัติ โดยหนึ่งหน้าที่สแกนใช้หนึ่งเครดิต และไฟล์หลายหน้าใช้ตามจำนวนหน้า ยอดที่เหลือยกไปเดือนถัดไปโดยไม่หมดอายุ แยกจากโควตารับ Document และไม่ใช่เงินคงเหลือ
_Avoid_: Stripe Billing Credit, Document quota, OCR retry attempt

**Document Allowance**:
จำนวน Document ใหม่ที่ Organization ยังรับได้จากโควตารายเดือนของ Plan โดยยอดที่ยังไม่ใช้ของ Plan ชำระเงินยกไปเดือนถัดไปและแต่ละยอดหมดอายุหลังได้รับสองปี ส่วน Free ไม่สะสมยอดรายเดือน การผิดนัดชำระเงินอาจพักยอดชั่วคราวตามช่วงกู้คืน และ Scan Credit เป็นสิทธิ์แยกต่างหาก
_Avoid_: Scan Credit, OCR pages, historical Document count

**Subscription**:
ความสัมพันธ์การชำระเงินของ Organization หนึ่งแห่งสำหรับ Plan และ Billing Interval ที่เลือก โดยสถานะที่ตรวจยืนยันแล้วเป็นข้อมูลสำหรับกำหนดสิทธิ์เชิงพาณิชย์
_Avoid_: Trial, Firm Access Grant, browser checkout result

**Grace Period**:
ช่วงเจ็ดวันหลังการชำระเงินต่ออายุล้มเหลวที่ Organization ยังใช้ Entitlement ของ Plan ที่ชำระไว้ได้ ระหว่างรอแก้การชำระเงิน โดยไม่ลบข้อมูลเมื่อช่วงนี้สิ้นสุด
_Avoid_: Trial, free extension, data retention deadline

**Scheduled Cancellation**:
การหยุดต่ออายุ Subscription ณ สิ้นสุดรอบที่ชำระแล้ว โดย Entitlement เดิมคงอยู่จนถึงวันสิ้นสุดนั้นและ Owner ยกเลิกคำขอนี้ได้ก่อนถึงวันดังกล่าว
_Avoid_: immediate revocation, refund request

**Accounting Firm Plan**:
Plan ของ Accounting Firm Organization ที่กำหนด capability และขีดจำกัดการใช้งานของสำนักงาน โดยไม่ให้สิทธิ์เข้าถึงข้อมูลของ Client Organization แทน Firm Access Grant
_Avoid_: Client Organization Plan, Firm Access Grant, staff role

**Trial**:
สถานะชั่วคราวที่ Owner เปิดให้ Organization ได้รับ capability แบบ Business พร้อม Usage Limit เฉพาะ Trial เป็นเวลา 14 วันหนึ่งครั้งโดยไม่ต้องใช้บัตร เมื่อสิ้นสุด Organization กลับสู่ Free โดยยังอ่านข้อมูลเดิมและใช้การควบคุมความปลอดภัยได้
_Avoid_: Trial Plan, free subscription

**Upgrade Request**:
คำขอของ Owner หรือ Admin ที่แจ้งความสนใจเปลี่ยน Plan โดยไม่ให้ Entitlement หรือเปลี่ยน Plan จนกว่า operator ที่เชื่อถือได้จะดำเนินการ
_Avoid_: Subscription, checkout, plan assignment

**Plan Over Limit**:
สถานะของ Organization หลัง Trial หรือ Plan สิ้นสุดเมื่อจำนวนทรัพยากรที่มีอยู่เกิน Usage Limit ของ Plan ปัจจุบัน โดยยังอ่าน ใช้ CSV Export ตาม role และจัดการความปลอดภัยได้แต่หยุด mutation ปกติจนกว่าจะลดจำนวนหรืออัปเกรด
_Avoid_: Account suspension, data deletion, automatic member removal

**Document Quota Exhausted**:
สถานะที่ยอดโควตารับ Document ที่ใช้ได้ของ Organization หมดลง จึงหยุดรับ Document ใหม่ที่ไม่ซ้ำ แต่ยังอ่าน ส่งออก และทำงานอื่นตามสิทธิ์เดิมได้ โดยโควตารายเดือนที่ยังไม่ใช้สามารถยกไปเดือนถัดไปตามกติกาของ Plan
_Avoid_: Plan Over Limit, account suspension, overage billing

**Document**:
เอกสารธุรกิจต้นฉบับหนึ่งฉบับที่ผ่านการตรวจรับและเก็บสำเนาต้นฉบับไว้ใน PlaiFlow แล้ว โดยไม่นับ Task, แถวจากการนำเข้า, แถวจากการส่งออก หรือจำนวนครั้งที่ OCR/AI ประมวลผลเป็น Document
_Avoid_: Import row, Task, OCR Usage

**Document Source**:
บันทึกที่มาของการรับเอกสารจาก Web, LINE หรือ Google Drive ซึ่งชี้ไปยัง Document ที่รับแล้วภายใน Organization เดียวกัน โดย Document หนึ่งรายการอาจมีหลาย Document Source เมื่อส่งไฟล์เนื้อหาเดียวกันเข้ามาหลายครั้งหรือหลายช่องทาง
_Avoid_: Document copy, provider authorization, rejected intake attempt

**Document Intake Attempt**:
คำขอรับไฟล์หนึ่งครั้งที่อาจกำลังตรวจ ผ่านเป็น Document หรือถูกปฏิเสธ โดยรายการที่ถูกปฏิเสธไม่ใช่ Document และไม่ใช้โควตา Document
_Avoid_: Document, Document Source, accepted original

**Expected Document Item**:
รายการเอกสารที่บุคคลกำหนดว่าควรได้รับจาก Client Organization สำหรับรอบบัญชีหนึ่ง ซึ่งยังไม่ใช่ Document และจะถือว่าขาดหลัง Due Date ตาม timezone ลูกค้าเมื่อยังไม่มี Document ที่บุคคลผูกไว้
_Avoid_: Document, OCR prediction, missing-file inference

**Checklist Follow-up Assignment**:
ความรับผิดชอบของบุคลากรสำนักงานในการติดตาม Expected Document Item หนึ่งรายการ ซึ่งไม่ใช่ Client Assignment และไม่เปลี่ยนสถานะว่าลูกค้าส่งเอกสารครบแล้วหรือยัง
_Avoid_: Client Assignment, Task Assignee, Document match

**Expected Document Template**:
รายการที่คาดหวังซ้ำตามรอบบัญชีของ Client Organization ซึ่งสร้าง Expected Document Item สำหรับแต่ละรอบและยังแก้ไขรายการของรอบนั้นได้
_Avoid_: OCR prediction, Document, automatic file match

**Document Trash**:
สถานะเอกสารที่ Owner หรือ Admin นำออกจากงานปัจจุบันและกู้คืนได้ภายใน 30 วัน ก่อนลบต้นฉบับถาวร โดยไม่คืน Document Usage ที่เคยใช้
_Avoid_: Archived Document, immediate deletion, quota refund

**Business Contact**:
บุคคลหรือนิติบุคคลหนึ่งสาขาภายใน Organization ที่มีบทบาท Customer, Vendor หรือทั้งสองบทบาท โดยประเทศ Tax ID และ canonical branch identity เดียวกันใช้ข้อมูลระบุตัวตนทางธุรกิจชุดเดียวร่วมกัน
_Avoid_: Customer record, Vendor record, accounting-firm client

**Customer**:
บทบาทของ Business Contact ที่ Organization ขายสินค้าหรือให้บริการแก่ฝ่ายนั้น ไม่ใช่ entity แยกจาก Vendor
_Avoid_: Customer entity, client organization

**Vendor**:
บทบาทของ Business Contact ที่ขายสินค้าหรือให้บริการแก่ Organization ไม่ใช่ entity แยกจาก Customer
_Avoid_: Vendor entity, supplier record

**Business Reference Data**:
รายการอ้างอิงที่ Organization เป็นเจ้าของ เช่น Expense Category และ Payment Channel โดยไม่ใช่รายการบัญชี สมุดรายวัน หรือธุรกรรมทางการเงิน
_Avoid_: Ledger, journal entry, transaction

**Expense Category**:
หมวดค่าใช้จ่ายแบบแบนที่ Organization เป็นเจ้าของ ใช้จัดกลุ่มเอกสารค่าใช้จ่ายโดยไม่ใช่รหัสบัญชีหรือรายการลงบัญชี
_Avoid_: Chart of accounts, ledger account, journal entry

**Accounting Suggestion**:
ข้อเสนอหมวดค่าใช้จ่ายและคู่ค้าอย่างไม่เป็นทางการสำหรับ Document หนึ่งฉบับที่ยืนยันข้อมูลแล้ว ซึ่งต้องให้ผู้มีสิทธิรับรองก่อนส่งออก และไม่ใช่การลงบัญชี
_Avoid_: Accounting entry, authoritative posting, OCR result

**Document Review Item**:
งานตรวจที่ลงมือทำได้หนึ่งรายการสำหรับ Document และ revision ปัจจุบันภายใน Organization โดย revision ที่ถูกแทนที่ยังอยู่ในประวัติแต่ไม่รอการอนุมัติ
_Avoid_: Task, Document Source, historical revision

**Review Draft**:
ชุดการยอมรับ แก้ไข หรือระบุว่าไม่ทราบค่าของผู้ตรวจต่อ Document Review Item ซึ่งยังไม่ใช่การยืนยันข้อมูลหรืออนุมัติส่งออก
_Avoid_: Confirmed extraction, Approved Export Row, OCR result

**Approved Export Row**:
แถวข้อมูลของ Document revision ปัจจุบันที่มีทั้งข้อมูลสกัดที่ยืนยันแล้วและ Accounting Suggestion ที่อนุมัติแล้ว โดยไม่ใช่การลงบัญชี
_Avoid_: Raw Export, draft extraction, accounting entry

**Raw Document Export**:
ไฟล์ข้อมูล metadata ของ Document ที่ผู้ขอมีสิทธิ์ส่งออก โดยไม่รวมต้นฉบับ ข้อความ OCR หรือค่าที่สกัดแต่ยังไม่ยืนยัน
_Avoid_: Original-file bundle, OCR dump, Approved Export Row

**Suggestion Basis**:
ที่มาของ Accounting Suggestion เช่นกฎที่อนุมัติแล้ว การไม่มีคู่ที่แน่นอน หรือกฎที่ขัดกัน โดยไม่ใช่ค่าความน่าจะเป็นว่าข้อเสนอถูกต้อง
_Avoid_: OCR confidence, fabricated accuracy percentage

**Approved Mapping Rule**:
กฎใช้ซ้ำภายใน Organization ที่ Owner หรือ Admin อนุมัติอย่างชัดเจนเพื่อช่วยสร้าง Accounting Suggestion โดยการแก้ข้อเสนอเฉพาะเอกสารไม่สร้างกฎนี้เอง
_Avoid_: Automatic learning, cross-Organization mapping, one-off correction

**Archived**:
สถานะของ Business Contact หรือ Business Reference Data ที่ไม่ให้เลือกใช้กับงานใหม่ แต่ยังคงข้อมูลอ้างอิงและประวัติเดิมไว้และสามารถ Restore ได้
_Avoid_: Deleted, inactive Membership

**Import Job**:
คำขอนำเข้า Business Contact จากไฟล์ CSV หรือ XLSX ของ Organization เดียว ซึ่งตรวจทั้งไฟล์ก่อนสร้างรายการแบบ all-or-nothing และไม่แก้หรือ merge Business Contact เดิม
_Avoid_: Upsert, partial import, cross-Organization import

**Validation Preview**:
ผลตรวจ Import Job ก่อนสร้างข้อมูลที่แยกรายการพร้อมสร้าง รายการซ้ำ คำเตือน และข้อผิดพลาด โดยไม่มีการเปลี่ยน Business Contact
_Avoid_: Import result, partial commit

**Undo Import**:
การ Archive Business Contact ที่ Import Job หนึ่งสร้างขึ้นภายใน 24 ชั่วโมง โดยไม่ hard-delete หรือย้อนการแก้ข้อมูลเดิม
_Avoid_: Database rollback, delete import

**Export Template**:
ชุดคอลัมน์แบบคงที่และมี version สำหรับส่งออกข้อมูลชนิดหนึ่งจาก Organization โดย Phase 3 ไม่มี custom columns หรือ template ที่ผู้ใช้สร้างเอง
_Avoid_: Saved view, custom report, database dump

**Destination Mapping Template**:
รูปแบบไฟล์ส่งออกที่ PlaiFlow กำหนดและออกเวอร์ชันสำหรับโปรแกรมปลายทางหนึ่งแห่ง โดยแยกจากค่าจับคู่ที่ Organization กำหนดและไม่รับสูตรหรือคอลัมน์อิสระจากผู้ใช้
_Avoid_: Organization mapping values, generic Export Template, arbitrary spreadsheet

**Organization Drive Connection**:
การอนุญาต Google Drive แยกจาก Google Login ซึ่ง Owner หรือ Admin ของลูกค้าให้แก่ Organization โดยมีได้หนึ่ง active connection เพื่อเขียนไฟล์ส่งออกลงโฟลเดอร์ PlaiFlow ใน My Drive ของลูกค้า บัญชี Google ของลูกค้าเป็นเจ้าของพื้นที่และ PlaiFlow ไม่ได้จัดสรร Drive หรือโควตาพื้นที่ให้
_Avoid_: PlaiFlow-owned Drive, included storage, Google Login, whole-Drive access

**Reauthorization Required**:
สถานะของ Organization Drive Connection เมื่อ credential หมดอายุ ถูกถอน หรือใช้ต่อไม่ได้ โดยหยุดการส่งออกไป Drive จนกว่า Owner หรือ Admin จะให้ consent ใหม่
_Avoid_: Retryable provider error, disconnected

**Folder Action Required**:
สถานะของ Organization Drive Connection เมื่อโฟลเดอร์ปลายทางถูกลบหรือเข้าถึงไม่ได้ โดยต้องให้ Owner หรือ Admin ยืนยันการสร้างโฟลเดอร์ใหม่
_Avoid_: Automatic replacement, Reauthorization Required

**LINE Rich Menu**:
เมนูนำทางแบบคงที่ใน LINE ที่เปิดหน้าของ PlaiFlow โดยไม่ให้สิทธิ์เข้าถึงข้อมูล และให้ server ตรวจ Organization Context, Membership และ role ทุกครั้ง
_Avoid_: Authorization menu, role grant

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

**Durable Job**:
คำขอประมวลผลเบื้องหลังที่เก็บสถานะและผลลัพธ์ไว้จนตรวจสอบได้ โดย Phase 5 ใช้โมเดลกลางเดียวรองรับ `export` และ `ocr` แยกชนิดด้วย `kind`
_Avoid_: ad-hoc background task, request-local work

**Job Lease**:
สิทธิ์ชั่วคราวของ worker ที่ claim งานหนึ่งรายการ โดยมีวันหมดอายุและต่ออายุด้วย heartbeat งานที่ lease หมดอายุสามารถถูกนำกลับไปทำใหม่ได้
_Avoid_: permanent lock, worker ownership

**Export Artifact**:
ไฟล์ส่งออกที่สร้างเสร็จใน object storage และเข้าถึงผ่าน signed URL อายุสั้น โดยต้องตรวจ Organization และสิทธิ์เจ้าของงานทุกครั้ง
_Avoid_: database blob, public download URL

**OCR Result Artifact**:
ผลข้อความและตำแหน่งที่ OCR อ่านได้จากต้นฉบับ Document รุ่นหนึ่ง โดยผูกกับรุ่นโมเดลที่ใช้และเก็บแยกจากต้นฉบับ เอกสารเดียวอาจมีผลหลายรายการเมื่อประมวลผลใหม่ด้วยโมเดลคนละรุ่น
_Avoid_: Document original, extracted business fields, OCR Job

**Worker Protocol**:
ช่องทาง internal API สำหรับ claim, heartbeat, complete และ fail ของ Durable Job โดย worker ไม่มีสิทธิ์ต่อ PostgreSQL โดยตรง
_Avoid_: worker database credentials, direct table access

**Lease Token**:
ค่าที่ผูกกับการ claim และ attempt ของ Durable Job เพื่อป้องกัน worker เก่าส่ง completion หลัง lease หมดอายุ
_Avoid_: job ID เป็นตัวอนุมัติผลเพียงอย่างเดียว

**Export Artifact Retention**:
ไฟล์ export เก็บ 24 ชั่วโมง signed URL ใช้ได้ 15 นาที ส่วน job metadata เก็บ 90 วันและ audit การสร้าง/ดาวน์โหลด/ลบเก็บ 1 ปี
_Avoid_: permanent public export
