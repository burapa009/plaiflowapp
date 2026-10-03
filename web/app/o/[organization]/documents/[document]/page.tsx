import { sessionGET, sessionPOST } from "@/lib/session-api";
import Link from "next/link";
import { notFound, redirect } from "next/navigation";
import { DocumentStatus } from "../document-visuals";
import OCRPanel from "./ocr-panel";
import ExtractionPanel, { documentTypeLabels, type Extraction } from "./extraction-panel";
import AccountingPanel from "./accounting-panel";
import ReviewShortcuts from "./review-shortcuts";
import DocumentPreview from "./document-preview";
import ReviewWorkspace from "./review-workspace";


type Source = {
  id: string;
  channel: string;
  submitted_by: string;
  accepted_at: string;
  provider_filename?: string;
  provider_mime?: string;
  provider_size?: number;
  selected_at?: string;
  drive_file_id?: string;
  drive_revision?: string;
};
type Detail = {
  document: { id: string; filename: string; mime: string; size: number; status: string; accepted_at: string };
  sources: Source[];
  attachments: { id: string; filename: string; mime: string }[];
};

async function archiveDocument(organization: string, document: string) {
  "use server";
  const response = await sessionPOST(`/v1/o/${encodeURIComponent(organization)}/documents/${encodeURIComponent(document)}/archive`, new URLSearchParams());
  if (response?.status === 401) redirect("/");
  redirect(`/o/${encodeURIComponent(organization)}/documents?${response?.ok ? "updated=1" : "error=status"}`);
}

