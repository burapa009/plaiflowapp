import Image from "next/image";
import { LoginButton } from "./login-button";

const capabilities = [
  ["01", "รวมงานไว้ที่เดียว", "สร้าง มอบหมาย และติดตามงานของแต่ละ Organization ได้จากพื้นที่เดียว"],
  ["02", "กำหนดส่งชัดเจน", "เห็นลำดับความสำคัญ สถานะ และงานที่ใกล้ครบกำหนดโดยไม่ต้องไล่ถาม"],
  ["03", "แจ้งเตือนผ่าน LINE", "ส่งการแจ้งเตือนงานไปยังผู้รับผิดชอบผ่านช่องทางที่ทีมใช้อยู่แล้ว"],
  ["04", "แยกข้อมูลแต่ละบริษัท", "สมาชิก บทบาท และงานถูกแยกตาม Organization เพื่อให้จัดการทีมได้อย่างมั่นใจ"],
];

export default function Welcome() {
  return (
    <section className="auth-page landing-page">
      <header className="landing-header">
        <a className="landing-brand" href="#top" aria-label="PlaiFlow หน้าหลัก">
          <Image className="landing-logo-image" src="/plaiflow-logo.png" alt="PlaiFlow — Work Flows, Made Simple" width={240} height={120} priority />
        </a>
        <nav className="landing-nav" aria-label="เมนูหลัก">
          <a href="#features">จุดเด่น</a>
          <a href="#workflow">การทำงาน</a>
          <a href="#security">ความปลอดภัย</a>
          <a href="/pricing">แพ็กเกจ</a>
        </nav>
        <a className="header-cta" href="#start">เริ่มต้นใช้งาน</a>
      </header>

      <div id="top" className="landing-content">
        <section className="landing-hero" aria-labelledby="hero-title">
          <div className="hero-copy">
            <p className="hero-eyebrow"><span aria-hidden="true">✦</span> LINE-FIRST BUSINESS WORKSPACE</p>
            <h1 id="hero-title">ให้งานทีมเดินหน้า<br /><span>ไม่ตกหล่นระหว่างทาง</span></h1>
            <p className="hero-intro">PlaiFlow รวมงาน ผู้รับผิดชอบ กำหนดส่ง และการแจ้งเตือนไว้ในระบบเดียว เพื่อให้ทีมธุรกิจทำงานร่วมกันได้ง่ายทั้งบนเว็บและ LINE</p>
            <div className="hero-actions" id="start">
              <LoginButton />
              <a className="secondary-cta" href="#workflow">ดูวิธีการทำงาน <span aria-hidden="true">→</span></a>
              <a className="secondary-cta" href="/pricing">ดูแพ็กเกจ</a>
            </div>
            <ul className="hero-points" aria-label="จุดเด่นสำคัญ">
              <li><span className="check-mark" aria-hidden="true" />เริ่มต้นจาก LINE</li>
              <li><span className="check-mark" aria-hidden="true" />แยกพื้นที่แต่ละบริษัท</li>
              <li><span className="check-mark" aria-hidden="true" />ใช้ได้ทุกขนาดหน้าจอ</li>
            </ul>
          </div>

          <div className="preview-wrap" aria-label="ตัวอย่างพื้นที่จัดการงานของ PlaiFlow" role="img">
            <div className="line-float"><span aria-hidden="true" />LINE พร้อมแจ้งเตือน</div>
            <div className="product-preview">
              <div className="preview-windowbar"><span /><span /><span /><p>พื้นที่ทำงาน PlaiFlow</p></div>
              <div className="preview-appbar"><b>PlaiFlow</b><div className="preview-search">ค้นหางานหรือสมาชิก…</div><span className="preview-avatar">PB</span></div>
              <div className="preview-layout">
                <aside className="preview-sidebar"><strong>เมนูหลัก</strong><span className="active">ภาพรวม</span><span>งานของทีม</span><span>การแจ้งเตือน</span><span>สมาชิก</span></aside>
                <div className="preview-main">
                  <div className="preview-heading"><div><small>วันพุธที่ 16 กันยายน</small><h2>ภาพรวมงานวันนี้</h2></div><button type="button" tabIndex={-1}>+ สร้างงาน</button></div>
                  <div className="preview-metrics">
                    <article><small>งานทั้งหมด</small><b>24</b><span>ในพื้นที่นี้</span></article>
                    <article><small>กำลังทำ</small><b>8</b><span>ต้องติดตาม</span></article>
                    <article><small>เสร็จแล้ว</small><b>16</b><span>เดินหน้าตามแผน</span></article>
                  </div>
                  <div className="preview-panels">
                    <article className="chart-card"><div><b>งานในรอบสัปดาห์</b><small>อัปเดตล่าสุดวันนี้</small></div><div className="mini-chart" aria-hidden="true"><i /><i /><i /><i /><i /><i /><i /><i /><i /></div></article>
                    <article className="task-card-preview"><b>งานล่าสุด</b><p><span className="task-dot teal" />เตรียมเอกสารประชุม <small>วันนี้</small></p><p><span className="task-dot orange" />ติดตามลูกค้าใหม่ <small>พรุ่งนี้</small></p><p><span className="task-dot blue" />สรุปงานประจำสัปดาห์ <small>ศุกร์</small></p></article>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </section>

        <section className="workflow-section" id="workflow" aria-labelledby="workflow-title">
          <div className="section-copy"><p className="hero-eyebrow">ทำงานร่วมกันอย่างเป็นระบบ</p><h2 id="workflow-title">เริ่มง่าย และเห็นความคืบหน้าตลอดทาง</h2></div>
          <ol className="workflow-steps">
            <li><span>1</span><div><b>เข้าสู่ระบบด้วย LINE</b><p>ใช้บัญชีที่คุ้นเคยเพื่อเริ่มพื้นที่ทำงาน</p></div></li>
            <li><span>2</span><div><b>สร้าง Organization และทีม</b><p>แยกสมาชิก บทบาท และงานของแต่ละบริษัท</p></div></li>
            <li><span>3</span><div><b>มอบหมายและติดตามงาน</b><p>ทุกคนเห็นเจ้าของงาน สถานะ และกำหนดส่งตรงกัน</p></div></li>
          </ol>
        </section>

        <section className="capabilities-section" id="features" aria-labelledby="features-title">
          <div className="section-copy"><p className="hero-eyebrow">สิ่งที่ทีมได้ใช้จริง</p><h2 id="features-title">เครื่องมือที่พอดีกับงานประจำวัน</h2></div>
          <div className="capability-grid">
            {capabilities.map(([number, title, description]) => <article key={number}><span aria-hidden="true">{number}</span><h3>{title}</h3><p>{description}</p></article>)}
          </div>
        </section>

        <section className="security-section" id="security" aria-labelledby="security-title">
          <div><p className="hero-eyebrow">ออกแบบเพื่อความไว้วางใจ</p><h2 id="security-title">ข้อมูลของแต่ละ Organization ไม่ปะปนกัน</h2><p>PlaiFlow ตรวจสิทธิ์ทุกคำขอ เก็บ session ฝั่ง server และไม่ส่ง token สำคัญไปยัง browser</p></div>
          <a className="secondary-cta" href="#start">เริ่มจัดการงาน <span aria-hidden="true">→</span></a>
        </section>
      </div>

      <footer className="landing-footer"><span>© 2026 PlaiFlow</span><span>งานชัด ทีมคล่อง ธุรกิจเดินหน้า</span></footer>
    </section>
  );
}
