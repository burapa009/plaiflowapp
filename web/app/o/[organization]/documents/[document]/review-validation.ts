export type FieldError = { id: string; label: string; message: string };
export function validateReviewField(key: string, value: string, required: boolean): string {
 value = value.trim();
 if (!value) return required ? "กรุณากรอกข้อมูลจำเป็น" : "";
 if ((key === "seller_tax_id" || key === "buyer_tax_id") && !/^\d{13}$/.test(value)) return "เลขผู้เสียภาษีต้องเป็นตัวเลข 13 หลัก";
 if (key === "seller_branch" && !/^\d{5}$/.test(value)) return "รหัสสาขาต้องเป็นตัวเลข 5 หลัก";
 if (key === "issue_date") {
  const date = new Date(value + "T00:00:00Z");
  if (!/^\d{4}-\d{2}-\d{2}$/.test(value) || !Number.isFinite(date.getTime()) || date.toISOString().slice(0, 10) !== value) return "วันที่ต้องถูกต้องในรูปแบบ YYYY-MM-DD";
 }
 if (["subtotal", "vat_amount", "total_amount"].includes(key) && !/^\d+\.\d{2}$/.test(value)) return "กรอกยอดเป็นตัวเลขพร้อมทศนิยม 2 ตำแหน่ง เช่น 1000.00";
 if (key === "currency" && !["THB", "USD", "EUR", "GBP", "JPY", "CNY", "SGD", "MYR", "VND", "KRW", "AUD", "HKD", "TWD", "IDR", "PHP", "LAK"].includes(value)) return "กรอกสกุลเงินที่รองรับ เช่น THB หรือเว้นว่างถ้าไม่ทราบ";
 return "";
}
