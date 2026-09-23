package accounting_test

import (
	"testing"

	"plaiflow/api/internal/accounting"
)

func TestApprovedRulesAreDeterministicAndConflictsNeedReview(t *testing.T) {
	rules := []accounting.Rule{
		{ID: "default", VendorID: "vendor-1", CategoryID: "office", Version: 1},
		{ID: "specific", VendorID: "vendor-1", DocumentType: "tax_invoice", CategoryID: "travel", Version: 2},
	}
	got := accounting.Suggest("vendor-1", "tax_invoice", rules)
	if got.CategoryID != "travel" || got.Basis != "approved_vendor_document_type" || len(got.Warnings) != 0 {
		t.Fatalf("specific rule must win: %+v", got)
	}
	rules = append(rules, accounting.Rule{ID: "conflict", VendorID: "vendor-1", DocumentType: "tax_invoice", CategoryID: "meals", Version: 3})
	got = accounting.Suggest("vendor-1", "tax_invoice", rules)
	if got.CategoryID != "" || got.Basis != "rule_conflict" || len(got.Warnings) != 1 || got.Warnings[0] != "rule_conflict" {
		t.Fatalf("conflicting rules must not guess: %+v", got)
	}
	got = accounting.Suggest("", "tax_invoice", rules)
	if got.CategoryID != "" || got.Warnings[0] != "vendor_unmatched" {
		t.Fatalf("unmatched vendor must remain blank: %+v", got)
	}
}
