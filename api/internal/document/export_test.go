package document

import (
	"archive/zip"
	"bytes"
	"encoding/csv"
	"encoding/xml"
	"io"
	"strings"
	"testing"
	"time"
)

func sampleExportRow() ExportRow {
	now := time.Date(2026, 9, 20, 0, 0, 0, 0, time.UTC)
	return ExportRow{ID: "doc-1", Filename: "=unsafe.pdf", MIME: "application/pdf", Size: 12, Status: "Available", SourceChannel: "Web", SourceCount: 1, SubmitterName: "=name", AcceptedAt: now, UpdatedAt: now}
}

func TestWriteExportCSVProtectsFormulaCells(t *testing.T) {
	var output bytes.Buffer
	if err := WriteExport(&output, "csv", []ExportRow{sampleExportRow()}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), "'=unsafe.pdf") || !strings.Contains(output.String(), "'=name") {
		t.Fatalf("csv formula protection missing: %q", output.String())
	}
}

func TestWriteExportXLSXIsReadable(t *testing.T) {
	var output bytes.Buffer
	if err := WriteExport(&output, "xlsx", []ExportRow{sampleExportRow()}); err != nil {
		t.Fatal(err)
	}
	archive, err := zip.NewReader(bytes.NewReader(output.Bytes()), int64(output.Len()))
	if err != nil {
		t.Fatal(err)
	}
	var sheet string
	for _, file := range archive.File {
		if file.Name == "xl/worksheets/sheet1.xml" {
			body, readErr := file.Open()
			if readErr != nil {
				t.Fatal(readErr)
			}
			data, readErr := io.ReadAll(body)
			body.Close()
			if readErr != nil {
				t.Fatal(readErr)
			}
			sheet = string(data)
		}
	}
	if !strings.Contains(sheet, "=unsafe.pdf") || !strings.Contains(sheet, "document_id") {
		t.Fatalf("xlsx sheet missing values: %q", sheet)
	}
}

func TestStructuredTableDoesNotExecuteUntrustedSpreadsheetValues(t *testing.T) {
	var output bytes.Buffer
	if err := WriteTable(&output, "csv", []string{"seller_name"}, [][]string{{"  ＝HYPERLINK(1)"}, {" \ufeff =HYPERLINK(1)"}}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), "'  ＝HYPERLINK(1)") {
		t.Fatalf("full-width formula prefix not neutralized: %q", output.String())
	}
	if !strings.Contains(output.String(), "' \ufeff =HYPERLINK(1)") {
		t.Fatalf("BOM formula prefix not neutralized: %q", output.String())
	}
	output.Reset()
	if err := WriteTable(&output, "xlsx", []string{"seller_name"}, [][]string{{"\x01=HYPERLINK(1)"}}); err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(output.Bytes(), []byte("PK")) {
		t.Fatal("structured XLSX is not a zip archive")
	}
}

func TestStructuredTableProtectsHeaderAndOpensAsText(t *testing.T) {
	var output bytes.Buffer
	if err := WriteTable(&output, "csv", []string{"\u200b=header"}, [][]string{{"\x01=payload"}}); err != nil {
		t.Fatal(err)
	}
	records, err := csv.NewReader(strings.NewReader(strings.TrimPrefix(output.String(), "\ufeff"))).ReadAll()
	if err != nil || len(records) != 2 || records[0][0] != "'\u200b=header" || records[1][0] != "'\x01=payload" {
		t.Fatalf("unsafe CSV records=%q err=%v", records, err)
	}
	output.Reset()
	if err := WriteTable(&output, "xlsx", []string{"=header"}, [][]string{{"\x00=payload"}}); err != nil {
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
		reader, err := file.Open()
		if err != nil {
			t.Fatal(err)
		}
		decoder := xml.NewDecoder(reader)
		cells, formulas := 0, 0
		for {
			token, err := decoder.Token()
			if err == io.EOF {
				break
			}
			if err != nil {
				t.Fatal(err)
			}
			if start, ok := token.(xml.StartElement); ok {
				if start.Name.Local == "f" {
					formulas++
				}
				if start.Name.Local == "c" {
					cells++
					if len(start.Attr) < 2 || start.Attr[1].Value != "inlineStr" {
						t.Fatalf("cell is not text: %+v", start.Attr)
					}
				}
			}
		}
		reader.Close()
		if formulas != 0 || cells != 2 {
			t.Fatalf("formulas=%d cells=%d", formulas, cells)
		}
		return
	}
	t.Fatal("missing worksheet")
}
