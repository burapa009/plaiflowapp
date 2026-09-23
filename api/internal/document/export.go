package document

import (
	"archive/zip"
	"bufio"
	"encoding/csv"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

var exportHeader = []string{"document_id", "display_filename", "detected_type", "byte_size", "status", "first_source_channel", "source_count", "submitter", "assignee", "received_at", "updated_at"}

func WriteExport(output io.Writer, format string, rows []ExportRow) error {
	values := make([][]string, 0, len(rows))
	for _, row := range rows {
		values = append(values, exportValues(row))
	}
	return WriteTable(output, format, exportHeader, values)
}

// WriteTable reuses the fixed CSV/XLSX safety rules for other structured document exports.
func WriteTable(output io.Writer, format string, header []string, values [][]string) error {
	switch format {
	case "csv":
		return writeCSV(output, header, values)
	case "xlsx":
		return writeXLSX(output, header, values)
	default:
		return errors.New("unsupported document export format")
	}
}

func writeCSV(output io.Writer, header []string, rows [][]string) error {
	if _, err := io.WriteString(output, "\ufeff"); err != nil {
		return err
	}
	w := csv.NewWriter(output)
	w.UseCRLF = true
	if err := w.Write(header); err != nil {
		return err
	}
	for _, row := range rows {
		values := append([]string(nil), row...)
		for i, value := range values {
			if spreadsheetFormula(value) {
				values[i] = "'" + value
			}
		}
		if err := w.Write(values); err != nil {
			return err
		}
	}
	w.Flush()
	return w.Error()
}

func exportValues(row ExportRow) []string {
	return []string{row.ID, row.Filename, row.MIME, strconv.FormatInt(row.Size, 10), row.Status, row.SourceChannel,
		strconv.FormatInt(row.SourceCount, 10), row.SubmitterName, row.AssigneeName, row.AcceptedAt.UTC().Format("2006-01-02T15:04:05Z"), row.UpdatedAt.UTC().Format("2006-01-02T15:04:05Z")}
}

func spreadsheetFormula(value string) bool {
	trimmed := strings.TrimLeftFunc(value, func(r rune) bool { return unicode.IsSpace(r) || r == '\ufeff' })
	first, _ := utf8.DecodeRuneInString(trimmed)
	return strings.ContainsRune("=+-@＝＋－＠", first)
}

func writeXLSX(output io.Writer, header []string, rows [][]string) error {
	archive := zip.NewWriter(output)
	parts := map[string]string{
		"[Content_Types].xml":        `<?xml version="1.0" encoding="UTF-8"?><Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types"><Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/><Default Extension="xml" ContentType="application/xml"/><Override PartName="/xl/workbook.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.sheet.main+xml"/><Override PartName="/xl/worksheets/sheet1.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.worksheet+xml"/></Types>`,
		"_rels/.rels":                `<?xml version="1.0" encoding="UTF-8"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="xl/workbook.xml"/></Relationships>`,
		"xl/workbook.xml":            `<?xml version="1.0" encoding="UTF-8"?><workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"><sheets><sheet name="Documents" sheetId="1" r:id="rId1"/></sheets></workbook>`,
		"xl/_rels/workbook.xml.rels": `<?xml version="1.0" encoding="UTF-8"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="worksheets/sheet1.xml"/></Relationships>`,
	}
	for name, content := range parts {
		part, err := archive.Create(name)
		if err != nil {
			return err
		}
		if _, err := io.WriteString(part, content); err != nil {
			return err
		}
	}
	sheet, err := archive.Create("xl/worksheets/sheet1.xml")
	if err != nil {
		return err
	}
	buffer := bufio.NewWriter(sheet)
	if _, err := io.WriteString(buffer, `<?xml version="1.0" encoding="UTF-8"?><worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><sheetData>`); err != nil {
		return err
	}
	writeRow := func(index int, values []string) error {
		if _, err := fmt.Fprintf(buffer, `<row r="%d">`, index); err != nil {
			return err
		}
		for column, value := range values {
			if _, err := fmt.Fprintf(buffer, `<c r="%s%d" t="inlineStr"><is><t xml:space="preserve">`, columnName(column), index); err != nil {
				return err
			}
			value = strings.Map(func(r rune) rune {
				if r < 0x20 && r != '\t' && r != '\n' && r != '\r' {
					return '\ufffd'
				}
				return r
			}, value)
			if err := xml.EscapeText(buffer, []byte(value)); err != nil {
				return err
			}
			if _, err := io.WriteString(buffer, `</t></is></c>`); err != nil {
				return err
			}
		}
		_, err := io.WriteString(buffer, `</row>`)
		return err
	}
	if err := writeRow(1, header); err != nil {
		return err
	}
	for i, row := range rows {
		if err := writeRow(i+2, row); err != nil {
			return err
		}
	}
	if _, err := io.WriteString(buffer, `</sheetData></worksheet>`); err != nil {
		return err
	}
	if err := buffer.Flush(); err != nil {
		return err
	}
	return archive.Close()
}

func columnName(index int) string {
	name := ""
	for index >= 0 {
		name = string(rune('A'+index%26)) + name
		index = index/26 - 1
	}
	return name
}
