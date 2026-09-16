# 04: Preview Vendor Import โดยไม่เขียนข้อมูล

**What to build:** Owner/Admin ที่มี Import Entitlement อัปโหลด CSV/XLSX, จับคู่คอลัมน์ และดู Validation Preview ของ Vendors ก่อนเกิดการเขียน Business Contact ใด ๆ

**Blocked by:** 01/เปรียบเทียบ Plan จาก authoritative catalog; 02/สร้างและค้นหา Vendor พร้อมป้องกัน Tax ID ซ้ำ.

**Status:** ready-for-agent

- [ ] รับเฉพาะ CSV UTF-8 และ XLSX หนึ่ง data sheet ภายใต้ขนาด จำนวนแถว คอลัมน์ cell และ parse-time limits ที่กำหนด
- [ ] Auto-map เฉพาะ Thai/English header aliases ที่ allowlist และให้หนึ่ง source column map ได้ไม่เกินหนึ่ง target
- [ ] Preview แยก Ready, Duplicate to Skip, Warning และ Error พร้อม representative rows โดยยังไม่มี Business Contact ถูกสร้าง
- [ ] Formula, macro, encrypted/legacy workbook, malformed archive และชนิดไฟล์ที่ extension/MIME/signature ไม่ตรงกัน fail safely โดยไม่คืน raw parser error หรือเนื้อหาไฟล์
- [ ] ตรวจ duplicate กับทั้ง active/Archived records และภายในไฟล์; แถวซ้ำไม่ถูกเสนอเป็น update หรือ merge
- [ ] มี runnable check ยืนยัน write-free Preview, entitlement enforcement และ bounded hostile-file handling
