import { sessionGET, sessionPOST } from "@/lib/session-api";
import Link from "next/link";
import { redirect } from "next/navigation";

type Item = { document_id: string; status: string; accepted_at: string; ocr_job_id: string;
  review_revision: number; draft_revision: number; task_id?: string; assignee_id?: string };
type Page = { items: Item[]; next_cursor?: string };
type Member = { user_id: string; role: string; display_name?: string };

export default async function ReviewQueue({ params, searchParams }: {
  params: Promise<{ organization: string }>;
  searchParams: Promise<{ cursor?: string; view?: string; status?: string; result?: string }>;
}) {
  const { organization } = await params;
  const { cursor, view, status, result } = await searchParams;
  const root = `/o/${encodeURIComponent(organization)}`;
  const filters = new URLSearchParams({ view: view === "mine" || view === "unassigned" ? view : "all",
    status: status === "Archived" ? "Archived" : "Available", limit: "20" });
  if (cursor) filters.set("cursor", cursor);
  const [response, organizationResponse, membersResponse] = await Promise.all([
    sessionGET(`/v1${root}/review-queue?${filters}`), sessionGET(`/v1${root}`), sessionGET(`/v1${root}/memberships`),
  ]);
  if (response?.status === 401) redirect("/");
  if (!response?.ok) return <section className="error-state" role="alert"><h1>เปิดคิวตรวจเอกสารไม่ได้</h1><Link href={`${root}/documents`}>กลับไปรายการเอกสาร</Link></section>;
  const page = await response.json() as Page;
  const membership = organizationResponse?.ok ? await organizationResponse.json() as { membership: { role: string } } : null;
  const canAssign = membership?.membership.role === "Owner" || membership?.membership.role === "Admin";
  const members = membersResponse?.ok ? ((await membersResponse.json() as { memberships: Member[] }).memberships ?? []) : [];

  async function assign(formData: FormData) {
    "use server";
    const selected = formData.getAll("selected").map(String);
    const body = new URLSearchParams({ assignee_user_id: String(formData.get("assignee_user_id") ?? "") });
    for (const id of selected) {
      body.append("document_id", id);
      body.append("ocr_job_id", String(formData.get(`ocr_${id}`) ?? ""));
      body.append("review_revision", String(formData.get(`review_${id}`) ?? ""));
      body.append("draft_revision", String(formData.get(`draft_${id}`) ?? ""));
    }
    const saved = await sessionPOST(`/v1${root}/review-queue/assign`, body);
    redirect(`${root}/review?result=${saved?.ok ? "assigned" : "changed"}`);
  }

  return <section className="mx-auto max-w-[1100px]">
    <Link className="secondary-button mb-5" href={`${root}/documents`}>← กลับไปรายการเอกสาร</Link>
    <header className="work-header"><div><p className="eyebrow">DOCUMENT REVIEW</p><h1>คิวตรวจเอกสาร</h1><p className="intro">ตรวจต้นฉบับ แก้ข้อมูล และอนุมัติก่อนส่งออก</p></div></header>
    {result && <p role="status" className={result === "assigned" || result === "complete" ? "success-message" : "form-error"}>{result === "assigned" ? "มอบหมายงานแล้ว" : result === "complete" ? "ตรวจครบรายการในคิวนี้แล้ว" : "รายการเปลี่ยนไปแล้ว กรุณาโหลดใหม่"}</p>}
    <form method="get" className="mb-5 flex flex-wrap gap-3 rounded-2xl border border-line bg-surface p-4">
      <label className="grid gap-1">ผู้รับผิดชอบ<select name="view" defaultValue={filters.get("view") ?? "all"} className="min-h-11 rounded-xl border border-line px-3"><option value="all">ทั้งหมด</option><option value="mine">ของฉัน</option><option value="unassigned">ยังไม่มอบหมาย</option></select></label>
      <label className="grid gap-1">สถานะเอกสาร<select name="status" defaultValue={filters.get("status") ?? "Available"} className="min-h-11 rounded-xl border border-line px-3"><option value="Available">พร้อมใช้งาน</option><option value="Archived">เก็บถาวร</option></select></label>
      <button className="secondary-button self-end" type="submit">กรอง</button>
    </form>
    <form action={assign} className="rounded-[1.2rem] border border-line bg-surface p-5 shadow-panel">
      {page.items.length === 0 ? <p className="text-muted">ไม่มีเอกสารที่รอตรวจในหน้านี้</p> : <ul className="grid gap-3">{page.items.map((item, index) => <li key={item.document_id} className="rounded-xl border border-line p-4 hover:border-brand-strong">
        <div className="flex items-start gap-3">{canAssign && <input type="checkbox" name="selected" value={item.document_id} aria-label={`เลือกเอกสาร ${item.document_id}`} className="mt-1 size-5 accent-brand-strong" />}
          <div><Link className="font-semibold text-brand-strong underline-offset-2 hover:underline" href={`${root}/documents/${encodeURIComponent(item.document_id)}?${new URLSearchParams({
            queue_cursor: Buffer.from(JSON.stringify({ at: item.accepted_at, id: item.document_id })).toString("base64url"),
            queue_view: filters.get("view") ?? "all", queue_status: filters.get("status") ?? "Available",
            ...(index > 0 ? { previous: page.items[index-1].document_id } : {}),
          })}`}>เปิดเอกสารเพื่อตรวจ</Link>
            <p className="mt-1 text-sm text-muted">รับเมื่อ {new Intl.DateTimeFormat("th-TH", { dateStyle: "medium", timeStyle: "short" }).format(new Date(item.accepted_at))} · {item.review_revision ? "ยืนยันข้อมูลแล้ว รออนุมัติ" : "รอยืนยันข้อมูล"} · {item.assignee_id ? `มอบหมาย ${item.assignee_id.slice(0, 8)}` : "ยังไม่มอบหมาย"}</p>
          </div></div>
        <input type="hidden" name={`ocr_${item.document_id}`} value={item.ocr_job_id} /><input type="hidden" name={`review_${item.document_id}`} value={item.review_revision} /><input type="hidden" name={`draft_${item.document_id}`} value={item.draft_revision} />
      </li>)}</ul>}
      {canAssign && page.items.length > 0 && <div className="mt-5 flex flex-wrap items-end gap-3 border-t border-line pt-4"><label className="grid gap-1">มอบหมายรายการที่เลือก<select name="assignee_user_id" className="min-h-11 rounded-xl border border-line px-3"><option value="">ยกเลิกการมอบหมาย</option>{members.map((member) => <option key={member.user_id} value={member.user_id}>{member.display_name || member.user_id.slice(0, 8)} · {member.role}</option>)}</select></label><button type="submit" className="button">บันทึกการมอบหมาย</button></div>}
    </form>
    {page.next_cursor && <Link className="secondary-button mt-4" href={`${root}/review?${new URLSearchParams({ view: filters.get("view") ?? "all", status: filters.get("status") ?? "Available", cursor: page.next_cursor })}`}>หน้าถัดไป →</Link>}
  </section>;
}
