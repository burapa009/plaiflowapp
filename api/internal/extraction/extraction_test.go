package extraction_test

import (
	"testing"

	"plaiflow/api/internal/extraction"
	"plaiflow/api/internal/ocr"
)

func TestThaiTaxInvoiceProducesTraceableFieldsWithoutInventingMissingValues(t *testing.T) {
	input := ocr.Result{SchemaVersion: 1, Pages: []ocr.Page{{Number: 1, Lines: []ocr.Line{
		{Text: "ใบกำกับภาษี", Confidence: .99},
		{Text: "เลขที่ INV-42", Confidence: .98},
		{Text: "วันที่ 23/09/2569", Confidence: .97},
		{Text: "ผู้ขาย: บริษัท ตัวอย่าง จำกัด", Confidence: .95},
		{Text: "เลขประจำตัวผู้เสียภาษี 0123456789012", Confidence: .92},
		{Text: "มูลค่าก่อนภาษี 1,000.00", Confidence: .98},
		{Text: "ภาษีมูลค่าเพิ่ม 70.00", Confidence: .94},
		{Text: "ยอดรวม 1,070.00", Confidence: .99},
	}}}}
	draft := extraction.Extract(input)
	if draft.DocumentType != "tax_invoice" || draft.Fields["issue_date"].Normalized != "2026-09-23" || draft.Fields["total_amount"].Normalized != "1070.00" {
		t.Fatalf("unexpected extraction: %+v", draft)
	}
	if draft.Fields["total_amount"].Raw != "1,070.00" || draft.Fields["total_amount"].Confidence != "unrated" || draft.Fields["total_amount"].Evidence[0].Line != 8 {
		t.Fatalf("raw value, confidence or evidence lost: %+v", draft.Fields["total_amount"])
	}
	if draft.Fields["buyer_tax_id"].Normalized != "" || draft.Fields["buyer_tax_id"].Presence != "not_found" {
		t.Fatalf("missing value fabricated: %+v", draft.Fields["buyer_tax_id"])
	}
	for _, warning := range draft.Warnings {
		if warning.Code == "amount_mismatch" {
			t.Fatal("correct amounts were marked mismatched")
		}
	}
}

func TestAmbiguousOrContradictoryValuesStayUnconfirmed(t *testing.T) {
	input := ocr.Result{Pages: []ocr.Page{{Number: 1, Lines: []ocr.Line{
		{Text: "ใบกำกับภาษี", Confidence: .99},
		{Text: "ยอดรวม 100.00", Confidence: .4},
		{Text: "ยอดรวม 200.00", Confidence: .8},
	}}}}
	draft := extraction.Extract(input)
	if draft.Fields["total_amount"].Presence != "ambiguous" || draft.Fields["total_amount"].Normalized != "" {
		t.Fatalf("ambiguous total selected: %+v", draft.Fields["total_amount"])
	}
	if draft.Fields["total_amount"].Raw != "100.00 | 200.00" {
		t.Fatalf("ambiguous raw candidates lost: %+v", draft.Fields["total_amount"])
	}
	if !draft.HasWarning("multiple_candidates", "total_amount") || !draft.HasWarning("required_missing", "issue_date") || !draft.HasWarning("ocr_low_quality", "total_amount") {
		t.Fatalf("missing review warnings: %+v", draft.Warnings)
	}
}

func TestEnglishAmountLabelsAreCaseInsensitive(t *testing.T) {
	draft := extraction.Extract(ocr.Result{Pages: []ocr.Page{{Number: 1, Lines: []ocr.Line{
		{Text: "Tax Invoice", Confidence: .99},
		{Text: "Subtotal 1,000.00", Confidence: .99},
		{Text: "VAT 70.00", Confidence: .99},
		{Text: "Grand Total 1,070.00", Confidence: .99},
	}}}})
	if draft.DocumentType != "tax_invoice" || draft.Fields["subtotal"].Normalized != "1000.00" || draft.Fields["vat_amount"].Normalized != "70.00" || draft.Fields["total_amount"].Normalized != "1070.00" {
		t.Fatalf("English amount labels were skipped: %+v", draft.Fields)
	}
}

func TestThaiNumeralsNormalizeWithoutChangingRawEvidence(t *testing.T) {
	draft := extraction.Extract(ocr.Result{Pages: []ocr.Page{{Number: 1, Lines: []ocr.Line{
		{Text: "ใบกำกับภาษี", Confidence: .99},
		{Text: "วันที่ ๒๓/๐๙/๒๕๖๙", Confidence: .99},
		{Text: "ยอดรวม ๑,๐๗๐.๐๐", Confidence: .99},
	}}}})
	if draft.Fields["issue_date"].Normalized != "2026-09-23" || draft.Fields["total_amount"].Normalized != "1070.00" || draft.Fields["total_amount"].Raw != "๑,๐๗๐.๐๐" {
		t.Fatalf("Thai numerals lost or not normalized: %+v", draft.Fields)
	}
}

func TestConfirmationRejectsMissingRequiredAndInventedSystemDefaults(t *testing.T) {
	draft := extraction.Extract(ocr.Result{Pages: []ocr.Page{{Number: 1, Lines: []ocr.Line{{Text: "ใบกำกับภาษี", Confidence: .99}}}}})
	if err := extraction.ValidateReview(draft, map[string]string{"total_amount": "0.00"}); err == nil {
		t.Fatal("incomplete tax invoice was confirmed")
	}
	values := map[string]string{
		"document_number": "INV-42", "issue_date": "2026-09-23", "seller_name": "บริษัท ตัวอย่าง จำกัด",
		"seller_tax_id": "0123456789012", "total_amount": "1070.00", "subtotal": "1000.00", "vat_amount": "70.00",
	}
	if err := extraction.ValidateReview(draft, values); err != nil {
		t.Fatalf("human-confirmed values rejected: %v", err)
	}
	values["total_amount"] = "100.00"
	if err := extraction.ValidateReview(draft, values); err == nil {
		t.Fatal("contradictory amounts were confirmed")
	}
}

func TestSellerBranchMustBeReviewedAndNeverDefaultsToHeadOffice(t *testing.T) {
	draft := extraction.Extract(ocr.Result{Pages: []ocr.Page{{Number: 1, Lines: []ocr.Line{
		{Text: "ใบกำกับภาษี", Confidence: .99}, {Text: "สาขาที่ 00012", Confidence: .98},
	}}}})
	if draft.Fields["seller_branch"].Normalized != "00012" || draft.Fields["seller_branch"].Confidence != "unrated" {
		t.Fatalf("branch candidate missing: %+v", draft.Fields["seller_branch"])
	}
	values := map[string]string{"document_number": "INV-42", "issue_date": "2026-09-23", "seller_name": "ร้าน",
		"seller_tax_id": "0123456789012", "total_amount": "0.00"}
	if err := extraction.ValidateReview(draft, values); err != nil {
		t.Fatalf("unknown branch should remain empty: %v", err)
	}
	values["seller_branch"] = "head"
	if err := extraction.ValidateReview(draft, values); err == nil {
		t.Fatal("invalid branch accepted")
	}
}
