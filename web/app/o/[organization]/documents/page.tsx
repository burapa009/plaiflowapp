import { Card } from "@/components/ui/card";
import { sessionGET, sessionPOST } from "@/lib/session-api";
import Link from "next/link";
import { redirect } from "next/navigation";
import { UploadForm } from "./upload-form";
import { DriveImportForm } from "./drive-import-form";


type Document = { id: string; filename: string; mime: string; size: number; status: string; accepted_at: string; source_count: number; source_channel?: string };
type Page = { documents: Document[]; next_cursor?: string };
type Summary = { used: number; limit: number; reset_at: string; available: number; archived: number; trash: number; checking: number; rejected: number; warning?: string };
type Membership = { role: "Owner" | "Admin" | "Member" };

async function changeStatus(organization: string, documentID: string, action: "archive" | "trash" | "restore") {
  "use server";
  const destination = `/o/${encodeURIComponent(organization)}/documents`;
  const response = await sessionPOST(`/v1/o/${encodeURIComponent(organization)}/documents/${encodeURIComponent(documentID)}/${action}`, new URLSearchParams());
  if (response?.status === 401) redirect("/");
  redirect(`${destination}?${response?.ok ? "updated=1" : "error=status"}`);
}

export default async function DocumentsPage({ params, searchParams }: {
  params: Promise<{ organization: string }>;
  searchParams: Promise<{ cursor?: string; status?: string; source?: string; filename?: string; from?: string; to?: string; updated?: string; error?: string }>;
}) {
  const { organization } = await params;
  const { cursor, status, source, filename, from, to, updated, error } = await searchParams;
  const filters = new URLSearchParams();
  for (const [key, value] of Object.entries({ status, source, filename, from, to })) if (value) filters.set(key, value);
  const filterQuery = filters.toString();
  const listQuery = new URLSearchParams(filters);
  listQuery.set("limit", "20");
  if (cursor) listQuery.set("cursor", cursor);
  const base = `/v1/o/${encodeURIComponent(organization)}/documents`;
  const [response, summaryResponse, organizationResponse] = await Promise.all([sessionGET(`${base}?${listQuery.toString()}`), sessionGET(`${base}/summary`), sessionGET(`/v1/o/${encodeURIComponent(organization)}`)]);
  if (response?.status === 401 || organizationResponse?.status === 401) redirect("/");
  if (!response?.ok || !organizationResponse?.ok) return <section className="error-state" role="alert"><h1>ยังเปิดเอกสารไม่ได้</h1><p>ตรวจสอบการเชื่อมต่อแล้วลองอีกครั้ง</p><Link href={`/o/${encodeURIComponent(organization)}/documents`}>ลองใหม่</Link></section>;
  const page = await response.json() as Page;
  const summary = summaryResponse?.ok ? await summaryResponse.json() as Summary : null;
  const { membership } = await organizationResponse.json() as { membership: Membership };
  const canManage = membership.role === "Owner" || membership.role === "Admin";
  const countResponse = canManage ? await sessionGET(`${base}/export/count${filterQuery ? `?${filterQuery}` : ""}`) : null;
  const exportCount = countResponse?.ok ? (await countResponse.json() as { count: number }).count : null;
  return <section className="mx-auto max-w-[1270px]">
    <div className="mb-5 flex flex-wrap items-center justify-between gap-4 rounded-2xl border border-line bg-surface p-4 text-muted shadow-panel max-md:items-stretch sm:px-5 [&_a]:text-brand-strong"><span>PlaiFlow / เอกสาร</span><Link href={`/o/${encodeURIComponent(organization)}/tasks`}>กลับไปงานของทีม</Link></div>
    <div className="work-header"><div><p className="eyebrow">DOCUMENT INBOX</p><h1>เอกสารของทีม</h1><p className="intro">ส่งเอกสารและเปิดต้นฉบับที่คุณมีสิทธิ์ดู</p></div>{canManage && <div className="card-actions">{exportCount !== null && <span>ผลลัพธ์ที่กรอง {exportCount.toLocaleString("th-TH")} รายการ</span>}{exportCount !== null && exportCount <= 5000 && <><a className="secondary-button" href={`/api/o/${encodeURIComponent(organization)}/documents/export.csv${filterQuery ? `?${filterQuery}` : ""}`}>ส่งออก CSV</a><a className="secondary-button" href={`/api/o/${encodeURIComponent(organization)}/documents/export.xlsx${filterQuery ? `?${filterQuery}` : ""}`}>ส่งออก XLSX</a></>}{exportCount !== null && exportCount > 5000 && <span role="status">กรองให้เหลือไม่เกิน 5,000 รายการก่อนส่งออก</span>}</div>}</div>
    {updated && <p className="success-message" role="status">อัปเดตสถานะเอกสารแล้ว</p>}
    {error && <p className="form-error" role="alert">เปลี่ยนสถานะไม่ได้ กรุณารีเฟรชรายการแล้วลองอีกครั้ง</p>}
    {summary && <section className="mb-5 flex flex-wrap items-center justify-between gap-4 rounded-2xl border border-line bg-surface p-4 text-muted shadow-panel max-md:items-stretch sm:px-5 [&_a]:text-brand-strong" aria-label="Document usage">{canManage && <><strong>ใช้ไป {summary.used}/{summary.limit} เอกสาร</strong>{summary.warning && <span role="status">ใกล้เต็ม ({summary.warning}%)</span>}</>}<span>พร้อมใช้ {summary.available}</span><span>กำลังตรวจ {summary.checking}</span><span>ปฏิเสธ {summary.rejected}</span>{canManage && <span>รีเซ็ต {new Intl.DateTimeFormat("th-TH", { dateStyle: "medium" }).format(new Date(summary.reset_at))}</span>}</section>}
    <form className="mb-5 flex flex-wrap items-center justify-between gap-4 rounded-2xl border border-line bg-surface p-4 text-muted shadow-panel max-md:items-stretch sm:px-5 [&_a]:text-brand-strong [&_label]:flex [&_label]:flex-wrap [&_label]:items-center [&_label]:gap-2 [&_label]:font-semibold [&_label]:text-fg max-md:[&_label]:w-full [&_input]:min-h-11 [&_input]:rounded-xl [&_input]:border [&_input]:border-line [&_input]:bg-surface [&_input]:px-3 [&_input]:py-2 [&_input]:text-fg max-md:[&_input]:w-full [&_select]:min-h-11 [&_select]:rounded-xl [&_select]:border [&_select]:border-line [&_select]:bg-surface [&_select]:px-3 [&_select]:py-2 [&_select]:text-fg max-md:[&_select]:w-full" method="get"><label>ค้นหาชื่อไฟล์ <input name="filename" defaultValue={filename} /></label><label>สถานะ <select name="status" defaultValue={status}><option value="">ทั้งหมด</option><option value="Available">Available</option><option value="Archived">Archived</option><option value="Trash">Trash</option></select></label><label>ช่องทาง <select name="source" defaultValue={source}><option value="">ทั้งหมด</option><option value="Web">Web</option><option value="LINE">LINE</option><option value="Drive">Drive</option></select></label><button className="secondary-button" type="submit">กรอง</button></form>
    <div className="work-layout">
      <div className="work-stack"><section className="rounded-[1.2rem] border border-line bg-surface p-5 shadow-panel" aria-labelledby="documents-heading"><div className="flex items-baseline justify-between gap-4 [&_h2]:m-0"><h2 id="documents-heading">รายการเอกสาร</h2></div><p className="mt-2 text-sm text-muted">ต้องการเก็บใน Google Drive? กดดาวน์โหลด แล้วอัปโหลดเข้า Drive ด้วยตนเอง</p>
        {page.documents.length === 0 ? <p className="empty-state">ยังไม่มีเอกสารที่คุณเข้าถึงได้</p> : <div className="task-list">{page.documents.map((doc) => <Card key={doc.id} className="notification-card"><div><strong><Link href={`/o/${encodeURIComponent(organization)}/documents/${encodeURIComponent(doc.id)}`}>{doc.filename}</Link></strong><p className="field-help">{doc.status} · {doc.source_channel || "ไม่ทราบช่องทาง"} · {new Intl.DateTimeFormat("th-TH", { dateStyle: "medium" }).format(new Date(doc.accepted_at))} · {Math.ceil(doc.size / 1024)} KB</p></div><div className="card-actions">{doc.status !== "Trash" && <a className="secondary-button" href={`/api/o/${encodeURIComponent(organization)}/documents/${encodeURIComponent(doc.id)}/original`}>ดาวน์โหลด</a>}{canManage && doc.status === "Available" && <form action={changeStatus.bind(null, organization, doc.id, "archive")}><button className="secondary-button" type="submit">เก็บถาวร</button></form>}{canManage && doc.status !== "Trash" && <form action={changeStatus.bind(null, organization, doc.id, "trash")}><button className="secondary-button" type="submit">ย้ายไปถังขยะ</button></form>}{canManage && doc.status === "Trash" && <form action={changeStatus.bind(null, organization, doc.id, "restore")}><button className="secondary-button" type="submit">กู้คืน</button></form>}</div></Card>)}</div>}
        {page.next_cursor && <Link className="secondary-button" href={`/o/${encodeURIComponent(organization)}/documents?${new URLSearchParams({ ...(status ? { status } : {}), ...(source ? { source } : {}), ...(filename ? { filename } : {}), ...(from ? { from } : {}), ...(to ? { to } : {}), cursor: page.next_cursor }).toString()}`}>ดูรายการถัดไป</Link>}
      </section></div>
      <div className="work-stack"><Card className="work-panel rounded-[1.2rem] border border-line bg-surface p-5 shadow-panel"><h2>ส่งเอกสารใหม่</h2><UploadForm organization={organization} /></Card>{canManage && <Card className="work-panel rounded-[1.2rem] border border-line bg-surface p-5 shadow-panel"><h2>นำเข้าจาก Google Drive</h2><DriveImportForm organization={organization} /></Card>}</div>
    </div>
  </section>;
}
