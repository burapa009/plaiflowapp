# Plaiflow Design System

> **Product:** Plaiflow<br>
> **Document:** Design System Specification<br>
> **Version:** 1.0<br>
> **Status:** Project-ready<br>
> **Primary use:** Web application / SaaS dashboard<br>
> **Design direction:** Minimal + Friendly + Professional + AI-first

---

## 1. Purpose

เอกสารนี้เป็นมาตรฐานกลางสำหรับการออกแบบและพัฒนา UI ของ **Plaiflow** เพื่อให้ทุกหน้าในระบบมีรูปแบบเดียวกัน ใช้งานง่าย ขยายระบบต่อได้ง่าย และรองรับการทำงานร่วมกันระหว่าง Designer, Frontend Developer และ AI Coding Agent

Design System นี้ใช้กับหน้าหลักของระบบ เช่น:

- Dashboard
- สร้างรายจ่าย
- รายการรายจ่าย
- อัปโหลดเอกสาร
- Document Review
- OCR / AI Extraction
- Accounting Workflow
- Reports
- Settings
- Billing / Subscription
- Admin
- Integrations

---

# 2. Brand Principles

Plaiflow ต้องสื่อภาพลักษณ์ 4 อย่างพร้อมกัน:

1. **Simple** — ใช้งานง่าย ไม่ซับซ้อน
2. **Smart** — มี AI ช่วยทำงาน
3. **Trustworthy** — เหมาะกับงานธุรกิจและเอกสารทางบัญชี
4. **Friendly** — ไม่แข็งหรือดูเป็น Enterprise แบบเกินไป

### Brand Keywords

- Clean
- Calm
- Smart
- Friendly
- Reliable
- Modern
- Productive
- AI-powered

---

# 3. Visual Direction

## 3.1 Default Theme

ธีมหลักของ Plaiflow ใช้:

- พื้นหลังสีขาว / ฟ้าอ่อน
- Primary color เป็น Blue
- Accent ใช้ Cyan / Teal
- Gradient ใช้เฉพาะจุดสำคัญ
- Card มุมโค้ง
- Border บาง
- Shadow เบา
- UI โปร่ง
- Icon แบบ outline / rounded
- AI elements มี glow ได้เล็กน้อย

## 3.2 Avoid

หลีกเลี่ยง:

- สีฉูดฉาดหลายสีในหน้าจอเดียว
- Gradient ทุก component
- Shadow หนัก
- Border หนามาก
- Glassmorphism จนอ่านข้อความยาก
- Card ซ้อนหลายชั้นเกินไป
- Animation มากเกินไป
- Mascot ที่แย่งความสนใจจากข้อมูลหลัก

---

# 4. Design Tokens

## 4.1 Colors

### Primary

```css
--pf-primary-50: #EEF5FF;
--pf-primary-100: #D9E9FF;
--pf-primary-200: #B8D6FF;
--pf-primary-300: #8CB9FF;
--pf-primary-400: #5C97FF;
--pf-primary-500: #2F73FF;
--pf-primary-600: #1F5CE6;
--pf-primary-700: #1848B8;
--pf-primary-800: #143A91;
--pf-primary-900: #112F73;
```

### Accent

```css
--pf-cyan-500: #22C7F2;
--pf-teal-500: #1FD6C2;
--pf-purple-500: #7A6BFF;
```

### Neutral

```css
--pf-white: #FFFFFF;

--pf-gray-25: #FCFDFE;
--pf-gray-50: #F8FAFC;
--pf-gray-100: #F2F5F9;
--pf-gray-200: #E7ECF2;
--pf-gray-300: #D7DFE9;
--pf-gray-400: #A9B4C4;
--pf-gray-500: #7B8798;
--pf-gray-600: #5E6B7C;
--pf-gray-700: #3D4756;
--pf-gray-800: #242D3B;
--pf-gray-900: #121926;
```

### Semantic

