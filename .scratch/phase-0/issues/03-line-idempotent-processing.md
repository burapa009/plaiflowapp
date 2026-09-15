# 03: LINE event ที่ส่งซ้ำถูกเก็บและประมวลผลเพียงครั้งเดียว

**What to build:** การส่ง LINE event เดิมซ้ำได้รับ acknowledgement อย่างปลอดภัย ถูกเก็บเพียงครั้งเดียว และ worker ประมวลผล event ที่ยอมรับแล้วเพียงครั้งเดียว พร้อม state transition ที่ตรวจสอบได้

**Blocked by:** 02 — LINE webhook ที่ถูกต้องได้รับการตรวจสอบและตอบรับอย่างปลอดภัย

**Status:** ready-for-agent

- [ ] เปิด fresh Codex context ก่อนเริ่มงาน ticket นี้
- [ ] database constraint บังคับ uniqueness ของ provider, channel และ webhook event ID
- [ ] duplicate delivery ไม่สร้างแถวหรือ side effect ใหม่ และยังตอบรับได้อย่างปลอดภัย
- [ ] worker claim งานด้วย lease และไม่แย่งงานเดียวกันโดยไม่จำเป็น
- [ ] LINE message ถูกบันทึกเป็น `Processed` แบบ record-only; ยังไม่มี outbound LINE
- [ ] มี Domain Event envelope shell และ integration tests สำหรับ duplicate delivery
