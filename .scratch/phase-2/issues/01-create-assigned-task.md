# 01: ผู้ใช้สร้าง มอบหมาย และเปิดดู Task แรกได้

**What to build:** สมาชิกสร้าง Task ผ่าน Web ภายใน Organization ปัจจุบัน เลือก Assignee ได้ไม่เกินหนึ่งราย แล้วเปิดดู Task และรายการงานของ Organization ได้โดยข้อมูลและความสัมพันธ์ทุกส่วนถูกจำกัดอยู่ใน tenant เดียว

**Blocked by:** None (can start immediately)

**Status:** ready-for-agent

- [ ] เปิด fresh Codex context ก่อนเริ่มงาน ticket นี้
- [ ] Task มี Organization, title, optional description, immutable Task Creator, optional Assignee, Task Status, Priority, optional Due Date และ timestamps ตาม Phase 2 spec
- [ ] title ถูก trim และยาว 1–200 Unicode characters; description ยาวไม่เกิน 5,000 Unicode characters และไม่ปรากฏใน logs
- [ ] Task ใหม่เริ่ม Open และ Normal โดยไม่มี Due Date เว้นแต่ผู้ใช้ระบุ
- [ ] ผู้ใช้ที่ authenticated สร้าง Task ผ่าน Web flow ที่ตรวจ Organization Context, active Membership, same-origin และ CSRF ฝั่ง server
- [ ] Assignee ต้องเป็น active Membership ใน Organization เดียวกันทั้งจาก Go authorization, database constraint และ RLS
- [ ] active Membership อ่าน Task ใน Organization ตนได้ แต่ identifier จาก Organization อื่นให้ผลแบบ not found โดยไม่เปิดเผยว่ามีข้อมูลอยู่
- [ ] การสร้าง Task กับ initial Assignee บันทึก task.created Domain Event ใน transaction เดียวกันเพียงหนึ่งรายการ และไม่คัดลอก title/description ลง event payload
- [ ] Domain Event รองรับ Web command Causation โดยไม่สร้าง Inbound Event ปลอม
- [ ] Web UI มี create, list และ detail flow พร้อม loading, empty, validation และ safe error states ที่ใช้งานได้ด้วย keyboard
- [ ] integration test ครอบคลุม create/read สำเร็จ, invalid input, foreign Assignee, unauthenticated request, CSRF failure และ cross-tenant read

