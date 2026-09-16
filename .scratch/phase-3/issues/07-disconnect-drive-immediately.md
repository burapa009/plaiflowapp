# 07: Disconnect Drive และหยุดใช้ credential ทันที

**What to build:** OWNER ตัดการเชื่อมต่อ Google Drive แล้ว PlaiFlow หยุดเริ่มงานเขียนใหม่และไม่ใช้ credential เดิมอีกทันที โดยไฟล์และโฟลเดอร์ที่ลูกค้าเป็นเจ้าของยังคงอยู่

**Blocked by:** 06/OWNER เชื่อม Google Drive ด้วย scope ขั้นต่ำ.

**Status:** ready-for-agent

- [ ] Disconnect ตรวจ current OWNER, recent reauthentication และ Organization Context ใหม่ และไม่ถูก commercial gate ขวาง
- [ ] การเปลี่ยนสถานะกับการลบ token material ทำให้ worker claim หรือ retry หลัง disconnect ใช้ credential เดิมไม่ได้
- [ ] ระบบพยายาม revoke ที่ provider แบบ bounded แต่ local disconnect สำเร็จอย่างปลอดภัยแม้ provider ไม่ตอบ
- [ ] งานที่กำลังทำอยู่ตรวจ connection generation/status ก่อน provider write และหยุดโดยไม่สร้างงานซ้ำ
- [ ] ไม่ลบหรือแก้ไข customer-owned Drive folder/files และ audit เก็บเฉพาะ safe outcome
- [ ] มี runnable concurrency check ยืนยันว่าไม่มี provider call ใหม่หลัง disconnect commit
