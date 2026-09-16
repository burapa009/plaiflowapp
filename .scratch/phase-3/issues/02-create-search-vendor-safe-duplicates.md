# 02: สร้างและค้นหา Vendor พร้อมป้องกัน Tax ID ซ้ำ

**What to build:** Owner/Admin สร้าง Business Contact ที่มี Vendor role และค้นหา Vendor ภายใน Organization ได้ ส่วน Member ค้นหาและเลือกเฉพาะรายการ active ได้ โดย duplicate identity ทางภาษีไม่สร้างหรือเขียนทับข้อมูลเดิมโดยไม่ตั้งใจ

**Blocked by:** None (can start immediately).

**Status:** ready-for-agent

- [ ] Owner/Admin สร้าง Vendor ด้วยข้อมูลขั้นต่ำและกำหนด Customer role ร่วมกันได้ ส่วน Member ไม่มีสิทธิ์สร้างหรือแก้ไข
- [ ] Thai Tax ID ตัด visual separators แล้วต้องมี 13 หลักและ checksum ถูกต้อง; Head Office ใช้ branch `00000` และสาขาอื่นใช้ห้าหลัก
- [ ] Strong duplicate ภายใน Organization ใช้ country, normalized Tax ID และ canonical branch; ข้อขัดแย้งตอบผลลัพธ์ที่ปลอดภัยโดยไม่ overwrite, merge หรือเปิดเผยข้อมูลข้าม Organization
- [ ] Owner/Admin ค้นหา exact Tax ID และค้นหาชื่อ/Contact Code ได้; Member เห็นเฉพาะข้อมูลที่ได้รับอนุญาตและ Tax ID ถูก mask เมื่อจำเป็น
- [ ] รายการ Archived ยังร่วม uniqueness แต่ไม่ปรากฏในผลค้นหา active โดยปริยาย
- [ ] มี runnable check ครอบคลุม tenant isolation, role checks, Tax ID checksum และ duplicate race ที่ database boundary
