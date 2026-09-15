# สร้าง Organization Context ฝั่ง server สำหรับทุกคำขอ

เส้นทาง tenant ใช้ immutable Organization UUID เช่น `/o/{organization_id}` แต่ค่าจาก URL เป็นเพียง requested organization Go ต้องอ่าน User จาก session และตรวจ Membership ก่อนสร้าง Organization Context ทุกครั้ง การ switch organization จึงเป็น navigation ที่ไม่แก้ session ช่วยให้ role removal มีผลทันทีและหลีกเลี่ยง org spoofing หรือ stale tenant state
