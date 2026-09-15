# 09: Owner มอบสิทธิ์ Admin และโอน ownership

**What to build:** Organization ใช้ Owner/Admin/Member permissions แบบตายตัว โดย Owner จัดการ Admin และโอน ownership ได้อย่าง atomic ขณะที่ Admin และ Member ถูกจำกัดตามบทบาท

**Blocked by:** 08 — Owner/Admin เชิญ Member และผู้รับตอบรับคำเชิญ

**Status:** ready-for-agent

- [ ] เปิด fresh Codex context ก่อนเริ่มงาน ticket นี้
- [ ] permission checks อยู่ฝั่ง server และทดสอบ matrix ของ Owner, Admin และ Member กับทุก privileged endpoint ที่มีใน Phase 1
- [ ] Owner promote/demote Admin ได้; Admin จัดการ Member ได้แต่จัดการ Admin/Owner หรือโอน ownership ไม่ได้
- [ ] ownership transfer lock Organization/Membership ที่เกี่ยวข้องและเปลี่ยน Owner เดิม/ใหม่ใน transaction เดียว
- [ ] database บังคับ at-most-one Owner และ guards บังคับให้มี Owner อย่างน้อยหนึ่งคนหลัง transfer/leave
- [ ] concurrent ownership transfers มีผู้ชนะหนึ่งรายการและไม่ทิ้ง Organization ที่ไม่มีหรือมีหลาย Owner
- [ ] Admin และ Member leave ได้; Owner ต้อง transfer ownership ก่อน leave
- [ ] role change มีผลใน request ถัดไปเพราะไม่ cache role ใน session และ pending privileged action ต้อง re-check current role
- [ ] denied request ไม่เปลี่ยน target state; audit บันทึก role change, transfer, leave และ safe denial category
- [ ] tests ใช้ current Membership จริงและ runtime database role ไม่ mock ข้าม authorization boundary

