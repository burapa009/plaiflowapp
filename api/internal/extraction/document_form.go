package extraction

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"math/big"
	"regexp"
	"strconv"
	"strings"
	"time"
)

//go:embed document-forms.json
var formConfigJSON []byte

type FormField struct {
	Label   string
	Kind    string
	Options [][]string
}
type FormSection struct {
	Label  string
	Fields []string
}
type FormTable struct {
	Label   string
	Columns [][]string
}
type FormType struct {
	Label    string
	Sections []string
	Tables   []string
	Roles    []string
	Required []string
	Issuance bool
}
type FormConfig struct {
	Fields      map[string]FormField
	Sections    map[string]FormSection
	Tables      map[string]FormTable
	Types       map[string]FormType
	RuleVersion string
	Rules       map[string]map[string]FieldRule
}

var DocumentForms = func() FormConfig {
	var c FormConfig
	if err := json.Unmarshal(formConfigJSON, &c); err != nil {
		panic(err)
	}
	return c
}()

// DocumentForm is the canonical human-edited data. Hidden sections are retained
// but only the active type participates in validation, exports and printing.
type DocumentForm struct {
	Version    int                            `json:"version"`
	Type       string                         `json:"type"`
	Fields     map[string]string              `json:"fields"`
	Tables     map[string][]map[string]string `json:"tables"`
	Assessment *FormAssessment                `json:"assessment,omitempty"`
	Source     *FormSource                    `json:"source,omitempty"`
}
type FormRecord struct {
	Data      DocumentForm `json:"data"`
	Revision  int          `json:"revision"`
	OCRJobID  string       `json:"ocr_job_id"`
	Status    string       `json:"status"`
	UpdatedAt time.Time    `json:"updated_at"`
}
type FormStore interface {
	ListFormReferences(context.Context, string, string) ([]FormReference, error)
	GetDocumentForm(context.Context, string, string, string) (FormRecord, error)
	SaveDocumentForm(context.Context, string, string, string, string, DocumentForm, int, int, string, time.Time) (FormRecord, error)
}
type FormReference struct {
	ID   string       `json:"id"`
	Data DocumentForm `json:"data"`
}
type FormErrors map[string]string

func (e FormErrors) Error() string { return "invalid document form" }

var formMoney = regexp.MustCompile(`^(0|[1-9][0-9]{0,11})(\.[0-9]{1,2})?$`)
var formDecimal = regexp.MustCompile(`^(0|[1-9][0-9]{0,11})(\.[0-9]{1,4})?$`)
var formTax = regexp.MustCompile(`^[0-9]{13}$`)
var formBranch = regexp.MustCompile(`^[0-9]{5}$`)

