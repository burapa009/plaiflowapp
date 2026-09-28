package extraction

import (
	"encoding/json"
	"fmt"
	"math/big"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"
	"unicode/utf8"

	"plaiflow/api/internal/ocr"
)

// AccountingDocument is the nullable, reviewable accounting view of an OCR result.
// Monetary values are JSON numbers; no missing field is filled by arithmetic.
type AccountingDocument struct {
	Document struct {
		DocumentType   *string `json:"document_type"`
		DocumentNumber *string `json:"document_number"`
		DocumentDate   *string `json:"document_date"`
		DueDate        *string `json:"due_date"`
		Currency       *string `json:"currency"`
	} `json:"document"`
	Seller struct {
		Name    *string `json:"name"`
		Branch  *string `json:"branch"`
		Address *string `json:"address"`
		TaxID   *string `json:"tax_id"`
		Phone   *string `json:"phone"`
	} `json:"seller"`
	Buyer struct {
		Name    *string `json:"name"`
		Branch  *string `json:"branch"`
		Address *string `json:"address"`
		TaxID   *string `json:"tax_id"`
	} `json:"buyer"`
	Items   []AccountingItem `json:"items"`
	Summary struct {
		Subtotal        *json.Number `json:"subtotal"`
		Discount        *json.Number `json:"discount"`
		AmountBeforeVAT *json.Number `json:"amount_before_vat"`
		VATRate         *json.Number `json:"vat_rate"`
		VATAmount       *json.Number `json:"vat_amount"`
		WithholdingTax  *json.Number `json:"withholding_tax"`
		ServiceCharge   *json.Number `json:"service_charge"`
		OtherCharges    *json.Number `json:"other_charges"`
		TotalAmount     *json.Number `json:"total_amount"`
		PaidAmount      *json.Number `json:"paid_amount"`
		ChangeAmount    *json.Number `json:"change_amount"`
	} `json:"summary"`
	Payment struct {
		PaymentMethod        *string `json:"payment_method"`
		BankName             *string `json:"bank_name"`
		TransactionReference *string `json:"transaction_reference"`
	} `json:"payment"`
	Confidence map[string]string `json:"confidence"`
	Validation struct {
		TaxIDValid            *bool `json:"tax_id_valid"`
		TotalCalculationValid *bool `json:"total_calculation_valid"`
		VATCalculationValid   *bool `json:"vat_calculation_valid"`
	} `json:"validation"`
	Warnings []string          `json:"warnings"`
	RawText  string            `json:"raw_text"`
	RawValue map[string]string `json:"raw_value,omitempty"`
}

type AccountingItem struct {
	Description *string      `json:"description"`
	Quantity    *json.Number `json:"quantity"`
	Unit        *string      `json:"unit"`
	UnitPrice   *json.Number `json:"unit_price"`
	Discount    *json.Number `json:"discount"`
	Amount      *json.Number `json:"amount"`
}

var accountingDate = regexp.MustCompile(`(?:[0-9๐-๙]{1,2}[/.-]){2}[0-9๐-๙]{2,4}|[0-9]{4}-[0-9]{2}-[0-9]{2}`)
var accountingThaiMonthDate = regexp.MustCompile(`[0-9๐-๙]{1,2}\s+(?:ม\.ค\.|ก\.พ\.|มี\.ค\.|เม\.ย\.|พ\.ค\.|มิ\.ย\.|ก\.ค\.|ส\.ค\.|ก\.ย\.|ต\.ค\.|พ\.ย\.|ธ\.ค\.)\s*[0-9๐-๙]{4}`)
var accountingMoney = regexp.MustCompile(`^(?:[0-9]+|[0-9]{1,3}(?:,[0-9]{3})+)(?:\.[0-9]{1,2})?$`)
var accountingTax = regexp.MustCompile(`(?i)(?:tax\s*id|เลข(?:ประจำตัว)?ผู้เสียภาษี)(?:ผู้ซื้อ|ผู้ขาย)?\s*[:：]?\s*(.+)$`)
var accountingVATRate = regexp.MustCompile(`(?i)(?:vat|ภาษีมูลค่าเพิ่ม|ภาษี)\s*([0-9]{1,2})\s*%`)
var accountingAmount = regexp.MustCompile(`(?i)^\s*([0-9๐-๙][0-9๐-๙,]*(?:\.[0-9๐-๙]{1,2})?)\s*(?:บาท|THB)?\s*$`)
var accountingPatternCache sync.Map // Keys are fixed source literals below.

