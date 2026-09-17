package business

import (
	"archive/zip"
	"bytes"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"
)

func TestNormalizeVendorRejectsInvalidAndCanonicalizesThaiTaxIdentity(t *testing.T) {
	vendor, err := NormalizeVendor(VendorInput{DisplayName: " บริษัท ตัวอย่าง ", Country: "TH", TaxID: "010-5552-11771-8", BranchCode: ""})
	if err != nil {
		t.Fatal(err)
	}
	if vendor.DisplayName != "บริษัท ตัวอย่าง" || vendor.TaxID != "0105552117718" || vendor.BranchCode != "00000" {
		t.Fatalf("vendor=%+v", vendor)
	}
	if _, err := NormalizeVendor(VendorInput{DisplayName: "Bad", Country: "TH", TaxID: "0105552117719"}); err == nil {
		t.Fatal("invalid checksum accepted")
	}
	if _, err := NormalizeVendor(VendorInput{DisplayName: "Bad\x00Name"}); err == nil {
		t.Fatal("NUL accepted")
	}
}

func TestCSVImportRejectsFormulaLikeInput(t *testing.T) {
	_, err := ParseImport("vendors.csv", "text/csv", []byte("display_name,tax_id\r\n=HYPERLINK(\"https://evil\"),0105552117718\r\n"))
	if err == nil {
		t.Fatal("formula-like CSV value accepted")
	}
}

func TestCSVImportRejectsTwoColumnsForTheSameField(t *testing.T) {
	_, err := ParseImport("vendors.csv", "text/csv", []byte("display_name,name\r\nAcme,Other\r\n"))
	if !errors.Is(err, ErrInvalid) {
		t.Fatalf("err=%v", err)
	}
}

func TestXLSXImportRejectsFormulaCells(t *testing.T) {
	var output bytes.Buffer
	archive := zip.NewWriter(&output)
	file, _ := archive.Create("xl/worksheets/sheet1.xml")
	_, _ = io.WriteString(file, `<worksheet><sheetData><row><c r="A1"><f>HYPERLINK("https://evil")</f><v>1</v></c></row></sheetData></worksheet>`)
	_ = archive.Close()
	if _, err := ParseImport("vendors.xlsx", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", output.Bytes()); err == nil {
		t.Fatal("formula cell accepted")
	}
}

func TestXLSXRoundTripUsesTextCells(t *testing.T) {
	contacts := []Contact{{ID: "v1", DisplayName: "=อันตราย", ContactCode: "V-001", Country: "TH", TaxID: "0105552117718", BranchCode: "00000", Vendor: true}}
	var output bytes.Buffer
	if err := WriteXLSX(&output, contacts); err != nil {
		t.Fatal(err)
	}
	archive, err := zip.NewReader(bytes.NewReader(output.Bytes()), int64(output.Len()))
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range archive.File {
		if file.Name != "xl/worksheets/sheet1.xml" {
			continue
		}
		reader, _ := file.Open()
		sheet, _ := io.ReadAll(reader)
		_ = reader.Close()
		if strings.Contains(string(sheet), "<f>") || !strings.Contains(string(sheet), `t="inlineStr"`) {
			t.Fatalf("unsafe sheet=%s", sheet)
		}
	}
	rows, err := ParseImport("vendors.xlsx", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", output.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].DisplayName != "=อันตราย" {
		t.Fatalf("rows=%+v", rows)
	}
}

func TestCSVExportProtectsFormulaInjection(t *testing.T) {
	var output bytes.Buffer
	if err := WriteCSV(&output, []Contact{{DisplayName: "=cmd", Vendor: true}}); err != nil {
		t.Fatal(err)
	}
	reader := csv.NewReader(strings.NewReader(strings.TrimPrefix(output.String(), "\ufeff")))
	records, err := reader.ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	if records[1][1] != "'=cmd" {
		t.Fatalf("display_name=%q", records[1][1])
	}
}

func BenchmarkCSVImport10000Rows(b *testing.B) {
	var input strings.Builder
	input.WriteString("display_name,contact_code\r\n")
	for index := 0; index < 10000; index++ {
		_, _ = fmt.Fprintf(&input, "Vendor %d,V-%d\r\n", index, index)
	}
	data := []byte(input.String())
	b.ResetTimer()
	for range b.N {
		if _, err := ParseImport("vendors.csv", "text/csv", data); err != nil {
			b.Fatal(err)
		}
	}
}
