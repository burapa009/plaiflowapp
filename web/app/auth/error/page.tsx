import Link from "next/link";

const errors = {
  access_denied: ["ยกเลิกการเข้าสู่ระบบแล้ว", "คุณยังไม่ได้อนุญาตให้ PlaiFlow เข้าสู่ระบบ"],
  expired_attempt: ["คำขอหมดอายุแล้ว", "เริ่มการเข้าสู่ระบบใหม่เพื่อรับคำขอที่ปลอดภัยอีกครั้ง"],
  provider_unavailable: ["ยังเชื่อมต่อ LINE ไม่ได้", "ผู้ให้บริการอาจไม่พร้อมใช้งานชั่วคราว กรุณาลองอีกครั้ง"],
  identity_conflict: ["บัญชีนี้ยังใช้ต่อไม่ได้", "บัญชีผู้ให้บริการเชื่อมกับผู้ใช้อื่นอยู่ กรุณาติดต่อผู้ดูแล"],
  session_expired: ["เซสชันหมดอายุแล้ว", "เข้าสู่ระบบใหม่ก่อนดำเนินการต่อ"],
  invalid_invite: ["คำเชิญใช้ไม่ได้แล้ว", "ขอคำเชิญใหม่จากผู้ดูแล Organization"],
  external_browser_required: ["กรุณาเปิดในเบราว์เซอร์", "เปิดลิงก์นี้ในเบราว์เซอร์ของเครื่อง แล้วเริ่มอีกครั้ง"],
} as const;

export default async function AuthError({ searchParams }: { searchParams: Promise<{ code?: string; request_id?: string }> }) {
  const params = await searchParams;
  const copy = errors[params.code as keyof typeof errors] ?? ["เข้าสู่ระบบไม่สำเร็จ", "เริ่มใหม่อีกครั้ง หรือแจ้งผู้ดูแลหากปัญหายังเกิดซ้ำ"];
  const requestID = /^[A-Za-z0-9_-]{8,64}$/.test(params.request_id ?? "") ? params.request_id : "";

  return (
    <section className="auth-page">
      <div autoFocus className="auth-card" role="alert" tabIndex={-1}>
        <p className="auth-brand">PlaiFlow</p>
        <p className="eyebrow">เข้าสู่ระบบไม่สำเร็จ</p>
        <h1>{copy[0]}</h1>
        <p className="intro">{copy[1]}</p>
        {requestID && <p className="request-id">รหัสอ้างอิง: <code>{requestID}</code></p>}
        <Link className="button inline-button" href="/">เริ่มใหม่</Link>
      </div>
    </section>
  );
}
