# นำเข้า Business Contact แบบตรวจทั้งไฟล์และสร้างรายการใหม่เท่านั้น

Import Job ของ Phase 3 ตรวจ CSV/XLSX ทั้งไฟล์ก่อน commit และสร้าง Business Contact แบบ all-or-nothing โดยไม่ update หรือ merge รายการเดิม strong duplicate ถูกข้ามและรายงาน ส่วน Undo ภายใน 24 ชั่วโมงใช้การ Archive รายการที่ batch นั้นสร้าง แนวทางนี้ยอมลดความสะดวกของการ upsert เพื่อป้องกันการเขียนทับข้อมูลภาษีจำนวนมากและทำให้ผลลัพธ์กับการย้อนกลับตรวจสอบได้
