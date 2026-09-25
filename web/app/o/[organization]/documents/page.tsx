import { Card } from "@/components/ui/card";
import { sessionGET, sessionPOST } from "@/lib/session-api";
import Link from "next/link";
import { redirect } from "next/navigation";
import { UploadForm } from "./upload-form";
import { DocumentIcon, DocumentSummary } from "./document-visuals";
import { SearchShortcut } from "./search-shortcut";
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
  searchParams: Promise<{ cursor?: string; status?: string; source?: string; filename?: string; from?: string; to?: string; view?: string; updated?: string; error?: string }>;
}) {
  const { organization } = await params;
  const { cursor, status, source, filename, from, to, view, updated, error } = await searchParams;
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
  const root = `/o/${encodeURIComponent(organization)}`;
  const currentView = view === "compact" ? "compact" : "cards";
  const tabHref = (nextStatus: string) => {
    const query = new URLSearchParams(filters);
    if (nextStatus) query.set("status", nextStatus); else query.delete("status");
    if (currentView === "compact") query.set("view", currentView);
    return `${root}/documents${query.size ? `?${query}` : ""}`;
  };
  const viewHref = (nextView: string) => {
    const query = new URLSearchParams(filters);
    if (nextView === "compact") query.set("view", nextView);
    return `${root}/documents${query.size ? `?${query}` : ""}`;
  };
  const tabs = [["", "ทั้งหมด"], ["Available", "พร้อมใช้"], ["Archived", "เก็บถาวร"], ["Trash", "ถังขยะ"], ["Checking", "กำลังตรวจ"]];
  return <section className="documents-inbox mx-auto max-w-[1270px]">
    <SearchShortcut />
    <div className="documents-search-row"><form className="documents-search" method="get" role="search">
      <DocumentIcon kind="search" /><input id="document-search" name="filename" aria-label="ค้นหาชื่อไฟล์เอกสาร" placeholder="ค้นหาชื่อไฟล์เอกสาร..." defaultValue={filename} />
      {status && <input type="hidden" name="status" value={status} />}{source && <input type="hidden" name="source" value={source} />}{from && <input type="hidden" name="from" value={from} />}{to && <input type="hidden" name="to" value={to} />}{currentView === "compact" && <input type="hidden" name="view" value="compact" />}
      <kbd>Ctrl K</kbd>
    </form><a className="documents-upload-action" href="#document-upload"><DocumentIcon kind="upload" />อัปโหลดเอกสาร</a></div>
    <header className="documents-toolbar"><div className="documents-title"><span className="document-icon-circle"><DocumentIcon kind="file" /></span><div><h1>เอกสารของทีม</h1><p>จัดเก็บ ค้นหา และจัดการเอกสารของทีมง่ายๆ ในที่เดียว</p></div></div>
      <nav className="documents-toolbar-actions" aria-label="เครื่องมือเอกสาร">
        <details className="documents-menu"><summary><DocumentIcon kind="layers" />มุมมอง</summary><div><Link href={viewHref("cards")} aria-current={currentView === "cards" ? "page" : undefined}>การ์ด</Link><Link href={viewHref("compact")} aria-current={currentView === "compact" ? "page" : undefined}>รายการย่อ</Link></div></details>
        {canManage && <details className="documents-menu"><summary><DocumentIcon kind="settings" />ตั้งค่า</summary><div><Link href={`${root}/connections`}>การเชื่อมต่อ</Link><Link href={`${root}/business`}>ข้อมูลธุรกิจ</Link></div></details>}
        {canManage && <details className="documents-menu"><summary><DocumentIcon kind="download" />ส่งออก</summary><div>{reviewEnabled && <Link href={`${root}/exports`}>เปิดหน้าส่งออก</Link>}{!reviewEnabled && exportCount !== null && exportCount <= 5000 && <><a href={`/api${root}/documents/export.csv${filterQuery ? `?${filterQuery}` : ""}`}>CSV</a><a href={`/api${root}/documents/export.xlsx${filterQuery ? `?${filterQuery}` : ""}`}>XLSX</a></>}{!reviewEnabled && exportCount !== null && exportCount > 5000 && <span>กรองให้เหลือไม่เกิน 5,000 รายการ</span>}{extractionEnabled && <><a href={`/api${root}/documents/extraction.csv`}>ข้อมูลยืนยันแล้ว CSV</a><a href={`/api${root}/documents/extraction.xlsx`}>ข้อมูลยืนยันแล้ว XLSX</a></>}{accountingEnabled && <><a href={`/api${root}/documents/accounting.csv`}>ข้อเสนอที่อนุมัติแล้ว CSV</a><a href={`/api${root}/documents/accounting.xlsx`}>ข้อเสนอที่อนุมัติแล้ว XLSX</a></>}</div></details>}
      </nav>
    </header>
    {(reviewEnabled || canManage && accountingEnabled) && <div className="documents-work-links">{reviewEnabled && <Link href={`${root}/review`}>เปิดคิวตรวจเอกสาร</Link>}{canManage && reviewEnabled && <Link href={`${root}/exports`}>ส่งออกข้อมูล</Link>}{canManage && accountingEnabled && <Link href={`${root}/accounting`}>ข้อเสนอที่รอตรวจ</Link>}</div>}
    <nav className="documents-tabs" aria-label="สถานะเอกสาร">{tabs.map(([value, label]) => <Link key={label} href={tabHref(value)} aria-current={(status || "") === value ? "page" : undefined}>{label}</Link>)}<span>รายการหน้านี้ <b>{page.documents.length}</b> รายการ</span></nav>
    <form className="documents-filters" method="get">
      {filename && <input type="hidden" name="filename" value={filename} />}{currentView === "compact" && <input type="hidden" name="view" value="compact" />}
      <label>ช่วงวันที่<div className="documents-date-range"><DocumentIcon kind="calendar" /><input type="date" name="from" defaultValue={from} aria-label="ตั้งแต่วันที่" /><span>–</span><input type="date" name="to" defaultValue={to} aria-label="ถึงวันที่" /></div></label>
      <label>ช่องทาง<select name="source" defaultValue={source || ""}><option value="">ทั้งหมด</option><option value="Web">Web</option><option value="LINE">LINE</option><option value="Drive">Drive</option></select></label>
      <label>สถานะ<select name="status" defaultValue={status || ""}><option value="">ทั้งหมด</option><option value="Available">พร้อมใช้</option><option value="Archived">เก็บถาวร</option><option value="Trash">ถังขยะ</option><option value="Checking">กำลังตรวจ</option><option value="Rejected">ปฏิเสธ</option></select></label>
      <Link className="documents-reset" href={`${root}/documents`}>รีเซ็ตตัวกรอง</Link><button type="submit"><DocumentIcon kind="search" />ค้นหา</button>
    </form>
    {updated && <p className="success-message" role="status">อัปเดตสถานะเอกสารแล้ว</p>}
    {error && <p className="form-error" role="alert">เปลี่ยนสถานะไม่ได้ กรุณารีเฟรชรายการแล้วลองอีกครั้ง</p>}
    {summary && <details className="documents-usage"><summary>การใช้เอกสารของทีม {canManage && summary.warning && <span role="status">· ใกล้เต็ม ({summary.warning}%)</span>}</summary><div>{canManage && <span>ใช้ไป {summary.used}/{summary.limit} เอกสาร</span>}<span>พร้อมใช้ {summary.available}</span><span>กำลังตรวจ {summary.checking}</span><span>ปฏิเสธ {summary.rejected}</span>{canManage && <span>รีเซ็ต {new Intl.DateTimeFormat("th-TH", { dateStyle: "medium" }).format(new Date(summary.reset_at))}</span>}</div></details>}
    <div className={`work-layout document-layout ${currentView === "compact" ? "is-compact" : ""}`}>
      <div className="work-stack"><section className="rounded-[1.2rem] border border-line bg-surface p-5 shadow-panel" aria-labelledby="documents-heading"><h2 id="documents-heading" className="document-panel-heading"><span className="document-icon-circle"><DocumentIcon kind="file" /></span>เอกสารที่อัปโหลด</h2>
        {page.documents.length === 0 ? <p className="empty-state mt-4">ยังไม่มีเอกสารที่คุณเข้าถึงได้</p> : <div className="task-list">{page.documents.map((doc) => <div key={doc.id} className="notification-card">
          <Link href={`/o/${encodeURIComponent(organization)}/documents/${encodeURIComponent(doc.id)}`}><DocumentSummary filename={doc.filename} size={doc.size} status={doc.status} /></Link>
          <div className="card-actions document-actions">{doc.status !== "Trash" && <a className="secondary-button" href={`/api/o/${encodeURIComponent(organization)}/documents/${encodeURIComponent(doc.id)}/original`}><DocumentIcon kind="download" />ดาวน์โหลด</a>}{canManage && doc.status !== "Trash" && <form action={changeStatus.bind(null, organization, doc.id, "trash")}><button className="danger-button document-trash-button" type="submit"><DocumentIcon kind="trash" />ย้ายไปถังขยะ</button></form>}{canManage && doc.status === "Trash" && <form action={changeStatus.bind(null, organization, doc.id, "restore")}><button className="secondary-button" type="submit">กู้คืน</button></form>}</div>
          {canManage && doc.status !== "Trash" && <p className="document-trash-help">กู้คืนได้ภายใน 30 วันหลังย้าย</p>}
        </div>)}</div>}
        {page.next_cursor && <Link className="secondary-button" href={`/o/${encodeURIComponent(organization)}/documents?${new URLSearchParams({ ...(status ? { status } : {}), ...(source ? { source } : {}), ...(filename ? { filename } : {}), ...(from ? { from } : {}), ...(to ? { to } : {}), ...(currentView === "compact" ? { view: "compact" } : {}), cursor: page.next_cursor }).toString()}`}>ดูรายการถัดไป</Link>}
      </section></div>
      <div className="work-stack"><section id="document-upload" className="rounded-[1.2rem] border border-line bg-surface p-5 shadow-panel"><div className="document-panel-heading"><span className="document-icon-circle"><DocumentIcon kind="upload" /></span><div><h2>อัปโหลดเอกสารใหม่</h2><p>ลากไฟล์มาวาง หรือเลือกไฟล์จากอุปกรณ์ของคุณ</p></div></div><UploadForm organization={organization} /></section>{canManage && <Card className="work-panel rounded-[1.2rem] border border-line bg-surface p-5 shadow-panel"><h2>นำเข้าจาก Google Drive</h2><DriveImportForm organization={organization} /></Card>}</div>
    </div>
  </section>;
}
