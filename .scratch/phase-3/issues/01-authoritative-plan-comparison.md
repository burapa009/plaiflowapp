# 01: เปรียบเทียบ Plan จาก authoritative catalog

**What to build:** ผู้ใช้ที่เข้าจาก LINE หรือ Web เปรียบเทียบ Free, Starter และ Business แบบ Monthly, 6-month และ Yearly ได้ โดยราคา รายละเอียด Entitlement และ effective Plan ของ Organization มาจาก Plan Catalog ฝั่ง server เท่านั้น การเลือก Plan เป็นการดูข้อมูลหรือส่ง Upgrade Request และไม่เปลี่ยนสิทธิ์จริงเอง

**Blocked by:** None (can start immediately).

**Status:** ready-for-agent

- [ ] หน้าเปรียบเทียบแบบ responsive ใช้งานได้ทั้ง Web และ LINE in-app browser โดยแสดงยอดชำระล่วงหน้าและราคาต่อเดือนที่แท้จริงของทั้งสามช่วงเวลา
- [ ] 6-month ใช้สมมติฐาน Early Access ลด 8% จากยอด Monthly หกเดือน โดยคำนวณและส่งจำนวนเงินจาก server ไม่คำนวณจากค่าที่ client ส่งมา
- [ ] Server ตอบ Plan Catalog และ effective Entitlements จาก fixed Plan/Capability identifiers ที่ server เป็นเจ้าของ
- [ ] การแก้ Plan, interval, price, capability หรือ Usage identifier ใน request ไม่เปลี่ยน effective Entitlements และไม่ปลดล็อก paid capability
- [ ] Unknown หรือ stale Plan key fail closed สำหรับ paid mutation โดยไม่ปิดกั้น login, read หรือ security/recovery action ที่ต้องคงไว้
- [ ] มี runnable check ครอบคลุมการเปลี่ยน identifier ฝั่ง client และยืนยันว่า Upgrade Request ไม่ให้สิทธิ์
