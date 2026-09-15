# ใช้ repository เดียวโดยแยก web และ API

PlaiFlow จะเก็บ Next.js dashboard ไว้ใน `web/` และ Go service ไว้ใน `api/` ภายใน repository เดียวกัน โค้ด Go ใช้ module เดียวที่มีรากอยู่ใน `api/` เพื่อให้การประสานงานและ CI ของ Phase 0 เรียบง่าย โดยไม่ผูกสอง runtime เข้าด้วยกันผ่าน source-code package ร่วม
