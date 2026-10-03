package extraction

import (
	"plaiflow/api/internal/classification"
	"plaiflow/api/internal/ocr"
	"testing"
)

func TestBankTypesKeepDifferentAmountSemantics(t *testing.T) {
	result := ocr.Result{Pages: []ocr.Page{{Number: 1, Lines: []ocr.Line{
		{Text: "จำนวนเงิน 1,070.00", Confidence: .96, BBox: &ocr.BoundingBox{XMin: 1, YMin: 2, XMax: 50, YMax: 15}},
		{Text: "ยอดรวม 2,000.00", Confidence: .9},
	}}}}
	slip := ExtractClassified(result, classification.Record{EffectiveType: "bank_slip"})
	if slip.Fields["total_amount"].Normalized != "1070.00" || slip.Fields["total_amount"].Evidence[0].BBox == nil {
		t.Fatalf("slip amount evidence: %+v", slip.Fields["total_amount"])
	}
	statement := ExtractClassified(result, classification.Record{EffectiveType: "bank_statement"})
	if statement.Fields["total_amount"].Presence != "not_found" || statement.Accounting.Summary.TotalAmount != nil {
		t.Fatalf("statement cannot have a single total: %+v", statement.Fields["total_amount"])
	}
}