```css
--pf-success-50: #EAFBF4;
--pf-success-500: #18A66A;
--pf-success-700: #0F7A4E;

--pf-warning-50: #FFF7E8;
--pf-warning-500: #D88A00;
--pf-warning-700: #9B6500;

--pf-danger-50: #FDEEEE;
--pf-danger-500: #D64545;
--pf-danger-700: #A92F2F;

--pf-info-50: #EEF6FF;
--pf-info-500: #2F73FF;
```

---

## 4.2 Background Tokens

```css
--pf-bg-app: #F7FAFD;
--pf-bg-page: #F8FBFF;
--pf-bg-soft: #F2F7FC;
--pf-bg-muted: #EEF3F8;
--pf-surface: #FFFFFF;
--pf-surface-hover: #F8FBFF;
```

---

## 4.3 Text Tokens

```css
--pf-text-primary: #14213D;
--pf-text-secondary: #5E6B85;
--pf-text-muted: #8E9AB0;
--pf-text-disabled: #B6BFCC;
--pf-text-inverse: #FFFFFF;
```

---

## 4.4 Border Tokens

```css
--pf-border-default: #D9E4F0;
--pf-border-soft: #E7EDF5;
--pf-border-strong: #C6D1E0;
--pf-border-focus: #2F73FF;
--pf-border-danger: #D64545;
```

---

# 5. Gradient System

ใช้ Gradient เฉพาะ:

- CTA หลัก
- AI badge
- AI helper
- onboarding
- background decoration
- selected premium features

### Brand Gradient

```css
background: linear-gradient(
  135deg,
  #2F73FF 0%,
  #22C7F2 55%,
  #1FD6C2 100%
);
```

### AI Gradient

```css
background: linear-gradient(
  135deg,
  #5C7CFF 0%,
  #7A6BFF 45%,
  #22C7F2 100%
);
```

### Soft AI Surface

```css
background:
  linear-gradient(
    135deg,
    rgba(47,115,255,0.10),
    rgba(122,107,255,0.08),
    rgba(31,214,194,0.08)
  );
```

---

# 6. Typography

## 6.1 Recommended Fonts

ลำดับแนะนำ:

1. `LINE Seed Sans TH`
2. `IBM Plex Sans Thai`
3. `Noto Sans Thai`
4. system sans-serif

### CSS

```css
font-family:
  "LINE Seed Sans TH",
  "IBM Plex Sans Thai",
  "Noto Sans Thai",
  system-ui,
  sans-serif;
```

---

## 6.2 Font Scale

| Token | Size | Weight | Usage |
|---|---:|---:|---|
| Display | 40px | 700 | Hero / Marketing |
| H1 | 32px | 700 | Page Title |
| H2 | 24px | 700 | Section Heading |
| H3 | 20px | 600 | Card Heading |
| Body LG | 18px | 400/500 | Important copy |
| Body | 16px | 400 | Default |
| Body SM | 14px | 400 | Secondary info |
| Caption | 12px | 400/500 | Hint / metadata |

### Line Height

```css
--pf-leading-tight: 1.25;
--pf-leading-normal: 1.5;
--pf-leading-relaxed: 1.65;
```

---

# 7. Spacing System

ใช้ระบบ 4px base scale

```txt
4
8
12
16
20
24
32
40
48
64
80
96
```

### Recommended Usage

| Area | Spacing |
|---|---:|
| Icon + Text | 8px |
| Input stack | 12px |
| Card inner gap | 16px |
| Form sections | 20–24px |
| Card padding | 20–24px |
| Page section gap | 24–32px |
| Desktop page padding | 24–32px |

---

# 8. Border Radius

```css
--pf-radius-xs: 6px;
--pf-radius-sm: 10px;
--pf-radius-md: 14px;
--pf-radius-lg: 18px;
--pf-radius-xl: 24px;
--pf-radius-pill: 999px;
```

### Usage

- Button: 10–12px
- Input: 10–12px
- Dropdown: 12px
- Card: 16–20px
- Modal: 20–24px
- Badge: pill

---

# 9. Shadow System

### Soft

```css
box-shadow:
  0 4px 16px rgba(20, 33, 61, 0.06);
```

### Card

