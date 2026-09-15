# 07: ผู้ใช้สร้าง Organization เพิ่ม

**What to build:** ผู้ใช้ที่มี session อยู่แล้วสามารถสร้าง Organization เพิ่มและเป็น Owner ได้ โดย tenant ใหม่แยกข้อมูลและไม่ถูกบันทึกเป็น authority ใน session

**Blocked by:** 01 — ผู้ใช้ใหม่เข้าสู่ระบบด้วย LINE และสร้าง Organization แรก

**Status:** ready-for-agent

- [ ] เปิด fresh Codex context ก่อนเริ่มงาน ticket นี้
- [ ] authenticated User สร้าง Organization จากชื่อเพียงช่องเดียว พร้อม UUID ที่ไม่เดาตามลำดับ
- [ ] Organization และ Owner Membership ถูกสร้างใน transaction เดียวและ rollback พร้อมกันเมื่อส่วนใดล้มเหลว
- [ ] session เดิมไม่ถูกแก้ให้มี current Organization หรือ role; tenant context ทุก request มาจาก URL และ current Membership
- [ ] Organization ใหม่เริ่มด้วย Owner เพียงคนเดียวและ database constraint ป้องกัน Owner มากกว่าหนึ่งคน
- [ ] Organization list โหลดทุก Membership-backed Organization ด้วย bounded indexed query และไม่ทำ N+1
- [ ] failure และ duplicate submission ให้ผล idempotent/safe โดยไม่สร้าง Organization ที่ไม่มี Owner
- [ ] audit ระบุ actor, Organization, outcome และ request ID โดยไม่เก็บ form/session secrets
- [ ] tests ครอบคลุม success, transaction rollback, repeated submission และ User หนึ่งคนเป็นสมาชิกหลาย Organization

