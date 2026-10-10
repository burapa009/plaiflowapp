package extraction

import "testing"

func TestOCRAccountingValuesReachCanonicalForm(t *testing.T) {
	draft := Extract(accountingResult(
		"ใบเสร็จรับเงิน/ใบกำกับภาษี", "Receipt No: INV-42", "วันที่ 28 ก.ย. 2569",
		"Seller: บริษัท ตัวอย่าง จำกัด", "Buyer: บริษัท ลูกค้า จำกัด",
		"มูลค่าก่อนภาษี 1000.00", "ภาษีมูลค่าเพิ่ม 70.00", "Net total 1070.00 บาท",
	))
	form := FormFromDraft(draft)
	if form.Type != "tax_invoice_receipt" {
		t.Errorf("combined receipt/invoice opened the wrong form: %s", form.Type)
	}
	for key, want := range map[string]string{
		"document_number": "INV-42", "issue_date": "2026-09-28", "seller_name": "บริษัท ตัวอย่าง จำกัด",
		"buyer_name": "บริษัท ลูกค้า จำกัด", "subtotal": "1000.00", "vat_amount": "70.00", "total_amount": "1070.00",
	} {
		if form.Fields[key] != want {
			t.Errorf("%s: OCR accounting value did not reach form: got %q want %q", key, form.Fields[key], want)
		}
	}
}

func TestOCRPrefillKeepsConflictsAndEmptyTemplateUnfilled(t *testing.T) {
	for _, input := range [][]string{
		{"ใบกำกับภาษี", "Net total 100.00 บาท", "Net total 200.00 บาท"},
		{"ใบกำกับภาษี", "ยอดสุทธิ", "ภาษีมูลค่าเพิ่ม", "รายการ | จำนวน | ราคา | ยอดเงิน"},
	} {
		form := FormFromDraft(Extract(accountingResult(input...)))
		if form.Fields["total_amount"] != "" || len(form.Tables["items"]) != 0 {
			t.Fatalf("missing or conflicting source became form data: %+v", form)
		}
	}
}

func TestOCRMissingThaiToneInLabelKeepsDocumentNumber(t *testing.T) {
	form := FormFromDraft(Extract(accountingResult("ใบเสร็จรับเงิน ตัวอย่าง", "เลขที TEST-0001")))
	if form.Fields["document_number"] != "TEST-0001" {
		t.Fatalf("clear value lost when OCR missed the label tone mark: %+v", form.Fields)
	}
}

func TestOCRHeadOfficePrefillsValidBranchCode(t *testing.T) {
	for _, label := range []string{"สาขา สำนักงานใหญ่", "Branch: HEAD OFFICE"} {
		form := FormFromDraft(Extract(accountingResult("ใบเสร็จรับเงิน", "Seller: บริษัท ตัวอย่าง จำกัด", label)))
		if form.Fields["seller_branch"] != "00000" || !validFormValue("branch", form.Fields["seller_branch"]) {
			t.Fatalf("head office must use the form's five-digit branch code: %+v", form.Fields)
		}
	}
}
