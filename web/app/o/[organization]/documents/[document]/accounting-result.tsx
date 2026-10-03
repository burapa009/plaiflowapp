"use client";

type Value = string | number | boolean | null;
export type AccountingResult = {
  document: Record<string, Value>;
  seller: Record<string, Value>;
  buyer: Record<string, Value>;
  items: Record<string, Value>[];
  summary: Record<string, Value>;
  payment: Record<string, Value>;
  confidence: Record<string, Value>;
  validation: Record<string, Value>;
  warnings: string[];
  raw_text: string;
};

const labels: Record<string, string> = {
  document_type: "ชนิดเอกสาร", document_number: "เลขที่เอกสาร", document_date: "วันที่เอกสาร", due_date: "วันครบกำหนด", currency: "สกุลเงิน",
  name: "ชื่อ", branch: "สาขา", address: "ที่อยู่", tax_id: "เลขผู้เสียภาษี", phone: "โทรศัพท์",
  description: "สินค้า / บริการ", quantity: "จำนวน", unit: "หน่วย", unit_price: "ราคาต่อหน่วย", discount: "ส่วนลด", amount: "ยอดรายการ",
  subtotal: "รวมรายการ", amount_before_vat: "ยอดก่อน VAT", vat_rate: "VAT (%)", vat_amount: "ภาษีมูลค่าเพิ่ม",
  withholding_tax: "ภาษีหัก ณ ที่จ่าย", service_charge: "ค่าบริการ", other_charges: "ค่าใช้จ่ายอื่น", total_amount: "ยอดสุทธิ", paid_amount: "รับชำระ", change_amount: "เงินทอน",
  payment_method: "วิธีชำระเงิน", bank_name: "ธนาคาร", transaction_reference: "เลขอ้างอิงการชำระ",
};

export default function AccountingResultPanel({ result }: { result: AccountingResult }) {
  function download() {
    const url = URL.createObjectURL(new Blob([JSON.stringify(result, null, 2)], { type: "application/json;charset=utf-8" }));
    const link = document.createElement("a");
    link.href = url;
    link.download = "ocr-accounting-draft.json";
    link.click();
    setTimeout(() => URL.revokeObjectURL(url), 1000);
  }

  return <details className="mt-4 rounded-xl border border-line p-4">
    <summary className="cursor-pointer font-semibold">ข้อมูลบัญชีแบบละเอียดจาก OCR</summary>
    <p className="mt-3 text-sm text-muted">ผลอ่านอัตโนมัติ ยังไม่ผ่านการยืนยัน · ช่องว่างหมายถึงไม่พบหรือไม่มั่นใจ · ระดับความมั่นใจเป็นคุณภาพหลักฐาน ไม่ใช่ค่าความแม่นยำที่ผ่านการปรับเทียบ</p>
    <div className="mt-4 grid gap-4 md:grid-cols-2">{([
      ["เอกสาร", result.document], ["ผู้ขาย", result.seller], ["ผู้ซื้อ", result.buyer], ["ยอดเงิน", result.summary], ["การชำระเงิน", result.payment],
    ] as const).map(([title, values]) => <section key={title} className="rounded-xl border border-line p-3">
      <h3 className="font-semibold">{title}</h3>
      <dl className="mt-2 grid gap-2 text-sm">{Object.entries(values).map(([key, value]) => <div key={key} className="grid grid-cols-2 gap-2"><dt className="text-muted">{labels[key] ?? key}</dt><dd className="break-words">{value === null ? "—" : String(value)}</dd></div>)}</dl>
    </section>)}</div>
    <h3 className="mt-4 font-semibold">รายการสินค้าและบริการ</h3>
    {result.items.length ? <div className="mt-2 overflow-x-auto"><table className="w-full text-left text-sm"><thead><tr>{["description", "quantity", "unit", "unit_price", "discount", "amount"].map(key => <th key={key} className="p-2">{labels[key]}</th>)}</tr></thead><tbody>{result.items.map((item, index) => <tr key={index}>{["description", "quantity", "unit", "unit_price", "discount", "amount"].map(key => <td key={key} className="border-t border-line p-2">{item[key] == null ? "—" : String(item[key])}</td>)}</tr>)}</tbody></table></div> : <p className="mt-2 text-sm text-muted">ไม่พบรายการที่แยกคอลัมน์ได้อย่างมั่นใจ</p>}
    {result.warnings.length > 0 && <ul className="mt-4 list-disc rounded-xl bg-amber-50 p-4 pl-8 text-sm text-amber-950" aria-label="ข้อควรตรวจสอบ">{result.warnings.map((warning, index) => <li key={index}>{warning}</li>)}</ul>}
    <button type="button" onClick={download} className="secondary-button mt-4">ดาวน์โหลด JSON ฉบับร่าง</button>
    <details className="mt-3"><summary className="cursor-pointer text-sm">JSON และหลักฐานต้นฉบับ</summary><pre className="mt-2 max-h-96 overflow-auto whitespace-pre-wrap break-words rounded-xl border border-line p-3 text-xs">{JSON.stringify(result, null, 2)}</pre></details>
  </details>;
}
