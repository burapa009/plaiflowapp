# ADR-0032: ใช้ Durable Job กลางและให้ Worker ผ่าน Internal API

## Status

Accepted for Phase 5 planning.

## Context

PlaiFlow มี `export_jobs` และ worker ที่เข้าถึง PostgreSQL โดยตรงอยู่แล้ว ขณะที่ Phase 5 ต้องรองรับงานเบื้องหลังที่ retry ได้ ตรวจสอบสถานะได้ และปลอดภัยเมื่อ worker ถูก compromise. OCR และ export ต้องใช้ claim, lease, heartbeat, retry และ stale recovery แบบเดียวกัน

## Decision

- ใช้ Durable Job model กลางเดียว โดยแยกชนิดงานด้วย `kind` (`export` และ `ocr`)
- เปิดใช้ export ก่อน และเตรียม seam สำหรับ OCR โดยไม่ผูกกับ provider ใด
- Worker เรียก internal API สำหรับ claim, heartbeat, complete และ fail; worker ไม่มี PostgreSQL credential
- Claim ใช้ bounded batch, lease มีวันหมดอายุ และ stale job ถูก reclaim ได้
- Export ขนาดใหญ่สร้างเป็น chunk ไป object storage และคืน signed URL อายุสั้นที่ตรวจ tenant/ownership ฝั่ง server
- Worker ใช้ service token อายุสั้นที่จำกัด scope และส่ง `job_id`, `attempt_id` และ `lease_token` ใน lifecycle calls
- Completion ซ้ำเป็น no-op เมื่อเป็น attempt เดิมที่สำเร็จแล้ว และ completion จาก lease เก่าถูกปฏิเสธ
- Retry ใช้ allowlist ของ error code ฝั่ง API ไม่เชื่อค่า retryable จาก worker โดยตรง
- Artifact เก็บ 24 ชั่วโมง, signed URL 15 นาที, job metadata 90 วัน และ audit 1 ปี
- Canonical lifecycle คือ `Queued → Running → Completed | Failed`; `Cancelled` ใช้ได้เฉพาะก่อนเริ่ม `Running`
- เริ่ม migration ด้วย generic `jobs`, ย้ายงานเดิมที่ยัง active และคง compatibility read path ของ `export_jobs` จนผ่าน retention window
- Large export อ่านครั้งละ 500 rows และเขียนแบบ streaming โดยจำกัด concurrency ต่อ Organization

## Consequences

- ต้องมี internal worker protocol และ authentication เพิ่ม
- ต้อง migrate จาก flow ของ `export_jobs` เดิมอย่างระมัดระวัง
- API เป็นจุดควบคุม authorization และ audit ของ worker
- มี overhead จาก network hop แต่ลดสิทธิ์และ blast radius ของ worker

## Rejected alternatives

- แยกตารางและกลไก claim สำหรับ OCR กับ export เพราะทำให้ retry/stale recovery/metrics ซ้ำกัน
- ให้ worker ต่อ PostgreSQL โดยตรง เพราะขัด least privilege และทำให้ข้อมูลทั้งฐานอยู่ใน blast radius เดียว
