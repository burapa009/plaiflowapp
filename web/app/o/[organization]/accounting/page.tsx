import { sessionGET } from "@/lib/session-api";
import Link from "next/link";
import { redirect } from "next/navigation";

type Item = { document_id: string; suggestion_basis: string; warning_codes: string[] };

export default async function AccountingReviewQueue({ params, searchParams }: { params: Promise<{ organization: string }>; searchParams: Promise<{ offset?: string }> }) {
  const { organization } = await params;
  const { offset: requestedOffset } = await searchParams;
  const offset = /^\d{1,5}$/.test(requestedOffset ?? "") ? Number(requestedOffset) : 0;
  const base = `/o/${encodeURIComponent(organization)}`;
  const response = await sessionGET(`/v1${base}/accounting/review-queue?offset=${offset}`);
  if (response?.status === 401) redirect("/");
  if (!response?.ok) return <section className="error-state" role="alert"><h1>ยังเปิดรายการรอตรวจไม่ได้</h1><Link href={`${base}/documents`}>กลับไปเอกสาร</Link></section>;
  const { items, limit, has_more } = await response.json() as { items: Item[]; limit: number; has_more: boolean };
  return <section className="mx-auto max-w-[1100px]">
    <Link href={`${base}/documents`}>← กลับไปเอกสาร</Link>
    <header className="work-header"><div><p className="eyebrow">ACCOUNTING REVIEW</p><h1>ข้อเสนอที่รอตรวจ</h1><p className="intro">รายการล่าสุด {limit} เอกสาร · ไม่มีการลงบัญชีอัตโนมัติ</p></div></header>
    <div className="rounded-[1.2rem] border border-line bg-surface p-5 shadow-panel">
      {items.length === 0 ? <p className="text-muted">ไม่มีข้อเสนอที่ต้องตรวจในรายการล่าสุด</p> : <ul className="grid gap-3">{items.map((item) => <li key={item.document_id} className="rounded-xl border border-line p-4">
        <Link className="font-semibold text-brand-strong" href={`${base}/documents/${encodeURIComponent(item.document_id)}`}>เปิดเอกสารเพื่อตรวจและอนุมัติ</Link>
        <p className="mt-1 text-sm text-muted">ที่มา: {item.suggestion_basis} · คำเตือน: {item.warning_codes.join(", ") || "ยังไม่ได้อนุมัติ"}</p>
      </li>)}</ul>}
    </div>
    <nav className="mt-4 flex gap-3" aria-label="หน้ารายการรอตรวจ">{offset > 0 && <Link className="secondary-button" href={`${base}/accounting?offset=${Math.max(0, offset - limit)}`}>ก่อนหน้า</Link>}{has_more && <Link className="secondary-button" href={`${base}/accounting?offset=${offset + limit}`}>ถัดไป</Link>}</nav>
  </section>;
}
