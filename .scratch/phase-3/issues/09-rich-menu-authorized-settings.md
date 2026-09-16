# 09: Rich Menu เปิด Settings ที่ตรวจ authorization ใหม่

**What to build:** ผู้ใช้เปิด Organization Settings จาก LINE Rich Menu หรือ Web navigation ได้ โดยปลายทางสร้าง authorization context จาก session และ Membership ปัจจุบันทุกครั้ง ไม่เชื่อค่า Organization, role หรือ Entitlement จาก URL, LINE หรือ browser state

**Blocked by:** 01/เปรียบเทียบ Plan จาก authoritative catalog; 06/OWNER เชื่อม Google Drive ด้วย scope ขั้นต่ำ.

**Status:** ready-for-agent

- [ ] Static Rich Menu มี HTTPS URI action ไป Settings ที่ถูกต้องและไม่ฝัง credential, role, Plan หรือ trusted Organization Context
- [ ] ผู้ใช้ที่มีหนึ่ง Membership เข้า Organization นั้นได้ ผู้ใช้หลาย Organization ต้องเลือก Organization และผู้ไม่มี Membership ถูกปฏิเสธอย่างปลอดภัย
- [ ] Settings แสดง Plan/Usage และ Drive status ที่ไม่อ่อนไหวตามสิทธิ์; OWNER เท่านั้นเห็น Drive connect/disconnect controls ตาม tickets ชุดนี้
- [ ] Direct URL, modified query, local storage และ last-selected Organization ไม่ข้าม session, Membership, RBAC หรือ Entitlement checks
- [ ] Web navigation มี Settings destination เดียวกันเมื่อ Rich Menu ใช้ไม่ได้
- [ ] มี runnable provider-double check สำหรับ Rich Menu object และ authorization check สำหรับ single-, multi- และ zero-membership paths
