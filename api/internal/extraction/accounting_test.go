package extraction

import (
	"encoding/json"
	"strings"
	"testing"

	"plaiflow/api/internal/ocr"
)

func accountingResult(lines ...string) ocr.Result {
	page := ocr.Page{Number: 1, Text: strings.Join(lines, "\n")}
	for _, line := range lines {
		page.Lines = append(page.Lines, ocr.Line{Text: line, Confidence: .98})
	}
	return ocr.Result{Pages: []ocr.Page{page}}
}

func TestAccountingThaiInvoiceAndItems(t *testing.T) {
	ocrText := []string{
		"ใบเสร็จรับเงิน/ใบกำกับภาษี", "เลขที่ INV-42", "วันที่ 27-09-2569", "ผู้ขาย: บริษัท ตัวอย่าง จำกัด",
		"เลขประจำตัวผู้เสียภาษี 0-1234-56789-01-2", "ผู้ซื้อ: บริษัท ลูกค้า จำกัด",
		"เลขประจำตัวผู้เสียภาษี 0123456789012", "รายการ | จำนวน | หน่วย | ราคา | ยอดเงิน",
		"ค่าบริการ | 2 | ครั้ง | 500.00 | 1,000.00", "มูลค่าก่อนภาษี 1,000.00",
		"VAT 7% 70.00", "ยอดสุทธิ 1,070.00 บาท",
	}
	a := ExtractAccounting(accountingResult(ocrText...))
	if a.RawText != strings.Join(ocrText, "\n") || *a.Document.DocumentType != "receipt_tax_invoice" || *a.Document.DocumentDate != "2026-09-27" {
		t.Fatalf("document lost: %+v", a.Document)
	}
	if *a.Seller.TaxID != "0123456789012" || a.RawValue["seller_tax_id"] != "0-1234-56789-01-2" {
		t.Fatalf("tax ID evidence lost: %+v", a)
	}
	if len(a.Items) != 1 || a.Items[0].Amount.String() != "1000.00" || a.Items[0].Quantity.String() != "2.00" {
		t.Fatalf("items: %+v", a.Items)
	}
	if a.Summary.TotalAmount.String() != "1070.00" || a.Summary.VATRate.String() != "7" || !*a.Validation.VATCalculationValid {
		t.Fatalf("summary: %+v", a.Summary)
	}
	encoded, err := json.Marshal(a)
	if err != nil || !strings.Contains(string(encoded), `"total_amount":1070.00`) {
		t.Fatalf("money JSON: %s %v", encoded, err)
	}
}

func TestAccountingUncertainOCRStaysNull(t *testing.T) {
	a := ExtractAccounting(accountingResult("Tax Invoice", "Date 01/02/2026", "Seller Tax ID 0123456789O12", "Total 1,2O0.00", "VAT 7%"))
	if a.Document.DocumentDate != nil || a.Seller.TaxID != nil || a.Summary.TotalAmount != nil || a.Summary.VATAmount != nil {
		t.Fatalf("uncertain values promoted: %+v", a)
	}
	if a.RawValue["seller_tax_id"] != "0123456789O12" || a.Validation.TaxIDValid == nil || *a.Validation.TaxIDValid {
		t.Fatalf("raw invalid tax ID lost: %+v", a)
	}
}

func TestAccountingContradictionsWarnWithoutChangingNumbers(t *testing.T) {
	a := ExtractAccounting(accountingResult("Receipt", "Subtotal 100.00", "VAT 7% 7.00", "Total 120.00", "รายการ | จำนวน | ราคา | ยอดเงิน", "A | 2 | 10.00 | 25.00"))
	if a.Summary.TotalAmount.String() != "120.00" || a.Validation.TotalCalculationValid == nil || *a.Validation.TotalCalculationValid || a.Validation.VATCalculationValid != nil {
		t.Fatalf("arithmetic result: %+v", a)
	}
	if len(a.Warnings) < 2 {
		t.Fatalf("missing arithmetic warnings: %+v", a.Warnings)
	}
	a = ExtractAccounting(accountingResult("Invoice", "Total 100.00", "Total 200.00"))
	if a.Summary.TotalAmount != nil || len(a.Warnings) == 0 {
		t.Fatalf("conflicting totals accepted: %+v", a)
	}
}

func TestAccountingPaddleCellsAndUnsafeCandidates(t *testing.T) {
	cell := func(text string, x, y float64) ocr.Line {
		return ocr.Line{Text: text, Confidence: .98, Polygon: [][2]float64{{x, y}, {x + .1, y}, {x + .1, y + .01}, {x, y + .01}}}
	}
	lines := []ocr.Line{
		cell("รายการ", .1, .2), cell("จำนวน", .35, .2), cell("ราคา/หน่วย", .55, .2), cell("ยอดเงิน", .8, .2),
		cell("บริการ", .1, .3), cell("2", .35, .3), cell("500.00", .55, .3), cell("1,000.00", .8, .3),
		{Text: "Date 2026-09-27", Confidence: .98}, {Text: "Total 1O0 50", Confidence: .98},
		{Text: "Seller Tax ID 0123456789012XYZ", Confidence: .98}, {Text: "Total 500.00", Confidence: .3},
	}
	textLines := make([]string, len(lines))
	for i := range lines {
		textLines[i] = lines[i].Text
	}
	a := ExtractAccounting(ocr.Result{Pages: []ocr.Page{{Number: 1, Text: strings.Join(textLines, "\n"), Lines: lines}}})
	if len(a.Items) != 1 || a.Items[0].Amount.String() != "1000.00" {
		t.Fatalf("Paddle cells: %+v", a.Items)
	}
	if a.Document.DocumentDate == nil || *a.Document.DocumentDate != "2026-09-27" {
		t.Fatalf("ISO date: %+v", a.Document)
	}
	if a.Seller.TaxID != nil || a.Summary.TotalAmount != nil {
		t.Fatalf("unsafe value promoted: %+v", a)
	}
	if a.RawValue["seller_tax_id"] != "0123456789012XYZ" {
		t.Fatalf("truncated tax evidence: %+v", a.RawValue)
	}
}