```css
box-shadow:
  0 8px 28px rgba(20, 33, 61, 0.08);
```

### Floating

```css
box-shadow:
  0 16px 40px rgba(20, 33, 61, 0.12);
```

### AI Glow

```css
box-shadow:
  0 0 0 1px rgba(47,115,255,0.12),
  0 10px 30px rgba(47,115,255,0.16);
```

---

# 10. Layout System

## 10.1 Main App Container

```css
max-width: 1680px;
margin: 0 auto;
padding-inline: 24px;
```

จอ 1440px ขึ้นไปใช้พื้นที่ได้กว้างขึ้น แต่ไม่ควรทำให้ content หลัก stretch มากเกินไป

---

## 10.2 SaaS Shell

โครงสร้างมาตรฐาน:

```txt
AppShell
 ├─ Sidebar / TopNav
 ├─ Header
 └─ MainContent
```

รองรับทั้ง:

- top navigation
- left sidebar
- hybrid navigation

---

# 11. Grid

ใช้ 12-column grid

### Desktop

```txt
12 columns
gap: 24px
```

### Create Expense

```txt
Left document workspace: 7 columns
Right form panel: 5 columns
```

หรือประมาณ:

```txt
55% / 45%
```

---

# 12. Breakpoints

```css
--pf-bp-sm: 640px;
--pf-bp-md: 768px;
--pf-bp-lg: 1024px;
--pf-bp-xl: 1280px;
--pf-bp-2xl: 1536px;
```

### Layout Behavior

#### ≥1280
- 2-column layout
- full document preview

#### 1024–1279
- 2-column compact
- ลด padding / gap

#### 768–1023
- stacked หรือ split view แบบ adaptive

#### <768
แนะนำเป็น workflow:

```txt
Upload
↓
Preview
↓
Review
↓
Confirm
```

---

# 13. Icon System

แนะนำ:

- Lucide
- Phosphor Icons

กฎ:

- ใช้ icon family เดียวทั้งระบบ
- 16px สำหรับ inline
- 18–20px สำหรับ input/button
- 20–24px สำหรับ navigation
- stroke width ประมาณ 1.75–2

---

# 14. Core Components

# 14.1 Button

## Primary

ใช้สำหรับ action หลัก เช่น:

- สร้างรายจ่าย
- บันทึก
- ยืนยัน
- อัปโหลด

```txt
Background: Primary 500
Text: White
Height: 44–48px
Padding: 16–20px
Radius: 10–12px
```

### Hover

```txt
Primary 600
```

### Active

```txt
Primary 700
```

---

## Secondary

```txt
Background: White
Border: Default
Text: Primary/Dark
```

---

## Ghost

ใช้กับ:

- toolbar
- small actions
- navigation

---

## Danger

เฉพาะ action ที่ destructive

ตัวอย่าง:

- ลบเอกสาร
- ยกเลิก subscription
- reset data

---

# 14.2 Input

### Default

```txt
Height: 44–48px
Radius: 10–12px
Border: #D9E4F0
Background: White
```

### Focus

```css
border-color: #2F73FF;
box-shadow: 0 0 0 3px rgba(47,115,255,0.12);
```

### Error

```txt
Border: Danger 500
Helper text: Danger 500
```

### Disabled

```txt
Background: Gray 100
Text: Gray 400
```

---

# 14.3 Select / Combobox

ต้องมี:

- clear active state
- keyboard navigation
- search สำหรับ list ยาว
- selected icon หรือ checkmark
- max-height + scroll

---

# 14.4 Checkbox / Radio

ขนาดแนะนำ:

```txt
18–20px
```

Selected:

```txt
Primary 500
```

ต้องมีทั้ง:

- color
- icon/check
- label

เพื่อไม่พึ่งสีอย่างเดียว

---

# 14.5 Card

### Default Card

```txt
Background: White
Border: 1px solid Border Soft
Radius: 18px
Padding: 20–24px
Shadow: Soft
```

### Interactive Card

Hover:

- border darken เล็กน้อย
- shadow เพิ่มเล็กน้อย
- translateY ไม่เกิน 1–2px

