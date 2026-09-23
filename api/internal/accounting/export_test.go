package accounting_test

import (
	"testing"
	"time"

	"plaiflow/api/internal/accounting"
	"plaiflow/api/internal/extraction"
)

func TestApprovedSuggestionExportsOnlyReviewedValuesInStableV1Columns(t *testing.T) {
	review := extraction.Review{ID: "review-1", DocumentID: "doc-1", Revision: 2, DocumentType: "tax_invoice", Values: map[string]string{
		"issue_date": "2026-09-23", "document_number": "INV-42", "seller_name": "=HYPERLINK(1)",
		"seller_tax_id": "0123456789012", "total_amount": "0.00",
	}}
	approved := accounting.Approval{ID: "approved-1", DocumentID: "doc-1", ReviewID: "review-1", ReviewRevision: 2,
		CategoryID: "category-1", CategoryName: "เดินทาง", Basis: "human_reviewed", ApprovedAt: time.Date(2026, 9, 23, 12, 0, 0, 0, time.UTC)}
	row, err := accounting.ExportRow(approved, review)
	if err != nil {
		t.Fatal(err)
	}
	if len(row) != 21 || row[0] != "approved-1" || row[11] != "เดินทาง" || row[15] != "0.00" || row[12] != "" || row[20] != "accounting_suggestions_v1" {
		t.Fatalf("unstable or fabricated row: %#v", row)
	}
	if _, err := accounting.ExportRow(approved, extraction.Review{ID: "new-review", Revision: 3}); err == nil {
		t.Fatal("stale review exported")
	}
}
