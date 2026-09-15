# แยก staging web และ backend ระหว่าง Vercel กับ Railway

PlaiFlow จะ deploy Next.js web บน Vercel project สำหรับ staging โดยเฉพาะ และ deploy Go API, worker และ PostgreSQL บน Railway staging environment การแบ่งนี้ยอมรับต้นทุนการดูแลผู้ให้บริการสองแห่งเพื่อใช้ deployment path ที่ตรงกับแต่ละ runtime และแยก secrets จาก production โดยไม่สร้าง production environment ใน Phase 0 GitHub Actions จะควบคุมลำดับ `migration → API → worker → smoke test` แทน Railway auto-deploy
