# 08: ผู้ใช้ตั้ง NotificationPreference และรับ Digest ได้

**What to build:** สมาชิกกำหนด Immediate, Digest หรือ Off สำหรับ Assignment, TaskChange และ DueReminder แยก Web/LINE ภายในแต่ละ Organization จัดการ Web inbox ได้ และรับ Digest รายวันที่รวมเหตุการณ์โดยไม่ส่งซ้ำหรือส่งข้อความว่าง

**Blocked by:** 03 — Assigned Task ปรากฏใน Web และ LINE; 05 — Watcher ติดตาม Task และถูกถอดเมื่อ Membership สิ้นสุด; 07 — Due Task สร้าง Reminder เพียงครั้งเดียว

**Status:** ready-for-agent

- [ ] เปิด fresh Codex context ก่อนเริ่มงาน ticket นี้
- [ ] NotificationPreference ถูก scope ด้วย Membership, Organization, Notification Category และ channel
- [ ] ค่าเริ่มต้นคือ Web Assignment Immediate, Web TaskChange Digest, Web DueReminder Immediate และ LINE ทุก category Off
- [ ] preference changes ใช้กับ notification ใหม่เท่านั้นและไม่ redeliver เหตุการณ์เดิม
- [ ] Notification opt-out และ preference mutation ไม่ถูก Plan/Entitlement gate แต่ยังต้องผ่าน session, Membership และ CSRF
- [ ] Web Immediate แสดงรายการเดี่ยว; Web Digest แสดงผ่าน daily Digest; Web Off ไม่แสดง
- [ ] Digest ทำงาน 09:00 Organization Timezone, รวมตาม Task, แสดง latest relevant state/change count, ไม่รวม Immediate และไม่ส่งเมื่อว่าง
- [ ] ไม่มี hourly, weekly, custom send time, per-Task reminder หรือ per-event preference UI
- [ ] Web inbox รองรับ unread, mark one, mark all, cursor 50/100 และไม่มี delete-one action
- [ ] Notification และ Delivery Attempt ถูกเก็บ 90 วันและ cleanup ไม่แตะ export/security audit
- [ ] worker claim ไม่เกิน 100 notification jobs ต่อ batch, LINE concurrency ไม่เกิน 10 และ transient retry ตาม 1m/5m/30m/2h/12h ก่อนหยุดครั้งที่ห้า
- [ ] permanent provider errors ไม่ retry; retry ใช้ Delivery Attempt เดิมและไม่สร้าง Notification ซ้ำ
- [ ] removed Membership, unlinked identity และ blocked provider ไม่ได้รับ LINE และไม่มี group/recipient fallback
- [ ] tests ครอบคลุมทุก default/mode/category/channel, prospective change, empty digest, no immediate duplication, batching, retry และ retention

