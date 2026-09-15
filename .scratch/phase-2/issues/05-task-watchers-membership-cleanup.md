# 05: Watcher ติดตาม Task และถูกถอดเมื่อ Membership สิ้นสุด

**What to build:** Task Creator หรือผู้ดูแลเพิ่มและถอด Watcher ภายใน Organization ได้ ผู้ติดตามใหม่ได้รับ Notification โดยไม่รับ edit permission และเมื่อ Membership สิ้นสุด assignment/watch relationships ถูกล้างอย่างปลอดภัยโดยประวัติผู้สร้างยังอยู่

**Blocked by:** 03 — Assigned Task ปรากฏใน Web และ LINE

**Status:** ready-for-agent

- [ ] เปิด fresh Codex context ก่อนเริ่มงาน ticket นี้
- [ ] Watcher relation unique และอ้าง active Membership ใน Organization เดียวกับ Task ด้วย application check, composite constraint และ RLS
- [ ] Task Creator และ Owner/Admin เพิ่ม/ถอด Watcher ได้; Watcher status อย่างเดียวไม่ให้สิทธิ์แก้ Task
- [ ] Assignee เป็น implicit recipient และไม่ถูกเก็บซ้ำเป็น Watcher; เมื่อ Watcher กลายเป็น Assignee ระบบลบ relation ที่ซ้ำ
- [ ] การเพิ่ม Watcher สร้าง task.watcher_added และแจ้งเฉพาะผู้ถูกเพิ่มหลัง actor suppression
- [ ] การถอด Watcher สร้าง task.watcher_removed แต่ไม่สร้าง Notification ให้ผู้ถูกถอดหรือ Watcher คนอื่น
- [ ] Task changes หลังจากนั้นแจ้ง Assignee และ current Watchers ตาม recipient rules โดยไม่แจ้ง Task Creator/Owner/Admin อัตโนมัติ
- [ ] การนำ Membership ออกจาก Organization ทำงาน atomic: unassign active Tasks, ลบ Watcher relations และหยุด pending notifications/deliveries ของ Membership นั้น
- [ ] Membership removal ไม่ถูกบล็อกเพราะยังมี Task และ former Assignee ไม่กลายเป็น Watcher อัตโนมัติ
- [ ] Task Creator identity และ Domain Event history ยังคงตรวจสอบได้หลัง Membership สิ้นสุด
- [ ] UI แสดง Assignee/Watcher แยกความหมายและมี loading/empty/error states ที่ปลอดภัย
- [ ] integration test ครอบคลุม duplicate watcher, foreign Membership, actor suppression, membership cleanup, historical creator และ cross-tenant attempts

