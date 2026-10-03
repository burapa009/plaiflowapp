import { sessionPOST } from "@/lib/session-api";
import { revalidatePath } from "next/cache";
import { redirect } from "next/navigation";
import Link from "next/link";
import AccountingResultPanel, { type AccountingResult } from "./accounting-result";
import ReviewTabs from "./review-tabs";
import OCRItems from "./ocr-items";
import { ReviewForm, ReviewSubmit, ReviewTotal, type ReviewState } from "./review-form";

type Field = { presence: string; raw: string; normalized: string; confidence: string; evidence: { page: number; line: number }[] };
type Warning = { code: string; field: string; severity: string };
export type Extraction = {
  draft: { document_type: string; fields: Record<string, Field>; warnings: Warning[]; accounting?: AccountingResult; classification?: { document_type: string; effective_type: string; confidence: number; requires_review: boolean; candidate_types: { type: string; confidence: number }[]; signals: string[] } };
  ocr_job_id: string;
  revision: number;
  source_superseded: boolean;
  possible_duplicate?: { possible_duplicate: boolean; matched_document_id: string; duplicate_confidence: number } | null;
  confirmed: { values: Record<string, string>; confirmed_at: string } | null;
  review_enabled?: boolean;
  saved_review?: { revision: number; values: Record<string, string>; decisions: Record<string, string> } | null;
	returned_review?: { reason_code: string; private_note: string; returned_at: string } | null;
	review_history?: { revision: number; values: Record<string, string>; decisions: Record<string, string>; updated_by: string; updated_at: string }[];
};

const fields: [string, string][] = [
  ["document_number", "เลขที่เอกสาร / ใบกำกับภาษี"], ["issue_date", "วันที่เอกสาร"],
  ["seller_name", "ชื่อผู้ขาย"], ["seller_tax_id", "เลขผู้เสียภาษีผู้ขาย"], ["seller_branch", "สาขาผู้ขาย (5 หลัก; เว้นว่างถ้าไม่ทราบ)"],
  ["buyer_name", "ชื่อผู้ซื้อ"], ["buyer_tax_id", "เลขผู้เสียภาษีผู้ซื้อ"],
  ["currency", "สกุลเงิน"], ["subtotal", "ยอดก่อนภาษี"],
  ["vat_amount", "ภาษีมูลค่าเพิ่ม"], ["total_amount", "ยอดรวมสุทธิ"],
];

const warningLabels: Record<string, string> = {
  required_missing: "ไม่พบข้อมูลจำเป็น", multiple_candidates: "พบค่าหลายแบบ ต้องเลือกเอง",
  ocr_low_quality: "OCR อ่านบรรทัดนี้ไม่มั่นใจ", amount_mismatch: "ยอดเงินไม่สัมพันธ์กัน",
  invalid_value: "พบข้อความแต่แปลงค่าไม่ได้",
};
export const documentTypeLabels: Record<string, string> = {
  tax_invoice: "ใบกำกับภาษี", receipt: "ใบเสร็จรับเงิน", pre_receipt: "ใบเสร็จก่อนรับเงิน", invoice: "ใบแจ้งหนี้", unknown: "ยังระบุไม่ได้",
  tax_invoice_receipt: "ใบกำกับภาษี / ใบเสร็จรับเงิน",
  billing_note: "ใบวางบิล", credit_note: "ใบลดหนี้", debit_note: "ใบเพิ่มหนี้",
  withholding_tax_certificate: "หนังสือรับรองหัก ณ ที่จ่าย", payment_voucher: "ใบสำคัญจ่าย", receipt_voucher: "ใบสำคัญรับเงิน",
  receipt_substitute: "ใบแทนใบเสร็จรับเงิน", expense_claim: "ใบเบิกค่าใช้จ่าย", petty_cash: "เอกสารเงินสดย่อย",
  quotation: "ใบเสนอราคา", purchase_order: "ใบสั่งซื้อ", bank_slip: "หลักฐานการโอนเงิน", bank_statement: "รายการเดินบัญชี",
};

