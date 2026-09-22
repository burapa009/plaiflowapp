import type { Metadata } from "next";
import Link from "next/link";
import Image from "next/image";
import "./globals.css";
import { NavLinks } from "./nav-links";

export const metadata: Metadata = { title: "PlaiFlow", description: "ศูนย์ดูแลเวิร์กโฟลว์ธุรกิจ" };

const Brand = () => <Link className="brand app-brand" href="/" aria-label="PlaiFlow หน้าแรก"><Image src="/plaiflow-logo.png" alt="PlaiFlow" width={256} height={128} priority /></Link>;

export default function RootLayout({ children }: Readonly<{ children: React.ReactNode }>) {
  return (
    <html lang="th">
      <body>
        <a className="skip-link" href="#main-content">ข้ามไปยังเนื้อหา</a>
        <div className="shell">
          <aside className="sidebar"><Brand /><NavLinks /></aside>
          <div>
            <header className="topbar"><Brand /><details className="mobile-nav"><summary aria-label="เปิดเมนู"><svg aria-hidden="true" width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><path d="M4 7h16M4 12h16M4 17h16" /></svg></summary><div className="mobile-sheet"><NavLinks /></div></details></header>
            <main className="main" id="main-content" tabIndex={-1}>{children}</main>
          </div>
        </div>
      </body>
    </html>
  );
}
