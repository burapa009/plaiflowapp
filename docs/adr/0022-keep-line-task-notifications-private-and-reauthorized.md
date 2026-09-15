# จำกัด LINE Task Notification เป็น direct message ที่เปิดเผยข้อมูลขั้นต่ำ

Task อาจมีข้อมูลส่วนบุคคล Phase 2 จึงส่ง LINE เฉพาะ direct message แบบ opt-in ที่ไม่มี title, description, assignee หรือข้อมูลลูกค้า ไม่ fallback ไป LINE Group และให้ deep link ที่ไม่มี credential ตรวจ session, Membership, role และ Entitlement ใหม่ที่ปลายทางทุกครั้ง เพื่อไม่ให้ group membership หรือ frontend state กลายเป็นสิทธิ์เข้าถึงข้อมูล
