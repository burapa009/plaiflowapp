export const documentTypeLabels: Record<string, string> = {
  tax_invoice: "ใบกำกับภาษี", receipt: "ใบเสร็จรับเงิน", pre_receipt: "ใบเสร็จก่อนรับเงิน", invoice: "ใบแจ้งหนี้", unknown: "ยังระบุไม่ได้",
  tax_invoice_receipt: "ใบกำกับภาษี / ใบเสร็จรับเงิน",
  billing_note: "ใบวางบิล", credit_note: "ใบลดหนี้", debit_note: "ใบเพิ่มหนี้",
  withholding_tax_certificate: "หนังสือรับรองหัก ณ ที่จ่าย", payment_voucher: "ใบสำคัญจ่าย", receipt_voucher: "ใบสำคัญรับเงิน",
  receipt_substitute: "ใบแทนใบเสร็จรับเงิน", expense_claim: "ใบเบิกค่าใช้จ่าย", petty_cash: "เอกสารเงินสดย่อย",
  quotation: "ใบเสนอราคา", purchase_order: "ใบสั่งซื้อ", bank_slip: "หลักฐานการโอนเงิน", bank_statement: "รายการเดินบัญชี",
};

export const documentTypeGroups = [
 { label: "ซื้อขายและภาษี", types: ["tax_invoice", "invoice", "tax_invoice_receipt", "billing_note", "credit_note", "debit_note", "withholding_tax_certificate"] },
 { label: "รับและจ่ายเงิน", types: ["receipt", "pre_receipt", "payment_voucher", "receipt_voucher", "receipt_substitute", "expense_claim", "petty_cash"] },
 { label: "ธนาคาร", types: ["bank_slip", "bank_statement"] },
 { label: "เอกสารประกอบ", types: ["quotation", "purchase_order", "unknown"] },
];
export function documentCategory(type: string) {
 return documentTypeGroups.find(group => group.types.includes(type))?.label ?? "เอกสารประกอบ";
}
