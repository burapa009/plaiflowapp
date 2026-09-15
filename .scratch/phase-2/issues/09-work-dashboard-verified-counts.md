# 09: Work Dashboard แสดง verified counts ตาม role

**What to build:** สมาชิกเปิด Work Dashboard แล้วเห็นจำนวนและรายการที่ตรวจสอบกลับกับ Task/Notification จริงได้ โดย Member เห็นเฉพาะ personal aggregates ส่วน Owner/Admin เห็น Team Queue เพิ่มเติมและ Operations Dashboard เดิมยังแยก authorization ชัดเจน

**Blocked by:** 06 — ผู้ใช้กรองและแบ่งหน้ารายการ Task ได้อย่างเสถียร; 08 — ผู้ใช้ตั้ง NotificationPreference และรับ Digest ได้

**Status:** ready-for-agent

- [ ] เปิด fresh Codex context ก่อนเริ่มงาน ticket นี้
- [ ] Work Dashboard ใช้ authenticated Organization Context และไม่ใช้ service-token boundary ของ Operations Dashboard
- [ ] read model มี generated time, effective role, My Focus, Watching และ unread Notifications พร้อมรายการสูงสุด widget ละ 10
- [ ] My Focus แยก overdue, due today, due next seven days และ Urgent assigned Tasks จากกฎ Task เดียวกับหน้ารายการ
- [ ] Owner/Admin ได้ Team Queue สำหรับ unassigned, overdue, Urgent และ counts by Assignee
- [ ] Member response ไม่มี per-person aggregates ของสมาชิกอื่น; ทุก role ไม่มี leaderboard, completion rate, ranking หรือ productivity score
- [ ] verified counts และ item lists มาจาก authorized indexed data เดียวกันและมี tests เทียบกับ Task/Notification fixtures ที่ทราบผล
- [ ] dashboard ใช้หนึ่ง HTTP request, ไม่เกินห้า SQL statements, ไม่มี N+1 และมีเป้าหมาย p95 ไม่เกิน 500 ms ที่ reference data
- [ ] query-on-read เท่านั้น ไม่มี materialized view, shared cross-tenant cache, WebSocket หรือ automatic polling
- [ ] Web UI มี manual refresh, responsive/loading/empty/error states และ deep links ที่ตรวจสิทธิ์ใหม่
- [ ] Operations Dashboard และ system-health route เดิมยังทำงานและไม่เปิด Organization work data
- [ ] integration tests ครอบคลุม Member/Owner/Admin payloads, cross-tenant IDs, count accuracy, top-10 caps และ query budget evidence

