package extraction

import (
	"fmt"
	"sort"
	"strings"
)

type FieldRule struct {
	Field              string            `json:"field"`
	Requirement        string            `json:"requirement"`
	Condition          map[string]string `json:"condition"`
	ConditionLabel     string            `json:"conditionLabel"`
	Action             []string          `json:"action"`
	Reason             string            `json:"reason"`
	SourceURL          string            `json:"sourceUrl"`
	LegalReference     string            `json:"legalReference"`
	EffectiveFrom      string            `json:"effectiveFrom"`
	LegalEffectiveFrom *string           `json:"legalEffectiveFrom"`
	RuleVersion        string            `json:"ruleVersion"`
	HiddenWhen         map[string]string `json:"hiddenWhen,omitempty"`
	EvidenceCheck      *bool             `json:"evidenceCheck,omitempty"`
	AdditionalRules    []FieldRule       `json:"additionalRules,omitempty"`
}
type FormSource struct {
	OCRJobID string                         `json:"ocr_job_id"`
	Fields   map[string]string              `json:"fields"`
	Tables   map[string][]map[string]string `json:"tables"`
}
type FormAssessment struct {
	RuleVersion    string            `json:"ruleVersion"`
	Action         string            `json:"action"`
	EvidenceStatus string            `json:"evidenceStatus"`
	Missing        FormErrors        `json:"missing"`
	Unresolved     FormErrors        `json:"unresolved"`
	Differences    FormErrors        `json:"differences"`
	Calculated     map[string]string `json:"calculated"`
	TaxStatus      string            `json:"taxStatus"`
}

// Conditions have three outcomes: yes / no / unknown. Unknown never means no.
func FormCondition(d DocumentForm, row map[string]string, condition map[string]string) string {
	unknown := false
	for k, want := range condition {
		value := d.Fields[k]
		if strings.HasPrefix(k, "$row.") {
			value = row[strings.TrimPrefix(k, "$row.")]
		}
		if want == "@present" {
			if value == "" {
				return "no"
			}
			continue
		}
		if value == "" || value == "unknown" {
			unknown = true
			continue
		}
		if value != want {
			return "no"
		}
	}
	if unknown {
		return "unknown"
	}
	return "yes"
}
func FormFieldVisible(d DocumentForm, key string) bool {
	rule, ok := DocumentForms.Rules[d.Type][key]
	if !ok || rule.Requirement == "NOT_APPLICABLE" {
		return false
	}
	return len(rule.HiddenWhen) == 0 || FormCondition(d, nil, rule.HiddenWhen) != "yes"
}
func FormRuleIssues(d DocumentForm, action string) (FormErrors, FormErrors) {
	missing, unknown := FormErrors{}, FormErrors{}
	calculated := CalculateForm(d)
	for key, rule := range DocumentForms.Rules[d.Type] {
		if !FormFieldVisible(d, key) {
			continue
		}
		candidates := append([]FieldRule{rule}, rule.AdditionalRules...)
		for _, r := range candidates {
			if action == "confirm" && r.EvidenceCheck != nil && !*r.EvidenceCheck {
				continue
			}
			applies := action == "confirm"
			for _, a := range r.Action {
				if a == action {
					applies = true
				}
			}
			if !applies || r.Requirement == "OPTIONAL" || r.Requirement == "NOT_APPLICABLE" {
				continue
			}
			inspect := func(path, value string, row map[string]string) {
				condition := FormCondition(d, row, r.Condition)
				if condition == "unknown" {
					unknown[path] = "ต้องตรวจสอบเงื่อนไข: " + r.ConditionLabel
					return
				}
				if condition == "yes" && strings.TrimSpace(value) == "" && calculated[path] == "" {
					missing[path] = r.Reason
				}
			}
			parts := strings.SplitN(key, ".", 2)
			if len(parts) == 1 {
				inspect(key, d.Fields[key], nil)
				continue
			}
			rows := d.Tables[parts[0]]
			if len(rows) == 0 {
				inspect(key, "", nil)
			}
			for i, row := range rows {
				inspect(fmt.Sprintf("%s.%d.%s", parts[0], i, parts[1]), row[parts[1]], row)
			}
		}
	}
	return missing, unknown
}
func AssessDocumentForm(d DocumentForm, action string) *FormAssessment {
	missing, unknown := FormRuleIssues(d, action)
	differences := FormErrors{}
	validateFormAmounts(d, differences)
	if d.Type == "bank_statement" {
		validateStatementEvidence(d, differences)
	}
	status := "checked"
	if action == "draft" {
		status = "not_assessed"
	}
	if len(missing) > 0 || len(unknown) > 0 || len(differences) > 0 {
		status = "incomplete"
	}
	return &FormAssessment{RuleVersion: DocumentForms.RuleVersion, Action: action, EvidenceStatus: status, Missing: missing, Unresolved: unknown, Differences: differences, Calculated: CalculateForm(d), TaxStatus: "not_assessed"}
}

