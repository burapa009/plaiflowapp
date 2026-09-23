import { sessionGET } from "@/lib/session-api";
import Link from "next/link";
import { notFound, redirect } from "next/navigation";
import OCRPanel from "./ocr-panel";
import ExtractionPanel from "./extraction-panel";
import AccountingPanel from "./accounting-panel";


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

export default async function DocumentDetailPage({ params }: {
  params: Promise<{ organization: string; document: string }>;
}) {
  const { organization, document } = await params;
  const base = `/o/${encodeURIComponent(organization)}/documents`;
  const response = await sessionGET(`/v1${base}/${encodeURIComponent(document)}/sources`);
  if (response?.status === 401) redirect("/");
  if (response?.status === 404) notFound();
  if (!response?.ok) return <section className="error-state" role="alert"><h1>ยังเปิดรายละเอียดเอกสารไม่ได้</h1><Link href={base}>กลับไปรายการเอกสาร</Link></section>;
  const { document: item, sources } = await response.json() as Detail;
  const date = (value: string) => new Intl.DateTimeFormat("th-TH", { dateStyle: "medium", timeStyle: "short" }).format(new Date(value));
  return <section className="mx-auto max-w-[1270px]">
    <Link href={base}>← กลับไปรายการเอกสาร</Link>
    <header className="work-header"><div><p className="eyebrow">DOCUMENT</p><h1>{item.filename}</h1><p className="intro">{item.status} · {item.mime} · {Math.ceil(item.size / 1024)} KB · รับเมื่อ {date(item.accepted_at)}</p></div>{item.status !== "Trash" && <a className="secondary-button" href={`/api/o/${encodeURIComponent(organization)}/documents/${encodeURIComponent(item.id)}/original`}>ดาวน์โหลดต้นฉบับ</a>}</header>
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
    <OCRPanel organization={organization} document={document} />
    <ExtractionPanel organization={organization} document={document} />
    <AccountingPanel organization={organization} document={document} />
  </section>;
}
