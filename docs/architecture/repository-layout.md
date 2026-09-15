# Repository Layout

```text
api/
├─ cmd/server/
├─ cmd/worker/
├─ internal/config/
├─ internal/httpapi/
├─ internal/line/
├─ internal/inbound/
├─ internal/postgres/
└─ migrations/
web/
├─ app/
└─ components/ui/
```

- ไม่มี `pkg/`, common package, repository abstraction ทั่วไป หรือ DI container
- shadcn primitives อยู่ใน `web/components/ui` ส่วน product components อยู่ใกล้ feature ที่เป็นเจ้าของ
- Browser ติดต่อ Next.js แบบ same-origin; Next.js ฝั่ง server ติดต่อ Go API ด้วย service token
