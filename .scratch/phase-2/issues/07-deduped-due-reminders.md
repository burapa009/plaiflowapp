# 07: Due Task สร้าง Reminder เพียงครั้งเดียว

**What to build:** Task ที่มี Due Date สร้าง Reminder ก่อนกำหนด วันกำหนด และหลังเลยกำหนดตาม Organization Timezone โดย scheduler scan หรือ worker retry ซ้ำไม่สร้าง Notification ซ้ำ และการเปลี่ยนวันหรือปิดงานยกเลิก schedule เก่า

**Blocked by:** 04 — ผู้ใช้เปลี่ยน Task Status, Due Date และ Priority ได้; 05 — Watcher ติดตาม Task และถูกถอดเมื่อ Membership สิ้นสุด

**Status:** ready-for-agent

- [ ] เปิด fresh Codex context ก่อนเริ่มงาน ticket นี้
- [ ] logical milestones อยู่ที่ 09:00 Organization Timezone หนึ่งวันก่อน Due Date, วัน Due Date และหนึ่งวันหลัง Due Date
- [ ] หลัง Due Date ส่งเฉพาะเมื่อ Task ยัง Open/InProgress; Done/Cancelled ไม่มี pending Reminder
- [ ] candidate recipients คือ current Assignee และ Watchers เท่านั้น; Task Creator รับเมื่อเป็นหนึ่งในความสัมพันธ์นั้น
- [ ] Reminder ใช้ DueReminder preference และ actor suppression ไม่ใช้กับ schedule-generated reminder
- [ ] changing Due Date ทำให้ unsent milestones เดิมใช้การไม่ได้และคำนวณ schedule ใหม่โดยไม่ลบประวัติที่ส่งแล้ว
- [ ] การสร้างหรือแก้ Task หลังหลาย milestone ผ่านไปสร้าง catch-up reminder ปัจจุบันสูงสุดหนึ่งรายการ ไม่ replay ทุก milestone
- [ ] unique identity ประกอบด้วย Organization, Task, recipient, milestone และ Due Date จึงทน repeated scan, crash และ retry
- [ ] scheduler ที่รันช้ายังสร้าง logical occurrence ได้หนึ่งครั้งและใช้ IANA calendar rules แทน fixed UTC offset
- [ ] worker ประมวลผล reminder แบบ bounded batch โดยไม่บล็อก Task mutation หรือ LINE provider call ใน request
- [ ] integration tests ครอบคลุมก่อน/ตรง/หลัง milestone, non-Bangkok timezone, late scan, changed date, Done/Cancelled, duplicate scan และ removed Membership

