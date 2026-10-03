import Image from "next/image";
import Link from "next/link";
import { redirect } from "next/navigation";
import { sessionGET } from "@/lib/session-api";
import { LoginButton } from "./login-button";

const capabilities = [
  ["01", "รวมงานไว้ที่เดียว", "สร้าง มอบหมาย และติดตามงานของแต่ละ Organization ได้จากพื้นที่เดียว"],
  ["02", "กำหนดส่งชัดเจน", "เห็นลำดับความสำคัญ สถานะ และงานที่ใกล้ครบกำหนดโดยไม่ต้องไล่ถาม"],
  ["03", "แจ้งเตือนผ่าน LINE", "ส่งการแจ้งเตือนงานไปยังผู้รับผิดชอบผ่านช่องทางที่ทีมใช้อยู่แล้ว"],
  ["04", "แยกข้อมูลแต่ละบริษัท", "สมาชิก บทบาท และงานถูกแยกตาม Organization เพื่อให้จัดการทีมได้อย่างมั่นใจ"],
];

const highlights = [
  ["ใช้งานง่าย", "เริ่มจัดการงานได้ในไม่กี่ขั้นตอน"],
  ["ทำงานร่วมกันได้ดีขึ้น", "ทีมงานเห็นงานเดียวกัน"],
  ["ติดตามความคืบหน้า", "รู้ว่างานไหนต้องทำต่อ"],
  ["ปลอดภัย มั่นใจได้", "แยกข้อมูลแต่ละ Organization"],
];

const businessCategories = [
  { name: "SME", detail: "ธุรกิจเติบโต", icon: "store", tone: "bg-gradient-to-br from-[#eff6ff] to-[#dbeafe] text-[#2563eb]" },
  { name: "E-Commerce", detail: "ร้านค้าออนไลน์", icon: "cart", tone: "bg-gradient-to-br from-[#ecfeff] to-[#cffafe] text-[#0891b2]" },
  { name: "Agency", detail: "ทีมครีเอทีฟ", icon: "spark", tone: "bg-gradient-to-br from-[#f1f0ff] to-[#e0e7ff] text-[#6366f1]" },
  { name: "Healthcare", detail: "สถานพยาบาล", icon: "heart", tone: "bg-gradient-to-br from-[#ecfdf5] to-[#d1fae5] text-[#0f9f88]" },
  { name: "Education", detail: "สถาบันการศึกษา", icon: "book", tone: "bg-gradient-to-br from-[#fffbeb] to-[#fef3c7] text-[#d97706]" },
  { name: "Real Estate", detail: "ธุรกิจอสังหาฯ", icon: "building", tone: "bg-gradient-to-br from-[#f8fafc] to-[#e2e8f0] text-[#52677d]" },
] as const;

function BusinessIcon({ type }: { type: (typeof businessCategories)[number]["icon"] }) {
  const paths = {
    store: <><path d="M4 10h16v9H4z" /><path d="M3 10 5 4h14l2 6M4 10h16M8 10v2M12 10v2M16 10v2M8 14h3v5H8z" /></>,
    cart: <><path d="M4 5h2l1.5 9h9L19 8H7" /><path d="M10 8h5v4h-5zM11 19h.01M18 19h.01" strokeWidth="2.4" /></>,
    spark: <><rect x="4" y="6" width="14" height="13" rx="2" /><path d="M9 6V4h6v2M8 11h6M8 15h3M19 9v8M19 9l2 2M19 9l-2 2" /></>,
    heart: <><path d="M20.8 8.7c0 5.2-8.8 10.2-8.8 10.2S3.2 13.9 3.2 8.7A4.7 4.7 0 0 1 12 6.2a4.7 4.7 0 0 1 8.8 2.5Z" /><path d="M5.5 12h3l1.4-2.8 2.4 5.2 1.5-2.4H18" /></>,
    book: <><path d="M4 5.5A2.5 2.5 0 0 1 6.5 3H20v15H6.5A2.5 2.5 0 0 0 4 20.5z" /><path d="M4 5.5v15M8 7h8M8 10h5M16 3v5l-2-1.3L12 8V3" /></>,
    building: <><path d="M4 20V6l8-3 8 3v14M2 20h20M8 9h2M14 9h2M8 13h2M14 13h2M10 20v-4h4v4" /><path d="M5 6h14M12 3v3" /></>,
  };
  return <svg className="size-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">{paths[type]}</svg>;
}

