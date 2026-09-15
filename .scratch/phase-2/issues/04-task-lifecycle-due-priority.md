# 04: ผู้ใช้เปลี่ยน Task Status, Due Date และ Priority ได้

**What to build:** ผู้มีสิทธิ์เปลี่ยน Task ระหว่าง Open, InProgress, Done และ Cancelled กำหนด Due Date แบบวันที่และ Priority ได้ พร้อม completion attribution, derived Overdue และ Domain Event ที่ตรวจสอบย้อนหลังได้โดยไม่มี Task deletion

**Blocked by:** 03 — Assigned Task ปรากฏใน Web และ LINE

**Status:** ready-for-agent

- [ ] เปิด fresh Codex context ก่อนเริ่มงาน ticket นี้
- [ ] Owner/Admin แก้ Task ใดก็ได้; Task Creator แก้รายละเอียด/assignment/due/priority ของ Task ตน; Assignee เปลี่ยน Task Status ได้
- [ ] Task Status รองรับเฉพาะ Open, InProgress, Done และ Cancelled และไม่ใช้สถานะของ Inbound Event
- [ ] เข้า Done แล้วบันทึก current completion actor/time; reopen กลับ Open และล้าง current completion fields โดย event history เดิมยังอยู่
- [ ] Cancelled ถูกซ่อนจาก active list แต่ยังเปิดดู กรอง และใช้งานในอนาคตได้; ไม่มี hard-delete หรือ soft-delete action
- [ ] Due Date เป็น nullable date-only และ Organization Timezone เป็น IANA timezone ที่ default Asia/Bangkok
- [ ] Overdue คำนวณหลังสิ้นวัน Due Date เฉพาะ Open/InProgress และไม่ถูกเก็บเป็น Task Status
- [ ] Priority รองรับ Normal, High และ Urgent โดย default Normal และไม่เปลี่ยน reminder, authorization หรือ Entitlement อัตโนมัติ
- [ ] การเปลี่ยนรายละเอียด, assignment, due, priority และ status สร้าง Task Domain Event เฉพาะชนิด โดยไม่มี task.updated
- [ ] event payload บันทึก identifiers/changed fields ที่จำเป็นและไม่คัดลอก sensitive field values
- [ ] Web UI แสดงสถานะ วันที่ และ Priority พร้อม accessible validation, safe conflicts และ timezone ที่ชัดเจน
- [ ] integration test ครอบคลุม role matrix, transitions, completion/reopen, cancellation, timezone boundary, Overdue และ rejected mutation ที่ไม่สร้าง Domain Event

