# 01: นักพัฒนาเปิด PlaiFlow ในเครื่องได้จาก fresh checkout

**What to build:** นักพัฒนาสามารถ clone โปรเจกต์ เปิด PostgreSQL ใน Docker Compose, apply migrations, รัน Next.js dashboard shell, Go API และ Python shell ได้ด้วยขั้นตอนที่บันทึกไว้ และ CI ตรวจ baseline เดียวกันได้

**Blocked by:** None (can start immediately)

**Status:** ready-for-agent

- [ ] เปิด fresh Codex context ก่อนเริ่มงาน ticket นี้
- [ ] โครง repo รองรับ web, api, python และ docs โดยแต่ละ runtime มีคำสั่งตรวจสอบที่ชัดเจน
- [ ] PostgreSQL local boot และ migrations ทำงานซ้ำได้โดยไม่ทำข้อมูลเสียหาย
- [ ] Go health endpoint, Next dashboard shell และ Python shell ทำงานได้ใน local
- [ ] CI รัน format, lint/typecheck, tests และ migration smoke check โดยไม่ใช้ secrets จริง
- [ ] local database bind เฉพาะ localhost และ CI actions ใช้ permissions ต่ำสุดพร้อม version pinning
