import { Card } from "@/components/ui/card";
import { sessionGET } from "@/lib/session-api";
import Link from "next/link";
import { redirect } from "next/navigation";
import { UploadForm } from "./upload-form";
import { DriveImportForm } from "./drive-import-form";

type Document = { id: string; filename: string; mime: string; size: number; status: string; accepted_at: string; source_count: number; source_channel?: string };
type Page = { documents: Document[]; next_cursor?: string };
type Summary = { used: number; limit: number; reset_at: string; available: number; archived: number; trash: number; checking: number; rejected: number; warning?: string };

export default async function DocumentsPage({ params, searchParams }: {
  params: Promise<{ organization: string }>;
  searchParams: Promise<{ cursor?: string; status?: string; source?: string; filename?: string; from?: string; to?: string }>;
}) {
  const { organization } = await params;
  const { cursor, status, source, filename, from, to } = await searchParams;
  const filters = new URLSearchParams();
  for (const [key, value] of Object.entries({ status, source, filename, from, to })) if (value) filters.set(key, value);
  const filterQuery = filters.toString();
  const listQuery = new URLSearchParams(filters);
  listQuery.set("limit", "20");
  if (cursor) listQuery.set("cursor", cursor);
  const base = `/v1/o/${encodeURIComponent(organization)}/documents`;
  const [response, summaryResponse] = await Promise.all([sessionGET(`${base}?${listQuery.toString()}`), sessionGET(`${base}/summary`)]);
  if (response?.status === 401) redirect("/");
  if (!response?.ok) return <section className="error-state" role="alert"><h1>ยังเปิดเอกสารไม่ได้</h1><p>ตรวจสอบการเชื่อมต่อแล้วลองอีกครั้ง</p><Link href={`/o/${encodeURIComponent(organization)}/documents`}>ลองใหม่</Link></section>;
  const page = await response.json() as Page;
  const summary = summaryResponse?.ok ? await summaryResponse.json() as Summary : null;
  return <section className="org-documents-page">
    <div className="org-documents-toolbar"><span>PlaiFlow / เอกสาร</span><Link href={`/o/${encodeURIComponent(organization)}/tasks`}>กลับไปงานของทีม</Link></div>
    <div className="work-header"><div><p className="eyebrow">DOCUMENT INBOX</p><h1>เอกสารของทีม</h1><p className="intro">ส่งเอกสารและเปิดต้นฉบับที่คุณมีสิทธิ์ดู</p></div><div className="card-actions"><a className="secondary-button" href={`/api/o/${encodeURIComponent(organization)}/documents/export.csv${filterQuery ? `?${filterQuery}` : ""}`}>ส่งออก CSV</a><a className="secondary-button" href={`/api/o/${encodeURIComponent(organization)}/documents/export.xlsx${filterQuery ? `?${filterQuery}` : ""}`}>ส่งออก XLSX</a></div></div>
    {summary && <section className="org-documents-toolbar" aria-label="Document usage"><strong>ใช้ไป {summary.used}/{summary.limit} เอกสาร</strong>{summary.warning && <span role="status">ใกล้เต็ม ({summary.warning}%)</span>}<span>พร้อมใช้ {summary.available}</span><span>กำลังตรวจ {summary.checking}</span><span>ปฏิเสธ {summary.rejected}</span><span>รีเซ็ต {new Intl.DateTimeFormat("th-TH", { dateStyle: "medium" }).format(new Date(summary.reset_at))}</span></section>}
    <form className="org-documents-toolbar" method="get"><label>ค้นหาชื่อไฟล์ <input name="filename" defaultValue={filename} /></label><label>สถานะ <select name="status" defaultValue={status}><option value="">ทั้งหมด</option><option value="Available">Available</option><option value="Archived">Archived</option><option value="Trash">Trash</option></select></label><label>ช่องทาง <select name="source" defaultValue={source}><option value="">ทั้งหมด</option><option value="Web">Web</option><option value="LINE">LINE</option><option value="Drive">Drive</option></select></label><button className="secondary-button" type="submit">กรอง</button></form>
    <div className="work-layout">
      <div className="work-stack"><section className="org-documents-list" aria-labelledby="documents-heading"><div className="org-documents-section"><h2 id="documents-heading">รายการเอกสาร</h2></div>
        {page.documents.length === 0 ? <p className="empty-state">ยังไม่มีเอกสารที่คุณเข้าถึงได้</p> : <div className="task-list">{page.documents.map((doc) => <Card key={doc.id} className="notification-card"><div><strong>{doc.filename}</strong><p className="field-help">{doc.status} · {doc.source_channel || "ไม่ทราบช่องทาง"} · {new Intl.DateTimeFormat("th-TH", { dateStyle: "medium" }).format(new Date(doc.accepted_at))} · {Math.ceil(doc.size / 1024)} KB</p></div><a className="secondary-button" href={`/api/o/${encodeURIComponent(organization)}/documents/${encodeURIComponent(doc.id)}/original`}>ดาวน์โหลด</a></Card>)}</div>}
        {page.next_cursor && <Link className="secondary-button" href={`/o/${encodeURIComponent(organization)}/documents?${new URLSearchParams({ ...(status ? { status } : {}), ...(source ? { source } : {}), ...(filename ? { filename } : {}), ...(from ? { from } : {}), ...(to ? { to } : {}), cursor: page.next_cursor }).toString()}`}>ดูรายการถัดไป</Link>}
      </section></div>
      <div className="work-stack"><Card className="work-panel"><h2>ส่งเอกสารใหม่</h2><UploadForm organization={organization} /></Card><Card className="work-panel"><h2>นำเข้าจาก Google Drive</h2><DriveImportForm organization={organization} /></Card></div>
    </div>
  </section>;
}