---

# 14.6 Badge

ใช้กับ:

- AI
- VAT
- สถานะ
- plan
- sync status

ตัวอย่าง:

```txt
AI
Auto
Draft
Paid
Pending
Failed
```

---

# 14.7 Alert / Banner

ประเภท:

- info
- success
- warning
- error
- AI

ต้องประกอบด้วย:

```txt
Icon
Title/Message
Optional CTA
Dismiss
```

---

# 14.8 Tabs

Active tab:

```txt
Text: Primary
Border-bottom: 2px Primary
Background: optional Primary 50
```

สำหรับ high-level tabs ใช้ icon + label ได้

---

# 14.9 Modal

```txt
Radius: 20–24px
Max width: 520–720px
Padding: 24px
```

Modal ต้องมี:

- title
- optional description
- close
- content
- footer actions

---

# 14.10 Tooltip

ใช้กับ:

- unfamiliar icons
- AI explanation
- finance/accounting terminology

ไม่ใช้แทน label ที่สำคัญ

---

# 15. Navigation

## Top Navigation

Height:

```txt
64–72px
```

องค์ประกอบ:

```txt
Logo
Primary navigation
Search
Notifications
User / Organization switcher
```

### Active Item

ใช้ pill:

```txt
background: Primary 50
color: Primary 600
```

---

# 16. Forms

## Form Structure

```txt
Section Title
↓
Description (optional)
↓
Fields
↓
Validation
↓
Action
```

อย่าวาง field มากเกินไปในหนึ่งแถว

### Desktop

แนะนำไม่เกิน:

```txt
2 fields / row
```

ยกเว้น field สั้นมาก เช่น:

```txt
Quantity
Unit
Tax
```

---

# 17. Data Tables

Plaiflow มีข้อมูลบัญชีจำนวนมาก ตารางจึงเป็น component สำคัญ

Table ต้องรองรับ:

- sticky header
- sorting
- filtering
- pagination
- row selection
- bulk action
- empty state
- loading state

### Row Height

```txt
48–56px
```

### Numeric Alignment

ตัวเลข:

```txt
right aligned
```

ชื่อ / รายละเอียด:

```txt
left aligned
```

---

# 18. Status Colors

| Status | Color |
|---|---|
| Success / Paid | Green |
| Pending | Amber |
| Draft | Gray |
| Processing | Blue |
| AI Processing | Purple/Blue |
| Failed | Red |
| Synced | Teal |

ห้ามใช้สีอย่างเดียวในการสื่อสถานะ

ตัวอย่าง:

```txt
✓ ชำระแล้ว
◷ รอดำเนินการ
! ไม่สำเร็จ
```

---

# 19. AI Components

AI เป็น feature สำคัญของ Plaiflow แต่ UI ต้องไม่ทำให้รู้สึก gimmicky

## AI Badge

```txt
Plaiflow AI
AI อ่านแล้ว
AI Suggestion
Auto-filled
```

ใช้ gradient ได้

---

## AI Confidence

ถ้าจำเป็นต้องแสดง confidence:

```txt
มั่นใจสูง
ควรตรวจสอบ
ต้องตรวจสอบ
```

หลีกเลี่ยงการแสดง score ทางเทคนิคให้ผู้ใช้ทั่วไปโดยไม่จำเป็น

---

## AI Generated Fields

field ที่ AI กรอกให้:

- icon sparkle
- soft blue background
- tooltip:
  `Plaiflow AI อ่านข้อมูลนี้จากเอกสาร`

ผู้ใช้ต้องสามารถแก้ไขได้เสมอ

---

# 20. Document Viewer

Component สำคัญของ Plaiflow

ต้องรองรับ:

- PDF
- image
- multi-page
- zoom
- rotate
- fit
- fullscreen
- page navigation
- thumbnails

### Viewer Background

```txt
Gray 50 / Blue 50
```

ไม่ควรใช้ background ที่ลด readability ของเอกสาร

---

# 21. Upload

Upload component ต้องรองรับ:

