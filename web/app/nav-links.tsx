"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";

const links = [["ภาพรวม", "/"], ["สถานะระบบ", "/system-status"]] as const;

export function NavLinks() {
  const pathname = usePathname();
  const organization = pathname.match(/^\/o\/([^/]+)/)?.[1];
  const workspaceLinks = organization ? [["งานของทีม", `/o/${organization}/tasks`], ["เอกสาร", `/o/${organization}/documents`], ["คู่ค้า", `/o/${organization}/vendors`], ["Connections", `/o/${organization}/connections`], ["เปลี่ยน Organization", "/organizations"]] as const : pathname === "/design-preview" ? [["งานของทีม", "/design-preview"], ["หน้าแรก", "/"], ["สถานะระบบ", "/system-status"]] as const : links;
  return <nav aria-label="เมนูหลัก"><ul className="nav-list">{workspaceLinks.map(([label, href]) => <li key={href}><Link aria-current={pathname === href ? "page" : undefined} className="nav-link" href={href}>{label}</Link></li>)}</ul></nav>;
}
