# Phase 0 Shared Understanding

## เป้าหมาย

สร้าง Repository Foundation, LINE Bootstrap และ Inbound Event Skeleton ที่ deploy และตรวจสอบได้จริง พร้อม Dashboard shell และ PlaiFlow Design System ขั้นต้น โดยยังไม่สร้าง authentication จริง เอกสาร/OCR บัญชี AI advisor ระบบแจ้งเตือน หรือ export service

## Repository และ Stack

- Repository เดียว แบ่ง `web/` และ `api/`
- Web: Next.js App Router, TypeScript, Tailwind, shadcn/ui และ pnpm
- API/worker: Go module เดียวใน `api/` ใช้ standard library และ `pgx/v5`
- API และ worker เป็นคนละ executable ที่ `cmd/server` และ `cmd/worker`
- PostgreSQL เป็น durable store และ queue; ยังไม่มี message broker
- SQL migrations อยู่ใน `api/migrations` และใช้ `golang-migrate` รุ่นที่ pin ไว้

## LINE Webhook Boundary

- Public route คือ `POST /webhooks/line`
- จำกัด raw body 1 MiB และตรวจ `x-line-signature` ด้วย HMAC-SHA256 ก่อน parse
- Signature หายหรือผิดต้องไม่ประมวลผล ส่วน JSON หรือ event ID ที่ไม่ถูกต้องตอบ `400`
- Delivery ที่ `events: []` ถูกต้องและตอบ `200` โดยไม่สร้างงาน
- บันทึกทุก event ใน delivery ด้วย transaction เดียว และตอบ `200` หลัง durable commit เท่านั้น
- ใช้ `(provider, channel, webhook_event_id)` เป็น database unique constraint
- รองรับ LINE channel เดียวผ่าน environment แต่ schema ไม่ผูกกับจำนวน channel
- ไม่เก็บ raw delivery หรือ signature และไม่ log payload/message text

## Inbound Event Lifecycle

- สถานะหลักคือ Received, Processing, Processed, Retryable, Failed และ Ignored
- Worker claim สูงสุด 25 รายการด้วย `FOR UPDATE SKIP LOCKED` และ lease 60 วินาที
- เริ่ม concurrency 1 และไม่รับประกัน global ordering
- Retry หลัง 1 นาที, 5 นาที, 30 นาที, 2 ชั่วโมง และ 12 ชั่วโมง ก่อนเป็น Failed
- LINE `message` ใช้ record-only handler เพื่อพิสูจน์ end-to-end pipeline โดยไม่มี outbound action
- Event type อื่นถูกบันทึกแล้วเป็น Ignored โดยไม่ retry
- Payload อยู่ 30 วัน; metadata/idempotency อยู่ 90 วัน; worker ล้างวันละครั้ง
- Worker update heartbeat ทุก 30 วินาที และถือว่าผิดปกติเมื่อเก่ากว่า 2 นาที

## Future Seams โดยไม่มี Implementation

- User ภายในหนึ่งรายเชื่อม External Identity ได้หลาย provider
- Go API สงวน callback convention `/v1/auth/{provider}/callback`; web แสดงผลที่ `/auth/callback`
- Domain Event ในอนาคตมี `id`, `type`, `occurred_at`, `subject_id`, `source_event_id` และ `data`
- ยังไม่มี auth/session/token storage, provider implementation, event bus, publisher, Notifier หรือ export package

## API และ Operations

- Application API ใช้ prefix `/v1`; operational routes คือ `/healthz` และ `/readyz`
- `/healthz` ตรวจ process; `/readyz` ตรวจ database และ migration version โดยไม่เรียก LINE
- Browser เรียก Next.js same-origin; Next.js server เรียก Railway ด้วย `DASHBOARD_API_TOKEN`
- Error response มี stable code, safe message และ request ID โดยไม่เปิดเผย internal error
- Public requests ได้ request ID ใหม่; trusted server-to-server request ส่งต่อ ID เดิมได้
- JSON logs มี method, route, status, duration และ request ID พร้อม redaction
- ค่าเริ่มต้น: read-header 5s, read 5s, write 10s, idle 60s, webhook deadline 3s
- Database pool เริ่มที่ API 10 connections และ worker 5 connections โดยปรับผ่าน environment ได้

## Dashboard และ Design System

- รูปลักษณ์ minimal, friendly, slightly cute, professional และ trustworthy โดยไม่เลียนแบบผลิตภัณฑ์อื่น
- ใช้ semantic OKLCH tokens, teal เป็น primary, warm orange เป็น accent และผ่าน WCAG 2.2 AA
- ใช้ `Noto Sans Thai` แบบ self-host สำหรับไทยและ Latin
- Desktop ใช้ sidebar ถาวร; mobile ใช้ top bar กับ Sheet menu
- มีเพียง `ภาพรวม` และ `สถานะระบบ`; ไม่แสดง feature อนาคต
- Dashboard ใช้ข้อมูลจริง ไม่มี mock data และมี loading/empty/error/retry states
- Theme ตาม system preference และ motion ใช้ CSS transform/opacity พร้อม reduced-motion

## Security และข้อมูล

- Secrets อยู่ใน environment-scoped secret stores เท่านั้นและไม่มีค่าใน repository
- ไม่มี secret ใดใช้ `NEXT_PUBLIC_`
- Staging dashboard ใช้ Vercel Deployment Protection
- LINE staging ใช้เฉพาะบัญชีทดสอบและข้อมูล synthetic
- Gitleaks ที่ pin ด้วย commit SHA ตรวจ secrets ใน CI
- Webhook ไม่จำกัดตาม source IP; ใช้ signature, body limit, timeout และ idempotency

## CI, Testing และ Deployment

- Pull request รัน format, lint/typecheck, Go/frontend tests, migration validation, secret scan และ production builds
- PostgreSQL integration tests รับ `TEST_DATABASE_URL`; local ใช้ Docker Compose และ CI ใช้ service container
- Vercel สร้าง web preview สำหรับ pull request
- Merge เข้า `main` deploy staging ตามลำดับ migration, API, worker และ smoke test
- ก่อนสร้าง cloud resources, เพิ่ม secrets, เปิด LINE webhook/redelivery หรือยิง staging webhook ต้องขออนุมัติอีกครั้ง

## Baseline Performance

- Webhook p95 ต่ำกว่า 500ms
- API read p95 ต่ำกว่า 300ms ใน staging
- Query เกิน 200ms ต้อง log
- Queue age เกิน 60 วินาทีถือว่าผิดปกติ
- Frontend ใช้ production build output, Lighthouse และ Vercel metrics โดยยังไม่มี analytics SDK

## Definition of Done

Phase 0 จบเมื่อ local checks และ CI ผ่าน, staging deploy สำเร็จ, LINE Verify ได้ `200`, signature ผิดและ malformed requests ถูกปฏิเสธอย่างปลอดภัย, duplicate event ไม่สร้างงานซ้ำ, synthetic `message` เปลี่ยนจาก Received เป็น Processed, dashboard แสดงข้อมูลจริงและทุก UI state, logs มี request ID/timing โดยไม่รั่วข้อมูล และ rollback migration ถูกทดสอบแล้ว
