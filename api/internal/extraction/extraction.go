package extraction

import (
	"errors"
	"regexp"
	"strconv"
	"strings"
	"time"

	"plaiflow/api/internal/ocr"
)

const SchemaVersion = 1

var Keys = []string{"document_number", "issue_date", "seller_name", "seller_tax_id", "buyer_name", "buyer_tax_id", "currency", "subtotal", "vat_amount", "total_amount"}

type Evidence struct {
	Page int `json:"page"`
	Line int `json:"line"`
}

type Field struct {
	Presence   string     `json:"presence"`
	Raw        string     `json:"raw"`
	Normalized string     `json:"normalized"`
	Confidence string     `json:"confidence"`
	Evidence   []Evidence `json:"evidence"`
}

type Warning struct {
	Code     string `json:"code"`
	Field    string `json:"field"`
	Severity string `json:"severity"`
}

type Draft struct {
	SchemaVersion int              `json:"schema_version"`
	DocumentType  string           `json:"document_type"`
	Fields        map[string]Field `json:"fields"`
	Warnings      []Warning        `json:"warnings"`
}

var patterns = map[string]*regexp.Regexp{
	"document_number": regexp.MustCompile(`(?i)(?:เลขที่(?:เอกสาร)?|invoice\s*(?:no\.?|number))\s*[:：]?\s*([A-Za-z0-9/-]+)`),
	"issue_date":      regexp.MustCompile(`(?i)(?:วันที่|date)\s*[:：]?\s*([0-9๐-๙]{1,2}[/.-][0-9๐-๙]{1,2}[/.-][0-9๐-๙]{4})`),
	"seller_name":     regexp.MustCompile(`(?:ผู้ขาย|ชื่อผู้ขาย)\s*[:：]\s*(.+)`),
	"seller_tax_id":   regexp.MustCompile(`(?:เลขประจำตัวผู้เสียภาษี(?:อากร)?|เลขผู้เสียภาษี)\s*[:：]?\s*([0-9๐-๙ -]{13,20})`),
	"buyer_name":      regexp.MustCompile(`(?:ผู้ซื้อ|ชื่อลูกค้า)\s*[:：]\s*(.+)`),
	"buyer_tax_id":    regexp.MustCompile(`(?:เลขประจำตัวผู้เสียภาษีผู้ซื้อ|เลขผู้เสียภาษีผู้ซื้อ)\s*[:：]?\s*([0-9๐-๙ -]{13,20})`),
	"subtotal":        regexp.MustCompile(`(?i)(?:มูลค่าก่อนภาษี|รวมก่อนภาษี|subtotal)\s*[:：]?\s*([0-9๐-๙,]+(?:\.[0-9๐-๙]{2})?)`),
	"vat_amount":      regexp.MustCompile(`(?i)(?:ภาษีมูลค่าเพิ่ม|vat)\s*[:：]?\s*([0-9๐-๙,]+(?:\.[0-9๐-๙]{2})?)`),
	"total_amount":    regexp.MustCompile(`(?i)(?:ยอดรวม|รวมทั้งสิ้น|ยอดสุทธิ|grand total)\s*[:：]?\s*([0-9๐-๙,]+(?:\.[0-9๐-๙]{2})?)`),
}

var moneyPattern = regexp.MustCompile(`^(?:[0-9]+|[0-9]{1,3}(?:,[0-9]{3})+)(?:\.[0-9]{2})?$`)

// Extract keeps OCR candidates separate from human-reviewed values. A rule match is never a calibrated confidence score.
func Extract(result ocr.Result) Draft {
	draft := Draft{SchemaVersion: SchemaVersion, DocumentType: "unknown", Fields: make(map[string]Field, len(Keys))}
	for _, key := range Keys {
		draft.Fields[key] = Field{Presence: "not_found", Confidence: "unrated", Evidence: []Evidence{}}
	}
	for _, page := range result.Pages {
		for i, line := range page.Lines {
			lower := strings.ToLower(line.Text)
			if strings.Contains(line.Text, "ใบกำกับภาษี") || strings.Contains(lower, "tax invoice") {
				draft.DocumentType = "tax_invoice"
			} else if draft.DocumentType == "unknown" && (strings.Contains(line.Text, "ใบเสร็จ") || strings.Contains(lower, "receipt")) {
				draft.DocumentType = "receipt"
			} else if draft.DocumentType == "unknown" && (strings.Contains(line.Text, "ใบแจ้งหนี้") || strings.Contains(lower, "invoice")) {
				draft.DocumentType = "invoice"
			}
			for key, pattern := range patterns {
				match := pattern.FindStringSubmatch(line.Text)
				if len(match) < 2 {
					continue
				}
				raw := strings.TrimSpace(match[1])
				normalized, ok := normalize(key, raw)
				if !ok {
					draft.warn("invalid_value", key, "review")
					continue
				}
				field := draft.Fields[key]
				field.Evidence = append(field.Evidence, Evidence{Page: page.Number, Line: i + 1})
				if field.Presence == "not_found" {
					field.Presence, field.Raw, field.Normalized = "found", raw, normalized
				} else if field.Presence == "ambiguous" || field.Normalized != normalized {
					field.Presence = "ambiguous"
					field.Raw += " | " + raw
					field.Normalized = ""
					draft.warn("multiple_candidates", key, "blocker")
				}
				draft.Fields[key] = field
				if line.Confidence < .8 {
					draft.warn("ocr_low_quality", key, "review")
				}
			}
			currencyRaw := ""
			if strings.Contains(line.Text, "บาท") {
				currencyRaw = "บาท"
			} else if strings.Contains(strings.ToUpper(line.Text), "THB") {
				currencyRaw = "THB"
			}
			if currencyRaw != "" {
				draft.Fields["currency"] = Field{Presence: "found", Raw: currencyRaw, Normalized: "THB", Confidence: "unrated", Evidence: []Evidence{{Page: page.Number, Line: i + 1}}}
			}
		}
	}
	if draft.DocumentType == "tax_invoice" || draft.DocumentType == "receipt" || draft.DocumentType == "invoice" {
		for _, key := range []string{"issue_date", "total_amount"} {
			if draft.Fields[key].Presence != "found" {
				draft.warn("required_missing", key, "blocker")
			}
		}
		if draft.DocumentType == "tax_invoice" {
			for _, key := range []string{"document_number", "seller_name", "seller_tax_id"} {
				if draft.Fields[key].Presence != "found" {
					draft.warn("required_missing", key, "blocker")
				}
			}
		}
	}
	a, aok := cents(draft.Fields["subtotal"].Normalized)
	b, bok := cents(draft.Fields["vat_amount"].Normalized)
	c, cok := cents(draft.Fields["total_amount"].Normalized)
	if aok && bok && cok && (a+b-c > 1 || c-a-b > 1) {
		draft.warn("amount_mismatch", "total_amount", "blocker")
	}
	return draft
}

