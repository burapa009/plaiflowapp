# บังคับ tenant boundary ทั้งใน Go และ PostgreSQL

Go บังคับ Membership และ RBAC ส่วน tenant-owned tables ทุกตารางมี `organization_id`, ใช้ composite constraints เมื่อความสัมพันธ์ห้ามข้าม tenant และเปิด PostgreSQL Row-Level Security ด้วย transaction-local Organization Context runtime roles ไม่มี `BYPASSRLS` และแยกจาก migration/worker roles เพื่อให้ query ที่ลืม tenant predicate ไม่กลายเป็น cross-tenant disclosure ชั้นเดียว
