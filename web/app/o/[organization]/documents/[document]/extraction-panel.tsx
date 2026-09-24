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
  review_enabled?: boolean;
  saved_review?: { revision: number; values: Record<string, string>; decisions: Record<string, string> } | null;
	returned_review?: { reason_code: string; private_note: string; returned_at: string } | null;
	review_history?: { revision: number; values: Record<string, string>; decisions: Record<string, string>; updated_by: string; updated_at: string }[];
};

const fields: [string, string][] = [
  ["document_number", "เลขที่เอกสาร"], ["issue_date", "วันที่ออกเอกสาร (YYYY-MM-DD)"],
  ["seller_name", "ชื่อผู้ขาย"], ["seller_tax_id", "เลขผู้เสียภาษีผู้ขาย"], ["seller_branch", "สาขาผู้ขาย (5 หลัก; เว้นว่างถ้าไม่ทราบ)"],
  ["buyer_name", "ชื่อผู้ซื้อ"], ["buyer_tax_id", "เลขผู้เสียภาษีผู้ซื้อ"],
  ["currency", "สกุลเงิน"], ["subtotal", "มูลค่าก่อนภาษี"],
  ["vat_amount", "ภาษีมูลค่าเพิ่ม"], ["total_amount", "ยอดรวม"],
];

const warningLabels: Record<string, string> = {
  required_missing: "ไม่พบข้อมูลจำเป็น", multiple_candidates: "พบค่าหลายแบบ ต้องเลือกเอง",
  ocr_low_quality: "OCR อ่านบรรทัดนี้ไม่มั่นใจ", amount_mismatch: "ยอดเงินไม่สัมพันธ์กัน",
  invalid_value: "พบข้อความแต่แปลงค่าไม่ได้",
};

