package classification

import (
	"context"
	"math"
	"sort"
	"strings"
	"time"

	"plaiflow/api/internal/ocr"
)

const Version = "rules-v1"

var Labels = map[string]string{
	"receipt": "ใบเสร็จรับเงิน", "tax_invoice": "ใบกำกับภาษี", "tax_invoice_receipt": "ใบกำกับภาษี/ใบเสร็จรับเงิน",
	"invoice": "ใบแจ้งหนี้", "billing_note": "ใบวางบิล", "credit_note": "ใบลดหนี้", "debit_note": "ใบเพิ่มหนี้",
	"withholding_tax_certificate": "หนังสือรับรองหัก ณ ที่จ่าย", "payment_voucher": "ใบสำคัญจ่าย",
	"receipt_voucher": "ใบสำคัญรับเงิน", "receipt_substitute": "ใบแทนใบเสร็จรับเงิน", "expense_claim": "ใบเบิกค่าใช้จ่าย",
	"petty_cash": "เอกสารเงินสดย่อย", "quotation": "ใบเสนอราคา", "purchase_order": "ใบสั่งซื้อ",
	"bank_slip": "หลักฐานการโอนเงิน", "bank_statement": "รายการเดินบัญชี", "unknown": "ยังระบุไม่ได้",
}

type Block struct {
	Text       string           `json:"text"`
	Confidence float64          `json:"confidence"`
	BBox       *ocr.BoundingBox `json:"bbox,omitempty"`
}
type Page struct {
	Page   int     `json:"page"`
	Width  int     `json:"width"`
	Height int     `json:"height"`
	Blocks []Block `json:"blocks"`
}
type Input struct {
	DocumentID string `json:"document_id"`
	Pages      []Page `json:"pages"`
	FullText   string `json:"full_text"`
}
type Candidate struct {
	Type       string  `json:"type"`
	Confidence float64 `json:"confidence"`
}
type Result struct {
	DocumentType      string             `json:"document_type"`
	Confidence        float64            `json:"confidence"`
	Method            string             `json:"classification_method"`
	RequiresReview    bool               `json:"requires_review"`
	ReviewStatus      string             `json:"review_status"`
	CandidateTypes    []Candidate        `json:"candidate_types"`
	Signals           []string           `json:"signals"`
	ClassifierVersion string             `json:"classifier_version"`
	Accounting        AccountingMetadata `json:"accounting"`
}
type AccountingMetadata struct {
	ExpenseCandidate        bool `json:"expense_candidate"`
	VATDocument             bool `json:"vat_document"`
	InputVATCandidate       bool `json:"input_vat_candidate"`
	WithholdingTaxCandidate bool `json:"withholding_tax_candidate"`
	PaymentEvidence         bool `json:"payment_evidence"`
}
type Record struct {
	Result
	EffectiveType string     `json:"effective_type"`
	CorrectedBy   string     `json:"corrected_by,omitempty"`
	CorrectedAt   *time.Time `json:"corrected_at,omitempty"`
}
type Store interface {
	SaveClassification(context.Context, string, string, string, Result) error
	GetClassification(context.Context, string, string, string) (Record, error)
	CorrectDocumentType(context.Context, string, string, string, string, string, string, time.Time) (Record, error)
}