func (d DocumentForm) ActiveFields() map[string]string {
	out := map[string]string{}
	for _, section := range DocumentForms.Types[d.Type].Sections {
		for _, key := range DocumentForms.Sections[section].Fields {
			out[key] = d.Fields[key]
		}
	}
	return out
}
func (d DocumentForm) LegacyValues() map[string]string {
	active := d.ActiveFields()
	out := map[string]string{}
	for _, key := range Keys {
		out[key] = active[key]
	}
	return out
}
func validFormValue(kind, value string) bool {
	if value == "" {
		return true
	}
	switch kind {
	case "money":
		return formMoney.MatchString(value)
	case "signed_money":
		return formMoney.MatchString(strings.TrimPrefix(value, "-"))
	case "decimal":
		return formDecimal.MatchString(value)
	case "tax":
		return formTax.MatchString(value)
	case "branch":
		return formBranch.MatchString(value)
	case "date":
		_, err := time.Parse("2006-01-02", value)
		return err == nil
	case "time":
		_, err := time.Parse("15:04", value)
		return err == nil
	}
	return true
}
func ValidateDocumentForm(d DocumentForm, confirm bool) FormErrors {
	errs := FormErrors{}
	typ, ok := DocumentForms.Types[d.Type]
	if !ok || d.Version != 1 {
		errs["type"] = "เลือกประเภทเอกสารที่รองรับ"
		return errs
	}
	if len(d.Fields) > len(DocumentForms.Fields) || len(d.Tables) > len(DocumentForms.Tables) {
		errs["type"] = "ข้อมูลมากเกินขอบเขต"
		return errs
	}
	for key, value := range d.Fields {
		field, ok := DocumentForms.Fields[key]
		if !ok || len([]rune(value)) > 4000 {
			errs[key] = "ช่องข้อมูลไม่ถูกต้องหรือยาวเกินไป"
			continue
		}
		// Drafts may contain incomplete values. Confirmation validates visible fields only.
		if _, active := d.ActiveFields()[key]; active && FormFieldVisible(d, key) && !validFormValue(field.Kind, value) {
			errs[key] = "รูปแบบข้อมูลไม่ถูกต้อง"
		}
		if value != "" && len(field.Options) > 0 && FormFieldVisible(d, key) {
			found := false
			for _, option := range field.Options {
				if option[0] == value {
					found = true
				}
			}
			if !found {
				errs[key] = "เลือกค่าที่รองรับ"
			}
		}
	}
	for table, rows := range d.Tables {
		spec, ok := DocumentForms.Tables[table]
		if !ok || len(rows) > 200 {
			errs[table] = "ตารางไม่รองรับหรือเกิน 200 แถว"
			continue
		}
		columns := map[string]string{}
		for _, c := range spec.Columns {
			columns[c[0]] = c[2]
		}
		active := false
		for _, t := range typ.Tables {
			if t == table {
				active = true
			}
		}
		for i, row := range rows {
			for key, value := range row {
				kind, ok := columns[key]
				if !ok || len([]rune(value)) > 2000 || active && FormFieldVisible(d, table+"."+key) && !validFormValue(kind, value) {
					errs[fmt.Sprintf("%s.%d.%s", table, i, key)] = "ข้อมูลในแถวไม่ถูกต้อง"
				}
			}
		}
	}
	if d.Fields["currency"] != "" && !SupportedCurrency(d.Fields["currency"]) {
		errs["currency"] = "สกุลเงินไม่รองรับ"
	}
	if d.ActiveFields()["price_mode"] != "" && d.Fields["price_mode"] != "inclusive" && d.Fields["price_mode"] != "exclusive" {
		errs["price_mode"] = "เลือกโหมดรวม VAT หรือไม่รวม VAT"
	}
	// Amounts are entered from evidence; never infer an unknown amount or tax rate.
	if d.Type == "bank_statement" && d.Fields["period_from"] != "" && d.Fields["period_to"] != "" && d.Fields["period_from"] > d.Fields["period_to"] {
		errs["period_to"] = "วันสิ้นสุดต้องไม่ก่อนวันเริ่มต้น"
	}
	// Missing evidence and arithmetic differences are recorded separately; OCR
	// acknowledgement is not accounting approval or a tax eligibility decision.
	return errs
}

// MergeDocumentForm distinguishes omitted keys from explicit empty strings / empty rows.
// Each supplied table replaces that table; rows are never implicitly appended.
func MergeDocumentForm(previous, incoming DocumentForm) DocumentForm {
	fields := map[string]string{}
	for k, v := range previous.Fields {
		fields[k] = v
	}
	for k, v := range incoming.Fields {
		fields[k] = v
	}
	tables := map[string][]map[string]string{}
	for k, v := range previous.Tables {
		tables[k] = v
	}
	for k, v := range incoming.Tables {
		tables[k] = v
	}
	incoming.Fields, incoming.Tables = fields, tables
	return incoming
}

// FormCents parses bounded decimal text without floating point.
func FormCents(value string) (int64, bool) {
	if !formMoney.MatchString(strings.TrimPrefix(value, "-")) {
		return 0, false
	}
	r, ok := new(big.Rat).SetString(value)
	if !ok {
		return 0, false
	}
	r.Mul(r, big.NewRat(100, 1))
	return r.Num().Int64(), true
}

