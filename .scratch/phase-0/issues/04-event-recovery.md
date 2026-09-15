# 04: Inbound Event ที่ล้มเหลวหรือ worker ทิ้งงานสามารถฟื้นตัวได้

**What to build:** operator และระบบสามารถเห็นว่า Inbound Event ที่ประมวลผลไม่สำเร็จจะ retry ตามลำดับที่กำหนด, lease ที่หมดอายุจะกลับมาทำงานได้ และ event ที่ลองครบแล้วจะจบอย่างชัดเจน

**Blocked by:** 03 — LINE event ที่ส่งซ้ำถูกเก็บและประมวลผลเพียงครั้งเดียว

**Status:** ready-for-agent

- [ ] เปิด fresh Codex context ก่อนเริ่มงาน ticket นี้
- [ ] lease expiry ทำให้ event ที่ worker หยุดกลางทางกลับมา claim ได้
- [ ] retry schedule เป็น 1m, 5m, 30m, 2h และ 12h พร้อม terminal `Failed`
- [ ] event type ที่ไม่รู้จักจบเป็น `Ignored` โดยไม่ทำให้ queue ค้าง
- [ ] การ retry ไม่สร้าง duplicate side effect
- [ ] มี tests ครบสำหรับ success, retryable failure, exhausted failure และ abandoned lease