export default async function DocumentDetailPage({ params, searchParams }: {
  params: Promise<{ organization: string; document: string }>;
  searchParams: Promise<{ extraction?: string; ocr?: string; queue_cursor?: string; queue_view?: string; queue_status?: string; previous?: string }>;
}) {
  const { organization, document } = await params;
  const { extraction: extractionResult, ocr, queue_cursor, queue_view, queue_status, previous } = await searchParams;
  const base = `/o/${encodeURIComponent(organization)}/documents`;
  const [response, organizationResponse, businessResponse, ocrResponse] = await Promise.all([
    sessionGET(`/v1${base}/${encodeURIComponent(document)}/sources`),
    sessionGET(`/v1/o/${encodeURIComponent(organization)}`),
    sessionGET(`/v1/o/${encodeURIComponent(organization)}/business`),
    sessionGET(`/v1${base}/${encodeURIComponent(document)}/ocr`),
  ]);
  if (response?.status === 401) redirect("/");
  if (response?.status === 404) notFound();
  if (!response?.ok) return <section className="error-state" role="alert"><h1>ยังเปิดรายละเอียดเอกสารไม่ได้</h1><Link href={base}>กลับไปรายการเอกสาร</Link></section>;
  const { document: item, sources, attachments } = await response.json() as Detail;
  const membership = organizationResponse?.ok ? (await organizationResponse.json() as { membership: { role: string; organization_name: string } }).membership : null;
  const business = businessResponse?.ok ? await businessResponse.json() as { name_th: string; branch_type: string } : null;
  const ocrStatus = ocrResponse?.ok ? (await ocrResponse.json() as { ocr: { status: string } }).ocr.status : "";
  const extractionResponse = item.status !== "Trash" ? await sessionGET(`/v1${base}/${encodeURIComponent(document)}/extraction`) : null;
  const extraction = extractionResponse?.ok ? await extractionResponse.json() as Extraction : null;
  let nextDocument = "";
  let nextAcceptedAt = "";
  let queueLoaded = false;
  if (process.env.REVIEW_ENABLED === "true" && queue_cursor) {
    const queue = new URLSearchParams({ cursor: queue_cursor, limit: "1", view: queue_view || "all", status: queue_status || "Available" });
    const nextResponse = await sessionGET(`/v1/o/${encodeURIComponent(organization)}/review-queue?${queue}`);
    if (nextResponse?.ok) {
      queueLoaded = true;
      const next = await nextResponse.json() as { items: { document_id: string; accepted_at: string }[] };
      nextDocument = next.items[0]?.document_id ?? "";
      nextAcceptedAt = next.items[0]?.accepted_at ?? "";
    }
  }
  const queueOptions = { view: queue_view || "all", status: queue_status || "Available" };
  const nextHref = nextDocument ? `${base}/${encodeURIComponent(nextDocument)}?${new URLSearchParams({
    queue_cursor: Buffer.from(JSON.stringify({ at: nextAcceptedAt, id: nextDocument })).toString("base64url"),
    queue_view: queueOptions.view, queue_status: queueOptions.status, previous: document,
  })}` : queueLoaded ? `/o/${encodeURIComponent(organization)}/review?${new URLSearchParams({ ...queueOptions, result: "complete" })}` : "";
  const date = (value: string) => new Intl.DateTimeFormat("th-TH", { dateStyle: "medium", timeStyle: "short" }).format(new Date(value));
  const originalURL = `/api/o/${encodeURIComponent(organization)}/documents/${encodeURIComponent(document)}/original`;
  const sourcePanel = <section className="rounded-[1.2rem] border border-line bg-surface p-5 shadow-panel" aria-labelledby="source-heading"><h2 id="source-heading">แหล่งที่มา</h2>
    {sources.length === 0 ? <p>ไม่มีแหล่งที่มาที่คุณมีสิทธิ์ดู</p> : <ol className="grid gap-4 pl-6 [&>li]:break-words [&>li]:border-b [&>li]:border-line [&>li]:pb-4 [&>li:last-child]:border-0 [&_span]:block [&_span]:text-muted">{sources.map((source) => <li key={source.id}>
      <strong>{source.channel}</strong><span>รับเมื่อ {date(source.accepted_at)}</span>{source.submitted_by && <span>ส่งโดย {source.submitted_by}</span>}
      {source.selected_at && <span>เลือกจาก Drive เมื่อ {date(source.selected_at)}</span>}
      {source.provider_filename && <span>ชื่อไฟล์ใน Drive: {source.provider_filename}</span>}
      {source.provider_mime && <span>ชนิดไฟล์ที่ Drive ระบุ: {source.provider_mime}</span>}
      {source.provider_size !== undefined && <span>ขนาดที่ Drive ระบุ: {source.provider_size.toLocaleString("th-TH")} bytes</span>}
      {source.drive_file_id && <span>Drive file ID: {source.drive_file_id}</span>}
      {source.drive_revision && <span>Drive revision: {source.drive_revision}</span>}
    </li>)}</ol>}
  </section>;
  const documentType = extraction?.draft.document_type === "receipt_tax_invoice" ? "ใบเสร็จรับเงิน / ใบกำกับภาษี" : documentTypeLabels[extraction?.draft.document_type ?? ""] ?? "ยังระบุไม่ได้";
  const viewer = <DocumentPreview src={`${originalURL}?preview=1`} filename={item.filename} mime={item.mime} documentType={documentType} ocrStatus={ocrStatus} organization={organization} document={document} attachments={attachments ?? []} />;
  const documentActions = <div className="card-actions">
    {previous && <Link data-review-previous className="secondary-button" href={`${base}/${encodeURIComponent(previous)}`}>← เอกสารก่อนหน้า</Link>}
    {nextHref && <Link data-review-next className="secondary-button" href={nextHref}>{nextDocument ? "เอกสารถัดไป →" : "กลับคิวตรวจเอกสาร"}</Link>}
    {item.status !== "Trash" && <a className="secondary-button" href={originalURL}>ดาวน์โหลดต้นฉบับ</a>}
    {item.status === "Available" && (membership?.role === "Owner" || membership?.role === "Admin") && <form action={archiveDocument.bind(null, organization, document)}><button className="secondary-button" type="submit">เก็บถาวร</button></form>}
  </div>;
  return <section className={`document-detail-page ${item.status !== "Trash" ? "is-post-ocr" : "mx-auto max-w-[1270px]"}`}>
    {item.status === "Trash" && <Link className="mb-5 inline-flex min-h-11 w-fit items-center gap-2 rounded-full border border-line bg-surface px-4 py-2 text-sm font-semibold text-brand-strong shadow-sm transition-colors hover:border-brand-strong hover:bg-surface-soft focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-brand-strong" href={base}>← กลับไปรายการเอกสาร</Link>}
    <header className={item.status !== "Trash" ? "document-detail-header" : "work-header"}><div><p className="eyebrow">{extraction ? "DOCUMENT REVIEW" : "DOCUMENT"}</p><h1>{extraction ? "ตรวจข้อมูลเอกสาร" : item.filename}</h1><p className="intro">{extraction && <span className="document-detail-filename">{item.filename} · </span>}<DocumentStatus status={item.status} /> · {item.mime} · {Math.ceil(item.size / 1024)} KB · รับเมื่อ {date(item.accepted_at)}</p></div>{item.status !== "Trash" ? <Link data-review-exit className="document-detail-close" href={base} aria-label="กลับไปรายการเอกสาร" title="กลับไปรายการเอกสาร">×</Link> : documentActions}</header>
    {extraction ? <>
      <ReviewWorkspace key={`${extraction.ocr_job_id}:${extraction.revision}:${extraction.saved_review?.revision ?? 0}`} viewer={viewer} form={<ExtractionPanel organization={organization} document={document} nextHref={nextHref} data={extraction} role={membership?.role ?? ""} cancelHref={base} organizationName={business?.name_th || membership?.organization_name || ""} branchType={business?.branch_type || ""} result={extractionResult} attachmentCount={attachments?.length ?? 0} />} support={<div className="ocr-review-support">{documentActions}<details><summary>ข้อความที่ OCR อ่านได้</summary><OCRPanel organization={organization} document={document} unavailable={ocr === "unavailable"} /></details><details><summary>แหล่งที่มา</summary>{sourcePanel}</details>{process.env.REVIEW_ENABLED === "true" && <ReviewShortcuts nextHref={nextHref} previousHref={previous ? `${base}/${encodeURIComponent(previous)}` : ""} />}<AccountingPanel organization={organization} document={document} nextHref={nextHref} /></div>} />

    </> : item.status !== "Trash" ? <ReviewWorkspace viewer={viewer} formLabel="สถานะเอกสาร" form={<div className="document-before-reading"><section className="document-reading-status"><p className="eyebrow">อัปโหลด → อ่านข้อความ → จัดหมวด → ตรวจแก้ → ยืนยันข้อมูล</p><h2>{({ Queued: "รับเอกสารแล้ว · รออ่านข้อความ", Running: "กำลังอ่านข้อความจากเอกสาร", Completed: "อ่านเอกสารแล้ว", Failed: "อ่านเอกสารไม่สำเร็จ", Cancelled: "ยกเลิกการอ่านข้อความ", NotScheduled: "เอกสารพร้อมให้เปิดดู" } as Record<string, string>)[ocrStatus] ?? "ยังโหลดสถานะไม่ได้"}</h2><p>เมื่อข้อมูลพร้อม คุณจะตรวจแก้ทีละช่องและยืนยันเทียบกับต้นฉบับได้</p><Link className="secondary-button" href={`${base}/${encodeURIComponent(document)}`}>โหลดสถานะล่าสุด</Link></section><OCRPanel organization={organization} document={document} unavailable={ocr === "unavailable"} />{extractionResponse && !extractionResponse.ok && ![404, 409].includes(extractionResponse.status) && <p className="ocr-review-alert" role="alert">ยังโหลดฟอร์มตรวจไม่ได้ กรุณาโหลดสถานะล่าสุดอีกครั้ง</p>}<details className="document-source-details"><summary>แหล่งที่มาของเอกสาร</summary>{sourcePanel}</details></div>} support={documentActions} /> : sourcePanel}
    {!extraction && <AccountingPanel organization={organization} document={document} nextHref={nextHref} />}
  </section>;
}
