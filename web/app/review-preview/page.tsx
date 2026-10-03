import { notFound } from "next/navigation";
import Link from "next/link";
import ExtractionPanel, { type Extraction } from "../o/[organization]/documents/[document]/extraction-panel";
import DocumentPreview from "../o/[organization]/documents/[document]/document-preview";
import ReviewWorkspace from "../o/[organization]/documents/[document]/review-workspace";

const values: Record<string, string> = {
  document_number: "TEST-0001", issue_date: "2026-09-28", currency: "THB",
  seller_name: "", seller_tax_id: "", seller_branch: "", buyer_name: "", buyer_tax_id: "",
  subtotal: "1000.00", vat_amount: "70.00", total_amount: "1070.00",
};
const data: Extraction = {
  draft: { document_type: "receipt", warnings: [], accounting: {
    document: { document_type: "receipt", document_number: "TEST-0001", document_date: "2026-09-28", currency: "THB" },
    seller: { name: "ร้านตัวอย่าง", address: "ที่อยู่ตัวอย่าง" }, buyer: {},
    items: [{ description: "กระดาษ A4", quantity: 2, unit: "รีม", unit_price: 300, discount: null, amount: 600 }, { description: "หมึกพิมพ์", quantity: 1, unit: "กล่อง", unit_price: 400, discount: null, amount: 400 }],
    summary: { subtotal: 1000, vat_amount: 70, total_amount: 1070 }, payment: {}, confidence: {}, validation: {}, warnings: [], raw_text: "",
  }, fields: Object.fromEntries(Object.entries(values).map(([key, value]) => [key, {
    presence: value ? "present" : "missing", raw: value, normalized: value, confidence: "", evidence: [],
  }])) },
  ocr_job_id: "preview", revision: 0, source_superseded: false, confirmed: null, review_enabled: false,
};

export default function ReviewPreview() {
  if (process.env.NODE_ENV !== "development") notFound();
  return <section className="document-detail-page is-post-ocr">
    <header className="document-detail-header"><div><p className="eyebrow">LOCAL PREVIEW</p><h1>ตรวจข้อมูลเอกสาร</h1><p className="intro">ตัวอย่างใบเสร็จ.png · ข้อมูลจำลองสำหรับตรวจหน้าจอ</p></div><Link className="document-detail-close" href="/local-preview" aria-label="กลับหน้าพรีวิว" title="กลับหน้าพรีวิว">×</Link></header>
    <p role="status">พรีวิวข้อมูลจำลอง · ไม่บันทึกหรือยืนยันข้อมูลจริง</p>
    <ReviewWorkspace viewer={<DocumentPreview src="/review-preview.png" filename="ตัวอย่างใบเสร็จ.png" mime="image/png" documentType="ใบเสร็จรับเงิน" ocrStatus="Completed" />} form={<ExtractionPanel organization="preview" document="preview" data={data} role="Owner" cancelHref="/local-preview" organizationName="ธุรกิจตัวอย่าง" branchType="head" preview previewURL="/review-preview.png" />} />
  </section>;
}