func accountingPtr(s string) *string { return &s }
func accountingBool(b bool) *bool    { return &b }

func accountingDigits(s string) string {
	return strings.Map(func(r rune) rune {
		if r >= '๐' && r <= '๙' {
			return '0' + r - '๐'
		}
		return r
	}, s)
}

func accountingNumber(s string) (*json.Number, int64) {
	s = accountingDigits(strings.TrimSpace(s))
	if !accountingMoney.MatchString(s) {
		return nil, 0
	}
	s = strings.ReplaceAll(s, ",", "")
	parts := strings.Split(s, ".")
	if len(parts[0]) > 12 {
		return nil, 0
	}
	whole, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return nil, 0
	}
	fraction := "00"
	if len(parts) == 2 {
		fraction = parts[1]
		if len(fraction) == 1 {
			fraction += "0"
		}
	}
	f, err := strconv.ParseInt(fraction, 10, 64)
	if err != nil {
		return nil, 0
	}
	n := json.Number(fmt.Sprintf("%d.%02d", whole, f))
	return &n, whole*100 + f
}

func accountingCents(n *json.Number) (int64, bool) {
	if n == nil {
		return 0, false
	}
	_, c := accountingNumber(n.String())
	return c, true
}

func accountingDateValue(s string) *string {
	s = accountingDigits(s)
	if len(s) == 10 && s[4] == '-' && s[7] == '-' {
		date, err := time.Parse("2006-01-02", s)
		if err == nil && date.Format("2006-01-02") == s {
			return accountingPtr(s)
		}
		return nil
	}
	thaiMonths := map[string]int{"ม.ค.": 1, "ก.พ.": 2, "มี.ค.": 3, "เม.ย.": 4, "พ.ค.": 5, "มิ.ย.": 6,
		"ก.ค.": 7, "ส.ค.": 8, "ก.ย.": 9, "ต.ค.": 10, "พ.ย.": 11, "ธ.ค.": 12}
	parts := strings.FieldsFunc(s, func(r rune) bool { return r == '/' || r == '.' || r == '-' })
	thaiMonth := 0
	if fields := strings.Fields(s); len(fields) == 3 {
		thaiMonth = thaiMonths[fields[1]]
		if thaiMonth != 0 {
			parts = []string{fields[0], strconv.Itoa(thaiMonth), fields[2]}
		}
	}
	if len(parts) != 3 {
		return nil
	}
	d, e1 := strconv.Atoi(parts[0])
	m, e2 := strconv.Atoi(parts[1])
	y, e3 := strconv.Atoi(parts[2])
	if e1 != nil || e2 != nil || e3 != nil {
		return nil
	}
	if d <= 12 && m <= 12 && thaiMonth == 0 {
		return nil
	}
	if y >= 60 && y <= 99 {
		y += 2500 // Thai two-digit Buddhist year; only recent unambiguous years.
	}
	if y >= 2400 && y <= 2600 {
		y -= 543
	}
	if y < 1900 || y > 2100 || m < 1 || m > 12 {
		return nil
	}
	t := time.Date(y, time.Month(m), d, 0, 0, 0, 0, time.UTC)
	if t.Year() != y || int(t.Month()) != m || t.Day() != d {
		return nil
	}
	return accountingPtr(t.Format("2006-01-02"))
}

func accountingDateFromLine(line string) *string {
	spans := accountingDate.FindAllStringIndex(line, -1)
	if len(spans) == 0 {
		spans = accountingThaiMonthDate.FindAllStringIndex(line, -1)
	}
	if len(spans) != 1 {
		return nil
	}
	span := spans[0]
	if span[0] > 0 {
		r, _ := utf8.DecodeLastRuneInString(line[:span[0]])
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return nil
		}
	}
	if span[1] < len(line) {
		r, _ := utf8.DecodeRuneInString(line[span[1]:])
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return nil
		}
	}
	return accountingDateValue(line[span[0]:span[1]])
}

