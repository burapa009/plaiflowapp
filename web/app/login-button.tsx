"use client";

import { useState } from "react";

export function LoginButton() {
  const [pending, setPending] = useState(false);

  return (
    <a
      aria-busy={pending}
      aria-disabled={pending}
      className="line-login"
      href="/api/auth/line/start?return_to=/organizations"
      onClick={(event) => {
        if (pending) event.preventDefault();
        else setPending(true);
      }}
    >
      {pending ? "กำลังเปิด LINE…" : "ล็อกอินด้วย LINE"}
    </a>
  );
}
