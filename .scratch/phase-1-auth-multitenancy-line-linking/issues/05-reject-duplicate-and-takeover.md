# 05: ระบบปฏิเสธ duplicate identity และ account-takeover attempt

**What to build:** ระบบ fail closed เมื่อ login/link พยายามยึด identity ของ User อื่นหรืออาศัย email, callback mix-up, replay หรือ session อื่นเพื่อรวม account

**Blocked by:** 04 — ผู้ใช้ LINE เดิมเชื่อม Google เป็น login provider ที่สอง

**Status:** ready-for-agent

- [ ] เปิด fresh Codex context ก่อนเริ่มงาน ticket นี้
- [ ] LINE และ Google identities ที่รายงาน email เดียวกันไม่ถูก auto-link หรือ merge และสร้าง User แยกกันเมื่อเป็น first login
- [ ] linking `(issuer, subject)` ที่เป็นของ User อื่นยุติด้วย identity-conflict state โดยไม่ reassign identity, Membership หรือ Organization ใด
- [ ] callback ปฏิเสธ state ที่ missing, malformed, expired, consumed, provider-mismatched หรือผูกกับ session/User อื่น และปฏิเสธ LINE/Google mix-up
- [ ] callback replay, reused authorization code และ concurrent claim มีผู้ชนะได้ครั้งเดียวโดยไม่สร้าง duplicate User/AuthIdentity
- [ ] invalid/duplicate responses ไม่เปิดเผยว่า email, provider identity หรือ Organization มีอยู่หรือไม่ และแสดง recovery ที่ไม่เสนอ automatic merge
- [ ] การปฏิเสธไม่เปลี่ยน session, Membership หรือ target identity และ audit เก็บเพียง safe category กับ request ID
- [ ] security test ครอบคลุม wrong nonce/issuer/audience/authorized party, token expiry/signature failure, provider timeout และ open-redirect inputs
- [ ] log, rendered response และ frontend assets ไม่มี authorization code, state, nonce, verifier, provider token หรือ raw claims จากกรณีโจมตี

