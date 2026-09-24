import { sessionGET, sessionPOST } from "@/lib/session-api";
import Link from "next/link";
import { notFound, redirect } from "next/navigation";
import OCRPanel from "./ocr-panel";
import ExtractionPanel from "./extraction-panel";
import AccountingPanel from "./accounting-panel";
import ReviewShortcuts from "./review-shortcuts";


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
};

async function archiveDocument(organization: string, document: string) {
  "use server";
  const response = await sessionPOST(`/v1/o/${encodeURIComponent(organization)}/documents/${encodeURIComponent(document)}/archive`, new URLSearchParams());
  if (response?.status === 401) redirect("/");
  redirect(`/o/${encodeURIComponent(organization)}/documents?${response?.ok ? "updated=1" : "error=status"}`);
}

export default async function DocumentDetailPage({ params, searchParams }: {
  params: Promise<{ organization: string; document: string }>;
  searchParams: Promise<{ ocr?: string; queue_cursor?: string; queue_view?: string; queue_status?: string; previous?: string }>;
}) {
  const { organization, document } = await params;
  const { ocr, queue_cursor, queue_view, queue_status, previous } = await searchParams;
  const base = `/o/${encodeURIComponent(organization)}/documents`;
  const [response, organizationResponse] = await Promise.all([
    sessionGET(`/v1${base}/${encodeURIComponent(document)}/sources`),
    sessionGET(`/v1/o/${encodeURIComponent(organization)}`),
  ]);
  if (response?.status === 401) redirect("/");
  if (response?.status === 404) notFound();
  if (!response?.ok) return <section className="error-state" role="alert"><h1>ยังเปิดรายละเอียดเอกสารไม่ได้</h1><Link href={base}>กลับไปรายการเอกสาร</Link></section>;
  const { document: item, sources } = await response.json() as Detail;
  const membership = organizationResponse?.ok ? (await organizationResponse.json() as { membership: { role: string } }).membership : null;
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
  return <section className="mx-auto max-w-[1270px]">
    {process.env.REVIEW_ENABLED === "true" && <ReviewShortcuts nextHref={nextHref} previousHref={previous ? `${base}/${encodeURIComponent(previous)}` : ""} />}
    {previous && <Link data-review-previous className="secondary-button mb-5 ml-2" href={`${base}/${encodeURIComponent(previous)}`}>← เอกสารก่อนหน้า</Link>}
    {nextHref && <Link data-review-next className="secondary-button mb-5 ml-2" href={nextHref}>{nextDocument ? "เอกสารถัดไป →" : "กลับคิวตรวจเอกสาร"}</Link>}
    <Link className="mb-5 inline-flex min-h-11 w-fit items-center gap-2 rounded-full border border-line bg-surface px-4 py-2 text-sm font-semibold text-brand-strong shadow-sm transition-colors hover:border-brand-strong hover:bg-surface-soft focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-brand-strong" href={base}>
      <svg aria-hidden="true" className="size-4 shrink-0" fill="none" stroke="currentColor" strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" viewBox="0 0 24 24"><path d="m12 19-7-7 7-7M5 12h14" /></svg>
      กลับไปรายการเอกสาร
    </Link>
    <header className="work-header"><div><p className="eyebrow">DOCUMENT</p><h1>{item.filename}</h1><p className="intro">{item.status} · {item.mime} · {Math.ceil(item.size / 1024)} KB · รับเมื่อ {date(item.accepted_at)}</p></div><div className="card-actions">{item.status !== "Trash" && <a className="secondary-button" href={`/api/o/${encodeURIComponent(organization)}/documents/${encodeURIComponent(item.id)}/original`}>ดาวน์โหลดต้นฉบับ</a>}{item.status === "Available" && (membership?.role === "Owner" || membership?.role === "Admin") && <form action={archiveDocument.bind(null, organization, document)}><button className="secondary-button" type="submit">เก็บถาวร</button></form>}</div></header>
    <section className="rounded-[1.2rem] border border-line bg-surface p-5 shadow-panel" aria-labelledby="source-heading"><h2 id="source-heading">แหล่งที่มา</h2>
      {sources.length === 0 ? <p>ไม่มีแหล่งที่มาที่คุณมีสิทธิ์ดู</p> : <ol className="grid gap-4 pl-6 [&>li]:break-words [&>li]:border-b [&>li]:border-line [&>li]:pb-4 [&>li:last-child]:border-0 [&_span]:block [&_span]:text-muted">{sources.map((source) => <li key={source.id}>
        <strong>{source.channel}</strong><span>รับเมื่อ {date(source.accepted_at)}</span>{source.submitted_by && <span>ส่งโดย {source.submitted_by}</span>}
        {source.selected_at && <span>เลือกจาก Drive เมื่อ {date(source.selected_at)}</span>}
        {source.provider_filename && <span>ชื่อไฟล์ใน Drive: {source.provider_filename}</span>}
        {source.provider_mime && <span>ชนิดไฟล์ที่ Drive ระบุ: {source.provider_mime}</span>}
        {source.provider_size !== undefined && <span>ขนาดที่ Drive ระบุ: {source.provider_size.toLocaleString("th-TH")} bytes</span>}
        {source.drive_file_id && <span>Drive file ID: {source.drive_file_id}</span>}
        {source.drive_revision && <span>Drive revision: {source.drive_revision}</span>}
      </li>)}</ol>}
    </section>
    <OCRPanel organization={organization} document={document} unavailable={ocr === "unavailable"} />
    {item.status !== "Trash" && <div className="mt-5 grid gap-5 lg:grid-cols-2">
      <details className="group rounded-[1.2rem] border border-line bg-surface p-5 shadow-panel">
        <summary className="cursor-pointer font-semibold lg:hidden">ดูเอกสารต้นฉบับ</summary>
        <h2 id="original-preview-heading" className="hidden font-semibold lg:block">เอกสารต้นฉบับ</h2>
        <p className="mt-1 text-sm text-muted">เปิดต้นฉบับเพื่อตรวจเทียบก่อนยืนยันข้อมูล</p>
        <iframe title="ตัวอย่างเอกสารต้นฉบับ" loading="lazy" className="mt-4 hidden min-h-[32rem] w-full rounded-xl border border-line group-open:block lg:block" src={`/api/o/${encodeURIComponent(organization)}/documents/${encodeURIComponent(document)}/original?preview=1`} />
      </details>
      <ExtractionPanel organization={organization} document={document} nextHref={nextHref} />
    </div>}
    <AccountingPanel organization={organization} document={document} nextHref={nextHref} />
  </section>;
}
