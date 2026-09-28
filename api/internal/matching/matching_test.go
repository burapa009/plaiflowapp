package matching

import "testing"

func TestStrongSameTenantFactsProduceOnlyCandidate(t *testing.T) {
	source := Facts{DocumentID: "source", DocumentType: "invoice", IssueDate: "2026-09-28", TotalAmount: "1070.00", DocumentNumber: "INV-1", SellerTaxID: "0105558123456", SellerName: "ตัวอย่าง"}
	target := Facts{DocumentID: "target", DocumentType: "invoice", IssueDate: "2026-09-28", TotalAmount: "1070.00", DocumentNumber: "INV-1", SellerTaxID: "0105558123456", SellerName: "ตัวอย่าง", VendorSimilarity: 1}
	got := Score(source, target, .9, .7, false)
	if got.MatchType != "duplicate_document" || got.Status != "auto_match_candidate" || got.Score != 1 {
		t.Fatalf("candidate %+v", got)
	}
	got = Score(source, target, .9, .7, true)
	if got.Status != "review_required" {
		t.Fatalf("uncertain OCR candidate %+v", got)
	}
	target.SellerTaxID = "9999999999999"
	got = Score(source, target, .9, .7, false)
	if got.Status != "no_match" {
		t.Fatalf("tax conflict %+v", got)
	}
}
