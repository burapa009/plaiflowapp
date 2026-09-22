"use client";

import { useState } from "react";

export function LoginButton() {
  const [pending, setPending] = useState<"line" | "google" | null>(null);

  return (
    <div className="login-options">
      <a aria-busy={pending === "line"} aria-disabled={pending !== null} className="line-login hero-primary-cta" href="/api/auth/line/start?return_to=/organizations" onClick={(event) => pending ? event.preventDefault() : setPending("line")}>
        {pending === "line" ? "กำลังเปิด LINE…" : "เริ่มต้นใช้งานฟรี →"}
      </a>
      <a aria-disabled={pending !== null} className="secondary-cta line-connect" href="/api/auth/line/start?return_to=/organizations" onClick={(event) => pending ? event.preventDefault() : setPending("line")}><span className="line-brand-mark" aria-hidden="true">LINE</span>เชื่อมต่อ LINE</a>
      <a aria-busy={pending === "google"} aria-disabled={pending !== null} className="secondary-cta google-login" href="/api/auth/google/start?return_to=/organizations" onClick={(event) => pending ? event.preventDefault() : setPending("google")}>
        <span className="google-brand-mark" aria-hidden="true">G</span>{pending === "google" ? "กำลังเปิด Google…" : "ลงชื่อเข้าใช้ด้วย Google"}
      </a>
    </div>
  );
}
