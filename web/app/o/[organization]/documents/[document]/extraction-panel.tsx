import { sessionGET, sessionPOST } from "@/lib/session-api";
import { revalidatePath } from "next/cache";
import { redirect } from "next/navigation";

type Field = { presence: string; raw: string; normalized: string; confidence: string; evidence: { page: number; line: number }[] };
type Warning = { code: string; field: string; severity: string };
type Extraction = {
  draft: { document_type: string; fields: Record<string, Field>; warnings: Warning[] };
  ocr_job_id: string;
  revision: number;
  source_superseded: boolean;
  confirmed: { values: Record<string, string>; confirmed_at: string } | null;
};

const fields: [string, string][] = [
  ["document_number", "เลขที่เอกสาร"], ["issue_date", "วันที่ออกเอกสาร (YYYY-MM-DD)"],
  ["seller_name", "ชื่อผู้ขาย"], ["seller_tax_id", "เลขผู้เสียภาษีผู้ขาย"],
  ["buyer_name", "ชื่อผู้ซื้อ"], ["buyer_tax_id", "เลขผู้เสียภาษีผู้ซื้อ"],
  ["currency", "สกุลเงิน"], ["subtotal", "มูลค่าก่อนภาษี"],
  ["vat_amount", "ภาษีมูลค่าเพิ่ม"], ["total_amount", "ยอดรวม"],
];

const warningLabels: Record<string, string> = {
  required_missing: "ไม่พบข้อมูลจำเป็น", multiple_candidates: "พบค่าหลายแบบ ต้องเลือกเอง",
  ocr_low_quality: "OCR อ่านบรรทัดนี้ไม่มั่นใจ", amount_mismatch: "ยอดเงินไม่สัมพันธ์กัน",
  invalid_value: "พบข้อความแต่แปลงค่าไม่ได้",
};

export default async function ExtractionPanel({ organization, document }: { organization: string; document: string }) {
  const path = `/o/${encodeURIComponent(organization)}/documents/${encodeURIComponent(document)}`;
  const [response, membershipResponse] = await Promise.all([
    sessionGET(`/v1${path}/extraction`), sessionGET(`/v1/o/${encodeURIComponent(organization)}`),
  ]);
  if (!response?.ok) return null;
  const data = await response.json() as Extraction;
  const membership = membershipResponse?.ok ? await membershipResponse.json() as { role?: string; membership?: { role?: string } } : null;
  const role = membership?.role ?? membership?.membership?.role;
  const canConfirm = role === "Owner" || role === "Admin";

  async function confirm(formData: FormData) {
    "use server";
    const body = new URLSearchParams();
    for (const [key] of fields) body.set(key, String(formData.get(key) ?? ""));
    body.set("ocr_job_id", String(formData.get("ocr_job_id") ?? ""));
    body.set("expected_revision", String(formData.get("expected_revision") ?? "0"));
    body.set("review_ack", String(formData.get("review_ack") ?? ""));
    const result = await sessionPOST(`/v1${path}/extraction/confirm`, body);
    if (!result?.ok) redirect(`${path}?extraction=${result?.status === 409 ? "changed" : "invalid"}`);
    revalidatePath(path);
    redirect(`${path}?extraction=confirmed`);
  }

  return <section className="mt-5 rounded-[1.2rem] border border-line bg-surface p-5 shadow-panel" aria-labelledby="extraction-heading">
    <h2 id="extraction-heading">ข้อมูลที่สกัดจากเอกสาร</h2>
    <p className="mt-2 text-sm text-muted">ชนิดที่ระบบอ่าน: {data.draft.document_type} · ค่าจาก OCR เป็นข้อเสนอ ไม่ใช่ข้อมูลที่ยืนยันแล้ว</p>
    {data.confirmed && !data.source_superseded && <p className="mt-3 rounded-xl bg-emerald-50 p-3 text-emerald-900" role="status">ยืนยันข้อมูลแล้วเมื่อ {new Date(data.confirmed.confirmed_at).toLocaleString("th-TH")}</p>}
    {data.source_superseded && <p className="mt-3 rounded-xl bg-amber-50 p-3 text-amber-900" role="alert">OCR เปลี่ยนหลังการยืนยันครั้งก่อน กรุณาตรวจทานใหม่ก่อนส่งออก</p>}
    {data.draft.warnings.length > 0 && <div className="mt-4 rounded-xl border border-amber-300 bg-amber-50 p-4 text-amber-950" role="status"><h3 className="font-semibold">รายการที่ต้องตรวจ</h3><ul className="mt-2 list-disc pl-5">{data.draft.warnings.map((warning, index) => <li key={`${warning.code}-${warning.field}-${index}`}>{warningLabels[warning.code] ?? warning.code}: {fields.find(([key]) => key === warning.field)?.[1] ?? warning.field}</li>)}</ul></div>}
    <form action={confirm} className="mt-4 grid gap-4">
      <input type="hidden" name="ocr_job_id" value={data.ocr_job_id} />
      <input type="hidden" name="expected_revision" value={data.revision} />
      <div className="grid gap-4 md:grid-cols-2">{fields.map(([key, label]) => {
        const field = data.draft.fields[key];
        return <div key={key} className="rounded-xl border border-line p-3">
          <label htmlFor={`extraction-${key}`} className="block font-semibold">{label}</label>
          <p className="mt-1 break-words text-sm text-muted">ข้อความดิบ: {field?.raw || "ไม่พบ"}</p>
          <p className="break-words text-sm text-muted">ค่ามาตรฐานที่ระบบเสนอ: {field?.normalized || "ไม่มี"}</p>
          <p className="text-xs text-muted">ความมั่นใจรายฟิลด์: ยังไม่ผ่านการปรับเทียบ{field?.evidence?.length ? ` · หน้า ${field.evidence[0].page} บรรทัด ${field.evidence[0].line}` : ""}</p>
          {canConfirm && <input id={`extraction-${key}`} name={key} maxLength={240} defaultValue={data.confirmed?.values[key] ?? ""} className="mt-2 min-h-11 w-full rounded-xl border border-line bg-surface px-3 text-fg" />}
        </div>;
      })}</div>
      {canConfirm && <><label className="flex items-start gap-2 text-sm"><input required type="checkbox" name="review_ack" value="1" className="mt-1" />ฉันตรวจเทียบค่ากับเอกสารต้นฉบับแล้ว และยืนยันค่าที่กรอก</label><button type="submit" className="button w-fit">ยืนยันข้อมูลที่ตรวจแล้ว</button></>}
    </form>
  </section>;
}
