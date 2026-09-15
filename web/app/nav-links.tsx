"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";

const links = [["ภาพรวม", "/"], ["สถานะระบบ", "/system-status"]] as const;

export function NavLinks() {
  const pathname = usePathname();
  return <nav aria-label="เมนูหลัก"><ul className="nav-list">{links.map(([label, href]) => <li key={href}><Link aria-current={pathname === href ? "page" : undefined} className="nav-link" href={href}>{label}</Link></li>)}</ul></nav>;
}