func validateStatementEvidence(d DocumentForm, issues FormErrors) {
	balance, known := FormCents(d.Fields["opening_balance"])
	seen := map[string]bool{}
	for i, row := range d.Tables["transactions"] {
		path := fmt.Sprintf("transactions.%d.", i)
		credit, c := FormCents(row["credit"])
		debit, b := FormCents(row["debit"])
		next, n := FormCents(row["balance"])
		if known && c && b && n && next != balance+credit-debit {
			issues[path+"balance"] = "ยอดคงเหลือไม่ต่อเนื่องกับยอดก่อนหน้าและเงินเข้าออก"
		}
		balance, known = next, n
		if row["reference"] != "" {
			key := row["date"] + "|" + row["reference"] + "|" + row["credit"] + "|" + row["debit"]
			if seen[key] {
				issues[path+"reference"] = "ธุรกรรมอาจซ้ำ ตรวจเลขอ้างอิง วันที่ และยอด"
			}
			seen[key] = true
		}
	}
	closing, c := FormCents(d.Fields["closing_balance"])
	if known && c && closing != balance {
		issues["closing_balance"] = "ยอดปิดไม่ตรงกับยอดคงเหลือรายการสุดท้าย"
	}
}

// No official issuance, ledger posting, or tax claim is implemented by this
// document-data editor. Completeness must never become authority to do them.
func ValidateFormAction(d DocumentForm, action string) FormErrors {
	if action == "draft" || action == "confirm" {
		return ValidateDocumentForm(d, action == "confirm")
	}
	missing, unknown := FormRuleIssues(d, action)
	for k, v := range unknown {
		missing[k] = v
	}
	for k, v := range ValidateDocumentForm(d, true) {
		missing[k] = v
	}
	validateFormAmounts(d, missing)
	missing["action"] = "ฟอร์มนี้รองรับเก็บข้อมูลและส่งตรวจ ยังไม่รองรับออกเอกสาร ลงสมุดรายวัน หรือรับรองสิทธิภาษี"
	return missing
}
func formAmount(n int64) string {
	sign := ""
	if n < 0 {
		sign = "-"
		n = -n
	}
	return fmt.Sprintf("%s%d.%02d", sign, n/100, n%100)
}
func CalculateForm(d DocumentForm) map[string]string {
	out := map[string]string{}
	f := d.ActiveFields()
	sum := func(table, key string) (int64, bool) {
		var n int64
		rows := d.Tables[table]
		for i, row := range rows {
			v, ok := FormCents(row[key])
			if !ok && table == "items" && key == "amount" {
				v, ok = FormCents(out[fmt.Sprintf("items.%d.amount", i)])
			}
			if !ok {
				return 0, false
			}
			n += v
		}
		return n, len(rows) > 0
	}
	for _, table := range DocumentForms.Types[d.Type].Tables {
		if table == "items" {
			for i, row := range d.Tables[table] {
				n, ok := roundedFormProduct(row["quantity"], row["unit_price"], false)
				discount, known := FormCents(row["discount"])
				if ok && known && n >= discount {
					out[fmt.Sprintf("items.%d.amount", i)] = formAmount(n - discount)
				}
			}
		}
	}
	if _, active := f["price_mode"]; active {
		total, known := sum("items", "amount")
		discount, disc := FormCents(f["discount"])
		if known && disc && total >= discount {
			if f["price_mode"] == "exclusive" {
				out["subtotal"] = formAmount(total - discount)
			}
			if f["price_mode"] == "inclusive" {
				out["total_amount"] = formAmount(total - discount)
			}
		}
		base := f["subtotal"]
		if out["subtotal"] != "" {
			base = out["subtotal"]
		}
		vat, v := roundedFormProduct(base, f["vat_rate"], true)
		if v {
			out["vat_amount"] = formAmount(vat)
		} else {
			vat, v = FormCents(f["vat_amount"])
		}
		sub, b := FormCents(base)
		if b && v && out["total_amount"] == "" {
			out["total_amount"] = formAmount(sub + vat)
		}
	}
	if d.Type == "billing_note" {
		if n, ok := sum("references", "allocated"); ok {
			out["total_amount"] = formAmount(n)
		}
	}
	if d.Type == "withholding_tax_certificate" {
		for _, x := range [][2]string{{"amount", "income_total"}, {"tax", "withholding_tax"}} {
			if n, ok := sum("income", x[0]); ok {
				out[x[1]] = formAmount(n)
			}
		}
	}
	total, t := FormCents(f["total_amount"])
	tax, w := FormCents(f["withholding_tax"])
	if _, active := f["net_amount"]; active && t && w && total >= tax {
		out["net_amount"] = formAmount(total - tax)
	}
	if d.Type == "credit_note" || d.Type == "debit_note" {
		before, b := FormCents(f["before_amount"])
		if b && t {
			if d.Type == "credit_note" {
				total = -total
			}
			if before+total >= 0 {
				out["after_amount"] = formAmount(before + total)
			}
		}
	}
	return out
}

func SortedFormIssues(issues FormErrors) []string {
	keys := make([]string, 0, len(issues))
	for k := range issues {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