export default async function ExtractionPanel({ organization, document, nextHref = "", data, role, cancelHref, organizationName = "", branchType = "", preview = false, previewURL = "", result = "", attachmentCount = 0 }: {
  organization: string; document: string; nextHref?: string; data: Extraction; role: string; cancelHref: string; organizationName?: string; branchType?: string; preview?: boolean; previewURL?: string; result?: string; attachmentCount?: number;
}) {
  const path = `/o/${encodeURIComponent(organization)}/documents/${encodeURIComponent(document)}`;
  const canConfirm = role === "Owner" || role === "Admin";
  const originalURL = previewURL || `/api/o/${encodeURIComponent(organization)}/documents/${encodeURIComponent(document)}/original?preview=1`;

  async function submitReview(_state: ReviewState, formData: FormData): Promise<ReviewState> {
    "use server";
    if (preview) return { error: "หน้าพรีวิวใช้ข้อมูลจำลองและไม่บันทึก กรุณาเปิดเอกสารจริงเพื่อแก้ไข" };
    const intent = String(formData.get("intent") ?? "confirm");
    const saveDraft = intent === "draft" || intent === "draft-next";
    const body = new URLSearchParams();
    for (const [key] of fields) {
      body.set(key, String(formData.get(key) ?? ""));
      body.set(`${key}_decision`, String(formData.get(`${key}_decision`) ?? ""));
    }
    body.set("ocr_job_id", String(formData.get("ocr_job_id") ?? ""));
    body.set("expected_draft_revision", String(formData.get("expected_draft_revision") ?? "0"));
    if (!saveDraft) {
      body.set("expected_revision", String(formData.get("expected_revision") ?? "0"));
      body.set("review_ack", String(formData.get("review_ack") ?? ""));
    }
    const result = await sessionPOST(`/v1${path}/extraction/${saveDraft ? "draft" : "confirm"}`, body);
    if (!result?.ok) return { error: result?.status === 409 ? "ข้อมูล OCR หรือฉบับร่างเปลี่ยนแล้ว กรุณาตรวจและโหลดหน้าใหม่ก่อนบันทึก" : result?.status === 422 ? "ข้อมูลยังไม่ผ่านการตรวจสอบ กรุณาตรวจช่องที่กรอกและการตัดสินใจ" : "บันทึกไม่สำเร็จ กรุณาลองอีกครั้ง" };
    revalidatePath(path);
    if (intent === "draft-next" && nextHref) redirect(nextHref);
    redirect(`${path}?extraction=${saveDraft ? "draft-saved" : "confirmed"}`);
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

  async function correctType(formData: FormData) {
    "use server";
    if (preview) return;
    const body = new URLSearchParams({
      ocr_job_id: data.ocr_job_id,
      expected_type: data.draft.classification?.effective_type ?? "",
      document_type: String(formData.get("document_type") ?? ""),
    });
    const response = await sessionPOST(`/v1${path}/document-type`, body);
    if (!response?.ok) redirect(`${path}?extraction=type-error`);
    revalidatePath(path);
    redirect(`${path}?extraction=type-saved`);
  }

  const amountKeys = new Set(["subtotal", "vat_amount", "total_amount"]);
  const requiredKeys = new Set([
    ...(["receipt", "invoice", "tax_invoice", "tax_invoice_receipt", "billing_note", "credit_note", "debit_note"].includes(data.draft.document_type) ? ["issue_date", "total_amount"] : []),
    ...(["tax_invoice", "tax_invoice_receipt"].includes(data.draft.document_type) ? ["document_number", "seller_name", "seller_tax_id"] : []),
  ]);
  const needsAttention = new Set(data.draft.warnings.map((warning) => warning.field));
  const editable = canConfirm || data.review_enabled;
  const accountingItems = data.draft.accounting?.items ?? [];
  const accountingSummary = data.draft.accounting?.summary;
  const currency = data.saved_review?.values.currency ?? data.confirmed?.values.currency ?? data.draft.fields.currency?.normalized ?? "";
  const display = (value: string | number | boolean | null | undefined) => value === null || value === undefined || value === "" ? "—" : String(value);
  const displayMoney = (value: string | number | boolean | null | undefined) => typeof value === "number" ? value.toLocaleString("en-US", { minimumFractionDigits: 2, maximumFractionDigits: 2 }) : display(value);

  function renderField([key, label]: [string, string]) {
    const field = data.draft.fields[key];
    const value = data.saved_review?.values[key] ?? data.confirmed?.values[key] ?? field?.normalized ?? "";
    return <div key={key} className={`ocr-review-field${needsAttention.has(key) ? " needs-attention" : ""}`}>
      <div className="ocr-review-field-heading">{editable ? <label htmlFor={`extraction-${key}`}>{label}{requiredKeys.has(key) && <span aria-hidden="true"> *</span>}</label> : <span>{label}</span>}{needsAttention.has(key) && <span className="ocr-review-field-warning">ต้องตรวจ</span>}</div>
      {editable ? <input id={`extraction-${key}`} name={key} type={key === "issue_date" && (!value || /^\d{4}-\d{2}-\d{2}$/.test(value)) ? "date" : "text"} aria-required={requiredKeys.has(key)} maxLength={240} inputMode={amountKeys.has(key) ? "decimal" : undefined} defaultValue={value} className={amountKeys.has(key) ? "ocr-review-money-input" : ""} /> : <p className="ocr-review-readonly-value">{display(value)}</p>}
      {data.review_enabled && <label className="ocr-review-decision">การตัดสินใจ<select name={`${key}_decision`} defaultValue={data.saved_review?.decisions[key] ?? ""}>
        <option value="">ยังไม่ตัดสินใจ</option><option value="accepted">ยอมรับค่าที่เสนอ</option><option value="corrected">แก้ไขจากต้นฉบับ</option><option value="unknown">ไม่ทราบค่า</option>
      </select></label>}
    </div>;
  }

  const documentPanel = <div className="ocr-review-panel-content">
    {result === "draft-saved" && <p className="ocr-review-success" role="status">บันทึกการเปลี่ยนแปลงแล้ว</p>}
    {data.confirmed && !data.source_superseded && <p className="ocr-review-success" role="status">ยืนยันข้อมูลแล้วเมื่อ {new Date(data.confirmed.confirmed_at).toLocaleString("th-TH")}</p>}
    {data.source_superseded && <p className="ocr-review-alert" role="alert">OCR เปลี่ยนหลังการยืนยันครั้งก่อน กรุณาตรวจทานใหม่ก่อนส่งออก</p>}
    {data.returned_review && <div className="ocr-review-alert" role="status"><strong>ส่งกลับเพื่อแก้ไข: {({ missing_value: "ข้อมูลไม่ครบ", incorrect_value: "ข้อมูลไม่ถูกต้อง", unreadable_original: "ต้นฉบับอ่านไม่ได้", other: "อื่น ๆ" } as Record<string, string>)[data.returned_review.reason_code] ?? data.returned_review.reason_code}</strong>{data.returned_review.private_note && <p className="whitespace-pre-wrap">{data.returned_review.private_note}</p>}</div>}
    {data.draft.warnings.length > 0 && <div className="ocr-review-alert" role="status"><h3>รายการที่ต้องตรวจ</h3><ul>{data.draft.warnings.map((warning, index) => <li key={`${warning.code}-${warning.field}-${index}`}>{warningLabels[warning.code] ?? warning.code}: {fields.find(([key]) => key === warning.field)?.[1] ?? warning.field}</li>)}</ul></div>}

    <section className="ocr-review-card ocr-review-business-card"><h3>ธุรกิจ</h3><p className="ocr-review-business-name">{organizationName || "ยังไม่มีชื่อธุรกิจ"}</p>{branchType && <p className="ocr-review-card-intro">{({ head: "สำนักงานใหญ่", branch: "สาขา", none: "ไม่มีสาขา" } as Record<string, string>)[branchType] ?? branchType}</p>}</section>
    <section className="ocr-review-card ocr-review-evidence-card"><div className="ocr-review-card-heading"><div><h3>เอกสารหลักฐาน</h3><p className="ocr-review-card-intro">เอกสารประกอบเพิ่มเติม</p></div><a className="ocr-review-original-link" href={originalURL} target="_blank" rel="noopener noreferrer">เปิดเอกสารต้นฉบับ ↗</a></div><div className="ocr-review-evidence-empty"><span aria-hidden="true">▤</span><strong>{attachmentCount > 0 ? `แนบเอกสารเพิ่มเติมแล้ว ${attachmentCount} ไฟล์` : "ยังไม่มีเอกสารประกอบเพิ่มเติม"}</strong></div></section>
    <section className="ocr-review-card"><h3>ข้อมูลรายจ่าย</h3><p className="ocr-review-card-intro">ประเภทที่ OCR อ่านได้: {documentTypeLabels[data.draft.document_type] ?? (data.draft.document_type || "ยังระบุไม่ได้")}</p><div className="ocr-review-fields">{fields.filter(([key]) => ["document_number", "issue_date", "currency"].includes(key)).map(renderField)}</div>{data.draft.accounting?.summary.paid_amount != null && <p className="ocr-review-card-intro">ยอดรับชำระที่ OCR อ่านได้: {displayMoney(data.draft.accounting.summary.paid_amount)} {currency}</p>}</section>
    <section className="ocr-review-card"><div className="ocr-review-card-heading"><h3>ข้อมูลผู้ขาย</h3>{!preview && <Link className="ocr-review-original-link" href={`/o/${encodeURIComponent(organization)}/vendors`} target="_blank" rel="noopener noreferrer">ค้นหาผู้ขาย ↗</Link>}</div><div className="ocr-review-fields">{fields.filter(([key]) => key.startsWith("seller_")).map(renderField)}</div><div className="ocr-review-readonly-block"><label htmlFor="review-seller-address">ที่อยู่ที่ OCR อ่านได้ · อ่านอย่างเดียว</label><textarea id="review-seller-address" readOnly value={display(data.draft.accounting?.seller.address)} rows={3} /></div></section>
  </div>;
  const amountsPanel = <div className="ocr-review-panel-content">
    <section className="ocr-review-card ocr-review-items-card"><p className="ocr-review-item-info">รายการค่าใช้จ่ายจาก OCR ใช้ประกอบการตรวจ <span>{accountingItems.length} รายการ · {currency || "ไม่ระบุสกุลเงิน"}</span></p>
      <OCRItems items={accountingItems} currency={currency} />
    </section>
    <section className="ocr-review-card"><h3>สรุปรวมค่าใช้จ่าย</h3><p className="ocr-review-card-intro">ตัวเลขจาก OCR และค่าที่ตรวจแก้ได้ · ไม่มีการคำนวณภาษีใหม่ในหน้านี้</p><div className="ocr-review-summary-fields">{fields.filter(([key]) => amountKeys.has(key)).map(renderField)}</div>
      {accountingSummary && <dl className="ocr-review-summary-extra">{([
        ["discount", "ส่วนลดรวม"], ["amount_before_vat", "ยอดก่อน VAT ที่ OCR อ่านได้"], ["vat_rate", "อัตรา VAT (%)"],
        ["withholding_tax", "ภาษีหัก ณ ที่จ่าย"], ["service_charge", "ค่าบริการ"], ["other_charges", "ค่าใช้จ่ายอื่น"],
      ] as const).filter(([key]) => accountingSummary[key] != null).map(([key, label]) => <div key={key}><dt>{label}</dt><dd>{displayMoney(accountingSummary[key])}{key !== "vat_rate" ? ` ${currency}` : ""}</dd></div>)}</dl>}
    </section>
    {data.draft.accounting && <AccountingResultPanel result={data.draft.accounting} />}
    <details className="ocr-review-card"><summary>ข้อมูลผู้ซื้อ</summary><div className="ocr-review-fields">{fields.filter(([key]) => key.startsWith("buyer_")).map(renderField)}</div></details>
  </div>;

  return <section className="ocr-review-panel" aria-labelledby="extraction-heading">
    <h2 className="sr-only" id="extraction-heading">ตรวจข้อมูลจาก OCR</h2>
    {data.possible_duplicate && <p className="ocr-review-alert" role="status">อาจเป็นเอกสารซ้ำ · ความมั่นใจ {Math.round(data.possible_duplicate.duplicate_confidence * 100)}% <Link href={`/o/${encodeURIComponent(organization)}/documents/${encodeURIComponent(data.possible_duplicate.matched_document_id)}`}>ดูเอกสารที่อาจซ้ำ</Link></p>}
    {data.draft.classification && <section className="ocr-review-card" aria-labelledby="document-type-heading">
      <h3 id="document-type-heading">ประเภทเอกสาร</h3>
      <p className="ocr-review-card-intro">ระบบอ่านว่า {documentTypeLabels[data.draft.classification.document_type] ?? "ยังระบุไม่ได้"} · ความมั่นใจ {Math.round(data.draft.classification.confidence * 100)}%</p>
      {data.draft.classification.requires_review && <p className="ocr-review-alert" role="status">โปรดตรวจประเภทเอกสารกับต้นฉบับก่อนยืนยัน</p>}
      {data.draft.classification.signals.length > 0 && <p className="ocr-review-card-intro">หลักฐาน: {data.draft.classification.signals.join(" · ")}</p>}
      {result === "type-saved" && <p className="ocr-review-success" role="status">บันทึกประเภทเอกสารแล้ว</p>}
      {result === "type-error" && <p className="ocr-review-alert" role="alert">เปลี่ยนประเภทไม่สำเร็จ กรุณาโหลดหน้าใหม่และลองอีกครั้ง</p>}
      {canConfirm && !data.confirmed && !preview ? <form action={correctType} className="field"><label htmlFor="document-type-select">แก้ประเภทเอกสาร</label><select id="document-type-select" name="document_type" defaultValue={data.draft.classification.effective_type}>{Object.entries(documentTypeLabels).map(([value, label]) => <option key={value} value={value}>{label}</option>)}</select><button className="secondary-button" type="submit">บันทึกประเภท</button></form> : <p className="ocr-review-card-intro">ประเภทที่ใช้: {documentTypeLabels[data.draft.classification.effective_type] ?? "ยังระบุไม่ได้"}</p>}
    </section>}
    <ReviewForm action={submitReview}>
      <input type="hidden" name="ocr_job_id" value={data.ocr_job_id} />
      <input type="hidden" name="expected_revision" value={data.revision} />
      {data.review_enabled && <input type="hidden" name="expected_draft_revision" value={data.saved_review?.revision ?? 0} />}
      <ReviewTabs documentPanel={documentPanel} amountsPanel={amountsPanel} />
      <div className="ocr-review-actions">
        <ReviewTotal initial={data.saved_review?.values.total_amount ?? data.confirmed?.values.total_amount ?? data.draft.fields.total_amount?.normalized ?? ""} currency={currency} />
        <details className="ocr-review-footer-menu"><summary aria-label="เมนูเอกสาร" title="เมนูเอกสาร">⋮</summary><div><a href={originalURL} target="_blank" rel="noopener noreferrer">เปิดเอกสารต้นฉบับ ↗</a></div></details>
        {canConfirm && <label className="ocr-review-ack"><input required type="checkbox" name="review_ack" value="1" />ฉันตรวจเทียบค่ากับเอกสารต้นฉบับแล้ว และยืนยันค่าที่กรอก</label>}
        {data.review_enabled && <p className="ocr-review-draft-help">ตัดสินใจแต่ละช่องและบันทึกฉบับร่างก่อนยืนยัน</p>}
        <div className="ocr-review-action-buttons"><Link data-review-exit className="secondary-button" href={cancelHref}>ยกเลิก</Link>
          {data.review_enabled && <ReviewSubmit id="review-save" intent="draft" skipValidation className="button ocr-review-save-button"><svg aria-hidden="true" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" width="16" height="16"><path d="M4 3h14l3 3v15H4zM7 3v6h10V3M7 21v-8h11v8" /></svg>บันทึกการเปลี่ยนแปลง</ReviewSubmit>}
          {data.review_enabled && nextHref && <ReviewSubmit intent="draft-next" skipValidation className="secondary-button">บันทึกและไป{nextHref.includes("/review?") ? "คิว" : "เอกสารถัดไป"}</ReviewSubmit>}
          {canConfirm && <ReviewSubmit intent="confirm" className="button">บันทึกและยืนยันข้อมูล</ReviewSubmit>}
        </div>
      </div>
    </ReviewForm>
    {(data.review_enabled && (canConfirm || !!data.review_history?.length)) && <details className="ocr-review-tools"><summary aria-label="การตรวจเพิ่มเติม"><span aria-hidden="true">⋯</span><span>การตรวจเพิ่มเติม</span></summary>
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
    </details>}
  </section>;
}
