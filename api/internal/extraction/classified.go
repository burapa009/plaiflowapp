package extraction

import (
	"encoding/json"
	"plaiflow/api/internal/classification"
	"plaiflow/api/internal/ocr"
	"regexp"
	"strings"
)

var bankAmount = regexp.MustCompile(`(?i)^(?:จำนวนเงิน|amount(?: transferred)?)\s*[:：]?\s*([0-9๐-๙,]+(?:\.[0-9๐-๙]{2})?)`)

// ExtractClassified reuses the existing field parsers; the chosen type never changes OCR evidence.
func ExtractClassified(result ocr.Result, record classification.Record) Draft {
	d := Extract(result)
	switch record.EffectiveType {
	case "bank_slip":
		extractBankSlip(&d, result)
	case "bank_statement":
		clearStatementTotals(&d)
	}
	d.Classification = &record
	d.DocumentType = record.EffectiveType
	if d.Accounting != nil {
		kind := record.EffectiveType
		d.Accounting.Document.DocumentType = &kind
	}
	warnings := make([]Warning, 0, len(d.Warnings))
	for _, w := range d.Warnings {
		if w.Code != "required_missing" {
			warnings = append(warnings, w)
		}
	}
	d.Warnings = warnings
	required := []string{}
	switch d.DocumentType {
	case "receipt", "invoice", "tax_invoice", "tax_invoice_receipt", "billing_note", "credit_note", "debit_note":
		required = []string{"issue_date", "total_amount"}
	}
	if d.DocumentType == "tax_invoice" || d.DocumentType == "tax_invoice_receipt" {
		required = append(required, "document_number", "seller_name", "seller_tax_id")
	}
	for _, key := range required {
		if d.Fields[key].Presence != "found" {
			d.warn("required_missing", key, "blocker")
		}
	}
	return d
}

func extractBankSlip(d *Draft, result ocr.Result) {
	for _, page := range result.Pages {
		for i, line := range page.Lines {
			match := bankAmount.FindStringSubmatch(strings.TrimSpace(line.Text))
			if len(match) != 2 {
				continue
			}
			value, ok := normalize("total_amount", match[1])
			if !ok {
				continue
			}
			d.Fields["total_amount"] = Field{Presence: "found", Raw: match[1], Normalized: value, Confidence: "unrated", Source: "ocr_rule",
				Evidence: []Evidence{{Page: page.Number, Line: i + 1, BBox: line.BBox, OCRConfidence: line.Confidence}}}
			if d.Accounting != nil {
				amount := json.Number(value)
				d.Accounting.Summary.TotalAmount = &amount
			}
			return
		}
	}
}

func clearStatementTotals(d *Draft) {
	for _, key := range []string{"subtotal", "vat_amount", "total_amount"} {
		d.Fields[key] = Field{Presence: "not_found", Confidence: "unrated", Evidence: []Evidence{}}
	}
	if d.Accounting != nil {
		d.Accounting.Summary.Subtotal = nil
		d.Accounting.Summary.VATAmount = nil
		d.Accounting.Summary.TotalAmount = nil
	}
}
