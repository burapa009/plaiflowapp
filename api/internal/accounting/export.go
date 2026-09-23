package accounting

import (
	"errors"
	"strconv"
	"strings"
	"time"

	"plaiflow/api/internal/extraction"
)

const TemplateVersion = "accounting_suggestions_v1"

var ExportHeader = []string{"row_id", "document_id", "source_review_revision", "document_type", "reviewed_issue_date",
	"reviewed_document_number", "reviewed_seller_name", "reviewed_seller_tax_id", "reviewed_seller_branch", "vendor_contact_code",
	"expense_category_id", "expense_category_name", "reviewed_currency", "reviewed_subtotal", "reviewed_vat_amount",
	"reviewed_total_amount", "suggestion_basis", "rule_version", "warning_codes", "approved_at", "template_version"}

type Category struct {
	ID         string     `json:"id"`
	Name       string     `json:"name"`
	ArchivedAt *time.Time `json:"archived_at,omitempty"`
}

type Approval struct {
	ID             string    `json:"id"`
	Revision       int       `json:"revision"`
	DocumentID     string    `json:"document_id"`
	ReviewID       string    `json:"review_id"`
	ReviewRevision int       `json:"review_revision"`
	CategoryID     string    `json:"category_id"`
	CategoryName   string    `json:"category_name"`
	VendorID       string    `json:"vendor_id,omitempty"`
	ContactCode    string    `json:"vendor_contact_code,omitempty"`
	Basis          string    `json:"suggestion_basis"`
	RuleVersion    int       `json:"rule_version,omitempty"`
	Warnings       []string  `json:"warning_codes"`
	ApprovedAt     time.Time `json:"approved_at"`
}

func ExportRow(approval Approval, review extraction.Review) ([]string, error) {
	if approval.ID == "" || approval.ReviewID != review.ID || approval.ReviewRevision != review.Revision || approval.DocumentID != review.DocumentID || approval.CategoryID == "" {
		return nil, errors.New("approved source revision is unavailable")
	}
	v := review.Values
	ruleVersion := ""
	if approval.RuleVersion > 0 {
		ruleVersion = strconv.Itoa(approval.RuleVersion)
	}
	return []string{approval.ID, review.DocumentID, strconv.Itoa(review.Revision), review.DocumentType, v["issue_date"],
		v["document_number"], v["seller_name"], v["seller_tax_id"], v["seller_branch"], approval.ContactCode,
		approval.CategoryID, approval.CategoryName, v["currency"], v["subtotal"], v["vat_amount"], v["total_amount"],
		approval.Basis, ruleVersion, strings.Join(approval.Warnings, ";"), approval.ApprovedAt.UTC().Format(time.RFC3339), TemplateVersion}, nil
}
