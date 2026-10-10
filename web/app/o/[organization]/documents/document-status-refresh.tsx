"use client";

import { useEffect, useTransition } from "react";
import { useRouter } from "next/navigation";

export default function DocumentStatusRefresh({ active, manual = false }: { active: boolean; manual?: boolean }) {
  const router = useRouter();
  const [pending, startTransition] = useTransition();
  useEffect(() => {
    if (!active || pending) return;
    const timer = window.setInterval(() => {
      if (document.visibilityState === "visible") startTransition(() => router.refresh());
    }, 2000);
    return () => window.clearInterval(timer);
  }, [active, pending, router]);
  return manual ? <div>{active && <p role="status">กำลังอัปเดตสถานะอัตโนมัติ เมื่ออ่านเสร็จแบบฟอร์มจะปรากฏที่นี่</p>}<button className="secondary-button" type="button" disabled={pending} onClick={() => startTransition(() => router.refresh())}>{pending ? "กำลังอัปเดต…" : "โหลดสถานะล่าสุด"}</button></div> : null;
}
