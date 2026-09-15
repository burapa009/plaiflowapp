# ใช้ Task lifecycle แบบกะทัดรัดและไม่ลบประวัติ

Phase 2 ให้ Task มีหนึ่ง Assignee, หลาย Watcher, สถานะ Open, InProgress, Done หรือ Cancelled, Due Date แบบวันที่ตาม Organization Timezone และ Priority แบบ Normal, High หรือ Urgent โดย Overdue เป็นเงื่อนไขที่คำนวณและไม่มีการลบ Task เพื่อให้ assignment, reminder, filter, audit และ CSV ใช้ความหมายชุดเดียวโดยไม่สร้าง state ซ้ำซ้อน
