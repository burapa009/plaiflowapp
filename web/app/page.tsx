import { LoginButton } from "./login-button";

export default function Welcome() {
  return (
    <section className="auth-page">
      <div className="auth-card">
        <p className="auth-brand">PlaiFlow</p>
        <p className="eyebrow">เริ่มต้นใช้งาน</p>
        <h1>จัดการงานธุรกิจให้ไหลลื่นในที่เดียว</h1>
        <p className="intro">เข้าสู่ระบบด้วยบัญชี LINE ของคุณ แล้วตั้งชื่อ Organization แรกได้ทันที</p>
        <LoginButton />
        <p className="auth-note">PlaiFlow จะไม่เริ่มการเข้าสู่ระบบจนกว่าคุณจะกดปุ่ม</p>
      </div>
    </section>
  );
}
