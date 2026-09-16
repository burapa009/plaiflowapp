# 05: ยืนยัน Vendor Import แบบ all-or-nothing

**What to build:** Owner/Admin ยืนยัน Preview ที่ยังใช้ได้แล้วสร้าง Vendors ที่ผ่าน validation เป็น batch เดียวอย่างปลอดภัย โดย error ไม่ทิ้งข้อมูลครึ่งชุดและ duplicate ไม่เปลี่ยนข้อมูลเดิม

**Blocked by:** 04/Preview Vendor Import โดยไม่เขียนข้อมูล.

**Status:** ready-for-agent

- [ ] Commit ตรวจ session, Organization Context, Membership, RBAC, effective Entitlement และ Preview expiry ใหม่ก่อนเขียน
- [ ] Preview ที่มี Error ถูกปฏิเสธทั้งหมด; Warning ต้องได้รับ explicit confirmation
- [ ] Strong duplicate ถูก skip และรายงาน โดยไม่ update, merge, เพิ่ม role หรือเปิดใช้งานรายการเดิม
- [ ] Valid non-duplicate rows ถูกสร้างใน transaction แบบ all-or-nothing และ retry เดิมไม่สร้างซ้ำ
- [ ] ผลลัพธ์ผูกกับ Import Job และมีจำนวน created, skipped, warning และ safe failure category ที่ตรวจสอบย้อนหลังได้โดยไม่เก็บข้อมูลอ่อนไหวใน audit
- [ ] มี runnable check ครอบคลุม concurrent duplicate, rollback ทั้ง batch, stale Preview และ unauthorized commit