- drag & drop
- click upload
- upload progress
- retry
- cancel
- error
- multiple files

### Dropzone

```txt
Dashed border
Large icon
Short instruction
Supported file types
```

---

# 22. File State

แต่ละไฟล์มี state:

```txt
Queued
Uploading
Processing
AI Reading
Needs Review
Ready
Error
```

---

# 23. Empty States

ควรประกอบด้วย:

```txt
Illustration / Mascot
Short title
Short description
CTA
```

ตัวอย่าง:

```txt
ยังไม่มีเอกสาร

อัปโหลดใบเสร็จหรือใบกำกับภาษี
Plaiflow จะช่วยอ่านข้อมูลให้คุณอัตโนมัติ

[อัปโหลดเอกสาร]
```

---

# 24. Mascot

Plaiflow mascot ใช้ได้ใน:

- empty states
- onboarding
- AI assistant
- error แบบไม่ critical
- document reading state

ไม่ควรใช้:

- ทุก card
- financial summary หลัก
- audit views
- dense data table

---

# 25. Loading States

ต้องใช้ Skeleton สำหรับ:

- tables
- cards
- preview
- dashboard

ใช้ spinner สำหรับ:

- button action
- short loading state

AI Processing อาจใช้:

```txt
sparkle animation
progress text
```

---

# 26. Motion

Animation duration:

```css
--pf-motion-fast: 120ms;
--pf-motion-normal: 180ms;
--pf-motion-slow: 280ms;
```

### Easing

```css
cubic-bezier(0.2, 0, 0, 1)
```

Animation ควรใช้กับ:

- hover
- modal
- dropdown
- tabs
- accordion
- toast
- sidebar

หลีกเลี่ยง animation ที่รบกวนงานบัญชี

---

# 27. Toast

ตำแหน่ง:

```txt
top-right
```

ประเภท:

```txt
Success
Info
Warning
Error
```

ตัวอย่าง:

```txt
บันทึกรายจ่ายเรียบร้อยแล้ว
```

---

# 28. Notifications

แยกจาก Toast

ใช้กับ:

- OCR เสร็จแล้ว
- import เสร็จ
- sync failed
- Google Drive connection
- export complete
- subscription issue

---

# 29. Search

Global Search สามารถค้นหา:

- เอกสาร
- คู่ค้า
- รายจ่าย
- รายรับ
- invoice
- transaction

Input placeholder:

```txt
ค้นหาเอกสาร รายการ หรือคู่ค้า...
```

---

# 30. Create Expense Page Specification

## Desktop Structure

```txt
CreateExpensePage

├─ Header
│  ├─ BackButton
│  ├─ PageTitle
│  └─ AIHelperBanner
│
├─ DocumentControlBar
│
├─ MainGrid
│
│  ├─ DocumentWorkspace
│  │  ├─ Viewer
│  │  ├─ ViewerToolbar
│  │  └─ ThumbnailTray
│  │
│  └─ ExpensePanel
│     ├─ Tabs
│     ├─ BusinessCard
│     ├─ InputMethodCard
│     ├─ CreditAlert
│     ├─ ExpenseDetails
│     └─ ExpenseItems
│
└─ StickyActionBar
   ├─ Cancel
   └─ CreateExpense
```

---

# 31. Create Expense UX Rules

ผู้ใช้ต้องเข้าใจ 3 อย่างทันที:

```txt
1. เอกสารอะไรที่กำลังเปิด
2. AI อ่านอะไรมาแล้ว
3. ต้องตรวจหรือกดอะไรต่อ
```

### Priority

#### Highest
- document
- extracted fields
- errors
- CTA

#### Secondary
- thumbnails
- helper text
- optional settings

---

# 32. Sticky Actions

บน desktop:

```txt
sticky bottom
```

หรืออยู่ footer ของ right panel

Primary CTA:

```txt
สร้างรายจ่าย
```

Secondary:

```txt
ยกเลิก
```

---

# 33. Accounting UI Rules

ตัวเลขการเงิน:

- ใช้ tabular numbers ถ้ามี
- align right
- format comma
- decimal 2 ตำแหน่งเมื่อจำเป็น