func Normalize(documentID string, result ocr.Result) Input {
	in := Input{DocumentID: documentID, Pages: make([]Page, 0, len(result.Pages))}
	texts := make([]string, 0, len(result.Pages))
	for _, p := range result.Pages {
		page := Page{Page: p.Number, Width: p.Width, Height: p.Height, Blocks: make([]Block, 0, len(p.Lines))}
		for _, l := range p.Lines {
			box := l.BBox
			if box == nil && len(l.Polygon) == 4 && p.Width > 0 && p.Height > 0 {
				minX, minY, maxX, maxY := 1.0, 1.0, 0.0, 0.0
				for _, point := range l.Polygon {
					minX = math.Min(minX, point[0])
					minY = math.Min(minY, point[1])
					maxX = math.Max(maxX, point[0])
					maxY = math.Max(maxY, point[1])
				}
				box = &ocr.BoundingBox{XMin: int(math.Floor(minX * float64(p.Width))), YMin: int(math.Floor(minY * float64(p.Height))),
					XMax: int(math.Ceil(maxX * float64(p.Width))), YMax: int(math.Ceil(maxY * float64(p.Height)))}
			}
			page.Blocks = append(page.Blocks, Block{l.Text, l.Confidence, box})
		}
		if len(p.Lines) == 0 && p.Text != "" {
			page.Blocks = append(page.Blocks, Block{Text: p.Text})
		}
		pageText := p.Text
		if pageText == "" {
			lineTexts := make([]string, 0, len(p.Lines))
			for _, l := range p.Lines {
				lineTexts = append(lineTexts, l.Text)
			}
			pageText = strings.Join(lineTexts, "\n")
		}
		texts = append(texts, pageText)
		in.Pages = append(in.Pages, page)
	}
	in.FullText = strings.Join(texts, "\n")
	return in
}

type signal struct {
	words  []string
	weight float64
}
type rule struct {
	title   []string
	support []signal
}

var rules = map[string]rule{
	"receipt":                     {[]string{"ใบเสร็จรับเงิน", "receipt"}, []signal{{[]string{"ชำระแล้ว", "paid", "ได้รับเงิน"}, .25}, {[]string{"ยอดรวม", "total", "จำนวนเงิน"}, .15}}},
	"tax_invoice":                 {[]string{"ใบกำกับภาษี", "tax invoice"}, []signal{{[]string{"เลขประจำตัวผู้เสียภาษี", "tax id"}, .16}, {[]string{"vat", "ภาษีมูลค่าเพิ่ม"}, .16}, {[]string{"ยอดรวม", "total", "subtotal"}, .12}}},
	"tax_invoice_receipt":         {[]string{"ใบกำกับภาษี/ใบเสร็จรับเงิน", "ใบกำกับภาษี / ใบเสร็จรับเงิน", "tax invoice / receipt"}, []signal{{[]string{"เลขประจำตัวผู้เสียภาษี", "tax id"}, .13}, {[]string{"vat", "ภาษีมูลค่าเพิ่ม"}, .13}, {[]string{"ชำระแล้ว", "paid", "ได้รับเงิน"}, .12}, {[]string{"ยอดรวม", "total"}, .08}}},
	"invoice":                     {[]string{"ใบแจ้งหนี้", "invoice"}, []signal{{[]string{"payment due", "due date", "กำหนดชำระ", "ยอดที่ต้องชำระ"}, .24}, {[]string{"เลขที่", "invoice no", "ยอดรวม"}, .14}}},
	"billing_note":                {[]string{"ใบวางบิล", "billing note"}, []signal{{[]string{"กำหนดชำระ", "due date"}, .22}, {[]string{"ยอดรวม", "total"}, .14}}},
	"credit_note":                 {[]string{"ใบลดหนี้", "credit note"}, []signal{{[]string{"อ้างอิงใบกำกับภาษี", "original invoice", "reference"}, .22}, {[]string{"ยอดลด", "amount", "total"}, .14}}},
	"debit_note":                  {[]string{"ใบเพิ่มหนี้", "debit note"}, []signal{{[]string{"อ้างอิงใบกำกับภาษี", "original invoice", "reference"}, .22}, {[]string{"ยอดเพิ่ม", "amount", "total"}, .14}}},
	"withholding_tax_certificate": {[]string{"หนังสือรับรองการหักภาษี ณ ที่จ่าย", "หนังสือรับรองหัก ณ ที่จ่าย"}, []signal{{[]string{"ผู้มีหน้าที่หักภาษี", "ผู้ถูกหักภาษี"}, .24}, {[]string{"ภ.ง.ด.", "ภาษีที่หัก", "อัตราภาษี"}, .16}}},
	"payment_voucher":             {[]string{"ใบสำคัญจ่าย", "payment voucher"}, []signal{{[]string{"จ่ายให้", "ผู้รับเงิน"}, .23}, {[]string{"จำนวนเงิน", "total"}, .13}}},
	"receipt_voucher":             {[]string{"ใบสำคัญรับเงิน", "receipt voucher"}, []signal{{[]string{"รับจาก", "ผู้จ่ายเงิน"}, .23}, {[]string{"จำนวนเงิน", "total"}, .13}}},
	"receipt_substitute":          {[]string{"ใบแทนใบเสร็จรับเงิน"}, []signal{{[]string{"ผู้จ่ายเงิน", "รายการ"}, .23}, {[]string{"จำนวนเงิน", "ยอดรวม"}, .13}}},
	"expense_claim":               {[]string{"ใบเบิกค่าใช้จ่าย", "expense claim"}, []signal{{[]string{"ผู้เบิก", "พนักงาน"}, .23}, {[]string{"จำนวนเงิน", "ยอดรวม"}, .13}}},
	"petty_cash":                  {[]string{"เงินสดย่อย", "petty cash"}, []signal{{[]string{"ผู้เบิก", "ผู้รับเงิน"}, .23}, {[]string{"จำนวนเงิน", "ยอดรวม"}, .13}}},
	"quotation":                   {[]string{"ใบเสนอราคา", "quotation"}, []signal{{[]string{"ยืนราคา", "valid until", "ราคาเสนอ"}, .23}, {[]string{"ยอดรวม", "total"}, .13}}},
	"purchase_order":              {[]string{"ใบสั่งซื้อ", "purchase order"}, []signal{{[]string{"po no", "เลขที่ใบสั่งซื้อ"}, .23}, {[]string{"ผู้ขาย", "supplier", "ยอดรวม"}, .13}}},
	"bank_slip":                   {[]string{"โอนเงินสำเร็จ", "transfer successful", "หลักฐานการโอน"}, []signal{{[]string{"จากบัญชี", "from account"}, .18}, {[]string{"ไปยังบัญชี", "to account"}, .18}, {[]string{"reference no", "เลขที่รายการ"}, .12}}},
	"bank_statement":              {[]string{"รายการเดินบัญชี", "bank statement", "statement of account"}, []signal{{[]string{"ยอดคงเหลือ", "balance"}, .2}, {[]string{"ถอน", "ฝาก", "debit", "credit"}, .16}}},
}

