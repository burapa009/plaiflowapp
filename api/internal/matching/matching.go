package matching

import (
	"context"
	"math"
	"strings"
	"time"
)

// Facts are human-confirmed target fields or explicitly labelled OCR draft fields.
type Facts struct {
	DocumentID, DocumentType, IssueDate, TotalAmount string
	DocumentNumber, SellerTaxID, SellerName          string
	VendorSimilarity                                 float64
}

type Candidate struct {
	TargetDocumentID string             `json:"target_document_id"`
	MatchType        string             `json:"match_type"`
	Score            float64            `json:"score"`
	Signals          map[string]float64 `json:"signals"`
	Reasons          []string           `json:"reasons"`
	Status           string             `json:"status"`
}

type Store interface {
	FindMatchCandidates(context.Context, string, string, Facts) ([]Facts, error)
}

func Score(source, target Facts, autoThreshold, reviewThreshold float64, sourceNeedsReview bool) Candidate {
	result := Candidate{TargetDocumentID: target.DocumentID, MatchType: "possible_related", Status: "no_match",
		Signals: map[string]float64{}, Reasons: []string{}}
	if source.DocumentID == target.DocumentID || source.TotalAmount == "" || target.TotalAmount == "" || source.IssueDate == "" || target.IssueDate == "" {
		return result
	}
	if source.SellerTaxID != "" && target.SellerTaxID != "" && source.SellerTaxID != target.SellerTaxID {
		return result
	}
	if source.TotalAmount == target.TotalAmount {
		result.Signals["amount"] = 1
		result.Score += .30
		result.Reasons = append(result.Reasons, "same_amount")
	} else {
		result.Signals["amount"] = .8
		result.Score += .24
		result.Reasons = append(result.Reasons, "amount_within_one_satang")
	}
	a, ea := time.Parse("2006-01-02", source.IssueDate)
	b, eb := time.Parse("2006-01-02", target.IssueDate)
	if ea != nil || eb != nil {
		return Candidate{TargetDocumentID: target.DocumentID, MatchType: "possible_related", Status: "no_match", Signals: map[string]float64{}, Reasons: []string{}}
	}
	days := math.Abs(a.Sub(b).Hours() / 24)
	if days == 0 {
		result.Signals["date"] = 1
		result.Score += .15
		result.Reasons = append(result.Reasons, "same_date")
	} else if days <= 1 {
		result.Signals["date"] = .8
		result.Score += .12
		result.Reasons = append(result.Reasons, "date_within_one_day")
	} else if days <= 3 {
		result.Signals["date"] = .5
		result.Score += .075
		result.Reasons = append(result.Reasons, "date_within_three_days")
	} else {
		return Candidate{TargetDocumentID: target.DocumentID, MatchType: "possible_related", Status: "no_match", Signals: map[string]float64{}, Reasons: []string{}}
	}
	if source.SellerTaxID != "" && source.SellerTaxID == target.SellerTaxID {
		result.Signals["tax_id"] = 1
		result.Score += .20
		result.Reasons = append(result.Reasons, "same_seller_tax_id")
	}
	if source.DocumentNumber != "" && strings.EqualFold(source.DocumentNumber, target.DocumentNumber) {
		result.Signals["document_number"] = 1
		result.Score += .25
		result.Reasons = append(result.Reasons, "same_document_number")
	}
	if source.SellerName != "" && target.SellerName != "" {
		similarity := math.Max(0, math.Min(1, target.VendorSimilarity))
		result.Signals["vendor"] = similarity
		result.Score += .10 * similarity
		if similarity >= .7 {
			result.Reasons = append(result.Reasons, "vendor_similarity")
		}
	}
	result.Score = math.Round(result.Score*10000) / 10000
	if source.DocumentNumber != "" && strings.EqualFold(source.DocumentNumber, target.DocumentNumber) && source.SellerTaxID != "" && source.SellerTaxID == target.SellerTaxID && source.DocumentType == target.DocumentType {
		result.MatchType = "duplicate_document"
	} else if source.DocumentType == "invoice" && target.DocumentType == "receipt" || source.DocumentType == "receipt" && target.DocumentType == "invoice" {
		result.MatchType = "invoice_receipt"
	}
	if result.Score >= autoThreshold && !sourceNeedsReview {
		result.Status = "auto_match_candidate"
	} else if result.Score >= reviewThreshold {
		result.Status = "review_required"
	}
	return result
}