// ExtractAccounting only accepts values with a visible label or an unambiguous item row.
// It never repairs OCR characters in identifiers or amounts.
func ExtractAccounting(result ocr.Result) AccountingDocument {
	a := AccountingDocument{Items: []AccountingItem{}, Warnings: []string{}, Confidence: map[string]string{
		"document_number": "low", "document_date": "low", "seller_name": "low", "seller_tax_id": "low", "buyer_tax_id": "low", "total_amount": "low",
	}, RawValue: map[string]string{}}
	pages := make([]string, 0, len(result.Pages))
	for _, p := range result.Pages {
		if p.Text != "" {
			pages = append(pages, p.Text)
		} else {
			lines := make([]string, 0, len(p.Lines))
			for _, line := range p.Lines {
				lines = append(lines, line.Text)
			}
			pages = append(pages, strings.Join(lines, "\n"))
		}
	}
	a.RawText = strings.Join(pages, "\n")
	seen := map[string]string{}
	quality := map[string]float64{}
	totalPriority := 0
	set := func(key, value string, q float64) bool {
		if value == "" {
			return false
		}
		if strings.Contains("|subtotal|discount|amount_before_vat|vat_amount|withholding_tax|service_charge|other_charges|total_amount|paid_amount|change_amount|", "|"+key+"|") {
			if n, _ := accountingNumber(value); n != nil {
				value = n.String()
			}
		}
		if old, ok := seen[key]; ok && old != value {
			seen[key] = ""
			a.Warnings = append(a.Warnings, key+": conflicting values")
			return false
		}
		if _, ok := seen[key]; ok && seen[key] == "" {
			return false
		}
		if q < .8 {
			a.Warnings = append(a.Warnings, key+": low OCR confidence")
			return false
		}
		seen[key], quality[key] = value, q
		return true
	}
	setTotal := func(raw string, q float64, priority int) bool {
		n, _ := accountingNumber(raw)
		if n == nil || q < .8 {
			return false
		}
		value := n.String()
		if seen["total_amount"] == value {
			if priority > totalPriority {
				totalPriority = priority
			}
			if q > quality["total_amount"] {
				quality["total_amount"] = q
			}
			return true
		}
		if old, ok := seen["total_amount"]; ok && old != value {
			a.Warnings = append(a.Warnings, "total_amount: conflicting values")
			if priority < totalPriority {
				return false
			}
			if priority == totalPriority {
				seen["total_amount"] = ""
				return false
			}
		}
		if seen["total_amount"] == "" && totalPriority == priority {
			return false
		}
		totalPriority = priority
		seen["total_amount"], quality["total_amount"] = value, q
		return true
	}
	section := "seller"
	var itemColumns []string
	invalidTax := false
	sawReceipt, sawTaxInvoice := false, false
	for _, p := range result.Pages {
		groups := accountingPolygonRows(p.Lines)
		parseLines := make([]ocr.Line, 0, len(p.Lines)+len(groups))
		for i, line := range p.Lines {
			for _, row := range groups {
				if row.index == i {
					parseLines = append(parseLines, ocr.Line{Text: strings.Join(row.cells, " "), Confidence: row.confidence})
				}
			}
			parseLines = append(parseLines, line)
		}
		for _, l := range parseLines {
			line := strings.TrimSpace(l.Text)
			lower := strings.ToLower(line)
			if line == "" {
				continue
			}
			if strings.Contains(line, "|") && l.Confidence >= .8 {
				cells := strings.Split(line, "|")
				for i := range cells {
					cells[i] = strings.TrimSpace(cells[i])
				}
				if cols := accountingItemHeader(cells); cols != nil {
					itemColumns = cols
					continue
				}
				if len(cells) > 0 && accountingSummaryLabel(cells[0]) {
					itemColumns = nil
					line = strings.Join(cells, " ")
					lower = strings.ToLower(line)
				}
				if itemColumns != nil && len(cells) == len(itemColumns) {
					if item, ok := accountingItemRow(cells, itemColumns); ok {
						a.Items = append(a.Items, item)
						continue
					}
				}
			}
			if strings.Contains(lower, "ผู้ซื้อ") || strings.Contains(lower, "ลูกค้า") || strings.HasPrefix(lower, "buyer") || strings.HasPrefix(lower, "customer") {
				section = "buyer"
			}
			if strings.Contains(lower, "ผู้ขาย") || strings.HasPrefix(lower, "seller") || strings.HasPrefix(lower, "vendor") {
				section = "seller"
			}
			if a.Document.DocumentType == nil && l.Confidence >= .8 {
				kind := ""
				switch {
				case strings.Contains(lower, "ใบกำกับภาษีอย่างย่อ"):
					kind = "abbreviated_tax_invoice"
				case (strings.Contains(lower, "ใบเสร็จรับเงิน") && strings.Contains(lower, "ใบกำกับภาษี")) || (strings.Contains(lower, "receipt") && strings.Contains(lower, "tax invoice")):
					kind = "receipt_tax_invoice"
				case strings.Contains(lower, "ใบกำกับภาษี") || strings.Contains(lower, "tax invoice"):
					kind = "tax_invoice"
				case strings.Contains(lower, "ใบเสร็จ") || strings.Contains(lower, "receipt"):
					kind = "receipt"
				case strings.Contains(lower, "ใบวางบิล") || strings.Contains(lower, "billing note"):
					kind = "billing_note"
				case strings.Contains(lower, "ใบแจ้งหนี้") || strings.Contains(lower, "invoice"):
					kind = "invoice"
				}
				if kind != "" {
					a.Document.DocumentType = accountingPtr(kind)
				}
			}
			if l.Confidence >= .8 {
				sawReceipt = sawReceipt || strings.Contains(lower, "ใบเสร็จ") || strings.Contains(lower, "receipt")
				sawTaxInvoice = sawTaxInvoice || strings.Contains(lower, "ใบกำกับภาษี") || strings.Contains(lower, "tax invoice")
				if sawReceipt && sawTaxInvoice {
					a.Document.DocumentType = accountingPtr("receipt_tax_invoice")
				}
			}
			if strings.Contains(lower, "บาท") || strings.Contains(lower, "thb") {
				if set("currency", "THB", l.Confidence) {
					a.Document.Currency = accountingPtr("THB")
				}
			}
			if strings.Contains(lower, "usd") {
				if set("currency", "USD", l.Confidence) {
					a.Document.Currency = accountingPtr("USD")
				}
			}
			if v, ok := accountingLabel(line, `(?i)^(?:เลขที่(?:เอกสาร)?|document\s*(?:no\.?|number)|invoice\s*(?:no\.?|number)|receipt\s*(?:no\.?|number))\s*[:：#]?\s*`, `[A-Za-z0-9][A-Za-z0-9/-]*`); ok && set("document_number", v, l.Confidence) {
				a.Document.DocumentNumber = accountingPtr(v)
			}
			if strings.Contains(lower, "due") || strings.Contains(lower, "ครบกำหนด") || strings.Contains(lower, "กำหนดชำระ") {
				if v := accountingDateFromLine(line); v != nil && set("due_date", *v, l.Confidence) {
					a.Document.DueDate = v
				}
			} else if strings.Contains(lower, "วันที่") || strings.HasPrefix(lower, "date") || strings.Contains(lower, "issue date") {
				if v := accountingDateFromLine(line); v != nil && set("document_date", *v, l.Confidence) {
					a.Document.DocumentDate = v
				}
			}
			if v, ok := accountingLabel(line, `(?i)^(?:ผู้ขาย|ชื่อผู้ขาย|seller|vendor|ผู้ซื้อ|ชื่อลูกค้า|buyer|customer)\s*[:：]\s*`, `.+`); ok {
				key := section + "_name"
				if set(key, v, l.Confidence) {
					if section == "buyer" {
						a.Buyer.Name = accountingPtr(v)
					} else {
						a.Seller.Name = accountingPtr(v)
					}
				}
			}
			if m := accountingTax.FindStringSubmatch(line); len(m) == 2 {
				party := section
				if strings.Contains(lower, "ผู้ซื้อ") || strings.Contains(lower, "buyer") {
					party = "buyer"
				}
				if strings.Contains(lower, "ผู้ขาย") || strings.Contains(lower, "seller") {
					party = "seller"
				}
				key := party + "_tax_id"
				raw := strings.TrimSpace(m[1])
				if prior := a.RawValue[key]; prior == "" {
					a.RawValue[key] = raw
				} else if !strings.Contains(prior, raw) {
					a.RawValue[key] = prior + " | " + raw
				}
				id := accountingDigits(strings.NewReplacer(" ", "", "-", "").Replace(raw))
				if len(id) == 13 && strings.Trim(id, "0123456789") == "" {
					if set(key, id, l.Confidence) {
						if party == "buyer" {
							a.Buyer.TaxID = accountingPtr(id)
						} else {
							a.Seller.TaxID = accountingPtr(id)
						}
					}
				} else {
					invalidTax = true
					a.Warnings = append(a.Warnings, key+": invalid or uncertain OCR value")
				}
			}
			if v, ok := accountingLabel(line, `(?i)^(?:สาขา(?:ที่)?|branch(?:\s*(?:no\.?|number))?)\s*[:：]?\s*`, `(?:[0-9๐-๙]{5}|สำนักงานใหญ่|head\s*office)`); ok {
				v = accountingDigits(v)
				if set(section+"_branch", v, l.Confidence) {
					if section == "buyer" {
						a.Buyer.Branch = accountingPtr(v)
					} else {
						a.Seller.Branch = accountingPtr(v)
					}
				}
			}
			if v, ok := accountingLabel(line, `(?i)^(?:ที่อยู่|address)\s*[:：]\s*`, `.+`); ok {
				if set(section+"_address", v, l.Confidence) {
					if section == "buyer" {
						a.Buyer.Address = accountingPtr(v)
					} else {
						a.Seller.Address = accountingPtr(v)
					}
				}
			}
			if v, ok := accountingLabel(line, `(?i)^(?:โทร(?:ศัพท์)?|tel(?:ephone)?|phone)\s*[:：]?\s*`, `[0-9+() -]{8,20}`); ok && set("seller_phone", v, l.Confidence) && section == "seller" {
				a.Seller.Phone = accountingPtr(v)
			}
			if v, ok := accountingLabel(line, `(?i)^(?:วิธีชำระเงิน|payment\s*method)\s*[:：]\s*`, `.+`); ok && set("payment_method", v, l.Confidence) {
				a.Payment.PaymentMethod = accountingPtr(v)
			}
			if v, ok := accountingLabel(line, `(?i)^(?:ธนาคาร|bank)\s*[:：]\s*`, `.+`); ok && set("bank_name", v, l.Confidence) {
				a.Payment.BankName = accountingPtr(v)
			}
			if v, ok := accountingLabel(line, `(?i)^(?:เลขอ้างอิง|transaction\s*(?:reference|ref\.?))\s*[:：#]\s*`, `[A-Za-z0-9/-]+`); ok && set("transaction_reference", v, l.Confidence) {
				a.Payment.TransactionReference = accountingPtr(v)
			}
			if rate := accountingVATRate.FindStringSubmatch(line); len(rate) == 2 {
				parsed, _ := strconv.Atoi(rate[1])
				normalized := strconv.Itoa(parsed)
				if set("vat_rate", normalized, l.Confidence) {
					n := json.Number(normalized)
					a.Summary.VATRate = &n
				}
			}
			if v, ok := accountingLabeledAmount(line, `(?i)^(?:ยอดสุทธิ|รวมทั้งสิ้น|grand\s*total|net\s*total|total|ยอดรวม|รวม)\s*[:：]?\s*`); ok {
				priority := 1
				if strings.HasPrefix(lower, "ยอดสุทธิ") || strings.HasPrefix(lower, "รวมทั้งสิ้น") || strings.HasPrefix(lower, "grand total") || strings.HasPrefix(lower, "net total") {
					priority = 2
				}
				if setTotal(*v, l.Confidence, priority) {
					a.Summary.TotalAmount, _ = accountingNumber(*v)
				}
			}
			if v, ok := accountingLabeledAmount(line, `(?i)^(?:ยอดก่อนภาษี|มูลค่าก่อนภาษี|amount\s*before\s*vat)\s*[:：]?\s*`); ok {
				if set("amount_before_vat", *v, l.Confidence) {
					a.Summary.AmountBeforeVAT, _ = accountingNumber(*v)
				}
			}
			if v, ok := accountingLabeledAmount(line, `(?i)^(?:ยอดรวมก่อนส่วนลด|รวมก่อนภาษี|subtotal)\s*[:：]?\s*`); ok {
				if set("subtotal", *v, l.Confidence) {
					a.Summary.Subtotal, _ = accountingNumber(*v)
				}
			}
			if v, ok := accountingLabeledAmount(line, `(?i)^(?:ส่วนลด|discount)\s*[:：]?\s*`); ok {
				if set("discount", *v, l.Confidence) {
					a.Summary.Discount, _ = accountingNumber(*v)
				}
			}
			if v, ok := accountingLabeledAmount(line, `(?i)^(?:ภาษีมูลค่าเพิ่ม(?:\s*7\s*%)?|vat(?:\s*7\s*%)?)\s*[:：]?\s*`); ok {
				if set("vat_amount", *v, l.Confidence) {
					a.Summary.VATAmount, _ = accountingNumber(*v)
				}
			}
			if v, ok := accountingLabeledAmount(line, `(?i)^(?:ภาษีหัก ณ ที่จ่าย|withholding\s*tax)\s*[:：]?\s*`); ok {
				if set("withholding_tax", *v, l.Confidence) {
					a.Summary.WithholdingTax, _ = accountingNumber(*v)
				}
			}
			if v, ok := accountingLabeledAmount(line, `(?i)^(?:ค่าบริการ|service\s*charge)\s*[:：]?\s*`); ok {
				if set("service_charge", *v, l.Confidence) {
					a.Summary.ServiceCharge, _ = accountingNumber(*v)
				}
			}
			if v, ok := accountingLabeledAmount(line, `(?i)^(?:ค่าใช้จ่ายอื่น|other\s*charges?)\s*[:：]?\s*`); ok {
				if set("other_charges", *v, l.Confidence) {
					a.Summary.OtherCharges, _ = accountingNumber(*v)
				}
			}
			if v, ok := accountingLabeledAmount(line, `(?i)^(?:ชำระแล้ว|paid\s*amount|amount\s*paid)\s*[:：]?\s*`); ok {
				if set("paid_amount", *v, l.Confidence) {
					a.Summary.PaidAmount, _ = accountingNumber(*v)
				}
			}
			if v, ok := accountingLabeledAmount(line, `(?i)^(?:เงินทอน|change(?:\s*amount)?)\s*[:：]?\s*`); ok {
				if set("change_amount", *v, l.Confidence) {
					a.Summary.ChangeAmount, _ = accountingNumber(*v)
				}
			}
		}
	}
	for _, page := range result.Pages {
		columns := []string(nil)
		var headerX []float64
		for _, row := range accountingPolygonRows(page.Lines) {
			if header := accountingItemHeader(row.cells); header != nil && row.confidence >= .8 {
				columns = header
				headerX = row.xs
				continue
			}
			if columns != nil && accountingSummaryLabel(row.cells[0]) {
				columns = nil
				continue
			}
			if columns != nil && len(row.cells) == len(columns) && row.confidence >= .8 && accountingAligned(headerX, row.xs) {
				if item, ok := accountingItemRow(row.cells, columns); ok {
					a.Items = append(a.Items, item)
				}
			}
		}
	}
	// A conflicting candidate cannot be promoted, even if the first occurrence was clear.
	for k, v := range seen {
		if v != "" {
			continue
		}
		switch k {
		case "document_number":
			a.Document.DocumentNumber = nil
		case "document_date":
			a.Document.DocumentDate = nil
		case "due_date":
			a.Document.DueDate = nil
		case "seller_name":
			a.Seller.Name = nil
		case "buyer_name":
			a.Buyer.Name = nil
		case "seller_tax_id":
			a.Seller.TaxID = nil
		case "buyer_tax_id":
			a.Buyer.TaxID = nil
		case "total_amount":
			a.Summary.TotalAmount = nil
		case "seller_branch":
			a.Seller.Branch = nil
		case "buyer_branch":
			a.Buyer.Branch = nil
		case "seller_address":
			a.Seller.Address = nil
		case "buyer_address":
			a.Buyer.Address = nil
		case "seller_phone":
			a.Seller.Phone = nil
		case "currency":
			a.Document.Currency = nil
		case "subtotal":
			a.Summary.Subtotal = nil
		case "discount":
			a.Summary.Discount = nil
		case "amount_before_vat":
			a.Summary.AmountBeforeVAT = nil
		case "vat_amount":
			a.Summary.VATAmount = nil
		case "vat_rate":
			a.Summary.VATRate = nil
		case "withholding_tax":
			a.Summary.WithholdingTax = nil
		case "service_charge":
			a.Summary.ServiceCharge = nil
		case "other_charges":
			a.Summary.OtherCharges = nil
		case "paid_amount":
			a.Summary.PaidAmount = nil
		case "change_amount":
			a.Summary.ChangeAmount = nil
		case "payment_method":
			a.Payment.PaymentMethod = nil
		case "bank_name":
			a.Payment.BankName = nil
		case "transaction_reference":
			a.Payment.TransactionReference = nil
		}
	}
	for _, k := range []string{"document_number", "document_date", "seller_name", "seller_tax_id", "buyer_tax_id", "total_amount"} {
		if seen[k] != "" {
			if quality[k] >= .95 {
				a.Confidence[k] = "high"
			} else if quality[k] >= .8 {
				a.Confidence[k] = "medium"
			}
		}
	}
	if invalidTax {
		a.Validation.TaxIDValid = accountingBool(false)
	}
	for i, item := range a.Items {
		if item.Quantity == nil || item.UnitPrice == nil || item.Amount == nil {
			continue
		}
		q, _ := accountingCents(item.Quantity)
		price, _ := accountingCents(item.UnitPrice)
		amount, _ := accountingCents(item.Amount)
		product := new(big.Int).Mul(big.NewInt(q), big.NewInt(price))
		expected := new(big.Int).Mul(big.NewInt(amount), big.NewInt(100))
		delta := new(big.Int).Sub(product, expected)
		if delta.Abs(delta).Cmp(big.NewInt(100)) > 0 {
			a.Warnings = append(a.Warnings, fmt.Sprintf("item %d: quantity × unit price does not match amount", i+1))
		}
	}
	conflicted := func(keys ...string) bool {
		for _, key := range keys {
			if value, exists := seen[key]; exists && value == "" {
				return true
			}
		}
		return false
	}
	if a.Summary.AmountBeforeVAT != nil && a.Summary.VATAmount != nil && a.Summary.TotalAmount != nil && !conflicted("amount_before_vat", "vat_amount", "total_amount") {
		before, _ := accountingCents(a.Summary.AmountBeforeVAT)
		vat, _ := accountingCents(a.Summary.VATAmount)
		total, _ := accountingCents(a.Summary.TotalAmount)
		ok := accountingNear(before+vat, total)
		a.Validation.VATCalculationValid = accountingBool(ok)
		if !ok {
			a.Warnings = append(a.Warnings, "VAT amounts do not match total")
		}
	}
	if a.Summary.Subtotal != nil && a.Summary.TotalAmount != nil && !(a.Summary.VATRate != nil && a.Summary.VATAmount == nil) && !conflicted("subtotal", "discount", "vat_amount", "withholding_tax", "service_charge", "other_charges", "total_amount") {
		sub, _ := accountingCents(a.Summary.Subtotal)
		total, _ := accountingCents(a.Summary.TotalAmount)
		calc := sub
		for _, t := range []struct {
			n    *json.Number
			sign int64
		}{{a.Summary.Discount, -1}, {a.Summary.VATAmount, 1}, {a.Summary.ServiceCharge, 1}, {a.Summary.OtherCharges, 1}, {a.Summary.WithholdingTax, -1}} {
			if t.n != nil {
				v, _ := accountingCents(t.n)
				calc += t.sign * v
			}
		}
		ok := accountingNear(calc, total)
		a.Validation.TotalCalculationValid = accountingBool(ok)
		if !ok {
			a.Warnings = append(a.Warnings, "subtotal, discount, tax and charges do not match total")
		}
	}
	return a
}

