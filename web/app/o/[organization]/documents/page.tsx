import { sessionGET, sessionPOST } from "@/lib/session-api";
import Link from "next/link";
import { redirect } from "next/navigation";
import { UploadWorkspace } from "./upload-workspace";
import { DocumentIcon } from "./document-visuals";
import { SearchShortcut } from "./search-shortcut";
import WorkflowInbox from "./workflow-inbox";
import { workflowDocument } from "./workflow-data";
import type { InboxDocument } from "./document-table";
import "./inbox.css";
import DocumentStatusRefresh from "./document-status-refresh";

type Page = { documents: InboxDocument[]; next_cursor?: string };
type Summary = { used: number; limit: number; reset_at: string; available: number; checking: number; rejected: number; warning?: string };

async function changeStatus(organization: string, documentID: string, action: "trash" | "restore") {
  "use server";
  const destination = `/o/${encodeURIComponent(organization)}/documents`;
  const response = await sessionPOST(`/v1${destination}/${encodeURIComponent(documentID)}/${action}`, new URLSearchParams());
  if (response?.status === 401) redirect("/");
  redirect(`${destination}?${response?.ok ? "updated=1" : "error=status"}`);
}
export default async function DocumentsPage({ params, searchParams }: {
  params: Promise<{ organization: string }>;
  searchParams: Promise<{ cursor?: string; status?: string; source?: string; filename?: string; from?: string; to?: string; updated?: string; error?: string; selected?: string }>;
}) {
  const { organization } = await params;
  const { cursor, status, source, filename, from, to, updated, error, selected } = await searchParams;
  const root = `/o/${encodeURIComponent(organization)}`;
  const filters = new URLSearchParams();
  for (const [key, value] of Object.entries({ status, source, filename, from, to })) if (value) filters.set(key, value);
  const filterQuery = filters.toString();
  const listQuery = new URLSearchParams(filters);
  listQuery.set("limit", "20");
  if (cursor) listQuery.set("cursor", cursor);
  const base = `/v1${root}/documents`;
  const [response, summaryResponse, organizationResponse] = await Promise.all([sessionGET(`${base}?${listQuery}`), sessionGET(`${base}/summary`), sessionGET(`/v1${root}`)]);
  if (response?.status === 401 || organizationResponse?.status === 401) redirect("/");
  if (!response?.ok || !organizationResponse?.ok) return <section className="error-state" role="alert"><h1>ยังเปิดเอกสารไม่ได้</h1><p>ตรวจสอบการเชื่อมต่อแล้วลองอีกครั้ง</p><Link href={`${root}/documents`}>ลองใหม่</Link></section>;
  const page = await response.json() as Page;
  const summary = summaryResponse?.ok ? await summaryResponse.json() as Summary : null;
  const { membership } = await organizationResponse.json() as { membership: { role: string } };
  const canManage = membership.role === "Owner" || membership.role === "Admin";
  if (selected && page.documents.some(doc => doc.id === selected)) redirect(`${root}/documents/${encodeURIComponent(selected)}`);
  const extractionEnabled = process.env.EXTRACTION_ENABLED === "true";
  const accountingEnabled = process.env.ACCOUNTING_ENABLED === "true";
  const reviewEnabled = process.env.REVIEW_ENABLED === "true";
  const countResponse = canManage && !reviewEnabled ? await sessionGET(`${base}/export/count${filterQuery ? `?${filterQuery}` : ""}`) : null;
  const exportCount = countResponse?.ok ? (await countResponse.json() as { count: number }).count : null;
  // shortcut: status reads are bounded to 20 documents; add a batch API before increasing page size.
  const documents = await Promise.all(page.documents.map(doc => workflowDocument(organization, doc)));
  const search = <form className="workflow-search" method="get" role="search"><div><DocumentIcon kind="search" /><input id="document-search" name="filename" aria-label="ค้นหาชื่อไฟล์เอกสาร" placeholder="ค้นหาชื่อไฟล์เอกสาร…" defaultValue={filename} /><button type="submit" className="secondary-button">ค้นหา</button></div><details><summary>ตัวกรองเพิ่มเติม{filterQuery ? " · มีตัวกรอง" : ""}</summary><div className="workflow-search-options"><label>สถานะไฟล์<select name="status" defaultValue={status || ""}><option value="">ทั้งหมด</option><option value="Available">พร้อมใช้</option><option value="Archived">เก็บถาวร</option><option value="Trash">ถังขยะ</option><option value="Checking">กำลังตรวจไฟล์</option><option value="Rejected">ปฏิเสธ</option></select></label><label>ช่องทาง<select name="source" defaultValue={source || ""}><option value="">ทั้งหมด</option><option value="Web">อัปโหลด</option><option value="LINE">LINE</option><option value="Drive">Google Drive</option></select></label><label>ตั้งแต่วันที่<input type="date" name="from" defaultValue={from} /></label><label>ถึงวันที่<input type="date" name="to" defaultValue={to} /></label><Link href={`${root}/documents`}>ล้างตัวกรอง</Link></div></details></form>;
  const tools = <nav className="workflow-links" aria-label="เครื่องมือเอกสาร">{reviewEnabled && <Link className="secondary-button" href={`${root}/review`}>คิวตรวจเอกสาร</Link>}{canManage && <details><summary>เครื่องมืออื่น</summary><div><Link href={`${root}/connections`}>การเชื่อมต่อ</Link><Link href={`${root}/business`}>ข้อมูลธุรกิจ</Link>{reviewEnabled && <Link href={`${root}/exports`}>ส่งออกข้อมูล</Link>}{!reviewEnabled && exportCount !== null && exportCount <= 5000 && <><a href={`/api${root}/documents/export.csv${filterQuery ? `?${filterQuery}` : ""}`}>ส่งออก CSV</a><a href={`/api${root}/documents/export.xlsx${filterQuery ? `?${filterQuery}` : ""}`}>ส่งออก XLSX</a></>}{!reviewEnabled && exportCount !== null && exportCount > 5000 && <span>กรองให้เหลือไม่เกิน 5,000 รายการก่อนส่งออก</span>}{extractionEnabled && <><a href={`/api${root}/documents/extraction.csv`}>ข้อมูลยืนยันแล้ว CSV</a><a href={`/api${root}/documents/extraction.xlsx`}>ข้อมูลยืนยันแล้ว XLSX</a></>}{accountingEnabled && <><Link href={`${root}/accounting`}>ข้อเสนอที่รอตรวจ</Link><a href={`/api${root}/documents/accounting.csv`}>ข้อเสนอที่อนุมัติแล้ว CSV</a><a href={`/api${root}/documents/accounting.xlsx`}>ข้อเสนอที่อนุมัติแล้ว XLSX</a></>}</div></details>}</nav>;
  const notices = <>{updated && <p className="success-message" role="status">อัปเดตสถานะเอกสารแล้ว</p>}{error && <p className="form-error" role="alert">เปลี่ยนสถานะไม่ได้ กรุณารีเฟรชรายการแล้วลองอีกครั้ง</p>}{summary && <details className="documents-usage"><summary>การใช้เอกสารของทีม {canManage && summary.warning && <span>· ใกล้เต็ม ({summary.warning}%)</span>}</summary><div>{canManage && <span>ใช้ไป {summary.used}/{summary.limit} เอกสาร</span>}<span>พร้อมใช้ {summary.available}</span><span>กำลังตรวจ {summary.checking}</span><span>ปฏิเสธ {summary.rejected}</span>{canManage && <span>รีเซ็ต {new Intl.DateTimeFormat("th-TH", { dateStyle: "medium" }).format(new Date(summary.reset_at))}</span>}</div></details>}</>;
  return <><SearchShortcut /><DocumentStatusRefresh active={documents.some(doc => doc.status === "Checking" || doc.stage === "processing")} /><WorkflowInbox key={listQuery.toString()} documents={documents} organization={organization} canManage={canManage} changeStatus={changeStatus} upload={<UploadWorkspace organization={organization} canManage={canManage} />} search={search} tools={tools} notices={notices} pagination={page.next_cursor && <Link className="secondary-button" href={`${root}/documents?${new URLSearchParams({ ...Object.fromEntries(filters), cursor: page.next_cursor })}`}>หน้าถัดไป →</Link>} /></>;
}
