# ป้องกัน staging ด้วย server-to-server boundary

Phase 0 จะใช้ Vercel Deployment Protection จำกัดการเข้าถึง dashboard และให้ browser ติดต่อ Next.js แบบ same-origin จากนั้น Next.js ฝั่ง server จึงเรียก Railway API ด้วย service token ที่ไม่เปิดเผยต่อ browser แนวทางนี้แทนที่แผนเดิมที่ให้ browser เรียก Railway โดยตรงผ่าน CORS และไม่ถือเป็นระบบ user authentication ของ PlaiFlow LINE staging channel ใช้เฉพาะทีม บัญชีทดสอบ และข้อมูล synthetic เท่านั้น
