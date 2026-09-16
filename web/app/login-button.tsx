"use client";

import { useState } from "react";

export function LoginButton() {
  const [pending, setPending] = useState<"line" | "google" | null>(null);

  return (
    <div className="login-options">
      <a aria-busy={pending === "line"} aria-disabled={pending !== null} className="line-login" href="/api/auth/line/start?return_to=/organizations" onClick={(event) => pending ? event.preventDefault() : setPending("line")}>
        {pending === "line" ? "กำลังเปิด LINE…" : "ล็อกอินด้วย LINE"}
      </a>
      <a aria-busy={pending === "google"} aria-disabled={pending !== null} className="secondary-cta google-login" href="/api/auth/google/start?return_to=/organizations" onClick={(event) => pending ? event.preventDefault() : setPending("google")}>
        {pending === "google" ? "กำลังเปิด Google…" : "ล็อกอินด้วย Google"}
      </a>
    </div>
  );
}
