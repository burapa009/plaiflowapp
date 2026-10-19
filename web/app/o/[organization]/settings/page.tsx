import Link from "next/link";
import { redirect } from "next/navigation";
import { sessionGET } from "@/lib/session-api";

type Membership = { role: "Owner" | "Admin" | "Member"; organization_name: string };

export default async function SettingsPage({ params }: { params: Promise<{ organization: string }> }) {
  const { organization } = await params;
  const base = `/v1/o/${encodeURIComponent(organization)}`;
  const response = await sessionGET(base);
  if (response?.status === 401) redirect("/");
  if (!response?.ok) {
    return (
      <section className="error-state" role="alert">
        <h1>ยังเปิดหน้าตั้งค่าไม่ได้</h1>
        <Link className="secondary-button" href={`/o/${encodeURIComponent(organization)}/tasks`}>
          กลับหน้าหลัก
        </Link>
      </section>
    );
  }

  const { membership } = await response.json() as { membership: Membership };
  const root = `/o/${encodeURIComponent(organization)}`;

  const settingCards = [
    {
      title: "ข้อมูลธุรกิจ",
      description: "ชื่อองค์กร, เลขประจำตัวผู้เสียภาษี 13 หลัก, สาขา, ที่อยู่ และโลโก้สำหรับเอกสาร",
      href: `${root}/business`,
      icon: (
        <svg aria-hidden="true" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round">
          <path d="M3 21h18M5 21V7l7-4 7 4v14M9 10h.01M12 10h.01M15 10h.01M9 14h.01M12 14h.01M15 14h.01" />
        </svg>
      ),
      badge: "ข้อมูลหลัก",
    },
    {
      title: "คู่ค้าและผู้จำหน่าย",
      description: "จัดการรายชื่อผู้ขาย (Vendors) และลูกค้า เพื่อให้ AI จับคู่ชื่อและเลขภาษีในใบกำกับภาษีได้แม่นยำ",
      href: `${root}/vendors`,
      icon: (
        <svg aria-hidden="true" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round">
          <path d="M16 21v-2a4 4 0 0 0-4-4H6a4 4 0 0 0-4 4v2M9 11a4 4 0 1 0 0-8 4 4 0 0 0 0 8ZM22 21v-2a4 4 0 0 0-3-3.87M16 3.13a4 4 0 0 1 0 7.75" />
        </svg>
      ),
      badge: "ฐานข้อมูล",
    },
    {
      title: "สมาชิกและสิทธิ์",
      description: "เชิญเพื่อนร่วมงานเข้าร่วม จัดการบทบาทสิทธิ์ Owner, Admin, และ Member",
      href: `${root}/members`,
      icon: (
        <svg aria-hidden="true" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round">
          <path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2M9 11a4 4 0 1 0 0-8 4 4 0 0 0 0 8M23 21v-2a4 4 0 0 0-3-3.87M16 3.13a4 4 0 0 1 0 7.75" />
        </svg>
      ),
      badge: "ทีมงาน",
    },
    {
      title: "กลุ่ม LINE สำหรับสแกนเอกสาร",
      description: "เชื่อมต่อกลุ่ม LINE เพื่อให้พนักงานสามารถถ่ายรูปใบเสร็จและส่งเข้าสู่ระบบ PlaiFlow ได้ทันทีจากมือถือ",
      href: `${root}/line-groups`,
      icon: (
        <svg aria-hidden="true" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round">
          <path d="M4 5h16v11H8l-4 4V5ZM8 9h8M8 12h5" />
        </svg>
      ),
      badge: "LINE Chat",
    },
    {
      title: "การเชื่อมต่อ Google Drive",
      description: "เชื่อมต่อบัญชี Google Drive เพื่อนำเข้าไฟล์เอกสารและใบแจ้งหนี้อัตโนมัติ",
      href: `${root}/connections`,
      icon: (
        <svg aria-hidden="true" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round">
          <path d="M10 13a5 5 0 0 0 7.54.54l2-2a5 5 0 0 0-7.07-7.07l-1.15 1.15M14 11a5 5 0 0 0-7.54-.54l-2 2a5 5 0 0 0 7.07 7.07l1.15-1.15" />
        </svg>
      ),
      badge: "Cloud Storage",
    },
  ];

  return (
    <section className="workspace-page mx-auto max-w-[1270px]">
      <div className="work-header">
        <div>
          <p className="eyebrow">การจัดการและตั้งค่า</p>
          <h1>ตั้งค่าระบบ</h1>
          <p className="intro">
            จัดการข้อมูลธุรกิจ ทีมงาน และการเชื่อมต่อช่องทางภายนอกสำหรับ {membership.organization_name}
          </p>
        </div>
        <span className="status-pill">{membership.role}</span>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-4 mt-6">
        {settingCards.map((card) => (
          <Link
            key={card.href}
            href={card.href}
            className="card group hover:border-[var(--pf-primary-400)] transition-all duration-200 flex flex-col justify-between"
          >
            <div>
              <div className="flex items-center justify-between mb-3">
                <span className="document-icon-circle w-11 h-11 bg-[var(--pf-primary-50)] text-[var(--pf-primary-600)]">
                  {card.icon}
                </span>
                <span className="text-xs px-2.5 py-1 rounded-full bg-[var(--pf-bg-soft)] text-[var(--pf-text-secondary)] font-semibold">
                  {card.badge}
                </span>
              </div>
              <h2 className="text-lg font-bold text-[var(--pf-text-primary)] group-hover:text-[var(--pf-primary-600)] transition-colors">
                {card.title}
              </h2>
              <p className="text-sm text-[var(--pf-text-secondary)] mt-1.5 leading-relaxed">
                {card.description}
              </p>
            </div>
            <div className="mt-4 pt-3 border-t border-[var(--pf-border-soft)] flex items-center justify-between text-sm font-semibold text-[var(--pf-primary-600)]">
              <span>เปิดจัดการ</span>
              <span className="group-hover:translate-x-1 transition-transform">→</span>
            </div>
          </Link>
        ))}
      </div>
    </section>
  );
}
