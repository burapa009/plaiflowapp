import { sessionPOST } from "@/lib/session-api";
import { revalidatePath } from "next/cache";
import { redirect } from "next/navigation";
import Link from "next/link";
import AccountingResultPanel, { type AccountingResult } from "./accounting-result";
import ReviewTabs from "./review-tabs";

type Field = { presence: string; raw: string; normalized: string; confidence: string; evidence: { page: number; line: number }[] };
type Warning = { code: string; field: string; severity: string };
export type Extraction = {
  draft: { document_type: string; fields: Record<string, Field>; warnings: Warning[]; accounting?: AccountingResult };
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
const documentTypeLabels: Record<string, string> = {
  tax_invoice: "ใบกำกับภาษี", receipt: "ใบเสร็จรับเงิน", invoice: "ใบแจ้งหนี้", receipt_tax_invoice: "ใบเสร็จรับเงิน / ใบกำกับภาษี", unknown: "ยังระบุไม่ได้",
};

export default async function ExtractionPanel({ organization, document, nextHref = "", data, role, cancelHref }: {
  organization: string; document: string; nextHref?: string; data: Extraction; role: string; cancelHref: string;
}) {
  const path = `/o/${encodeURIComponent(organization)}/documents/${encodeURIComponent(document)}`;
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

  const amountKeys = new Set(["subtotal", "vat_amount", "total_amount"]);
  const needsAttention = new Set(data.draft.warnings.map((warning) => warning.field));
  const editable = canConfirm || data.review_enabled;
  const accountingItems = data.draft.accounting?.items ?? [];
  const display = (value: string | number | boolean | null | undefined) => value === null || value === undefined || value === "" ? "—" : String(value);
  const displayMoney = (value: string | number | boolean | null | undefined) => typeof value === "number" ? value.toLocaleString("en-US", { minimumFractionDigits: 2, maximumFractionDigits: 2 }) : display(value);

  function renderField([key, label]: [string, string]) {
    const field = data.draft.fields[key];
    const value = data.saved_review?.values[key] ?? data.confirmed?.values[key] ?? field?.normalized ?? "";
    return <div key={key} className={`ocr-review-field${needsAttention.has(key) ? " needs-attention" : ""}`}>
      <div className="ocr-review-field-heading">{editable ? <label htmlFor={`extraction-${key}`}>{label}</label> : <span>{label}</span>}{needsAttention.has(key) && <span className="ocr-review-field-warning">ต้องตรวจ</span>}</div>
      {editable ? <input id={`extraction-${key}`} name={key} maxLength={240} inputMode={amountKeys.has(key) ? "decimal" : undefined} defaultValue={value} className={amountKeys.has(key) ? "ocr-review-money-input" : ""} /> : <p className="ocr-review-readonly-value">{display(value)}</p>}
      {data.review_enabled && <label className="ocr-review-decision">การตัดสินใจ<select name={`${key}_decision`} defaultValue={data.saved_review?.decisions[key] ?? ""}>
        <option value="">ยังไม่ตัดสินใจ</option><option value="accepted">ยอมรับค่าที่เสนอ</option><option value="corrected">แก้ไขจากต้นฉบับ</option><option value="unknown">ไม่ทราบค่า</option>
      </select></label>}
      <details className="ocr-review-evidence"><summary>ข้อความ OCR และหลักฐาน</summary><p>ข้อความดิบ: {field?.raw || "ไม่พบ"}</p><p>ค่าที่ระบบเสนอ: {field?.normalized || "ไม่มี"}</p><p>ความมั่นใจรายฟิลด์ยังไม่ผ่านการปรับเทียบ</p>
        {field?.evidence?.length > 0 && <a href={`/api/o/${encodeURIComponent(organization)}/documents/${encodeURIComponent(document)}/original?preview=1#page=${field.evidence[0].page}`} target="_blank" rel="noopener noreferrer" aria-label={`เปิดหลักฐานหน้า ${field.evidence[0].page} บรรทัด ${field.evidence[0].line} ของ ${label}`}>เปิดต้นฉบับหน้า {field.evidence[0].page} บรรทัด {field.evidence[0].line} ↗</a>}
      </details>
    </div>;
  }

  const documentPanel = <div className="ocr-review-panel-content">
    <section className="ocr-review-card"><h3>ข้อมูลเอกสาร</h3><p className="ocr-review-card-intro">ประเภทที่ OCR อ่านได้: {documentTypeLabels[data.draft.document_type] ?? (data.draft.document_type || "ยังระบุไม่ได้")}</p><div className="ocr-review-fields">{fields.filter(([key]) => ["document_number", "issue_date", "currency"].includes(key)).map(renderField)}</div></section>
    <section className="ocr-review-card"><h3>ข้อมูลผู้ขาย</h3><div className="ocr-review-fields">{fields.filter(([key]) => key.startsWith("seller_")).map(renderField)}</div></section>
    <section className="ocr-review-card"><h3>ข้อมูลผู้ซื้อ</h3><div className="ocr-review-fields">{fields.filter(([key]) => key.startsWith("buyer_")).map(renderField)}</div></section>
  </div>;
  const amountsPanel = <div className="ocr-review-panel-content">
    <section className="ocr-review-card"><div className="ocr-review-card-heading"><h3>รายการที่ OCR อ่านได้</h3><span>{accountingItems.length} รายการ</span></div><p className="ocr-review-card-intro">รายการละเอียดเป็นข้อมูลประกอบการตรวจ ยังไม่ใช่รายการรายจ่ายที่บันทึกแล้ว</p>
      {accountingItems.length ? <ol className="ocr-review-items">{accountingItems.map((item, index) => <li key={index}><strong>{display(item.description)}</strong><span>จำนวน {display(item.quantity)}{item.unit ? ` ${item.unit}` : ""}</span><span>ยอดรายการ {displayMoney(item.amount)}</span></li>)}</ol> : <p className="ocr-review-empty">OCR ยังแยกรายการสินค้าและบริการไม่ได้ กรุณาตรวจเอกสารต้นฉบับ</p>}
    </section>
    <section className="ocr-review-card"><h3>ยอดเงินและภาษี</h3><p className="ocr-review-card-intro">สกุลเงิน: {data.draft.fields.currency?.normalized || "ไม่ระบุ"} · ตรวจตัวเลขกับต้นฉบับก่อนยืนยัน</p><div className="ocr-review-fields">{fields.filter(([key]) => amountKeys.has(key)).map(renderField)}</div></section>
    {data.draft.accounting && <AccountingResultPanel result={data.draft.accounting} />}
  </div>;

  return <section className="ocr-review-panel" aria-labelledby="extraction-heading">
    <div className="ocr-review-panel-head"><div><h2 id="extraction-heading">ตรวจข้อมูลจาก OCR</h2><p>ค่าที่ระบบอ่านได้เป็นข้อเสนอ กรุณาเทียบกับต้นฉบับ</p></div><span className="ocr-review-state">{data.confirmed && !data.source_superseded ? "ยืนยันแล้ว" : "รอตรวจข้อมูล"}</span></div>
    {data.confirmed && !data.source_superseded && <p className="ocr-review-success" role="status">ยืนยันข้อมูลแล้วเมื่อ {new Date(data.confirmed.confirmed_at).toLocaleString("th-TH")}</p>}
    {data.source_superseded && <p className="ocr-review-alert" role="alert">OCR เปลี่ยนหลังการยืนยันครั้งก่อน กรุณาตรวจทานใหม่ก่อนส่งออก</p>}
    {data.returned_review && <div className="ocr-review-alert" role="status"><strong>ส่งกลับเพื่อแก้ไข: {({ missing_value: "ข้อมูลไม่ครบ", incorrect_value: "ข้อมูลไม่ถูกต้อง", unreadable_original: "ต้นฉบับอ่านไม่ได้", other: "อื่น ๆ" } as Record<string, string>)[data.returned_review.reason_code] ?? data.returned_review.reason_code}</strong>{data.returned_review.private_note && <p className="whitespace-pre-wrap">{data.returned_review.private_note}</p>}</div>}
    {data.draft.warnings.length > 0 && <div className="ocr-review-alert" role="status"><h3>รายการที่ต้องตรวจ</h3><ul>{data.draft.warnings.map((warning, index) => <li key={`${warning.code}-${warning.field}-${index}`}>{warningLabels[warning.code] ?? warning.code}: {fields.find(([key]) => key === warning.field)?.[1] ?? warning.field}</li>)}</ul></div>}
    <form action={confirm} className="ocr-review-form">
      <input type="hidden" name="ocr_job_id" value={data.ocr_job_id} />
      <input type="hidden" name="expected_revision" value={data.revision} />
      {data.review_enabled && <input type="hidden" name="expected_draft_revision" value={data.saved_review?.revision ?? 0} />}
      <ReviewTabs documentPanel={documentPanel} amountsPanel={amountsPanel} />
      <div className="ocr-review-actions">
        {canConfirm && <label className="ocr-review-ack"><input required type="checkbox" name="review_ack" value="1" />ฉันตรวจเทียบค่ากับเอกสารต้นฉบับแล้ว และยืนยันค่าที่กรอก</label>}
        {data.review_enabled && <p className="ocr-review-draft-help">ตัดสินใจแต่ละช่องและบันทึกฉบับร่างก่อนยืนยัน</p>}
        <div className="ocr-review-action-buttons"><Link className="secondary-button" href={cancelHref}>กลับรายการเอกสาร</Link>
          {data.review_enabled && <button id="review-save" formAction={saveDraft} formNoValidate type="submit" className="secondary-button">บันทึกฉบับร่าง</button>}
          {data.review_enabled && nextHref && <button formAction={saveDraft} formNoValidate name="save_next" value="1" type="submit" className="secondary-button">บันทึกและไป{nextHref.includes("/review?") ? "คิว" : "เอกสารถัดไป"}</button>}
          {canConfirm && <button type="submit" className="button">ยืนยันข้อมูลที่ตรวจแล้ว</button>}
        </div>
      </div>
    </form>
    {data.review_enabled && canConfirm && <details className="ocr-review-secondary"><summary>ส่งกลับหรือประมวลผลใหม่</summary>
      <form action={returnReview} className="ocr-review-secondary-form"><h3>ส่งกลับเพื่อแก้ไข</h3>
        <input type="hidden" name="ocr_job_id" value={data.ocr_job_id} /><input type="hidden" name="review_revision" value={data.revision} /><input type="hidden" name="draft_revision" value={data.saved_review?.revision ?? 0} />
        <label>เหตุผล<select name="reason_code" required><option value="incorrect_value">ข้อมูลไม่ถูกต้อง</option><option value="missing_value">ข้อมูลไม่ครบ</option><option value="unreadable_original">ต้นฉบับอ่านไม่ได้</option><option value="other">อื่น ๆ</option></select></label>
        <label>บันทึกส่วนตัว<textarea name="private_note" maxLength={1000} /></label><button type="submit" className="secondary-button">ส่งกลับเพื่อแก้ไข</button>
      </form>
      <form action={reprocess} className="ocr-review-secondary-form"><h3>ประมวลผล OCR อีกครั้ง</h3><p>คำขอนี้ทำให้ข้อมูลที่เคยอนุมัติใช้ส่งออกไม่ได้จนกว่าจะตรวจและอนุมัติใหม่</p>
        <input type="hidden" name="ocr_job_id" value={data.ocr_job_id} /><input type="hidden" name="draft_revision" value={data.saved_review?.revision ?? 0} />
        <label>เหตุผล<select name="reason_code" required><option value="quality_issue">อ่านข้อมูลผิด</option><option value="missing_page">อ่านหน้าไม่ครบ</option><option value="other">อื่น ๆ</option></select></label>
        <label>บันทึกส่วนตัว<textarea name="private_note" maxLength={1000} /></label><button type="submit" className="secondary-button">เริ่มประมวลผลใหม่</button>
      </form>
    </details>}
    {data.review_enabled && !!data.review_history?.length && <details className="ocr-review-secondary"><summary>ประวัติการตรวจ {data.review_history.length} ฉบับล่าสุด</summary><ol>{data.review_history.map((item) => <li key={item.revision}><p>ฉบับ {item.revision} · {new Date(item.updated_at).toLocaleString("th-TH")}</p><ul>{fields.filter(([key]) => item.decisions[key] === "corrected" || item.decisions[key] === "unknown").map(([key, label]) => <li key={key}>{label}: {item.decisions[key] === "unknown" ? "ไม่ทราบค่า" : item.values[key]}</li>)}</ul></li>)}</ol></details>}
  </section>;
}
