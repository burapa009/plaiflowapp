import { Card } from "@/components/ui/card";
import { sessionGET, sessionPOST } from "@/lib/session-api";
import Link from "next/link";
import { redirect } from "next/navigation";
import { UploadForm } from "./upload-form";
import { DocumentIcon, DocumentSummary } from "./document-visuals";
import { DriveImportForm } from "./drive-import-form";


type Document = { id: string; filename: string; mime: string; size: number; status: string; accepted_at: string; source_count: number; source_channel?: string };
type Page = { documents: Document[]; next_cursor?: string };
type Summary = { used: number; limit: number; reset_at: string; available: number; archived: number; trash: number; checking: number; rejected: number; warning?: string };
type Membership = { role: "Owner" | "Admin" | "Member" };

async function changeStatus(organization: string, documentID: string, action: "trash" | "restore") {
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
  const extractionEnabled = process.env.EXTRACTION_ENABLED === "true";
  const accountingEnabled = process.env.ACCOUNTING_ENABLED === "true";
  const reviewEnabled = process.env.REVIEW_ENABLED === "true";
  const countResponse = canManage && !reviewEnabled ? await sessionGET(`${base}/export/count${filterQuery ? `?${filterQuery}` : ""}`) : null;
  const exportCount = countResponse?.ok ? (await countResponse.json() as { count: number }).count : null;
  return <section className="mx-auto max-w-[1270px]">
    {reviewEnabled && <div className="mb-5 flex flex-wrap gap-3"><Link className="secondary-button" href={`/o/${encodeURIComponent(organization)}/review`}>เปิดคิวตรวจเอกสาร</Link>{canManage && <Link className="secondary-button" href={`/o/${encodeURIComponent(organization)}/exports`}>ส่งออกข้อมูล</Link>}</div>}
    <div className="mb-5 flex flex-wrap items-center justify-between gap-4 rounded-2xl border border-line bg-surface p-4 text-muted shadow-panel max-md:items-stretch sm:px-5 [&_a]:text-brand-strong"><span>PlaiFlow / เอกสาร</span>{canManage && accountingEnabled && <Link href={`/o/${encodeURIComponent(organization)}/accounting`}>ข้อเสนอที่รอตรวจ</Link>}<Link href={`/o/${encodeURIComponent(organization)}/tasks`}>กลับไปงานของทีม</Link></div>
    <div className="work-header"><div><p className="eyebrow">DOCUMENT INBOX</p><h1>เอกสารของทีม</h1><p className="intro">ส่งเอกสารและเปิดต้นฉบับที่คุณมีสิทธิ์ดู</p></div>{canManage && !reviewEnabled && <div className="card-actions">{exportCount !== null && <span>ผลลัพธ์ที่กรอง {exportCount.toLocaleString("th-TH")} รายการ</span>}{exportCount !== null && exportCount <= 5000 && <><a className="secondary-button" href={`/api/o/${encodeURIComponent(organization)}/documents/export.csv${filterQuery ? `?${filterQuery}` : ""}`}>ส่งออก CSV</a><a className="secondary-button" href={`/api/o/${encodeURIComponent(organization)}/documents/export.xlsx${filterQuery ? `?${filterQuery}` : ""}`}>ส่งออก XLSX</a></>}{exportCount !== null && exportCount > 5000 && <span role="status">กรองให้เหลือไม่เกิน 5,000 รายการก่อนส่งออก</span>}{extractionEnabled && <><a className="secondary-button" href={`/api/o/${encodeURIComponent(organization)}/documents/extraction.csv`}>ข้อมูลยืนยันแล้ว CSV</a><a className="secondary-button" href={`/api/o/${encodeURIComponent(organization)}/documents/extraction.xlsx`}>ข้อมูลยืนยันแล้ว XLSX</a></>}{accountingEnabled && <><a className="secondary-button" href={`/api/o/${encodeURIComponent(organization)}/documents/accounting.csv`}>ข้อเสนอที่อนุมัติแล้ว CSV</a><a className="secondary-button" href={`/api/o/${encodeURIComponent(organization)}/documents/accounting.xlsx`}>ข้อเสนอที่อนุมัติแล้ว XLSX</a></>}</div>}</div>
    {updated && <p className="success-message" role="status">อัปเดตสถานะเอกสารแล้ว</p>}
    {error && <p className="form-error" role="alert">เปลี่ยนสถานะไม่ได้ กรุณารีเฟรชรายการแล้วลองอีกครั้ง</p>}
    {summary && <section className="mb-5 flex flex-wrap items-center justify-between gap-4 rounded-2xl border border-line bg-surface p-4 text-muted shadow-panel max-md:items-stretch sm:px-5 [&_a]:text-brand-strong" aria-label="Document usage">{canManage && <><strong>ใช้ไป {summary.used}/{summary.limit} เอกสาร</strong>{summary.warning && <span role="status">ใกล้เต็ม ({summary.warning}%)</span>}</>}<span>พร้อมใช้ {summary.available}</span><span>กำลังตรวจ {summary.checking}</span><span>ปฏิเสธ {summary.rejected}</span>{canManage && <span>รีเซ็ต {new Intl.DateTimeFormat("th-TH", { dateStyle: "medium" }).format(new Date(summary.reset_at))}</span>}</section>}
    <form className="mb-5 flex flex-wrap items-center justify-between gap-4 rounded-2xl border border-line bg-surface p-4 text-muted shadow-panel max-md:items-stretch sm:px-5 [&_a]:text-brand-strong [&_label]:flex [&_label]:flex-wrap [&_label]:items-center [&_label]:gap-2 [&_label]:font-semibold [&_label]:text-fg max-md:[&_label]:w-full [&_input]:min-h-11 [&_input]:rounded-xl [&_input]:border [&_input]:border-line [&_input]:bg-surface [&_input]:px-3 [&_input]:py-2 [&_input]:text-fg max-md:[&_input]:w-full [&_select]:min-h-11 [&_select]:rounded-xl [&_select]:border [&_select]:border-line [&_select]:bg-surface [&_select]:px-3 [&_select]:py-2 [&_select]:text-fg max-md:[&_select]:w-full" method="get"><label>ค้นหาชื่อไฟล์ <input name="filename" defaultValue={filename} /></label><label>สถานะ <select name="status" defaultValue={status}><option value="">ทั้งหมด</option><option value="Available">Available</option><option value="Archived">Archived</option><option value="Trash">Trash</option></select></label><label>ช่องทาง <select name="source" defaultValue={source}><option value="">ทั้งหมด</option><option value="Web">Web</option><option value="LINE">LINE</option><option value="Drive">Drive</option></select></label><button className="secondary-button" type="submit">กรอง</button></form>
    <div className="work-layout document-layout">
      <div className="work-stack"><section className="rounded-[1.2rem] border border-line bg-surface p-5 shadow-panel" aria-labelledby="documents-heading"><h2 id="documents-heading" className="document-panel-heading"><span className="document-icon-circle"><DocumentIcon kind="file" /></span>เอกสารที่อัปโหลด</h2>
        {page.documents.length === 0 ? <p className="empty-state mt-4">ยังไม่มีเอกสารที่คุณเข้าถึงได้</p> : <div className="task-list">{page.documents.map((doc) => <div key={doc.id} className="notification-card">
          <Link href={`/o/${encodeURIComponent(organization)}/documents/${encodeURIComponent(doc.id)}`}><DocumentSummary filename={doc.filename} size={doc.size} status={doc.status} /></Link>
          <div className="card-actions document-actions">{doc.status !== "Trash" && <a className="secondary-button" href={`/api/o/${encodeURIComponent(organization)}/documents/${encodeURIComponent(doc.id)}/original`}><DocumentIcon kind="download" />ดาวน์โหลด</a>}{canManage && doc.status !== "Trash" && <form action={changeStatus.bind(null, organization, doc.id, "trash")}><button className="danger-button document-trash-button" type="submit"><DocumentIcon kind="trash" />ย้ายไปถังขยะ</button></form>}{canManage && doc.status === "Trash" && <form action={changeStatus.bind(null, organization, doc.id, "restore")}><button className="secondary-button" type="submit">กู้คืน</button></form>}</div>
          {canManage && doc.status !== "Trash" && <p className="document-trash-help">กู้คืนได้ภายใน 30 วันหลังย้าย</p>}
        </div>)}</div>}
        {page.next_cursor && <Link className="secondary-button" href={`/o/${encodeURIComponent(organization)}/documents?${new URLSearchParams({ ...(status ? { status } : {}), ...(source ? { source } : {}), ...(filename ? { filename } : {}), ...(from ? { from } : {}), ...(to ? { to } : {}), cursor: page.next_cursor }).toString()}`}>ดูรายการถัดไป</Link>}
      </section></div>
      <div className="work-stack"><section className="rounded-[1.2rem] border border-line bg-surface p-5 shadow-panel"><div className="document-panel-heading"><span className="document-icon-circle"><DocumentIcon kind="upload" /></span><div><h2>อัปโหลดเอกสารใหม่</h2><p>ลากไฟล์มาวาง หรือเลือกไฟล์จากอุปกรณ์ของคุณ</p></div></div><UploadForm organization={organization} /></section>{canManage && <Card className="work-panel rounded-[1.2rem] border border-line bg-surface p-5 shadow-panel"><h2>นำเข้าจาก Google Drive</h2><DriveImportForm organization={organization} /></Card>}</div>
    </div>
  </section>;
}