ตัวอย่าง:

```txt
12,500.00 บาท
```

Tax:

```txt
VAT 7%
ภาษีซื้อ
ภาษีขาย
ภาษีหัก ณ ที่จ่าย
```

ต้องใช้ wording สม่ำเสมอทั้งระบบ

---

# 34. Responsive Create Expense

## Desktop

```txt
Viewer | Form
```

## Tablet

```txt
Viewer
Form
```

หรือ split view แบบ collapse ได้

## Mobile

แนะนำ stepper:

```txt
1. เอกสาร
2. ตรวจข้อมูล
3. รายการ
4. ยืนยัน
```

---

# 35. Dark Mode

Dark mode เป็น optional theme

Base:

```css
--pf-dark-bg: #071422;
--pf-dark-surface: #0C1B2C;
--pf-dark-surface-2: #10243A;
--pf-dark-border: #233A52;
--pf-dark-text: #EEF6FF;
--pf-dark-muted: #91A3B8;
```

Primary:

```txt
Blue 400 / Cyan
```

หลีกเลี่ยง:

```txt
pure black #000
```

---

# 36. Accessibility

ขั้นต่ำต้องรองรับ:

- WCAG AA contrast
- keyboard navigation
- focus indicator
- screen reader labels
- semantic HTML
- aria labels สำหรับ icon buttons
- form labels
- error message
- disabled state
- reduced motion

### Minimum Target

```txt
44 × 44px
```

สำหรับ touch target สำคัญ

---

# 37. Content Style

ภาษา Plaiflow:

- สั้น
- ชัดเจน
- เป็นธรรมชาติ
- ไม่ใช้ศัพท์เทคนิคเกินจำเป็น

### Good

```txt
อัปโหลดเอกสาร
```

### Avoid

```txt
ดำเนินการนำเข้าไฟล์เอกสารเพื่อเข้าสู่กระบวนการประมวลผล
```

---

# 38. Confirmation Dialog

ใช้ confirmation เฉพาะ action ที่:

- destructive
- irreversible
- มีผลต่อหลายรายการ

ไม่ถาม confirmation ทุกครั้งที่กด Save

---

# 39. Error Messages

Error ต้องตอบ 2 อย่าง:

```txt
เกิดอะไรขึ้น
ต้องทำอะไรต่อ
```

ตัวอย่าง:

```txt
อัปโหลดไฟล์ไม่สำเร็จ
กรุณาลองใหม่อีกครั้ง หรือตรวจสอบประเภทไฟล์
```

---

# 40. Component Naming Convention

React components ใช้ PascalCase

ตัวอย่าง:

```tsx
<AppShell />
<TopNavigation />
<PageHeader />
<DocumentViewer />
<DocumentToolbar />
<DocumentThumbnailList />
<ExpenseForm />
<ExpenseSummary />
<AiSuggestion />
<StatusBadge />
<StickyActionBar />
```

---

# 41. File Structure Recommendation

```txt
src/
├─ components/
│  ├─ ui/
│  │  ├─ button.tsx
│  │  ├─ input.tsx
│  │  ├─ select.tsx
│  │  ├─ checkbox.tsx
│  │  ├─ dialog.tsx
│  │  └─ tooltip.tsx
│  │
│  ├─ plaiflow/
│  │  ├─ ai-badge.tsx
│  │  ├─ ai-suggestion.tsx
│  │  ├─ document-viewer.tsx
│  │  ├─ document-thumbnail.tsx
│  │  └─ status-badge.tsx
│
├─ features/
│  ├─ expenses/
│  ├─ documents/
│  ├─ accounting/
│  └─ ai/
│
└─ styles/
   ├─ tokens.css
   └─ globals.css
```

---

# 42. Tailwind Token Example

