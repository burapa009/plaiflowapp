package extraction

import (
	"regexp"
	"strings"

	"plaiflow/api/internal/ocr"
)

// ponytail: the first business-name line is the issuer; add layout-aware sections if real invoices disprove it.
var transcriptionSellerName = regexp.MustCompile(`^(?:บริษัท|ห้างหุ้นส่วน(?:จำกัด)?|ร้าน)\s+.+`)
var transcriptionAmounts = map[string]*regexp.Regexp{
	"subtotal":     regexp.MustCompile(`(?i)^(?:มูลค่าก่อนภาษี|รวมก่อนภาษี|ราคาไม่รวมภาษีมูลค่าเพิ่ม|amount\s*before\s*vat|subtotal)\s*[:：]?\s*([0-9๐-๙,]+(?:\.[0-9๐-๙]{2})?)`),
	"vat_amount":   regexp.MustCompile(`(?i)^(?:ภาษีมูลค่าเพิ่ม|vat)(?:\s*[0-9๐-๙]{1,2}\s*%)?\s*[:：]?\s*([0-9๐-๙,]+(?:\.[0-9๐-๙]{2})?)`),
	"total_amount": regexp.MustCompile(`(?i)^(?:จำนวนเงินรวมทั้งสิ้น|ยอดรวม|รวมทั้งสิ้น|ยอดสุทธิ|grand total|รวม)\s*[:：]?\s*([0-9๐-๙,]+(?:\.[0-9๐-๙]{2})?)`),
}

func preferTranscriptionEvidence(d *Draft, page ocr.Page) {
	lines := strings.Split(page.Text, "\n")
	sellerFound := d.Fields["seller_name"].Presence == "found"
	for i, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if !sellerFound && transcriptionSellerName.MatchString(line) {
			d.Fields["seller_name"] = Field{Presence: "found", Raw: line, Normalized: line,
				Confidence: "unrated", Evidence: []Evidence{{Page: page.Number, Line: i + 1}}}
			sellerFound = true
		}
		for _, key := range []string{"subtotal", "vat_amount", "total_amount"} {
			match := transcriptionAmounts[key].FindStringSubmatch(line)
			if len(match) < 2 {
				continue
			}
			raw := strings.TrimSpace(match[1])
			normalized, ok := normalize(key, raw)
			if !ok {
				continue
			}
			if previous := d.Fields[key]; previous.Presence == "found" && previous.Normalized != normalized {
				d.warn("multiple_candidates", key, "review")
			}
			d.Fields[key] = Field{Presence: "found", Raw: raw, Normalized: normalized,
				Confidence: "unrated", Evidence: []Evidence{{Page: page.Number, Line: i + 1}}}
		}
	}
}