func roundedFormProduct(a, b string, percent bool) (int64, bool) {
	if !formDecimal.MatchString(a) || !formDecimal.MatchString(b) {
		return 0, false
	}
	x, _ := new(big.Rat).SetString(a)
	y, _ := new(big.Rat).SetString(b)
	x.Mul(x, y)
	if !percent {
		x.Mul(x, big.NewRat(100, 1))
	}
	x.Add(x, big.NewRat(1, 2)) // half up to the nearest satang
	n := new(big.Int).Quo(x.Num(), x.Denom())
	if !n.IsInt64() {
		return 0, false
	}
	return n.Int64(), true
}

func validateFormAmounts(d DocumentForm, errs FormErrors) {
	f := d.ActiveFields()
	// Calculated intermediate amounts can satisfy required fields without being
	// copied into the source evidence. Compare entered values against that same
	// calculation so leaving an intermediate blank does not hide a discrepancy.
	for path, calculated := range CalculateForm(d) {
		value := f[path]
		parts := strings.Split(path, ".")
		if len(parts) == 3 {
			index, err := strconv.Atoi(parts[1])
			rows := d.Tables[parts[0]]
			if err == nil && index >= 0 && index < len(rows) {
				value = rows[index][parts[2]]
			}
		}
		got, entered := FormCents(value)
		want, known := FormCents(calculated)
		if entered && known && got != want {
			errs[path] = "ยอดไม่ตรงกับรายการ กรุณาตรวจเทียบต้นฉบับ"
		}
	}
	equal := func(key string, want int64, known bool) {
		got, ok := FormCents(f[key])
		if known && ok && got != want {
			errs[key] = "ยอดไม่ตรงกับรายการ กรุณาตรวจเทียบต้นฉบับ"
		}
	}
	sum := func(table, key string) (int64, bool) {
		rows := d.Tables[table]
		var total int64
		for _, row := range rows {
			n, ok := FormCents(row[key])
			if !ok {
				return 0, false
			}
			total += n
		}
		return total, len(rows) > 0
	}
	for _, table := range DocumentForms.Types[d.Type].Tables {
		for i, row := range d.Tables[table] {
			path := func(key string) string { return fmt.Sprintf("%s.%d.%s", table, i, key) }
			if table == "items" {
				n, ok := roundedFormProduct(row["quantity"], row["unit_price"], false)
				discount, known := FormCents(row["discount"])
				amount, hasAmount := FormCents(row["amount"])
				if ok && known && hasAmount && n-discount != amount {
					errs[path("amount")] = "จำนวนเงินไม่ตรงกับจำนวน × ราคา หักส่วนลด"
				}
			}
			if table == "income" {
				for _, key := range []string{"description", "payment_date", "amount", "tax"} {
					if row[key] == "" {
						errs[path(key)] = "กรอกข้อมูลรายการเงินได้"
					}
				}
				n, ok := roundedFormProduct(row["amount"], row["rate"], true)
				tax, known := FormCents(row["tax"])
				if ok && known && n != tax {
					errs[path("tax")] = "ภาษีไม่ตรงกับยอดเงินได้และอัตรา"
				}
			}
			if table == "references" && d.Type == "billing_note" && row["allocated"] == "" {
				errs[path("allocated")] = "ระบุยอดที่นำมาวางบิล"
			}
			if table == "transactions" {
				if row["date"] == "" {
					errs[path("date")] = "ระบุวันที่ทำรายการ"
				}
				credit, c := FormCents(row["credit"])
				debit, b := FormCents(row["debit"])
				if !c && !b {
					errs[path("credit")] = "ระบุเงินเข้าหรือเงินออกจากต้นฉบับ"
				}
				if credit > 0 && debit > 0 {
					errs[path("debit")] = "แยกเงินเข้าและเงินออกคนละรายการ"
				}
				allocated, a := FormCents(row["allocated"])
				if a && allocated > credit+debit {
					errs[path("allocated")] = "ยอดจัดสรรเกินยอดธุรกรรม"
				}
				if row["document_id"] == "" && a && allocated > 0 {
					errs[path("document_id")] = "เลือกเอกสารสำหรับยอดจัดสรร"
				}
			}
		}
	}
	if _, active := f["price_mode"]; active {
		base, b := FormCents(f["subtotal"])
		vat, v := FormCents(f["vat_amount"])
		equal("total_amount", base+vat, b && v)
		calculated, ok := roundedFormProduct(f["subtotal"], f["vat_rate"], true)
		equal("vat_amount", calculated, ok)
		items, i := sum("items", "amount")
		discount, disc := FormCents(f["discount"])
		if f["price_mode"] == "exclusive" {
			equal("subtotal", items-discount, i && disc)
		}
		if f["price_mode"] == "inclusive" {
			equal("total_amount", items-discount, i && disc)
		}
	}
	if d.Type == "billing_note" {
		n, ok := sum("references", "allocated")
		equal("total_amount", n, ok)
	}
	if d.Type == "withholding_tax_certificate" {
		n, ok := sum("income", "amount")
		equal("income_total", n, ok)
		n, ok = sum("income", "tax")
		equal("withholding_tax", n, ok)
	}
	if _, active := f["net_amount"]; active {
		total, t := FormCents(f["total_amount"])
		tax, w := FormCents(f["withholding_tax"])
		if t && w && tax > total {
			errs["withholding_tax"] = "ภาษีหักเกินยอดก่อนหัก"
		}
		equal("net_amount", total-tax, t && w)
	}
	if d.Type == "credit_note" || d.Type == "debit_note" {
		before, b := FormCents(f["before_amount"])
		amount, a := FormCents(f["total_amount"])
		if d.Type == "credit_note" {
			amount = -amount
		}
		equal("after_amount", before+amount, b && a)
	}
}