export default async function ExtractionPanel({ organization, document, nextHref = "" }: { organization: string; document: string; nextHref?: string }) {
  const path = `/o/${encodeURIComponent(organization)}/documents/${encodeURIComponent(document)}`;
  const [response, membershipResponse] = await Promise.all([
    sessionGET(`/v1${path}/extraction`), sessionGET(`/v1/o/${encodeURIComponent(organization)}`),
  ]);
  if (!response?.ok) return null;
  const data = await response.json() as Extraction;
  const membership = membershipResponse?.ok ? await membershipResponse.json() as { role?: string; membership?: { role?: string } } : null;
  const role = membership?.role ?? membership?.membership?.role;
  const canConfirm = role === "Owner" || role === "Admin";

  async function saveDraft(formData: FormData) {
    "use server";
    const body = new URLSearchParams();
    for (const [key] of fields) {
      body.set(key, String(formData.get(key) ?? ""));
      body.set(`${key}_decision`, String(formData.get(`${key}_decision`) ?? ""));
    }
    body.set("ocr_job_id", String(formData.get("ocr_job_id") ?? ""));
    body.set("expected_draft_revision", String(formData.get("expected_draft_revision") ?? "0"));
    const result = await sessionPOST(`/v1${path}/extraction/draft`, body);
    if (!result?.ok) redirect(`${path}?extraction=${result?.status === 409 ? "changed" : "invalid"}`);
    revalidatePath(path);
    if (formData.get("save_next") === "1" && nextHref) redirect(nextHref);
    redirect(`${path}?extraction=draft-saved`);
  }

  async function confirm(formData: FormData) {
    "use server";
    const body = new URLSearchParams();
    for (const [key] of fields) {
      body.set(key, String(formData.get(key) ?? ""));
      body.set(`${key}_decision`, String(formData.get(`${key}_decision`) ?? ""));
    }
    body.set("ocr_job_id", String(formData.get("ocr_job_id") ?? ""));
    body.set("expected_revision", String(formData.get("expected_revision") ?? "0"));
    body.set("expected_draft_revision", String(formData.get("expected_draft_revision") ?? "0"));
    body.set("review_ack", String(formData.get("review_ack") ?? ""));
    const result = await sessionPOST(`/v1${path}/extraction/confirm`, body);
    if (!result?.ok) redirect(`${path}?extraction=${result?.status === 409 ? "changed" : "invalid"}`);
    revalidatePath(path);
    redirect(`${path}?extraction=confirmed`);
  }

  async function returnReview(formData: FormData) {
    "use server";
    const body = new URLSearchParams();
    for (const key of ["ocr_job_id", "review_revision", "draft_revision", "reason_code", "private_note"])
      body.set(key, String(formData.get(key) ?? ""));
    const result = await sessionPOST(`/v1${path}/review/return`, body);
    if (!result?.ok) redirect(`${path}?extraction=${result?.status === 409 ? "changed" : "invalid"}`);
    revalidatePath(path);
    redirect(`${path}?extraction=returned`);
  }

  async function reprocess(formData: FormData) {
    "use server";
    const result = await sessionPOST(`/v1${path}/review/reprocess`, new URLSearchParams({
	  ocr_job_id: String(formData.get("ocr_job_id") ?? ""), draft_revision: String(formData.get("draft_revision") ?? "0"),
      reason_code: String(formData.get("reason_code") ?? ""), private_note: String(formData.get("private_note") ?? ""),
    }));
    if (!result?.ok) redirect(`${path}?extraction=reprocess-error`);
    revalidatePath(path);
    redirect(`${path}?extraction=reprocessing`);
  }

  return <section className="mt-5 rounded-[1.2rem] border border-line bg-surface p-5 shadow-panel" aria-labelledby="extraction-heading">
    <h2 id="extraction-heading">ข้อมูลที่สกัดจากเอกสาร</h2>
    <p className="mt-2 text-sm text-muted">ชนิดที่ระบบอ่าน: {data.draft.document_type} · ค่าจาก OCR เป็นข้อเสนอ ไม่ใช่ข้อมูลที่ยืนยันแล้ว</p>
    {data.confirmed && !data.source_superseded && <p className="mt-3 rounded-xl bg-emerald-50 p-3 text-emerald-900" role="status">ยืนยันข้อมูลแล้วเมื่อ {new Date(data.confirmed.confirmed_at).toLocaleString("th-TH")}</p>}
    {data.source_superseded && <p className="mt-3 rounded-xl bg-amber-50 p-3 text-amber-900" role="alert">OCR เปลี่ยนหลังการยืนยันครั้งก่อน กรุณาตรวจทานใหม่ก่อนส่งออก</p>}
	{data.returned_review && <div className="mt-3 rounded-xl border border-amber-300 bg-amber-50 p-3 text-amber-950" role="status"><strong>ส่งกลับเพื่อแก้ไข: {({ missing_value: "ข้อมูลไม่ครบ", incorrect_value: "ข้อมูลไม่ถูกต้อง", unreadable_original: "ต้นฉบับอ่านไม่ได้", other: "อื่น ๆ" } as Record<string, string>)[data.returned_review.reason_code] ?? data.returned_review.reason_code}</strong>{data.returned_review.private_note && <p className="mt-1 whitespace-pre-wrap">{data.returned_review.private_note}</p>}</div>}
    {data.draft.warnings.length > 0 && <div className="mt-4 rounded-xl border border-amber-300 bg-amber-50 p-4 text-amber-950" role="status"><h3 className="font-semibold">รายการที่ต้องตรวจ</h3><ul className="mt-2 list-disc pl-5">{data.draft.warnings.map((warning, index) => <li key={`${warning.code}-${warning.field}-${index}`}>{warningLabels[warning.code] ?? warning.code}: {fields.find(([key]) => key === warning.field)?.[1] ?? warning.field}</li>)}</ul></div>}
    <form action={confirm} className="mt-4 grid gap-4">
      <input type="hidden" name="ocr_job_id" value={data.ocr_job_id} />
      <input type="hidden" name="expected_revision" value={data.revision} />
      {data.review_enabled && <input type="hidden" name="expected_draft_revision" value={data.saved_review?.revision ?? 0} />}
      <div className="grid gap-4 md:grid-cols-2">{fields.map(([key, label]) => {
        const field = data.draft.fields[key];
        return <div key={key} className="rounded-xl border border-line p-3">
          <label htmlFor={`extraction-${key}`} className="block font-semibold">{label}</label>
          <p className="mt-1 break-words text-sm text-muted">ข้อความดิบ: {field?.raw || "ไม่พบ"}</p>
          <p className="break-words text-sm text-muted">ค่ามาตรฐานที่ระบบเสนอ: {field?.normalized || "ไม่มี"}</p>
          <p className="text-xs text-muted">ความมั่นใจรายฟิลด์: ยังไม่ผ่านการปรับเทียบ{field?.evidence?.length ? ` · หน้า ${field.evidence[0].page} บรรทัด ${field.evidence[0].line}` : ""}</p>
          {field?.evidence?.length > 0 && <a className="text-sm font-semibold text-brand-strong underline-offset-2 hover:underline" href={`/api/o/${encodeURIComponent(organization)}/documents/${encodeURIComponent(document)}/original?preview=1#page=${field.evidence[0].page}`} target="_blank" rel="noopener noreferrer" aria-label={`เปิดหลักฐานหน้า ${field.evidence[0].page} บรรทัด ${field.evidence[0].line} ของ ${label}`}>เปิดหน้าต้นฉบับ ↗</a>}
          {(canConfirm || data.review_enabled) && <><input id={`extraction-${key}`} name={key} maxLength={240} defaultValue={data.saved_review?.values[key] ?? data.confirmed?.values[key] ?? field?.normalized ?? ""} className="mt-2 min-h-11 w-full rounded-xl border border-line bg-surface px-3 text-fg" />
            {data.review_enabled && <label className="mt-2 grid gap-1 text-sm">การตัดสินใจ
              <select name={`${key}_decision`} defaultValue={data.saved_review?.decisions[key] ?? ""} className="min-h-11 rounded-xl border border-line bg-surface px-3">
                <option value="">ยังไม่ตัดสินใจ</option><option value="accepted">ยอมรับค่าที่เสนอ</option><option value="corrected">แก้ไขจากต้นฉบับ</option><option value="unknown">ไม่ทราบค่า</option>
              </select>
            </label>}</>}
        </div>;
      })}</div>
      {data.review_enabled && <div className="flex flex-wrap gap-3"><button id="review-save" formAction={saveDraft} formNoValidate type="submit" className="secondary-button w-fit">บันทึกฉบับร่าง</button>{nextHref && <button formAction={saveDraft} formNoValidate name="save_next" value="1" type="submit" className="secondary-button w-fit">บันทึกและไป{nextHref.includes("/review?") ? "คิว" : "เอกสารถัดไป"}</button>}</div>}
      {canConfirm && <><label className="flex items-start gap-2 text-sm"><input required type="checkbox" name="review_ack" value="1" className="mt-1" />ฉันตรวจเทียบค่ากับเอกสารต้นฉบับแล้ว และยืนยันค่าที่กรอก</label><button type="submit" className="button w-fit">ยืนยันข้อมูลที่ตรวจแล้ว</button></>}
    </form>
    {data.review_enabled && canConfirm && <form action={returnReview} className="mt-5 grid max-w-xl gap-3 border-t border-line pt-4">
      <h3 className="font-semibold">ส่งกลับเพื่อแก้ไข</h3>
      <input type="hidden" name="ocr_job_id" value={data.ocr_job_id} /><input type="hidden" name="review_revision" value={data.revision} /><input type="hidden" name="draft_revision" value={data.saved_review?.revision ?? 0} />
      <label className="grid gap-1">เหตุผล<select name="reason_code" required className="min-h-11 rounded-xl border border-line px-3"><option value="incorrect_value">ข้อมูลไม่ถูกต้อง</option><option value="missing_value">ข้อมูลไม่ครบ</option><option value="unreadable_original">ต้นฉบับอ่านไม่ได้</option><option value="other">อื่น ๆ</option></select></label>
      <label className="grid gap-1">บันทึกส่วนตัว<textarea name="private_note" maxLength={1000} className="min-h-24 rounded-xl border border-line px-3 py-2" /></label>
      <button type="submit" className="secondary-button w-fit">ส่งกลับเพื่อแก้ไข</button>
    </form>}
    {data.review_enabled && canConfirm && <form action={reprocess} className="mt-5 grid max-w-xl gap-3 border-t border-line pt-4">
      <h3 className="font-semibold">ประมวลผล OCR อีกครั้ง</h3><p className="text-sm text-muted">คำขอนี้ทำให้ข้อมูลที่เคยอนุมัติใช้ส่งออกไม่ได้จนกว่าจะตรวจและอนุมัติใหม่</p>
	  <input type="hidden" name="ocr_job_id" value={data.ocr_job_id} /><input type="hidden" name="draft_revision" value={data.saved_review?.revision ?? 0} />
      <label className="grid gap-1">เหตุผล<select name="reason_code" required className="min-h-11 rounded-xl border border-line px-3"><option value="quality_issue">อ่านข้อมูลผิด</option><option value="missing_page">อ่านหน้าไม่ครบ</option><option value="other">อื่น ๆ</option></select></label>
      <label className="grid gap-1">บันทึกส่วนตัว<textarea name="private_note" maxLength={1000} className="min-h-24 rounded-xl border border-line px-3 py-2" /></label>
      <button type="submit" className="secondary-button w-fit">เริ่มประมวลผลใหม่</button>
    </form>}
	{data.review_enabled && !!data.review_history?.length && <details className="mt-5 rounded-xl border border-line p-4"><summary className="cursor-pointer font-semibold">ประวัติการตรวจ {data.review_history.length} ฉบับล่าสุด</summary><ol className="mt-3 grid gap-3">{data.review_history.map((item) => <li key={item.revision} className="rounded-lg border border-line p-3"><p className="text-sm font-semibold">ฉบับ {item.revision} · {new Date(item.updated_at).toLocaleString("th-TH")}</p><ul className="mt-2 list-disc pl-5 text-sm">{fields.filter(([key]) => item.decisions[key] === "corrected" || item.decisions[key] === "unknown").map(([key, label]) => <li key={key}>{label}: {item.decisions[key] === "unknown" ? "ไม่ทราบค่า" : item.values[key]}</li>)}</ul></li>)}</ol></details>}
  </section>;
}