func accountingNear(a, b int64) bool { return a-b <= 1 && b-a <= 1 }

func accountingLabel(line, prefix, value string) (string, bool) {
	m := accountingCachedPattern(prefix + `(` + value + `)\s*$`).FindStringSubmatch(line)
	if len(m) != 2 {
		return "", false
	}
	return strings.TrimSpace(m[1]), true
}

func accountingLabeledAmount(line, prefix string) (*string, bool) {
	m := accountingCachedPattern(prefix).FindStringIndex(line)
	if m == nil || m[0] != 0 {
		return nil, false
	}
	v := accountingAmount.FindStringSubmatch(line[m[1]:])
	if len(v) != 2 {
		return nil, false
	}
	if n, _ := accountingNumber(v[1]); n != nil {
		return accountingPtr(v[1]), true
	}
	return nil, false
}

func accountingCachedPattern(pattern string) *regexp.Regexp {
	if cached, ok := accountingPatternCache.Load(pattern); ok {
		return cached.(*regexp.Regexp)
	}
	compiled := regexp.MustCompile(pattern)
	actual, _ := accountingPatternCache.LoadOrStore(pattern, compiled)
	return actual.(*regexp.Regexp)
}

func accountingItemHeader(cells []string) []string {
	if len(cells) < 2 || len(cells) > 8 {
		return nil
	}
	cols := make([]string, len(cells))
	hasDescription, hasAmount := false, false
	used := map[string]bool{}
	for i, cell := range cells {
		s := strings.ToLower(cell)
		switch {
		case strings.Contains(s, "รายการ") || strings.Contains(s, "description") || strings.Contains(s, "สินค้า"):
			cols[i] = "description"
			hasDescription = true
		case strings.Contains(s, "จำนวน") || strings.Contains(s, "quantity") || s == "qty":
			cols[i] = "quantity"
		case strings.Contains(s, "ราคา") || strings.Contains(s, "unit price"):
			cols[i] = "unit_price"
		case strings.Contains(s, "หน่วย") || s == "unit":
			cols[i] = "unit"
		case strings.Contains(s, "ส่วนลด") || strings.Contains(s, "discount"):
			cols[i] = "discount"
		case strings.Contains(s, "ยอดเงิน") || strings.Contains(s, "amount") || strings.Contains(s, "รวมเงิน"):
			cols[i] = "amount"
			hasAmount = true
		}
		if cols[i] != "" {
			if used[cols[i]] {
				return nil
			}
			used[cols[i]] = true
		}
	}
	if hasDescription && hasAmount {
		return cols
	}
	return nil
}

