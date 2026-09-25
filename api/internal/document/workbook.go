package document

import (
	"archive/zip"
	"bufio"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"strings"
	"unicode/utf8"
)

// WorkbookSheet streams rows into one sheet; callers must reauthorize during Rows.
type WorkbookSheet struct {
	Name   string
	Header []string
	Rows   func(func([]string) error) error
}

func WriteWorkbook(output io.Writer, sheets []WorkbookSheet) error {
	if len(sheets) < 1 || len(sheets) > 21 {
		return errors.New("invalid sheet count")
	}
	seen := map[string]bool{}
	for _, sheet := range sheets {
		if sheet.Rows == nil || sheet.Name == "" || utf8.RuneCountInString(sheet.Name) > 31 ||
			strings.ContainsAny(sheet.Name, `[]:*?/\"`) || strings.IndexFunc(sheet.Name, func(r rune) bool { return r < 0x20 }) >= 0 ||
			strings.HasPrefix(sheet.Name, "'") || strings.HasSuffix(sheet.Name, "'") ||
			seen[strings.ToLower(sheet.Name)] {
			return errors.New("invalid sheet name")
		}
		seen[strings.ToLower(sheet.Name)] = true
	}
	archive := zip.NewWriter(&workbookLimit{writer: output, remaining: 128 << 20})
	part := func(name, content string) error {
		writer, err := archive.Create(name)
		if err != nil {
			return err
		}
		_, err = io.WriteString(writer, content)
		return err
	}
	contentTypes := `<?xml version="1.0" encoding="UTF-8"?><Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types"><Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/><Default Extension="xml" ContentType="application/xml"/><Override PartName="/xl/workbook.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.sheet.main+xml"/>`
	workbook := `<?xml version="1.0" encoding="UTF-8"?><workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"><sheets>`
	rels := `<?xml version="1.0" encoding="UTF-8"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">`
	totalRows := 0
	for i, sheet := range sheets {
		n := i + 1
		contentTypes += fmt.Sprintf(`<Override PartName="/xl/worksheets/sheet%d.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.worksheet+xml"/>`, n)
		var escaped strings.Builder
		if err := xml.EscapeText(&escaped, []byte(sheet.Name)); err != nil {
			return err
		}
		workbook += fmt.Sprintf(`<sheet name="%s" sheetId="%d" r:id="rId%d"/>`, escaped.String(), n, n)
		rels += fmt.Sprintf(`<Relationship Id="rId%d" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="worksheets/sheet%d.xml"/>`, n, n)
	}
	if err := part("[Content_Types].xml", contentTypes+`</Types>`); err != nil {
		return err
	}
	if err := part("_rels/.rels", `<?xml version="1.0" encoding="UTF-8"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="xl/workbook.xml"/></Relationships>`); err != nil {
		return err
	}
	if err := part("xl/workbook.xml", workbook+`</sheets></workbook>`); err != nil {
		return err
	}
	if err := part("xl/_rels/workbook.xml.rels", rels+`</Relationships>`); err != nil {
		return err
	}
	for i, sheet := range sheets {
		writer, err := archive.Create(fmt.Sprintf("xl/worksheets/sheet%d.xml", i+1))
		if err != nil {
			return err
		}
		buffer := bufio.NewWriter(writer)
		if _, err = io.WriteString(buffer, `<?xml version="1.0" encoding="UTF-8"?><worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><sheetData>`); err != nil {
			return err
		}
		index := 1
		if err = writeWorkbookRow(buffer, index, sheet.Header); err != nil {
			return err
		}
		err = sheet.Rows(func(row []string) error {
			if len(row) != len(sheet.Header) {
				return errors.New("workbook row width changed")
			}
			index++
			if sheet.Name != "Summary" {
				totalRows++
			}
			if totalRows > 50000 {
				return errors.New("workbook row limit exceeded")
			}
			return writeWorkbookRow(buffer, index, row)
		})
		if err != nil {
			return err
		}
		if _, err = io.WriteString(buffer, `</sheetData></worksheet>`); err != nil {
			return err
		}
		if err = buffer.Flush(); err != nil {
			return err
		}
	}
	return archive.Close()
}

type workbookLimit struct {
	writer    io.Writer
	remaining int64
}

func (l *workbookLimit) Write(p []byte) (int, error) {
	if int64(len(p)) > l.remaining {
		return 0, errors.New("workbook exceeds 128 MiB")
	}
	n, err := l.writer.Write(p)
	l.remaining -= int64(n)
	if err == nil && n != len(p) {
		err = io.ErrShortWrite
	}
	return n, err
}

func writeWorkbookRow(output io.Writer, index int, values []string) error {
	if _, err := fmt.Fprintf(output, `<row r="%d">`, index); err != nil {
		return err
	}
	for column, value := range values {
		if _, err := fmt.Fprintf(output, `<c r="%s%d" t="inlineStr"><is><t xml:space="preserve">`, columnName(column), index); err != nil {
			return err
		}
		value = strings.Map(func(r rune) rune {
			if r < 0x20 && r != '\t' && r != '\n' && r != '\r' {
				return '\ufffd'
			}
			return r
		}, value)
		if err := xml.EscapeText(output, []byte(value)); err != nil {
			return err
		}
		if _, err := io.WriteString(output, `</t></is></c>`); err != nil {
			return err
		}
	}
	_, err := io.WriteString(output, `</row>`)
	return err
}
