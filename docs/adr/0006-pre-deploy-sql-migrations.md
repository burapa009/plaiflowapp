# รัน SQL migrations ก่อน deployment

PlaiFlow จะเก็บ migration แบบขึ้นและลงใน `api/migrations` ใช้ `golang-migrate` รุ่นที่ pin ไว้ และรัน migration เป็น pre-deploy step เพียงจุดเดียว Application จะไม่แก้ schema อัตโนมัติระหว่าง startup เพื่อหลีกเลี่ยงการแข่งขันเมื่อมีหลาย instance
