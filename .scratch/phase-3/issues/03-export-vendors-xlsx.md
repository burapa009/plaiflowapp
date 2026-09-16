# 03: Export Vendors เป็น XLSX อย่างปลอดภัย

**What to build:** Owner/Admin ที่มี XLSX Export Entitlement ส่งออก Vendors ของ Organization เป็นไฟล์ XLSX ตาม template คงที่ได้ โดย authorization และ Entitlement ถูกตรวจจาก server และค่าจากผู้ใช้ไม่กลายเป็นสูตร spreadsheet

**Blocked by:** 01/เปรียบเทียบ Plan จาก authoritative catalog; 02/สร้างและค้นหา Vendor พร้อมป้องกัน Tax ID ซ้ำ.

**Status:** ready-for-agent

- [ ] Owner/Admin ส่งออกเฉพาะ Vendors ของ Organization ปัจจุบันด้วย template/version ที่ server รู้จัก และ Member ถูกปฏิเสธ
- [ ] Server ตรวจ session, Organization Context, Membership, RBAC และ effective Entitlement ใหม่ก่อน request และก่อน download
- [ ] การส่ง Plan หรือ Capability identifier ที่แก้ไขจาก client ไม่ปลดล็อก XLSX Export
- [ ] Thai text, comma, quote, line break, Unicode และค่าที่ขึ้นต้นเหมือน formula ถูกเขียนเป็น text อย่างปลอดภัย
- [ ] ผลลัพธ์บังคับ row/file limit และไม่โหลดข้อมูลทั้งชุดแบบไร้ขอบเขต
- [ ] มี runnable check เปิด workbook ที่สร้างแล้วตรวจ headers, Vendor rows, text cells และ tenant isolation
