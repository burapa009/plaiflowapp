export const workflowStages = {
  waiting: { label: "รออ่านเอกสาร", hint: "รับไฟล์แล้ว รอเริ่ม OCR", action: "เปิดเอกสาร" },
  processing: { label: "กำลังอ่าน", hint: "OCR กำลังเตรียมข้อมูล", action: "ดูความคืบหน้า" },
  review: { label: "รอตรวจข้อมูล", hint: "เทียบต้นฉบับก่อนยืนยัน", action: "ตรวจข้อมูล" },
  confirmed: { label: "ตรวจแล้ว", hint: "ยืนยันข้อมูลกับต้นฉบับแล้ว", action: "ดูข้อมูล" },
  error: { label: "ต้องแก้ไข", hint: "เปิดดูปัญหาของเอกสาร", action: "แก้ไขปัญหา" },
  unavailable: { label: "ยังโหลดสถานะไม่ได้", hint: "เปิดเอกสารเพื่อลองใหม่", action: "เปิดเอกสาร" },
  inactive: { label: "ถังขยะ", hint: "กู้คืนเพื่อทำงานต่อ", action: "ดูเอกสาร" },
} as const;
export type WorkflowStage = keyof typeof workflowStages;
export function documentWorkflow(fileStatus: string, ocrStatus?: string, form?: { status: string; source_superseded: boolean }): WorkflowStage {
  if (fileStatus === "Trash") return "inactive";
  if (fileStatus === "Rejected") return "error";
  if (fileStatus === "Checking") return "waiting";
  if (["Failed", "Cancelled"].includes(ocrStatus ?? "")) return "error";
  if (ocrStatus === "NotScheduled") return "waiting";
  if (["Queued", "Running"].includes(ocrStatus ?? "")) return "processing";
  if (ocrStatus !== "Completed" || !form) return "unavailable";
  return form.status === "Confirmed" && !form.source_superseded ? "confirmed" : "review";
}
export const expenseDocumentTypes = ["receipt", "tax_invoice", "tax_invoice_receipt", "invoice", "payment_voucher", "receipt_substitute", "expense_claim"];
// Keep an existing type selectable so opening older documents never converts them.
export function expenseTypeOptions(current: string) {
  return [...new Set([...(expenseDocumentTypes.includes(current) ? [] : [current]), ...expenseDocumentTypes])];
}
