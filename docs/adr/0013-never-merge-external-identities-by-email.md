# ไม่รวม External Identity อัตโนมัติจาก email

LINE และ Google ใช้สมัครหรือเข้าสู่ระบบได้ โดย External Identity อ้างอิงด้วย `(issuer, subject)` และห้ามรวม User จาก email ที่ตรงกันแม้ provider ระบุว่า verified การเชื่อม identity ใหม่ต้องเริ่มจาก authenticated session พร้อม strong verification และต้องหยุดเมื่อ identity นั้นเป็นของ User อื่นอยู่แล้ว ยอมให้เกิด User ซ้ำที่กู้คืนภายหลังได้เพื่อหลีกเลี่ยง accidental merge และ account takeover
