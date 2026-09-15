# 08: ข้อมูล Inbound Event หมดอายุตามนโยบายโดยไม่ทำลาย duplicate protection

**What to build:** ระบบลบ payload ที่หมดอายุตาม retention policy แต่ยังคง metadata และ idempotency information ไว้พอให้ duplicate protection ทำงานตามช่วงเวลาที่กำหนด

**Blocked by:** 03 — LINE event ที่ส่งซ้ำถูกเก็บและประมวลผลเพียงครั้งเดียว

**Status:** ready-for-agent

- [ ] เปิด fresh Codex context ก่อนเริ่มงาน ticket นี้
- [ ] payload retention เป็น 30 วัน
- [ ] metadata และ idempotency retention เป็น 90 วัน
- [ ] cleanup ทำซ้ำได้ ปลอดภัย และไม่ทำลาย event ที่ยังอยู่ใน retry/lease window
- [ ] มี tests สำหรับ boundary dates และ duplicate lookup หลัง payload ถูกลบ
- [ ] บันทึก residual risk เรื่อง replay หลังครบ 90 วันอย่างชัดเจน
