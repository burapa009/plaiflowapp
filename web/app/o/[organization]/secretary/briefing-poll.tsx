"use client";

import { useRouter } from "next/navigation";
import { useEffect } from "react";

export function BriefingPoll() {
  const router = useRouter();
  useEffect(() => {
    const timer = window.setInterval(() => router.refresh(), 4000);
    return () => window.clearInterval(timer);
  }, [router]);
  return <p role="status">กำลังเตรียมสรุปงาน ระบบจะอัปเดตหน้านี้อัตโนมัติ</p>;
}
