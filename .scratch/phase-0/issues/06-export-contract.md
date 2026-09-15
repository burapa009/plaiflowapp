# 06: Go และ Python ตรวจสอบ export contract เดียวกันได้

**What to build:** Go API และ Python shell สามารถแลกเปลี่ยน versioned export request/result fixture เดียวกันและตรวจ contract ร่วมกันได้ เพื่อวาง seam สำหรับอนาคตโดยยังไม่ทำ business export

**Blocked by:** 01 — นักพัฒนาเปิด PlaiFlow ในเครื่องได้จาก fresh checkout

**Status:** ready-for-agent

- [ ] เปิด fresh Codex context ก่อนเริ่มงาน ticket นี้
- [ ] contract มี version และ validation ที่ deterministic ระหว่าง Go กับ Python
- [ ] contract tests ผ่านใน CI และ local
- [ ] Python shell เริ่ม/หยุดได้และรายงานผลตาม contract
- [ ] ไม่มี OCR, accounting, AI advisor, export job จริง หรือ live export deployment
