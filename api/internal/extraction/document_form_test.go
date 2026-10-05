package extraction

import (
	"encoding/json"
	"os"
	"reflect"
	"testing"
)

func TestFormConfigurationMatchesUI(t *testing.T) {
	raw, err := os.ReadFile("../../../web/lib/document-forms.json")
	if err != nil {
		t.Fatal(err)
	}
	var c FormConfig
	if err = json.Unmarshal(raw, &c); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(c, DocumentForms) {
		t.Fatal("UI/backend schema drift")
	}
	for name, typ := range c.Types {
		seen := map[string]bool{}
		for _, section := range typ.Sections {
			for _, key := range c.Sections[section].Fields {
				if seen[key] {
					t.Errorf("%s duplicate field %s", name, key)
				}
				seen[key] = true
				if _, ok := c.Fields[key]; !ok {
					t.Errorf("%s unknown field %s", name, key)
				}
			}
		}
		for _, key := range typ.Required {
			if !seen[key] {
				t.Errorf("%s invisible required %s", name, key)
			}
		}
	}
}
func TestDocumentFormDraftsAndTypeIsolation(t *testing.T) {
	for name := range DocumentForms.Types {
		d := DocumentForm{Version: 1, Type: name, Fields: map[string]string{}, Tables: map[string][]map[string]string{}}
		if e := ValidateDocumentForm(d, false); len(e) > 0 {
			t.Errorf("%s draft: %v", name, e)
		}
	}
	d := DocumentForm{Version: 1, Type: "delivery_note", Fields: map[string]string{"document_number": "DN-1", "issue_date": "2026-10-04", "total_amount": "invalid hidden", "price_mode": "invalid hidden", "period_from": "z", "period_to": "a"}, Tables: map[string][]map[string]string{"items": {{"amount": "hidden"}}}}
	if e := ValidateDocumentForm(d, true); len(e) > 0 {
		t.Fatal(e)
	}
	if d.LegacyValues()["total_amount"] != "" {
		t.Fatal("hidden money exported")
	}
	d.Type = "bank_slip"
	d.Fields["total_amount"] = "12.30"
	d.Fields["payment_date"] = "2026-10-04"
	if e := ValidateDocumentForm(d, true); len(e) > 0 {
		t.Fatal(e)
	}
	d.Type = "tax_invoice"
	if a := AssessDocumentForm(d, "confirm"); a.Unresolved["buyer_tax_id"] == "" {
		t.Fatal("unknown buyer tax obligation was treated as false")
	}
	d.Type = "abbreviated_tax_invoice"
	delete(d.Fields, "price_mode")
	delete(d.Tables, "items")
	if e := ValidateDocumentForm(d, true); len(e) > 0 {
		t.Fatal(e)
	}
}
func TestFormMergePreservesOmissionAndClearsExplicitly(t *testing.T) {
	before := DocumentForm{Fields: map[string]string{"buyer_name": "ลูกค้า", "notes": "ล้าง"}, Tables: map[string][]map[string]string{"items": {{"description": "คงอยู่"}}, "income": {{"tax": "1"}}}}
	after := MergeDocumentForm(before, DocumentForm{Version: 1, Type: "receipt", Fields: map[string]string{"notes": ""}, Tables: map[string][]map[string]string{"income": {}}})
	if after.Fields["buyer_name"] != "ลูกค้า" || after.Fields["notes"] != "" || len(after.Tables["items"]) != 1 || len(after.Tables["income"]) != 0 {
		t.Fatal(after)
	}
	if before.Fields["notes"] != "ล้าง" {
		t.Fatal("input mutated")
	}
}
func TestDocumentFormExactAmounts(t *testing.T) {
	if n, ok := roundedFormProduct("0.335", "3", false); !ok || n != 101 {
		t.Fatalf("half up %d %v", n, ok)
	}
	if n, ok := FormCents("999999999999.99"); !ok || n != 99999999999999 {
		t.Fatal(n, ok)
	}
	d := DocumentForm{Version: 1, Type: "invoice", Fields: map[string]string{"document_number": "I-1", "issue_date": "2026-10-04", "subtotal": "0.30", "vat_rate": "7", "vat_amount": "0.02", "total_amount": "0.32"}, Tables: map[string][]map[string]string{}}
	if e := ValidateDocumentForm(d, true); len(e) > 0 {
		t.Fatal(e)
	}
	d.Fields["total_amount"] = "0.33"
	if e := AssessDocumentForm(d, "confirm").Differences; e["total_amount"] == "" {
		t.Fatal("mismatch accepted")
	}
	d.Type = "credit_note"
	d.Fields = map[string]string{"document_number": "C-1", "issue_date": "2026-10-04", "adjustment_reason": "คืน", "before_amount": "100", "total_amount": "20", "after_amount": "80"}
	d.Tables["references"] = []map[string]string{{"document_id": "source"}}
	if e := ValidateDocumentForm(d, true); len(e) > 0 {
		t.Fatal(e)
	}
	d.Type = "debit_note"
	if e := AssessDocumentForm(d, "confirm").Differences; e["after_amount"] == "" {
		t.Fatal("debit sign")
	}
}
func TestIncomeAndStatementValidation(t *testing.T) {
	d := DocumentForm{Version: 1, Type: "withholding_tax_certificate", Fields: map[string]string{"issue_date": "2026-10-04", "seller_name": "ผู้จ่าย", "seller_tax_id": "0105552117718", "buyer_name": "ผู้รับ", "buyer_tax_id": "0105552117718", "income_total": "100.00", "withholding_tax": "3.00"}, Tables: map[string][]map[string]string{"income": {{"description": "บริการ", "payment_date": "2026-10-04", "amount": "40", "rate": "3", "tax": "1.20"}, {"description": "บริการ", "payment_date": "2026-10-04", "amount": "60", "rate": "3", "tax": "1.80"}}}}
	if e := ValidateDocumentForm(d, true); len(e) > 0 {
		t.Fatal(e)
	}
	d.Tables["income"][1]["tax"] = "1.81"
	e := AssessDocumentForm(d, "confirm").Differences
	if e["income.1.tax"] == "" || e["withholding_tax"] == "" {
		t.Fatal(e)
	}
	d.Type = "bank_statement"
	d.Fields = map[string]string{"period_from": "2026-10-01", "period_to": "2026-10-31"}
	d.Tables["transactions"] = []map[string]string{{"date": "2026-10-04", "credit": "10", "debit": "1"}}
	if e := AssessDocumentForm(d, "confirm").Differences; e["transactions.0.debit"] == "" {
		t.Fatal(e)
	}
}
