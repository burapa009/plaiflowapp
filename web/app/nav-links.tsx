"use client";

import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import { useState } from "react";

const links = [["ภาพรวม", "/"], ["สถานะระบบ", "/system-status"]] as const;

function NavIcon({ href }: { href: string }) {
  const path = href.endsWith("/tasks") ? "M3 12l2-2m0 0l7-7 7 7M5 10v10a1 1 0 001 1h3m10-11l2 2m-2-2v10a1 1 0 01-1 1h-3m-6 0a1 1 0 001-1v-4a1 1 0 011-1h2a1 1 0 011 1v4a1 1 0 001 1m-6 0h6" :
    href.endsWith("/documents") ? "M7 3h7l4 4v14H7zM14 3v5h5M10 13h5M10 17h5" :
    href.endsWith("/exports") ? "M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-8l-4-4m0 0L8 8m4-4v12" :
    href.endsWith("/settings") ? "M12 15a3 3 0 1 0 0-6 3 3 0 0 0 0 6ZM19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1 0 2.83 2 2 0 0 1-2.83 0l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-2 2 2 2 0 0 1-2-2v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83 0 2 2 0 0 1 0-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1-2-2 2 2 0 0 1 2-2h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 0-2.83 2 2 0 0 1 2.83 0l.06.06a1.65 1.65 0 0 0 1.82.33H9a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 2-2 2 2 0 0 1 2 2v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 0 2 2 0 0 1 0 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 2 2 2 2 0 0 1-2 2h-.09a1.65 1.65 0 0 0-1.51 1Z" :
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

  const isSettingsActive = organization ? (
    pathname.startsWith(`/o/${organization}/settings`) ||
    pathname.startsWith(`/o/${organization}/business`) ||
    pathname.startsWith(`/o/${organization}/vendors`) ||
    pathname.startsWith(`/o/${organization}/members`) ||
    pathname.startsWith(`/o/${organization}/line-groups`) ||
    pathname.startsWith(`/o/${organization}/connections`)
  ) : false;

  const isDocumentsActive = organization ? (
    pathname.startsWith(`/o/${organization}/documents`) ||
    pathname.startsWith(`/o/${organization}/review`)
  ) : false;

  const workspaceLinks = organization ? [
    ["หน้าหลัก", `/o/${organization}/tasks`],
    ["กล่องเอกสาร", `/o/${organization}/documents`],
    ["ส่งออกข้อมูล", `/o/${organization}/exports`],
    ["ตั้งค่า", `/o/${organization}/settings`],
    ["เปลี่ยนองค์กร", "/organizations"]
  ] as const : pathname === "/design-preview" ? [
    ["งานของทีม", "/design-preview"],
    ["หน้าแรก", "/"],
    ["สถานะระบบ", "/system-status"]
  ] as const : links;

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

  function checkActive(href: string) {
    if (organization) {
      if (href.endsWith("/settings")) return isSettingsActive;
      if (href.endsWith("/documents")) return isDocumentsActive;
    }
    return pathname === href;
  }

  return (
    <nav aria-label="เมนูหลัก">
      <ul className="nav-list">
        {workspaceLinks.map(([label, href]) => (
          <li key={href}>
            <Link aria-current={checkActive(href) ? "page" : undefined} className="nav-link" href={href}>
              <NavIcon href={href} />
              <span>{label}</span>
            </Link>
          </li>
        ))}
        {process.env.NEXT_PUBLIC_BILLING_ENABLED === "true" && organization && (
          <li>
            <Link aria-current={pathname === `/o/${organization}/billing` ? "page" : undefined} className="nav-link" href={`/o/${organization}/billing`}>
              <NavIcon href="/billing" />
              <span>การชำระเงิน</span>
            </Link>
          </li>
        )}
        {process.env.NEXT_PUBLIC_FIRM_ENABLED === "true" && organization && (
          <li>
            <Link aria-current={pathname === `/o/${organization}/firm` ? "page" : undefined} className="nav-link" href={`/o/${organization}/firm`}>
              <NavIcon href="/firm" />
              <span>สำนักงานบัญชี</span>
            </Link>
          </li>
        )}
        {organization && (
          <li>
            <button className="nav-link w-full cursor-pointer border-0 bg-transparent text-left font-[inherit] text-inherit disabled:cursor-wait" type="button" onClick={logout} disabled={loggingOut} aria-busy={loggingOut}>
              <NavIcon href="/api/logout" />
              <span>{loggingOut ? "กำลังออกจากระบบ…" : "ออกจากระบบ"}</span>
            </button>
            {logoutError && <p className="px-3 text-sm text-danger" role="alert">ออกจากระบบไม่สำเร็จ กรุณาลองใหม่</p>}
          </li>
        )}
      </ul>
    </nav>
  );
}
