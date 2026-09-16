# 08: Token ใช้ไม่ได้แล้วสร้าง Reconnect Task อย่างปลอดภัย

**What to build:** เมื่อ Google credential หมดอายุ ใช้ refresh ไม่ได้ หรือถูก revoke งาน Drive หยุดอย่างปลอดภัย เปลี่ยน connection เป็น Reauthorization Required และสร้าง Task ให้ OWNER กลับมาเชื่อมต่อใหม่โดยไม่เกิด retry loop

**Blocked by:** 06/OWNER เชื่อม Google Drive ด้วย scope ขั้นต่ำ.

**Status:** ready-for-agent

- [ ] Provider authentication failure ที่ยืนยันแล้วเปลี่ยน connection เป็น Reauthorization Required และไม่ retry credential เดิม
- [ ] งาน Drive จบด้วย safe failure category โดยไม่ log token, provider response body, filename หรือ business content
- [ ] สร้าง Reconnect Task ที่ actionable ให้ current OWNER เพียงหนึ่งรายการต่อ invalid connection generation แม้ worker retry หรือเหตุการณ์มาพร้อมกัน
- [ ] Reconnect ต้องผ่าน consent ใหม่และแทนที่ credential เดิม โดยรักษา canonical folder เดิมเมื่อยังเข้าถึงได้
- [ ] การ reconnect สำเร็จปิดหรือ resolve Task เดิมอย่างตรวจสอบได้ และไม่ให้ URL/Task payload กลายเป็น authorization
- [ ] มี runnable check ผ่าน provider double ครอบคลุม expired token, external revocation, deduped Task และ successful reconnect