func Valid(code string) bool { _, ok := Labels[code]; return ok }

func Review(result *Result, autoThreshold, reviewThreshold float64) {
	if autoThreshold <= 0 || autoThreshold >= 1 {
		autoThreshold = .9
	}
	if reviewThreshold <= 0 || reviewThreshold >= autoThreshold {
		reviewThreshold = .7
	}
	result.RequiresReview = result.DocumentType == "unknown" || result.Confidence < autoThreshold
	switch {
	case result.DocumentType == "unknown" || result.Confidence < reviewThreshold:
		result.ReviewStatus = "REVIEW_REQUIRED"
	case result.Confidence < autoThreshold:
		result.ReviewStatus = "REVIEW_RECOMMENDED"
	default:
		result.ReviewStatus = "AUTO_ACCEPT"
	}
}

// Rule confidence measures independent evidence, not calibrated probability.
func ClassifyRule(in Input, reviewThreshold float64) Result {
	if reviewThreshold <= 0 || reviewThreshold >= 1 {
		reviewThreshold = .7
	}
	full := strings.ToLower(in.FullText)
	if strings.Contains(full, "ใบกำกับภาษี") && strings.Contains(full, "ใบเสร็จรับเงิน") &&
		!strings.Contains(full, "ใบกำกับภาษี/ใบเสร็จรับเงิน") && !strings.Contains(full, "ใบกำกับภาษี / ใบเสร็จรับเงิน") {
		full += "\nใบกำกับภาษี/ใบเสร็จรับเงิน"
	}
	candidates := make([]Candidate, 0, len(rules))
	signals := map[string][]string{}
	for code, r := range rules {
		score, found := 0.0, []string{}
		title := first(full, r.title)
		if title == "" {
			continue
		}
		score += .48
		found = append(found, "พบคำว่า "+title)
		for _, s := range r.support {
			if word := first(full, s.words); word != "" {
				score += s.weight
				found = append(found, "พบคำว่า "+word)
			}
		}
		for _, p := range in.Pages {
			for _, b := range p.Blocks {
				if p.Height == 0 || b.BBox == nil {
					continue
				}
				position := float64(b.BBox.YMin) / float64(p.Height)
				lower := strings.ToLower(b.Text)
				if position <= .25 && strings.Contains(lower, title) {
					score += .08
					found = append(found, "หัวเอกสารอยู่ช่วงบนของหน้า")
					break
				}
			}
		}
		for _, p := range in.Pages {
			for _, b := range p.Blocks {
				if p.Height == 0 || b.BBox == nil {
					continue
				}
				lower := strings.ToLower(b.Text)
				if float64(b.BBox.YMin)/float64(p.Height) >= .65 && (strings.Contains(lower, "ยอดรวม") || strings.Contains(lower, "grand total")) {
					score += .03
					found = append(found, "ยอดรวมอยู่ช่วงล่างของหน้า")
					break
				}
			}
		}
		if len(found) < 2 {
			score = .49
		}
		if score > .98 {
			score = .98
		}
		candidates = append(candidates, Candidate{code, score})
		signals[code] = found
	}
	if strings.Contains(full, "ใบกำกับภาษี") && strings.Contains(full, "ใบเสร็จรับเงิน") && !strings.Contains(full, "ยังไม่ได้รับชำระ") {
		for i := range candidates {
			if candidates[i].Type == "tax_invoice_receipt" {
				candidates[i].Confidence = max(candidates[i].Confidence, .91)
			}
		}
	}
	if strings.Contains(full, "ยังไม่ได้รับชำระ") || strings.Contains(full, "unpaid") {
		for i := range candidates {
			if candidates[i].Type == "receipt" || candidates[i].Type == "tax_invoice_receipt" {
				candidates[i].Confidence = min(candidates[i].Confidence, .49)
			}
		}
	}
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].Confidence == candidates[j].Confidence {
			return candidates[i].Type < candidates[j].Type
		}
		return candidates[i].Confidence > candidates[j].Confidence
	})
	out := Result{DocumentType: "unknown", Method: "rule", RequiresReview: true, CandidateTypes: candidates, Signals: []string{}, ClassifierVersion: Version}
	if len(candidates) > 0 && candidates[0].Confidence >= reviewThreshold {
		out.DocumentType, out.Confidence, out.Signals = candidates[0].Type, candidates[0].Confidence, signals[candidates[0].Type]
	}
	out.Accounting = AccountingFor(out.DocumentType)
	return out
}

func AccountingFor(kind string) AccountingMetadata {
	vat := kind == "tax_invoice" || kind == "tax_invoice_receipt" || kind == "receipt_tax_invoice" || kind == "abbreviated_tax_invoice"
	return AccountingMetadata{ExpenseCandidate: kind == "receipt" || vat || kind == "invoice" || kind == "expense_claim" || kind == "petty_cash" || kind == "receipt_substitute",
		VATDocument: vat, InputVATCandidate: vat, WithholdingTaxCandidate: kind == "withholding_tax_certificate",
		PaymentEvidence: kind == "receipt" || kind == "tax_invoice_receipt" || kind == "bank_slip" || kind == "payment_voucher"}
}

func first(text string, choices []string) string {
	for _, choice := range choices {
		if strings.Contains(text, choice) {
			return choice
		}
	}
	return ""
}
