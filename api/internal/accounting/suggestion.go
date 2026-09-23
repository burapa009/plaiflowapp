package accounting

// Rule is an already-approved rule from the current Organization and rule-set version.
type Rule struct {
	ID           string `json:"id"`
	VendorID     string `json:"vendor_id"`
	DocumentType string `json:"document_type"`
	CategoryID   string `json:"category_id"`
	Version      int    `json:"version"`
}

type Candidate struct {
	CategoryID string   `json:"category_id,omitempty"`
	RuleID     string   `json:"rule_id,omitempty"`
	Version    int      `json:"rule_version,omitempty"`
	Basis      string   `json:"suggestion_basis"`
	Confidence string   `json:"confidence"`
	Warnings   []string `json:"warning_codes"`
}

func Suggest(vendorID, documentType string, rules []Rule) Candidate {
	if vendorID == "" {
		return Candidate{Basis: "no_rule", Confidence: "unrated", Warnings: []string{"vendor_unmatched", "category_unmatched"}}
	}
	bestPriority := 0
	var chosen Rule
	conflict := false
	for _, rule := range rules {
		if rule.VendorID != vendorID || rule.CategoryID == "" || rule.DocumentType != "" && rule.DocumentType != documentType {
			continue
		}
		priority := 1
		if rule.DocumentType != "" {
			priority = 2
		}
		if priority > bestPriority {
			bestPriority, chosen, conflict = priority, rule, false
		} else if priority == bestPriority && rule.CategoryID != chosen.CategoryID {
			conflict = true
		}
	}
	if conflict {
		return Candidate{Basis: "rule_conflict", Confidence: "unrated", Warnings: []string{"rule_conflict"}}
	}
	if bestPriority == 0 {
		return Candidate{Basis: "no_rule", Confidence: "unrated", Warnings: []string{"category_unmatched"}}
	}
	basis := "approved_vendor_default"
	if bestPriority == 2 {
		basis = "approved_vendor_document_type"
	}
	return Candidate{CategoryID: chosen.CategoryID, RuleID: chosen.ID, Version: chosen.Version, Basis: basis, Confidence: "unrated", Warnings: []string{}}
}
