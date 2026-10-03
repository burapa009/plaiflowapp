"use client";

import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import { useState } from "react";

const links = [["ภาพรวม", "/"], ["สถานะระบบ", "/system-status"]] as const;

function NavIcon({ href }: { href: string }) {
  const path = href.endsWith("/tasks") ? "M9 6h11M9 12h11M9 18h11M4 6h.01M4 12h.01M4 18h.01" :
    href.endsWith("/documents") ? "M7 3h7l4 4v14H7zM14 3v5h5M10 13h5M10 17h5" :
    href.endsWith("/vendors") ? "M3 21h18M5 21V7l7-4 7 4v14M9 10h.01M12 10h.01M15 10h.01M9 14h.01M12 14h.01M15 14h.01" :
    href.endsWith("/members") ? "M16 21v-2a4 4 0 0 0-4-4H6a4 4 0 0 0-4 4v2M9 11a4 4 0 1 0 0-8 4 4 0 0 0 0 8ZM22 21v-2a4 4 0 0 0-3-3.87M16 3.13a4 4 0 0 1 0 7.75" :
    href.endsWith("/line-groups") ? "M4 5h16v11H8l-4 4V5ZM8 9h8M8 12h5" :
    href.endsWith("/connections") ? "M10 13a5 5 0 0 0 7.54.54l2-2a5 5 0 0 0-7.07-7.07l-1.15 1.15M14 11a5 5 0 0 0-7.54-.54l-2 2a5 5 0 0 0 7.07 7.07l1.15-1.15" :
    href === "/organizations" ? "M4 21v-7h16v7M7 14V7l5-4 5 4v7M9 10h.01M12 10h.01M15 10h.01" :
    href === "/system-status" ? "M4 14h4l2-8 4 12 2-7h4" :
    href === "/api/logout" ? "M10 17l5-5-5-5M15 12H3M12 3h7a2 2 0 0 1 2 2v14a2 2 0 0 1-2 2h-7" :
    "M3 12l9-9 9 9M5 10v11h14V10M9 21v-7h6v7";
  return <svg className="nav-icon" aria-hidden="true" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round"><path d={path} /></svg>;
}

export function NavLinks() {
  const pathname = usePathname();
  const router = useRouter();
  const [loggingOut, setLoggingOut] = useState(false);
  const [logoutError, setLogoutError] = useState(false);
  const organization = pathname.match(/^\/o\/([^/]+)/)?.[1];
  const workspaceLinks = organization ? [["งานของทีม", `/o/${organization}/tasks`], ["เอกสาร", `/o/${organization}/documents`], ["คู่ค้า", `/o/${organization}/vendors`], ["การจัดการธุรกิจ", `/o/${organization}/business`], ["สมาชิกและสิทธิ์", `/o/${organization}/members`], ["กลุ่ม LINE", `/o/${organization}/line-groups`], ["Connections", `/o/${organization}/connections`], ["เปลี่ยนองค์กร", "/organizations"]] as const : pathname === "/design-preview" ? [["งานของทีม", "/design-preview"], ["หน้าแรก", "/"], ["สถานะระบบ", "/system-status"]] as const : links;
  async function logout() {
    if (loggingOut) return;
    setLogoutError(false);
    const csrf = document.cookie.split("; ").find((entry) => entry.startsWith("__Host-plaiflow-csrf="))?.split("=")[1];
    if (!csrf) { setLogoutError(true); return; }
    setLoggingOut(true);
    try {
      const response = await fetch("/api/logout", { method: "POST", credentials: "same-origin", headers: { "X-CSRF-Token": csrf } });
      if (!response.ok && response.status !== 401) throw new Error("logout failed");
      router.replace("/");
      router.refresh();
    } catch {
      setLogoutError(true);
      setLoggingOut(false);
    }
  }

  return <nav aria-label="เมนูหลัก"><ul className="nav-list">{workspaceLinks.map(([label, href]) => <li key={href}><Link aria-current={pathname === href ? "page" : undefined} className="nav-link" href={href}><NavIcon href={href} /><span>{label}</span></Link></li>)}{process.env.NEXT_PUBLIC_BILLING_ENABLED === "true" && organization && <li><Link aria-current={pathname === `/o/${organization}/billing` ? "page" : undefined} className="nav-link" href={`/o/${organization}/billing`}><NavIcon href="/billing" /><span>การชำระเงิน</span></Link></li>}{process.env.NEXT_PUBLIC_FIRM_ENABLED === "true" && organization && <li><Link aria-current={pathname === `/o/${organization}/firm` ? "page" : undefined} className="nav-link" href={`/o/${organization}/firm`}><NavIcon href="/firm" /><span>สำนักงานบัญชี</span></Link></li>}{organization && <li><button className="nav-link w-full cursor-pointer border-0 bg-transparent text-left font-[inherit] text-inherit disabled:cursor-wait" type="button" onClick={logout} disabled={loggingOut} aria-busy={loggingOut}><NavIcon href="/api/logout" /><span>{loggingOut ? "กำลังออกจากระบบ…" : "ออกจากระบบ"}</span></button>{logoutError && <p className="px-3 text-sm text-danger" role="alert">ออกจากระบบไม่สำเร็จ กรุณาลองใหม่</p>}</li>}</ul></nav>;
}