```ts
export const plaiflowTheme = {
  colors: {
    primary: {
      50: "#EEF5FF",
      100: "#D9E9FF",
      200: "#B8D6FF",
      300: "#8CB9FF",
      400: "#5C97FF",
      500: "#2F73FF",
      600: "#1F5CE6",
      700: "#1848B8",
      800: "#143A91",
      900: "#112F73",
    },

    cyan: "#22C7F2",
    teal: "#1FD6C2",
    purple: "#7A6BFF",
  },

  borderRadius: {
    sm: "10px",
    md: "14px",
    lg: "18px",
    xl: "24px",
  },
};
```

---

# 43. CSS Variables Starter

```css
:root {
  --pf-primary: #2F73FF;
  --pf-primary-hover: #1F5CE6;

  --pf-cyan: #22C7F2;
  --pf-teal: #1FD6C2;
  --pf-purple: #7A6BFF;

  --pf-background: #F7FAFD;
  --pf-surface: #FFFFFF;

  --pf-text: #14213D;
  --pf-text-secondary: #5E6B85;
  --pf-text-muted: #8E9AB0;

  --pf-border: #D9E4F0;

  --pf-success: #18A66A;
  --pf-warning: #D88A00;
  --pf-danger: #D64545;

  --pf-radius-sm: 10px;
  --pf-radius-md: 14px;
  --pf-radius-lg: 18px;
  --pf-radius-xl: 24px;
}
```

---

# 44. AI Coding Agent Rules

เมื่อ AI เช่น Codex / Claude Code / Cursor แก้ UI ของ Plaiflow ให้ยึดกฎต่อไปนี้

## MUST

- ใช้ design tokens
- reuse component ก่อนสร้าง component ใหม่
- รองรับ desktop + mobile
- มี hover / focus / disabled
- ใช้ semantic HTML
- รองรับ keyboard
- รักษา spacing consistency
- ใช้ Plaiflow primary blue เป็นสีหลัก

## SHOULD

- ใช้ card แบบ soft
- ใช้ gradient เฉพาะจุด
- ใช้ AI highlight อย่างพอดี
- ลด cognitive load
- ใช้ skeleton สำหรับ data loading

## MUST NOT

- hardcode สีใหม่โดยไม่มีเหตุผล
- ใช้สี primary หลายเฉดแบบสุ่ม
- สร้าง button style ใหม่ทุกหน้า
- ใช้ shadow หนัก
- ทำ gradient background ในทุก card
- ใช้ animation มากเกินไป

---

# 45. UI Review Checklist

ก่อน merge UI ทุกครั้งตรวจ:

- [ ] ใช้ token หรือไม่
- [ ] spacing สม่ำเสมอหรือไม่
- [ ] typography ถูก hierarchy หรือไม่
- [ ] mobile ใช้งานได้หรือไม่
- [ ] keyboard ใช้งานได้หรือไม่
- [ ] focus state มีหรือไม่
- [ ] loading state มีหรือไม่
- [ ] empty state มีหรือไม่
- [ ] error state มีหรือไม่
- [ ] primary CTA ชัดหรือไม่
- [ ] contrast เพียงพอหรือไม่
- [ ] AI element แย่งความสนใจหรือไม่
- [ ] component สามารถ reuse ได้หรือไม่

---

# 46. Final Design Rule

ทุกหน้าของ Plaiflow ควรตอบคำถามนี้ได้:

> ผู้ใช้เข้าหน้านี้แล้วรู้ทันทีหรือไม่ว่า<br>
> **กำลังดูอะไร — ต้องตรวจอะไร — และต้องทำอะไรต่อ**

ถ้าคำตอบคือ “ไม่” ให้ลด UI ที่ไม่จำเป็นก่อนเพิ่ม UI ใหม่

---

# 47. Plaiflow Design Summary

Plaiflow =

```txt
Professional SaaS
+
Friendly Interface
+
AI Assistance
+
Document Workflow
+
Accounting Clarity
```

Visual formula:

```txt
White Space
+ Soft Blue
+ Rounded Cards
+ Clear Typography
+ Minimal Shadow
+ Smart AI Highlight
```

เป้าหมายสุดท้าย:

> **ทำให้ระบบบัญชีและเอกสารที่ซับซ้อน รู้สึกง่ายและทำงานได้เร็วขึ้น**