export default async function Welcome() {
  const session = await sessionGET("/v1/session");
  if (session?.ok) redirect("/open/dashboard");
  if (session?.status !== 401) return <section className="error-state" role="alert"><h1>ยังเปิดหน้าแรกไม่ได้</h1><Link className="button inline-button" href="/">ลองใหม่</Link></section>;
  return (
    <div className="auth-page landing-page" id="top">
      <header className="landing-header">
        <a className="landing-brand" href="#top" aria-label="PlaiFlow หน้าหลัก">
          <Image className="landing-logo-image" src="/plaiflow-logo.png" alt="PlaiFlow — Work Flows, Made Simple" width={240} height={120} priority />
        </a>
        <nav className="landing-nav" aria-label="เมนูหลัก">
          <a href="#features">จุดเด่น</a>
          <a href="#workflow">ฟีเจอร์</a>
          <a href="/pricing">ราคา</a>
          <a href="#reviews">รีวิว</a>
          <a href="#faq">คำถามที่พบบ่อย</a>
        </nav>
        <div className="landing-header-actions"><a className="header-signin" href="#start">เข้าสู่ระบบ</a><a className="header-cta" href="#start">เริ่มต้นใช้งานฟรี →</a></div>
        <details className="landing-mobile-menu"><summary aria-label="เปิดเมนู">☰</summary><nav aria-label="เมนูมือถือ"><a href="#features">จุดเด่น</a><a href="#workflow">ฟีเจอร์</a><a href="/pricing">ราคา</a><a href="#reviews">รีวิว</a><a href="#faq">คำถามที่พบบ่อย</a><a href="#start">เข้าสู่ระบบ</a></nav></details>
      </header>

      <div className="landing-content">
        <section className="landing-hero" aria-labelledby="hero-title">
          <div className="hero-copy">
            <p className="hero-eyebrow"><span aria-hidden="true">✦</span> LINE-FIRST BUSINESS WORKSPACE</p>
            <h1 id="hero-title">ให้งานทีมเดินหน้า<br /><span>ไม่ตกหล่นระหว่างทาง</span></h1>
            <p className="hero-intro">PlaiFlow รวมงาน ผู้รับผิดชอบ กำหนดส่ง และการแจ้งเตือนไว้ในระบบเดียว เพื่อให้ทีมธุรกิจทำงานร่วมกันได้ง่ายขึ้น ทั้งเว็บและ LINE</p>
            <div className="hero-actions" id="start">
              <LoginButton />
            </div>
            <ul className="hero-points" aria-label="จุดเด่นสำคัญ">
              <li><span className="check-mark" aria-hidden="true" />เริ่มใช้งานได้ทันที</li>
              <li><span className="check-mark" aria-hidden="true" />ไม่ต้องติดตั้งโปรแกรม</li>
              <li><span className="check-mark" aria-hidden="true" />แยกข้อมูลแต่ละทีม</li>
            </ul>
          </div>

          <div className="preview-wrap" aria-label="ตัวอย่างหน้าตาพื้นที่จัดการงาน PlaiFlow ใช้ข้อมูลสมมติ">
            <div className="preview-note">จัดการงานได้ครบ<br />ในที่เดียว <span aria-hidden="true">↘</span></div>
            <div className="product-preview">
              <div className="preview-windowbar"><span /><span /><span /><p>ตัวอย่างหน้าตา PlaiFlow</p></div>
              <div className="preview-appbar"><b>Plai<span>Flow</span></b><div className="preview-search">⌕ &nbsp; ค้นหางาน โปรเจกต์ หรือผู้รับผิดชอบ...</div><span className="preview-avatar">PB</span></div>
              <div className="preview-layout">
                <aside className="preview-sidebar" aria-hidden="true"><span className="active">⌂ &nbsp; หน้าหลัก</span><span>▣ &nbsp; งานของฉัน</span><span>▤ &nbsp; โปรเจกต์</span><span>□ &nbsp; ปฏิทิน</span><span>◫ &nbsp; ไฟล์เอกสาร</span><span>♧ &nbsp; ลูกค้า</span><span>▥ &nbsp; รายงาน</span><span>⚙ &nbsp; ตั้งค่า</span></aside>
                <div className="preview-main">
                  <div className="preview-heading"><div><h2>สวัสดีครับ บูรพา 👋</h2><small>นี่คือภาพรวมงานของคุณวันนี้</small></div><span className="preview-date">1 – 30 ก.ย. 2026 ⌄</span></div>
                  <div className="preview-metrics">
                    <article><span className="metric-icon blue">▤</span><small>งานทั้งหมด</small><b>24</b><em>+12%</em></article>
                    <article><span className="metric-icon cyan">▶</span><small>กำลังดำเนินการ</small><b>8</b><em>+2</em></article>
                    <article><span className="metric-icon amber">◷</span><small>ใกล้ครบกำหนด</small><b>5</b><em className="metric-down">−3</em></article>
                    <article><span className="metric-icon green">✓</span><small>เสร็จสิ้น</small><b>16</b><em>+8</em></article>
                  </div>
                  <div className="preview-panels">
                    <article className="chart-card"><div><b>ความคืบหน้างาน</b><small>6 เดือนล่าสุด</small></div><div className="mini-chart" aria-hidden="true"><i /><i /><i /><i /><i /><i /><i /><i /><i /><i /></div><div className="chart-months" aria-hidden="true"><span>ม.ค.</span><span>ก.พ.</span><span>มี.ค.</span><span>เม.ย.</span><span>พ.ค.</span><span>มิ.ย.</span><span>ก.ค.</span><span>ส.ค.</span><span>ก.ย.</span></div></article>
                    <article className="task-card-preview"><div className="preview-task-head"><b>งานที่ต้องทำวันนี้</b><span>ดูทั้งหมด →</span></div><p>□ &nbsp; สรุปรายงานยอดขาย <strong>ด่วน</strong><small>วันนี้</small></p><p>□ &nbsp; ประชุมทีมการตลาด <small>วันนี้</small></p><p>□ &nbsp; ตรวจเอกสารลูกค้า <small>พรุ่งนี้</small></p><p>□ &nbsp; อัปเดตเว็บไซต์ <small>พรุ่งนี้</small></p><p>□ &nbsp; เตรียมแคมเปญ LINE <small>12 ก.ย.</small></p></article>
                  </div>
                </div>
              </div>
            </div>
            <div className="line-float"><span aria-hidden="true" />ใช้งานได้ทั้งเว็บและ LINE</div>
          </div>
        </section>

        <section className="landing-proof" id="features" aria-label="จุดเด่นของ PlaiFlow"><div className="feature-strip">{highlights.map(([title, description], index) => <article key={title}><span className="feature-icon" aria-hidden="true">{["✦", "✧", "▥", "◇"][index]}</span><div><h2>{title}</h2><p>{description}</p></div></article>)}</div><div className="testimonial-card" id="reviews"><span aria-hidden="true">“</span><p>“ทีมเห็นงานตรงกัน รู้ว่าใครรับผิดชอบ และติดตามกำหนดส่งได้ง่ายขึ้น”</p><small>ตัวอย่างประสบการณ์การใช้งาน</small></div></section>
        <section className="business-categories" aria-labelledby="business-categories-title">
          <div className="flex flex-wrap items-end justify-between gap-4">
            <div>
              <h2 id="business-categories-title" className="m-0 text-sm font-extrabold tracking-[-.01em] text-[#183750]">ออกแบบให้ใช้ได้กับหลากหลายประเภท</h2>
            </div>
            <p className="m-0 text-xs text-[#7890a4]">เลือกมุมมองที่เหมาะกับทีมของคุณ</p>
          </div>
          <ul className="mt-6 grid list-none grid-cols-1 gap-3 p-0 min-[420px]:grid-cols-2 md:grid-cols-6 md:gap-4">
            {businessCategories.map(({ name, detail, icon, tone }) => (
              <li key={name} className="group flex min-w-0 items-center gap-3 rounded-[1rem] border border-[#dceaf3] bg-gradient-to-br from-white to-[#f7fbff] px-3.5 py-3 shadow-[0_6px_24px_rgba(15,41,66,.05)] transition duration-200 hover:-translate-y-1 hover:border-[#b9d9eb] hover:shadow-[0_12px_30px_rgba(37,99,235,.1)]">
                <span className={`-translate-y-1 grid size-9 flex-none place-items-center rounded-xl ring-2 ring-white/90 shadow-[0_4px_12px_rgba(15,41,66,.08)] md:-translate-y-4 ${tone}`}><BusinessIcon type={icon} /></span>
                <span className="min-w-0 flex-1">
                  <strong className="block truncate text-[.75rem] font-extrabold leading-tight tracking-[-.01em] text-[#183750]">{name}</strong>
                  <span className="mt-1 block truncate text-[.64rem] leading-tight text-[#74899c]">{detail}</span>
                </span>
              </li>
            ))}
          </ul>
        </section>

        <section className="workflow-section" id="workflow" aria-labelledby="workflow-title">
          <div className="section-copy"><p className="hero-eyebrow">ทำงานร่วมกันอย่างเป็นระบบ</p><h2 id="workflow-title">เริ่มง่าย และเห็นความคืบหน้าตลอดทาง</h2></div>
          <ol className="workflow-steps">
            <li><span>1</span><div><b>เข้าสู่ระบบด้วย LINE</b><p>ใช้บัญชีที่คุ้นเคยเพื่อเริ่มพื้นที่ทำงาน</p></div></li>
            <li><span>2</span><div><b>สร้าง Organization และทีม</b><p>แยกสมาชิก บทบาท และงานของแต่ละบริษัท</p></div></li>
            <li><span>3</span><div><b>มอบหมายและติดตามงาน</b><p>ทุกคนเห็นเจ้าของงาน สถานะ และกำหนดส่งตรงกัน</p></div></li>
          </ol>
        </section>

        <section className="capabilities-section" aria-labelledby="features-title">
          <div className="section-copy"><p className="hero-eyebrow">สิ่งที่ทีมได้ใช้จริง</p><h2 id="features-title">เครื่องมือที่พอดีกับงานประจำวัน</h2></div>
          <div className="capability-grid">
            {capabilities.map(([number, title, description]) => <article key={number}><span aria-hidden="true">{number}</span><h3>{title}</h3><p>{description}</p></article>)}
          </div>
        </section>

        <section className="security-section" id="security" aria-labelledby="security-title">
          <div><p className="hero-eyebrow">ออกแบบเพื่อความไว้วางใจ</p><h2 id="security-title">ข้อมูลของแต่ละ Organization ไม่ปะปนกัน</h2><p>PlaiFlow ตรวจสิทธิ์ทุกคำขอ เก็บ session ฝั่ง server และไม่ส่ง token สำคัญไปยัง browser</p></div>
          <a className="secondary-cta" href="#start">เริ่มจัดการงาน <span aria-hidden="true">→</span></a>
        </section>
        <section className="faq-section" id="faq" aria-labelledby="faq-title"><div className="section-copy"><p className="hero-eyebrow">คำถามที่พบบ่อย</p><h2 id="faq-title">เริ่มใช้งานอย่างมั่นใจ</h2></div><div><details><summary>เริ่มใช้งานอย่างไร?</summary><p>เข้าสู่ระบบด้วย LINE หรือ Google แล้วสร้าง Organization แรกเพื่อเริ่มจัดการงาน</p></details><details><summary>ใช้บนโทรศัพท์ได้ไหม?</summary><p>หน้าเว็บปรับตามขนาดหน้าจอ และเปิดผ่านเบราว์เซอร์บนโทรศัพท์ได้</p></details><details><summary>ข้อมูลของแต่ละบริษัทแยกกันไหม?</summary><p>งาน สมาชิก และสิทธิ์ถูกแยกตาม Organization</p></details></div></section>
      </div>

      <footer className="landing-footer"><span>© 2026 PlaiFlow</span><a className="underline underline-offset-4" href="/privacy">นโยบายความเป็นส่วนตัว / Privacy Policy</a><span>งานชัด ทีมคล่อง ธุรกิจเดินหน้า</span></footer>
    </div>
  );
}
