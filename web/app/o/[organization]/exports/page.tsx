import { sessionGET, sessionPOST } from "@/lib/session-api";
import Link from "next/link";
import { redirect } from "next/navigation";

type Product = "raw_documents" | "confirmed_values" | "approved_suggestions";
const products: { id: Product; label: string; help: string; path: string; dateBasis: string }[] = [
  { id: "raw_documents", label: "รายการเอกสาร", help: "ข้อมูลรายการเอกสารเท่านั้น ไม่มีค่า OCR หรือข้อมูลที่ยืนยันแล้ว", path: "export", dateBasis: "วันที่รับเอกสาร" },
  { id: "confirmed_values", label: "ข้อมูลยืนยัน", help: "ค่าที่ผู้มีสิทธิ์ตรวจและยืนยันจากต้นฉบับแล้ว", path: "extraction", dateBasis: "วันที่ยืนยัน" },
  { id: "approved_suggestions", label: "ข้อมูลอนุมัติ", help: "ข้อมูลยืนยันพร้อมหมวดค่าใช้จ่ายที่ Owner/Admin อนุมัติแล้ว · accounting_suggestions_v1", path: "accounting", dateBasis: "วันที่อนุมัติ" },
];
type Count = { count: number; synchronous_threshold: number; maximum_rows: number };
type State = { id: string; status: string; row_count: number; failure_code?: string };

