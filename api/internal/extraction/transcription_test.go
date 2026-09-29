package extraction

import (
	"plaiflow/api/internal/ocr"
	"testing"
)

func TestTranscriptionEvidenceAndAmbiguity(t *testing.T) {
	p := ocr.Page{Number: 1, Width: 100, Height: 100, DurationMS: 100, TranscriptionMS: 60, ExtractionMS: 40,
		Text:      "Tax invoice\nDate 27/02/2025\nTotal 11,875.00\nTax ID 0123456789012\n<figure>total 999</figure>",
		Proposals: []ocr.Proposal{{Field: "issue_date", Raw: "27/02/2025", Line: 2}, {Field: "total_amount", Raw: "11,875.00", Line: 3}, {Field: "seller_tax_id", Raw: "0123456789012", Line: 4}}}
	r := ocr.Result{SchemaVersion: 3, InputSHA256: "sha", ModelVersion: ocr.GenerativeModelVersion, PreprocessingVersion: ocr.GenerativePreprocessingVersion, Pages: []ocr.Page{p}}
	if err := r.Validate(ocr.Input{SHA256: "sha", ModelVersion: r.ModelVersion, PreprocessingVersion: r.PreprocessingVersion}); err != nil {
		t.Fatal(err)
	}
	d := Extract(r)
	if d.Fields["total_amount"].Normalized != "11875.00" || d.Fields["issue_date"].Normalized != "2025-02-27" || d.Fields["seller_tax_id"].Presence != "not_found" || d.Accounting.Summary.TotalAmount == nil {
		t.Fatalf("unexpected fields: %+v", d.Fields)
	}
	r.Pages[0].Proposals[1] = ocr.Proposal{Field: "total_amount", Raw: "999", Line: 5}
	if r.Pages[0].ValidateTranscription() == nil {
		t.Fatal("figure interpretation accepted as evidence")
	}
	r.Pages[0] = p
	r.Pages[0].Proposals = []ocr.Proposal{{Field: "issue_date", Raw: "03/04/2025", Line: 1}}
	r.Pages[0].Text = "03/04/2025"
	if Extract(r).Fields["issue_date"].Presence != "not_found" {
		t.Fatal("ambiguous date normalized")
	}
}

func TestTranscriptionPrefersExplicitThaiInvoiceEvidence(t *testing.T) {
	p := ocr.Page{Number: 1, Width: 100, Height: 100, DurationMS: 100, TranscriptionMS: 60, ExtractionMS: 40,
		Text:      "บริษัท โฟลว์แอคเคาท์ ทดสอบ จำกัด (สำนักงานใหญ่)\nลูกค้า\nบริษัท ตัวอย่าง จำกัด\nรวมเป็นเงิน: 12,500.00 บาท\nภาษีมูลค่าเพิ่ม 7%: 776.87 บาท\nราคาไม่รวมภาษีมูลค่าเพิ่ม: 11,098.13 บาท\nจำนวนเงินรวมทั้งสิ้น: 11,875.00 บาท",
		Proposals: []ocr.Proposal{{Field: "subtotal", Raw: "12,500.00", Line: 4}, {Field: "vat_amount", Raw: "776.87", Line: 5}, {Field: "total_amount", Raw: "11,875.00", Line: 7}}}
	r := ocr.Result{SchemaVersion: 3, InputSHA256: "sha", ModelVersion: ocr.GenerativeModelVersion, PreprocessingVersion: ocr.GenerativePreprocessingVersion, Pages: []ocr.Page{p}}
	d := Extract(r)
	if d.Fields["seller_name"].Normalized != "บริษัท โฟลว์แอคเคาท์ ทดสอบ จำกัด (สำนักงานใหญ่)" ||
		d.Fields["subtotal"].Normalized != "11098.13" || d.Fields["vat_amount"].Normalized != "776.87" ||
		d.Fields["total_amount"].Normalized != "11875.00" || !d.HasWarning("multiple_candidates", "subtotal") {
		t.Fatalf("explicit evidence not preferred: %+v", d.Fields)
	}
}