func accountingItemRow(cells, cols []string) (AccountingItem, bool) {
	item := AccountingItem{}
	for i, col := range cols {
		cell := cells[i]
		switch col {
		case "description":
			if cell != "" {
				item.Description = accountingPtr(cell)
			}
		case "quantity":
			item.Quantity, _ = accountingNumber(cell)
		case "unit":
			if cell != "" {
				item.Unit = accountingPtr(cell)
			}
		case "unit_price":
			item.UnitPrice, _ = accountingNumber(cell)
		case "discount":
			item.Discount, _ = accountingNumber(cell)
		case "amount":
			item.Amount, _ = accountingNumber(cell)
		}
	}
	return item, item.Description != nil && item.Amount != nil
}

type accountingRow struct {
	cells      []string
	xs         []float64
	confidence float64
	index      int
}

func accountingPolygonRows(lines []ocr.Line) []accountingRow {
	type cell struct {
		text                     string
		x, y, height, confidence float64
		index                    int
	}
	cells := make([]cell, 0, len(lines))
	for index, line := range lines {
		if len(line.Polygon) != 4 || strings.Contains(line.Text, "|") {
			continue
		}
		var x, y float64
		for _, point := range line.Polygon {
			x += point[0]
			y += point[1]
		}
		x /= 4
		y /= 4
		if x == 0 && y == 0 {
			continue
		}
		height := line.Polygon[3][1] - line.Polygon[0][1]
		if height <= 0 {
			continue
		}
		cells = append(cells, cell{line.Text, x, y, height, line.Confidence, index})
	}
	sort.Slice(cells, func(i, j int) bool {
		if cells[i].y == cells[j].y {
			return cells[i].x < cells[j].x
		}
		return cells[i].y < cells[j].y
	})
	var grouped [][]cell
	for _, c := range cells {
		if len(grouped) == 0 || c.y-grouped[len(grouped)-1][0].y > c.height*.5 {
			grouped = append(grouped, []cell{c})
		} else {
			grouped[len(grouped)-1] = append(grouped[len(grouped)-1], c)
		}
	}
	rows := make([]accountingRow, 0, len(grouped))
	for _, group := range grouped {
		if len(group) < 2 {
			continue
		}
		sort.Slice(group, func(i, j int) bool { return group[i].x < group[j].x })
		row := accountingRow{cells: make([]string, len(group)), xs: make([]float64, len(group)), confidence: 1, index: len(lines)}
		for i := range group {
			row.cells[i] = strings.TrimSpace(group[i].text)
			row.xs[i] = group[i].x
			if group[i].confidence < row.confidence {
				row.confidence = group[i].confidence
			}
			if group[i].index < row.index {
				row.index = group[i].index
			}
		}
		rows = append(rows, row)
	}
	return rows
}

func accountingSummaryLabel(s string) bool {
	s = strings.ToLower(strings.TrimSpace(s))
	for _, label := range []string{"total", "grand total", "net total", "subtotal", "vat", "discount", "ยอดสุทธิ", "รวมทั้งสิ้น", "ยอดรวม", "ภาษีมูลค่าเพิ่ม", "ส่วนลด"} {
		if strings.HasPrefix(s, label) {
			return true
		}
	}
	return false
}

func accountingAligned(header, row []float64) bool {
	if len(header) != len(row) {
		return false
	}
	for i := range header {
		if header[i]-row[i] > .06 || row[i]-header[i] > .06 {
			return false
		}
	}
	return true
}