export default async function ReviewExports({ params, searchParams }: {
  params: Promise<{ organization: string }>;
  searchParams: Promise<{ product?: string; status?: string; date_from?: string; date_to?: string; job?: string; error?: string }>;
}) {
  const { organization } = await params;
  const { product: productParam, status: statusParam, date_from: dateFrom = "", date_to: dateTo = "", job: jobID, error } = await searchParams;
  const root = `/o/${encodeURIComponent(organization)}`;
  const product = products.find((item) => item.id === productParam) ?? products[0];
  const status = statusParam === "Archived" || statusParam === "Trash" && product.id === "raw_documents" ? statusParam : "Available";
  const countQuery = new URLSearchParams({ product: product.id, status, date_from: dateFrom, date_to: dateTo });
  const [countResponse, jobResponse] = await Promise.all([
    sessionGET(`/v1${root}/review-exports/count?${countQuery}`),
    jobID ? sessionGET(`/v1${root}/review-exports/${encodeURIComponent(jobID)}`) : Promise.resolve(null),
  ]);
  if (countResponse?.status === 401) redirect("/");
  if (!countResponse?.ok) return <section className="error-state" role="alert"><h1>เปิดการส่งออกไม่ได้</h1><Link href={`${root}/documents`}>กลับไปรายการเอกสาร</Link></section>;
  const count = await countResponse.json() as Count;
  const state = jobResponse?.ok ? await jobResponse.json() as State : null;
  const canDownloadDirect = status === "Available" && !dateFrom && !dateTo && count.count <= count.synchronous_threshold;

  async function requestExport(formData: FormData) {
    "use server";
    const chosenProduct = String(formData.get("product") ?? "");
    const chosenStatus = String(formData.get("status") ?? "");
    const format = String(formData.get("format") ?? "");
    const dates = { date_from: String(formData.get("date_from") ?? ""), date_to: String(formData.get("date_to") ?? "") };
    const response = await sessionPOST(`/v1${root}/review-exports`, new URLSearchParams({ product: chosenProduct, status: chosenStatus, format, ...dates }));
    if (!response?.ok) redirect(`${root}/exports?${new URLSearchParams({ product: chosenProduct, status: chosenStatus, ...dates, error: "request" })}`);
    const queued = await response.json() as State;
    redirect(`${root}/exports?${new URLSearchParams({ product: chosenProduct, status: chosenStatus, ...dates, job: queued.id })}`);
  }

  let downloadURL = "";
  if (state?.status === "Ready") {
    const ready = await sessionGET(`/v1${root}/review-exports/${encodeURIComponent(state.id)}/download`);
    if (ready?.ok) {
      const data = await ready.json() as { download_url: string };
      if (data.download_url.startsWith("/v1/export-artifacts/")) downloadURL = data.download_url.replace("/v1/", "/api/");
    }
  }
  const directQuery = product.id === "raw_documents" ? "?status=Available" : "";
  return <section className="mx-auto max-w-[1100px]">
    <Link className="secondary-button mb-5" href={`${root}/documents`}>← กลับไปรายการเอกสาร</Link>
    <header className="work-header"><div><p className="eyebrow">EXPORTS</p><h1>ส่งออกข้อมูล</h1><p className="intro">เลือกชนิดข้อมูลที่ตรงกับระดับการตรวจทานก่อนดาวน์โหลด</p></div></header>
    <nav className="mb-5 flex flex-wrap gap-2" aria-label="ชนิดข้อมูลส่งออก">{products.map((tab) => <Link key={tab.id} href={`${root}/exports?product=${tab.id}`} aria-current={tab.id === product.id ? "page" : undefined} className={tab.id === product.id ? "button" : "secondary-button"}>{tab.label}</Link>)}</nav>
    <section className="rounded-[1.2rem] border border-line bg-surface p-5 shadow-panel">
      <h2 className="font-semibold">{product.label}</h2><p className="mt-1 text-sm text-muted">{product.help}</p>
      {product.id === "raw_documents" && <p className="mt-3 rounded-xl border border-amber-300 bg-amber-50 p-3 text-sm text-amber-950" role="note">ไฟล์นี้เป็นรายการเอกสารที่ยังไม่ผ่านการยืนยันข้อมูล ห้ามใช้แทนข้อมูลบัญชีที่ตรวจแล้ว</p>}
      <form method="get" className="mt-5 flex flex-wrap items-end gap-3"><input type="hidden" name="product" value={product.id} /><label className="grid gap-1">สถานะเอกสาร<select name="status" defaultValue={status} className="min-h-11 rounded-xl border border-line px-3"><option value="Available">พร้อมใช้งาน</option><option value="Archived">เก็บถาวร</option>{product.id === "raw_documents" && <option value="Trash">ถังขยะ</option>}</select></label><label className="grid gap-1">{product.dateBasis} ตั้งแต่<input type="date" name="date_from" defaultValue={dateFrom} className="min-h-11 rounded-xl border border-line px-3" /></label><label className="grid gap-1">ถึง<input type="date" name="date_to" defaultValue={dateTo} className="min-h-11 rounded-xl border border-line px-3" /></label><button className="secondary-button" type="submit">ดูจำนวน</button></form>
      <p className="mt-4" role="status">พบ {count.count.toLocaleString("th-TH")} แถว · สูงสุด {count.maximum_rows.toLocaleString("th-TH")} แถวต่อไฟล์</p>
      {count.count > count.maximum_rows && <p className="form-error" role="alert">ผลลัพธ์เกินขนาดสูงสุด กรุณากรองให้แคบลง</p>}
      {canDownloadDirect && <div className="mt-4 flex flex-wrap gap-3"><a className="secondary-button" href={`/api/o/${encodeURIComponent(organization)}/documents/${product.path}.csv${directQuery}`}>ดาวน์โหลด CSV</a><a className="secondary-button" href={`/api/o/${encodeURIComponent(organization)}/documents/${product.path}.xlsx${directQuery}`}>ดาวน์โหลด XLSX</a></div>}
      {!canDownloadDirect && count.count <= count.maximum_rows && <form action={requestExport} className="mt-4 flex flex-wrap gap-3"><input type="hidden" name="product" value={product.id} /><input type="hidden" name="status" value={status} /><input type="hidden" name="date_from" value={dateFrom} /><input type="hidden" name="date_to" value={dateTo} /><button className="button" name="format" value="csv" type="submit">สร้างไฟล์ CSV</button><button className="secondary-button" name="format" value="xlsx" type="submit">สร้างไฟล์ XLSX</button></form>}
      {error && <p className="form-error mt-3" role="alert">สร้างไฟล์ไม่ได้ กรุณาลองใหม่</p>}
      {state && <div className="mt-5 rounded-xl border border-line p-4" role="status"><p>งานส่งออก {state.id.slice(0, 8)} · {state.status} · {state.row_count.toLocaleString("th-TH")} แถว</p>
        {downloadURL ? <a className="button mt-3 inline-flex" href={downloadURL}>ดาวน์โหลดไฟล์ที่พร้อมแล้ว</a> : state.status === "Queued" || state.status === "Running" ? <Link className="secondary-button mt-3 inline-flex" href={`${root}/exports?${new URLSearchParams({ product: product.id, status, date_from: dateFrom, date_to: dateTo, job: state.id })}`}>ตรวจสถานะอีกครั้ง</Link> : <p className="mt-2 text-sm text-muted">ไฟล์ไม่พร้อมให้ดาวน์โหลด กรุณาสร้างคำขอใหม่</p>}
      </div>}
    </section>
  </section>;
}
