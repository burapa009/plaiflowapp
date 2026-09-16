# รักษาข้อมูลและสิทธิ์อ่านเมื่อ Plan เกินขีดจำกัด

เมื่อ Trial หรือ Plan สิ้นสุดแล้ว Organization มีสมาชิกหรือ LINE Group Connection เกินขีดจำกัด ระบบเข้าสู่ Plan Over Limit แทนการลบ ระงับสมาชิก หรือตัดการเชื่อมต่อโดยอัตโนมัติ ผู้ใช้เดิมยัง login และอ่านข้อมูล ส่วน Owner/Admin ยังส่งออก CSV ได้ตาม authorization เดิมและทุก role ยังใช้ security controls ได้ แต่ mutation ปกติและ provider delivery หยุดจนกว่า Owner/Admin จะลดทรัพยากรให้อยู่ในขีดจำกัดหรือได้รับ Plan ใหม่ แนวทางนี้ปิดช่องใช้ paid collaboration ต่อโดยไม่จ่ายพร้อมหลีกเลี่ยง data loss และ lockout
