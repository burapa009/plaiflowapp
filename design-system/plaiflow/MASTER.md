# PlaiFlow Design System

## ทิศทาง

Minimal, friendly, slightly cute, professional และ trustworthy โดยใช้พื้นที่ว่าง ลำดับชั้นที่ชัด มุมโค้งอย่างพอดี และรายละเอียดที่อบอุ่น ห้ามเลียนแบบ Paypers หรือผลิตภัณฑ์อื่นแบบ pixel-for-pixel และยังไม่ใช้ mascot

## Tokens

- เก็บ source of truth เป็น semantic OKLCH variables ใน `web/app/globals.css`
- Component เรียกผ่านชื่อ semantic ของ Tailwind/shadcn ห้ามใช้ค่าสีดิบ
- รองรับ light และ dark mode ครบทุก token
- Spacing ใช้จังหวะ 4/8px และใช้ breakpoint มาตรฐานของ Tailwind

## สีอ้างอิง

| บทบาท | Light | On color | Contrast |
|---|---|---|---:|
| Primary | `#0F766E` | `#FFFFFF` | 5.47:1 |
| Accent | `#C2410C` | `#FFFFFF` | 5.18:1 |
| Background / Foreground | `#F8FAFC` | `#0F172A` | 17.06:1 |
| Muted / Muted foreground | `#F1F5F9` | `#475569` | 6.92:1 |

| บทบาท | Dark | On color | Contrast |
|---|---|---|---:|
| Primary | `#5EEAD4` | `#042F2E` | 9.78:1 |
| Accent | `#FDBA74` | `#431407` | 9.28:1 |
| Background / Foreground | `#0F172A` | `#F8FAFC` | 17.06:1 |
| Muted / Muted foreground | `#1E293B` | `#CBD5E1` | 9.85:1 |

สีสถานะต้องมีข้อความหรือ icon ประกอบเสมอ ค่า focus และ control boundary ต้องตรวจ non-text contrast แยกจากคู่สีข้อความ

## Typography

- ใช้ `Noto Sans Thai` แบบ self-host สำหรับทั้งภาษาไทยและ Latin
- ใช้เฉพาะน้ำหนัก 400, 500, 600 และ 700
- Body เริ่มที่ 16px และ line-height อย่างน้อย 1.5
- ข้อความยาวบน desktop จำกัดประมาณ 65–75 ตัวอักษรต่อบรรทัด

## Application Shell

- Desktop ตั้งแต่ `lg`: sidebar ถาวร
- Mobile: top bar และ Sheet menu โดยรักษาลำดับ navigation เดิม
- Phase 0 มีเพียง `ภาพรวม` และ `สถานะระบบ`
- ไม่แสดงเมนูของ feature ที่ยังไม่มี และไม่ใช้ bottom navigation

## Brand และ Theme

- ใช้โลโก้หลักจาก `web/public/plaiflow-logo.png` โดยรักษาสัดส่วน สี และพื้นที่ว่างของไฟล์ต้นฉบับ
- เมื่อใช้บน dark mode ให้วางโลโก้บนพื้นสว่างแทนการเปลี่ยนสี asset
- เลือก light/dark mode จาก `prefers-color-scheme` โดยไม่มี theme toggle ใน Phase 0

## Components และสถานะ

- shadcn primitives อยู่ใน `web/components/ui` และแก้ไขโดยตรงได้
- Product components อยู่ใกล้ feature ที่เป็นเจ้าของ ไม่สร้าง wrapper ล่วงหน้า
- Page loading ใช้ skeleton ที่จองพื้นที่และมี `aria-busy`
- Action loading แสดงใน control และป้องกันการส่งซ้ำ
- Empty state อธิบายสถานการณ์และมี action เดียวเมื่อแก้ไขได้
- Error อยู่ใกล้ต้นเหตุ ระบุวิธีกู้คืน และมี retry เมื่อเหมาะสม
- Toast ใช้แจ้งผล action เท่านั้น ไม่ใช้แทน validation หรือ persistent error

## Motion

- ใช้ CSS `transform` และ `opacity` เท่านั้น โดยทั่วไป 150–200ms
- เคลื่อนไหวเฉพาะ feedback และการเปิดหรือปิด overlay
- รองรับ `prefers-reduced-motion` และไม่ติดตั้ง animation library ใน Phase 0

## Accessibility และ Responsive Review

- ข้อความปกติต้องมี contrast อย่างน้อย 4.5:1
- มี visible focus, keyboard navigation, semantic labels และ skip link
- เป้าหมายสัมผัสหลักอย่างน้อย 44x44px และไม่พึ่ง hover เพียงอย่างเดียว
- ทดสอบที่ 375, 768, 1024 และ 1440px โดยไม่มี horizontal scroll
- ทดสอบ light mode, dark mode และ reduced motion แยกกัน