func FormFromDraft(d Draft) DocumentForm {
	f := DocumentForm{Version: 1, Type: d.DocumentType, Fields: map[string]string{}, Tables: map[string][]map[string]string{}}
	if f.Type == "receipt_tax_invoice" {
		f.Type = "tax_invoice_receipt"
	}
	if _, ok := DocumentForms.Types[f.Type]; !ok {
		f.Type = "unknown"
	}
	for key, v := range d.Fields {
		if v.Presence == "found" {
			f.Fields[key] = v.Normalized
		}
	}
	if d.Accounting != nil {
		a := d.Accounting
		for key, p := range map[string]*string{"seller_address": a.Seller.Address, "buyer_address": a.Buyer.Address, "buyer_branch": a.Buyer.Branch, "due_date": a.Document.DueDate, "payment_method": a.Payment.PaymentMethod, "bank_name": a.Payment.BankName, "payment_reference": a.Payment.TransactionReference} {
			if p != nil {
				f.Fields[key] = *p
			}
		}
		for key, p := range map[string]*json.Number{"discount": a.Summary.Discount, "vat_rate": a.Summary.VATRate, "paid_amount": a.Summary.PaidAmount, "withholding_tax": a.Summary.WithholdingTax} {
			if p != nil {
				f.Fields[key] = p.String()
			}
		}
		for _, item := range a.Items {
			row := map[string]string{}
			if item.Description != nil {
				row["description"] = *item.Description
			}
			if item.Unit != nil {
				row["unit"] = *item.Unit
			}
			for key, p := range map[string]*json.Number{"quantity": item.Quantity, "unit_price": item.UnitPrice, "discount": item.Discount, "amount": item.Amount} {
				if p != nil {
					row[key] = p.String()
				}
			}
			f.Tables["items"] = append(f.Tables["items"], row)
		}
	}
	return f
}