func (d Draft) HasWarning(code, field string) bool {
	for _, warning := range d.Warnings {
		if warning.Code == code && warning.Field == field {
			return true
		}
	}
	return false
}

func (d *Draft) warn(code, field, severity string) {
	if !d.HasWarning(code, field) {
		d.Warnings = append(d.Warnings, Warning{Code: code, Field: field, Severity: severity})
	}
}

func normalize(key, raw string) (string, bool) {
	valueASCII := strings.Map(func(r rune) rune {
		if r >= '๐' && r <= '๙' {
			return '0' + r - '๐'
		}
		return r
	}, raw)
	switch key {
	case "issue_date":
		parts := strings.FieldsFunc(valueASCII, func(r rune) bool { return r == '/' || r == '.' || r == '-' })
		if len(parts) != 3 {
			return "", false
		}
		day, e1 := strconv.Atoi(parts[0])
		month, e2 := strconv.Atoi(parts[1])
		year, e3 := strconv.Atoi(parts[2])
		if e1 != nil || e2 != nil || e3 != nil {
			return "", false
		}
		if year >= 2400 && year <= 2600 {
			year -= 543
		}
		date := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
		if year < 1900 || month < 1 || month > 12 || date.Year() != year || int(date.Month()) != month || date.Day() != day {
			return "", false
		}
		return date.Format("2006-01-02"), true
	case "subtotal", "vat_amount", "total_amount":
		if !moneyPattern.MatchString(valueASCII) {
			return "", false
		}
		value := strings.ReplaceAll(valueASCII, ",", "")
		if _, ok := cents(value); !ok {
			return "", false
		}
		if !strings.Contains(value, ".") {
			value += ".00"
		}
		return value, true
	case "seller_tax_id", "buyer_tax_id":
		value := strings.NewReplacer(" ", "", "-", "").Replace(valueASCII)
		if len(value) != 13 {
			return "", false
		}
		return value, true
	default:
		return raw, raw != ""
	}
}

func cents(value string) (int64, bool) {
	if value == "" || len(value) > 20 || strings.HasPrefix(value, "-") {
		return 0, false
	}
	parts := strings.Split(value, ".")
	if len(parts) > 2 || len(parts[0]) == 0 || len(parts[0]) > 15 {
		return 0, false
	}
	whole, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return 0, false
	}
	fraction := int64(0)
	if len(parts) == 2 {
		if len(parts[1]) != 2 {
			return 0, false
		}
		fraction, err = strconv.ParseInt(parts[1], 10, 64)
		if err != nil {
			return 0, false
		}
	}
	return whole*100 + fraction, true
}

// ValidateReview checks values explicitly submitted by a reviewer; extraction candidates are never confirmation defaults.
func ValidateReview(draft Draft, values map[string]string) error {
	if draft.DocumentType != "tax_invoice" && draft.DocumentType != "receipt" && draft.DocumentType != "invoice" {
		return errors.New("unsupported document type")
	}
	allowed := make(map[string]bool, len(Keys))
	for _, key := range Keys {
		allowed[key] = true
	}
	for key, value := range values {
		if !allowed[key] || len(value) > 240 {
			return errors.New("invalid reviewed field")
		}
		if value == "" {
			continue
		}
		if key == "currency" {
			if value != "THB" {
				return errors.New("unsupported currency")
			}
		} else if key == "issue_date" {
			date, err := time.Parse("2006-01-02", value)
			if err != nil || date.Format("2006-01-02") != value {
				return errors.New("invalid reviewed date")
			}
		} else if key == "subtotal" || key == "vat_amount" || key == "total_amount" {
			if _, ok := cents(value); !ok || !strings.Contains(value, ".") {
				return errors.New("invalid reviewed amount")
			}
		} else if key == "seller_tax_id" || key == "buyer_tax_id" {
			if len(value) != 13 || strings.Trim(value, "0123456789") != "" {
				return errors.New("invalid reviewed tax ID")
			}
		}
	}
	required := []string{"issue_date", "total_amount"}
	if draft.DocumentType == "tax_invoice" {
		required = append(required, "document_number", "seller_name", "seller_tax_id")
	}
	for _, key := range required {
		if strings.TrimSpace(values[key]) == "" {
			return errors.New("required reviewed field missing")
		}
	}
	a, aok := cents(values["subtotal"])
	b, bok := cents(values["vat_amount"])
	c, cok := cents(values["total_amount"])
	if aok && bok && cok && (a+b-c > 1 || c-a-b > 1) {
		return errors.New("reviewed amounts disagree")
	}
	return nil
}
