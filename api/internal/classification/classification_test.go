package classification

import (
	"context"
	"errors"
	"testing"

	"plaiflow/api/internal/ocr"
)

type failingClassifier struct{}

func (failingClassifier) Classify(context.Context, Input) (Result, error) {
	return Result{}, errors.New("upstream unavailable")
}

func TestFallbackKeepsRuleAndRequestsReview(t *testing.T) {
	in := Normalize("doc", ocr.Result{Pages: []ocr.Page{{Number: 1, Text: "ใบเสร็จรับเงิน\nชำระแล้ว\nยอดรวม 100"}}})
	got, used, err := Hybrid(context.Background(), in, failingClassifier{}, .9, .7)
	if err == nil || used || got.DocumentType != "receipt" || !got.RequiresReview {
		t.Fatalf("fallback: %+v %t %v", got, used, err)
	}
}

func TestRules(t *testing.T) {
	cases := []struct{ name, text, want string }{
		{"receipt", "ใบเสร็จรับเงิน\nชำระแล้ว\nยอดรวม 100", "receipt"},
		{"before payment receipt", "ใบเสร็จก่อนรับเงิน\nรหัสลูกหนี้ ก-0002\nยอดชำระ 100\nได้รับเงินตามเอกสารเลขที่", "pre_receipt"},
		{"before payment receipt with later receipt reference", "ใบเสร็จก่อนรับเงิน\nรหัสลูกหนี้ ก-0002\nยอดชำระ 100\nโปรดขอใบเสร็จรับเงินภายหลัง", "pre_receipt"},
		{"tax", "ใบกำกับภาษี\nเลขประจำตัวผู้เสียภาษี\nVAT 7%\nยอดรวม", "tax_invoice"},
		{"combined", "ใบกำกับภาษี / ใบเสร็จรับเงิน\nเลขประจำตัวผู้เสียภาษี\nVAT 7%\nชำระแล้ว\nยอดรวม", "tax_invoice_receipt"},
		{"invoice", "ใบแจ้งหนี้\nกำหนดชำระ\nเลขที่ 123", "invoice"},
		{"credit", "ใบลดหนี้\nอ้างอิงใบกำกับภาษี\nยอดลด", "credit_note"},
		{"withholding", "หนังสือรับรองการหักภาษี ณ ที่จ่าย\nผู้มีหน้าที่หักภาษี\nภ.ง.ด.", "withholding_tax_certificate"},
		{"slip", "โอนเงินสำเร็จ\nจากบัญชี 123\nไปยังบัญชี 456", "bank_slip"},
		{"billing", "ใบวางบิล\nกำหนดชำระ\nยอดรวม", "billing_note"},
		{"debit", "ใบเพิ่มหนี้\nอ้างอิงใบกำกับภาษี\nยอดเพิ่ม", "debit_note"},
		{"payment voucher", "ใบสำคัญจ่าย\nจ่ายให้ นาย ก\nจำนวนเงิน", "payment_voucher"},
		{"receipt voucher", "ใบสำคัญรับเงิน\nรับจาก นาย ก\nจำนวนเงิน", "receipt_voucher"},
		{"substitute", "ใบแทนใบเสร็จรับเงิน\nผู้จ่ายเงิน\nจำนวนเงิน", "receipt_substitute"},
		{"claim", "ใบเบิกค่าใช้จ่าย\nผู้เบิก\nจำนวนเงิน", "expense_claim"},
		{"petty cash", "เงินสดย่อย\nผู้เบิก\nจำนวนเงิน", "petty_cash"},
		{"quotation", "ใบเสนอราคา\nยืนราคา\nยอดรวม", "quotation"},
		{"purchase", "ใบสั่งซื้อ\nPO No. 1\nผู้ขาย", "purchase_order"},
		{"statement", "รายการเดินบัญชี\nยอดคงเหลือ\nถอน ฝาก", "bank_statement"},
		{"unknown", "บันทึกประชุม\nเรื่องทั่วไป", "unknown"},
		{"single keyword", "ใบกำกับภาษี", "unknown"},
		{"collision", "ใบแจ้งหนี้\nกำหนดชำระ\nยอดที่ต้องชำระ\nโปรดขอใบกำกับภาษีภายหลัง ยังไม่ได้รับชำระ", "invoice"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			in := Normalize("doc", ocr.Result{Pages: []ocr.Page{{Number: 1, Width: 100, Height: 100, Text: tc.text}}})
			got := ClassifyRule(in, .7)
			if got.DocumentType != tc.want {
				t.Fatalf("got %s %.2f, want %s: %+v", got.DocumentType, got.Confidence, tc.want, got.CandidateTypes)
			}
		})
	}
}

func TestPreReceiptIsNotPaymentEvidenceOrExpense(t *testing.T) {
	meta := AccountingFor("pre_receipt")
	if meta.ExpenseCandidate || meta.PaymentEvidence || meta.VATDocument || meta.InputVATCandidate {
		t.Fatalf("pre-receipt must not imply payment or expense: %+v", meta)
	}
}

func TestNormalizationRetainsGeometry(t *testing.T) {
	box := &ocr.BoundingBox{XMin: 1, YMin: 2, XMax: 30, YMax: 40}
	in := Normalize("doc", ocr.Result{Pages: []ocr.Page{{Number: 1, Width: 100, Height: 200, Text: "ใบเสร็จ", Lines: []ocr.Line{{Text: "ใบเสร็จ", Confidence: .9, BBox: box}}}}})
	if in.DocumentID != "doc" || in.Pages[0].Blocks[0].BBox != box || in.Pages[0].Blocks[0].Confidence != .9 {
		t.Fatalf("geometry lost: %+v", in)
	}
}
