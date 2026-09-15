# 06: ผู้ใช้กรองและแบ่งหน้ารายการ Task ได้อย่างเสถียร

**What to build:** สมาชิกเปิด My Tasks, Watching หรือ All Tasks แล้วกรองสถานะ ผู้รับผิดชอบ ผู้สร้าง Priority, Overdue และช่วงวันที่ได้โดย cursor ไม่ทำรายการซ้ำ/ตกหล่นและ query อยู่ในงบเมื่อ Organization มีข้อมูลตาม reference scale

**Blocked by:** 04 — ผู้ใช้เปลี่ยน Task Status, Due Date และ Priority ได้; 05 — Watcher ติดตาม Task และถูกถอดเมื่อ Membership สิ้นสุด

**Status:** ready-for-agent

- [ ] เปิด fresh Codex context ก่อนเริ่มงาน ticket นี้
- [ ] default view คือ My Tasks ที่มี Open/InProgress และผู้ใช้สลับ Watching/All Tasks ได้โดยไม่เปลี่ยน authorization
- [ ] filters รองรับ multiple statuses, me/unassigned/specific Assignee, Watcher me, Task Creator, multiple priorities, Overdue, Due Date range และ created-time range
- [ ] Due Date ranges เป็น inclusive local dates; created ranges ใช้ local-day boundaries ที่แปลงเป็น UTC ตาม Organization Timezone
- [ ] ไม่มี title/description full-text search หรือ arbitrary sort/filter expression ใน v1
- [ ] active ordering คือ Overdue ก่อน, Due Date ใกล้สุดโดย null อยู่ท้าย, Urgent/High/Normal และ stable Task ID; Done/Cancelled history เรียง updated time descending แล้ว Task ID
- [ ] ใช้ opaque cursor 50 rows default และ 100 maximum; malformed cursor ถูกปฏิเสธและ cursor ไม่ให้สิทธิ์เข้าถึงข้อมูล
- [ ] ทุกหน้าประเมิน Organization Context, role และ normalized filters ใหม่ และ foreign identifiers ให้ผลแบบ not found
- [ ] tenant-first indexes รองรับ active status/due, Assignee, created range และ Watcher query โดยไม่สร้าง index ต่อทุก filter combination
- [ ] query plans ถูกตรวจด้วย reference data 100,000 Tasks, 10,000 active Tasks และ 100 Memberships ต่อ Organization
- [ ] Task list API มีเป้าหมาย p95 ไม่เกิน 300 ms ที่ reference data และไม่โหลด collection แบบ unbounded
- [ ] Web UI มี filter state ที่แชร์กลับ server ได้, clear filters, pagination, loading, empty และ safe error states
- [ ] integration tests พิสูจน์ filter combinations, date boundaries, stable pagination ระหว่างหน้าที่มีค่าซ้ำ, role behavior และ tenant isolation