func extractTranscription(result ocr.Result) Draft {
	d := Draft{SchemaVersion: SchemaVersion, DocumentType: "unknown", Fields: map[string]Field{}, Warnings: []Warning{}}
	a := AccountingDocument{Items: []AccountingItem{}, Confidence: map[string]string{}, Warnings: []string{"Generated suggestions require human review; confidence is not measured."}, RawValue: map[string]string{}}
	for _, key := range Keys {
		d.Fields[key] = Field{Presence: "not_found", Confidence: "unrated", Evidence: []Evidence{}}
	}
	texts := []string{}
	for _, page := range result.Pages {
		texts = append(texts, page.Text)
		if page.ValidateTranscription() != nil {
			d.warn("invalid_value", "document_number", "blocker")
			continue
		}
		lower := strings.ToLower(page.Text)
		if strings.Contains(lower, "ใบกำกับภาษี") || strings.Contains(lower, "tax invoice") {
			d.DocumentType = "tax_invoice"
		} else if d.DocumentType == "unknown" && (strings.Contains(lower, "ใบเสร็จ") || strings.Contains(lower, "receipt")) {
			d.DocumentType = "receipt"
		} else if d.DocumentType == "unknown" && (strings.Contains(lower, "ใบแจ้งหนี้") || strings.Contains(lower, "invoice")) {
			d.DocumentType = "invoice"
		}
		for _, p := range page.Proposals {
			value, ok := normalize(p.Field, p.Raw)
			switch p.Field {
			case "issue_date":
				date := accountingDateValue(p.Raw)
				ok = date != nil
				if ok {
					value = *date
				}
			case "currency":
				ok = strings.EqualFold(p.Raw, "THB") || p.Raw == "บาท" || p.Raw == "฿"
				value = "THB"
			case "seller_tax_id", "buyer_tax_id":
				ok = ok && strings.Trim(value, "0123456789") == ""
				if ok {
					sum := 0
					for i := 0; i < 12; i++ {
						sum += int(value[i]-'0') * (13 - i)
					}
					ok = (11-sum%11)%10 == int(value[12]-'0')
				}
			}
			if !ok {
				d.warn("invalid_value", p.Field, "review")
				continue
			}
			field := d.Fields[p.Field]
			field.Evidence = append(field.Evidence, Evidence{Page: page.Number, Line: p.Line})
			if field.Presence == "not_found" {
				field.Presence, field.Raw, field.Normalized = "found", p.Raw, value
			} else if field.Presence == "ambiguous" || field.Normalized != value {
				field.Presence, field.Normalized = "ambiguous", ""
				field.Raw += " | " + p.Raw
				d.warn("multiple_candidates", p.Field, "blocker")
			}
			d.Fields[p.Field] = field
		}
		preferTranscriptionEvidence(&d, page)
	}
	a.RawText = strings.Join(texts, "\n\n")
	for key, target := range map[string]**string{
		"document_number": &a.Document.DocumentNumber, "issue_date": &a.Document.DocumentDate,
		"seller_name": &a.Seller.Name, "seller_tax_id": &a.Seller.TaxID, "seller_branch": &a.Seller.Branch,
		"buyer_name": &a.Buyer.Name, "buyer_tax_id": &a.Buyer.TaxID, "currency": &a.Document.Currency,
	} {
		if f := d.Fields[key]; f.Presence == "found" {
			*target = accountingPtr(f.Normalized)
		}
	}
	if d.DocumentType != "unknown" {
		a.Document.DocumentType = accountingPtr(d.DocumentType)
	}
	for _, key := range Keys {
		a.Confidence[key] = "unrated"
		a.RawValue[key] = d.Fields[key].Raw
	}
	if f := d.Fields["subtotal"]; f.Presence == "found" {
		a.Summary.Subtotal, _ = accountingNumber(f.Normalized)
		a.Summary.AmountBeforeVAT = a.Summary.Subtotal
	}
	if f := d.Fields["vat_amount"]; f.Presence == "found" {
		a.Summary.VATAmount, _ = accountingNumber(f.Normalized)
	}
	if f := d.Fields["total_amount"]; f.Presence == "found" {
		a.Summary.TotalAmount, _ = accountingNumber(f.Normalized)
	}
	for _, key := range []string{"issue_date", "total_amount"} {
		if d.Fields[key].Presence != "found" {
			d.warn("required_missing", key, "blocker")
		}
	}
	if d.DocumentType == "tax_invoice" {
		for _, key := range []string{"document_number", "seller_name", "seller_tax_id"} {
			if d.Fields[key].Presence != "found" {
				d.warn("required_missing", key, "blocker")
			}
		}
	}
	sub, sok := cents(d.Fields["subtotal"].Normalized)
	vat, vok := cents(d.Fields["vat_amount"].Normalized)
	total, tok := cents(d.Fields["total_amount"].Normalized)
	if sok && vok && tok {
		valid := sub+vat-total <= 1 && total-sub-vat <= 1
		a.Validation.TotalCalculationValid = accountingBool(valid)
		if !valid {
			d.warn("amount_mismatch", "total_amount", "blocker")
		}
	}
	for _, warning := range d.Warnings {
		a.Warnings = append(a.Warnings, warning.Field+": "+warning.Code)
	}
	d.Accounting = &a
	return d
}
