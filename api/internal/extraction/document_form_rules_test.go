package extraction

import (
	"encoding/json"
	"os"
	"reflect"
	"testing"
)

func TestFormRuleContractAndActions(t *testing.T) {
	for name, typ := range DocumentForms.Types {
		d := DocumentForm{Version: 1, Type: name, Fields: map[string]string{}, Tables: map[string][]map[string]string{}}
		for _, action := range []string{"draft", "confirm"} {
			if e := ValidateFormAction(d, action); len(e) > 0 {
				t.Fatalf("empty %s %s: %#v", name, action, e)
			}
		}
		if a := AssessDocumentForm(d, "confirm"); a.TaxStatus != "not_assessed" || a.RuleVersion == "" {
			t.Fatal(a)
		}
		if a := AssessDocumentForm(d, "draft"); a.EvidenceStatus != "not_assessed" {
			t.Fatalf("draft claimed evidence checked: %#v", a)
		}
		for _, action := range []string{"issue", "tax", "account"} {
			if e := ValidateFormAction(d, action); e["action"] == "" {
				t.Fatalf("unsupported %s allowed", action)
			}
		}
		active := []string{}
		for _, section := range typ.Sections {
			active = append(active, DocumentForms.Sections[section].Fields...)
		}
		for _, table := range typ.Tables {
			for _, col := range DocumentForms.Tables[table].Columns {
				active = append(active, table+"."+col[0])
			}
		}
		for _, key := range active {
			r, ok := DocumentForms.Rules[name][key]
			if !ok || r.Requirement == "NOT_APPLICABLE" || r.Field != key || r.RuleVersion == "" || r.SourceURL == "" || r.Reason == "" || r.EffectiveFrom == "" {
				t.Fatalf("missing metadata %s %s", name, key)
			}
		}
	}
}
func TestFormDraftFormatsAndHiddenValues(t *testing.T) {
	d := DocumentForm{Version: 1, Type: "receipt", Fields: map[string]string{"paid_amount": "wrong", "payment_method": "cash", "payment_reference": "hidden"}, Tables: map[string][]map[string]string{}}
	if ValidateFormAction(d, "draft")["paid_amount"] == "" {
		t.Fatal("draft accepted malformed amount")
	}
	d.Fields["paid_amount"] = "0"
	if e := ValidateFormAction(d, "draft"); len(e) > 0 {
		t.Fatal(e)
	}
	if FormFieldVisible(d, "payment_reference") {
		t.Fatal("cash reference visible")
	}
	if d.Fields["payment_reference"] != "hidden" {
		t.Fatal("hidden value lost")
	}
	d.Type = "bank_slip"
	d.Fields["vat_amount"] = "invalid hidden"
	if e := ValidateFormAction(d, "confirm"); e["vat_amount"] != "" {
		t.Fatal("hidden vat blocked slip")
	}
}

func TestFormCalculatedIntermediateDifferences(t *testing.T) {
	for _, tc := range []struct {
		name, subtotal, vat, total, want string
	}{
		{"total with blank intermediates", "", "", "108", "total_amount"},
		{"VAT with calculated base", "", "8", "", "vat_amount"},
		{"base with calculated row amount", "101", "", "", "subtotal"},
		{"matching total", "", "", "107", ""},
		{"unentered total", "", "", "", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d := DocumentForm{Version: 1, Type: "tax_invoice", Fields: map[string]string{
				"price_mode": "exclusive", "discount": "0", "vat_rate": "7",
				"subtotal": tc.subtotal, "vat_amount": tc.vat, "total_amount": tc.total,
			}, Tables: map[string][]map[string]string{"items": {{"quantity": "1", "unit_price": "100", "discount": "0"}}}}
			a := AssessDocumentForm(d, "confirm")
			if tc.want != "" && (a.Differences[tc.want] == "" || a.EvidenceStatus != "incomplete") {
				t.Fatalf("missing arithmetic diagnostic: %#v", a)
			}
			if tc.want == "" && len(a.Differences) != 0 {
				t.Fatalf("unexpected arithmetic diagnostic: %#v", a.Differences)
			}
			if a.Calculated["total_amount"] != "107.00" || d.Tables["items"][0]["amount"] != "" || d.Fields["subtotal"] != tc.subtotal {
				t.Fatal("calculation changed source evidence or lost derived totals")
			}
		})
	}
}
func TestFormRuleParityFixtures(t *testing.T) {
	raw, e := os.ReadFile("../../../web/tests/fixtures/document-form-rules.json")
	if e != nil {
		t.Fatal(e)
	}
	var cases []struct {
		Name       string            `json:"name"`
		Data       DocumentForm      `json:"data"`
		Missing    []string          `json:"missing"`
		NotMissing []string          `json:"notMissing"`
		Unknown    []string          `json:"unknown"`
		Hidden     []string          `json:"hidden"`
		Calculated map[string]string `json:"calculated"`
	}
	if e = json.Unmarshal(raw, &cases); e != nil {
		t.Fatal(e)
	}
	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			m, u := FormRuleIssues(tc.Data, "confirm")
			for _, k := range tc.Missing {
				if m[k] == "" {
					t.Errorf("not missing %s", k)
				}
			}
			for _, k := range tc.NotMissing {
				if m[k] != "" || u[k] != "" {
					t.Errorf("unexpected rule %s", k)
				}
			}
			for _, k := range tc.Unknown {
				if u[k] == "" {
					t.Errorf("unknown lost %s", k)
				}
			}
			for _, k := range tc.Hidden {
				if FormFieldVisible(tc.Data, k) {
					t.Errorf("visible %s", k)
				}
			}
			if got := CalculateForm(tc.Data); !reflect.DeepEqual(got, tc.Calculated) {
				t.Fatalf("calculated %v expected %v", got, tc.Calculated)
			}
		})
	}
}
