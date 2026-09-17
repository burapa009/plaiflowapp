package business

import (
	"archive/zip"
	"bufio"
	"bytes"
	"encoding/csv"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strconv"
	"strings"
	"unicode/utf8"
)

const (
	maxImportBytes = 10 << 20
	maxExpanded    = 64 << 20
	maxRows        = 10000
	maxColumns     = 50
	maxCellRunes   = 10000
)

var exportHeader = []string{"vendor_id", "display_name", "contact_code", "country", "tax_id", "branch_code", "roles"}

func WriteCSV(output io.Writer, contacts []Contact) error {
	if _, err := io.WriteString(output, "\ufeff"); err != nil {
		return err
	}
	writer := csv.NewWriter(output)
	writer.UseCRLF = true
	if err := writer.Write(exportHeader); err != nil {
		return err
	}
	for _, contact := range contacts {
		if err := writer.Write(exportRecord(contact, true)); err != nil {
			return err
		}
	}
	writer.Flush()
	return writer.Error()
}

func exportRecord(contact Contact, protectCSV bool) []string {
	roles := "Vendor"
	if contact.Customer {
		roles = "Customer,Vendor"
	}
	values := []string{contact.ID, contact.DisplayName, contact.ContactCode, contact.Country, contact.TaxID, contact.BranchCode, roles}
	if protectCSV {
		for index, value := range values {
			if spreadsheetFormula(value) {
				values[index] = "'" + value
			}
		}
	}
	return values
}

func spreadsheetFormula(value string) bool {
	trimmed := strings.TrimLeft(value, " \t\r\n")
	return trimmed != "" && strings.ContainsRune("=+-@", rune(trimmed[0]))
}

func WriteXLSX(output io.Writer, contacts []Contact) error {
	archive := zip.NewWriter(output)
	files := map[string]string{
		"[Content_Types].xml":        `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types"><Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/><Default Extension="xml" ContentType="application/xml"/><Override PartName="/xl/workbook.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.sheet.main+xml"/><Override PartName="/xl/worksheets/sheet1.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.worksheet+xml"/></Types>`,
		"_rels/.rels":                `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="xl/workbook.xml"/></Relationships>`,
		"xl/workbook.xml":            `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"><sheets><sheet name="Vendors" sheetId="1" r:id="rId1"/></sheets></workbook>`,
		"xl/_rels/workbook.xml.rels": `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="worksheets/sheet1.xml"/></Relationships>`,
	}
	for name, value := range files {
		writer, err := archive.Create(name)
		if err != nil {
			return err
		}
		if _, err := io.WriteString(writer, value); err != nil {
			return err
		}
	}
	sheet, err := archive.Create("xl/worksheets/sheet1.xml")
	if err != nil {
		return err
	}
	buffer := bufio.NewWriter(sheet)
	_, _ = io.WriteString(buffer, `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><sheetData>`)
	rows := make([][]string, 0, len(contacts)+1)
	rows = append(rows, exportHeader)
	for _, contact := range contacts {
		rows = append(rows, exportRecord(contact, false))
	}
	for rowIndex, row := range rows {
		_, _ = fmt.Fprintf(buffer, `<row r="%d">`, rowIndex+1)
		for columnIndex, value := range row {
			_, _ = fmt.Fprintf(buffer, `<c r="%s%d" t="inlineStr"><is><t xml:space="preserve">`, columnName(columnIndex), rowIndex+1)
			if err := xml.EscapeText(buffer, []byte(value)); err != nil {
				return err
			}
			_, _ = io.WriteString(buffer, `</t></is></c>`)
		}
		_, _ = io.WriteString(buffer, `</row>`)
	}
	_, _ = io.WriteString(buffer, `</sheetData></worksheet>`)
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

func ParseImport(filename, contentType string, data []byte) ([]Contact, error) {
	if len(data) == 0 || len(data) > maxImportBytes {
		return nil, ErrInvalid
	}
	extension := strings.ToLower(filepath.Ext(filename))
	switch extension {
	case ".csv":
		mediaType := strings.ToLower(strings.TrimSpace(strings.SplitN(contentType, ";", 2)[0]))
		if mediaType != "" && mediaType != "text/csv" && mediaType != "application/csv" && mediaType != "application/vnd.ms-excel" && mediaType != "text/plain" && mediaType != "application/octet-stream" {
			return nil, ErrInvalid
		}
		return parseCSV(data)
	case ".xlsx":
		if contentType != "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet" || len(data) < 4 || !bytes.Equal(data[:2], []byte("PK")) {
			return nil, ErrInvalid
		}
		return parseXLSX(data)
	default:
		return nil, ErrInvalid
	}
}

func parseCSV(data []byte) ([]Contact, error) {
	data = bytes.TrimPrefix(data, []byte("\xef\xbb\xbf"))
	reader := csv.NewReader(bytes.NewReader(data))
	reader.FieldsPerRecord = -1
	reader.ReuseRecord = true
	var rows [][]string
	for len(rows) <= maxRows {
		record, err := reader.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil || len(record) > maxColumns {
			return nil, ErrInvalid
		}
		copyRecord := append([]string(nil), record...)
		for _, value := range copyRecord {
			if utf8.RuneCountInString(value) > maxCellRunes || strings.ContainsRune(value, '\x00') || spreadsheetFormula(value) {
				return nil, ErrInvalid
			}
		}
		rows = append(rows, copyRecord)
	}
	return importRows(rows)
}

func parseXLSX(data []byte) ([]Contact, error) {
	archive, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil || len(archive.File) > 100 {
		return nil, ErrInvalid
	}
	var expanded uint64
	var sheet *zip.File
	var shared []string
	worksheetCount := 0
	for _, file := range archive.File {
		expanded += file.UncompressedSize64
		name := strings.ToLower(file.Name)
		if expanded > maxExpanded || strings.Contains(name, "vbaproject") || strings.Contains(name, "externallinks/") || strings.Contains(name, "../") {
			return nil, ErrInvalid
		}
		if strings.HasPrefix(name, "xl/worksheets/") && strings.HasSuffix(name, ".xml") {
			worksheetCount++
			sheet = file
		}
		if name == "xl/sharedstrings.xml" {
			shared, err = readSharedStrings(file)
			if err != nil {
				return nil, ErrInvalid
			}
		}
	}
	if worksheetCount != 1 || sheet == nil {
		return nil, ErrInvalid
	}
	rows, err := readSheet(sheet, shared)
	if err != nil {
		return nil, ErrInvalid
	}
	return importRows(rows)
}

func readSharedStrings(file *zip.File) ([]string, error) {
	reader, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	decoder := xml.NewDecoder(io.LimitReader(reader, maxExpanded))
	var values []string
	var current strings.Builder
	inItem, inText := false, false
	for {
		token, err := decoder.Token()
		if errors.Is(err, io.EOF) {
			return values, nil
		}
		if err != nil {
			return nil, err
		}
		switch value := token.(type) {
		case xml.StartElement:
			if value.Name.Local == "si" {
				inItem = true
				current.Reset()
			}
			inText = inItem && value.Name.Local == "t"
		case xml.CharData:
			if inText {
				current.Write(value)
			}
		case xml.EndElement:
			if value.Name.Local == "t" {
				inText = false
			}
			if value.Name.Local == "si" {
				values = append(values, current.String())
				inItem = false
			}
		}
	}
}

func readSheet(file *zip.File, shared []string) ([][]string, error) {
	reader, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	decoder := xml.NewDecoder(io.LimitReader(reader, maxExpanded))
	var rows [][]string
	var row []string
	cellType, cellRef, cellValue := "", "", ""
	inValue, inText := false, false
	for {
		token, err := decoder.Token()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, err
		}
		switch value := token.(type) {
		case xml.StartElement:
			switch value.Name.Local {
			case "row":
				row = nil
			case "c":
				cellType, cellRef, cellValue = attribute(value.Attr, "t"), attribute(value.Attr, "r"), ""
			case "f":
				return nil, ErrInvalid
			case "v":
				inValue = true
			case "t":
				inText = true
			}
		case xml.CharData:
			if inValue || inText {
				cellValue += string(value)
			}
		case xml.EndElement:
			switch value.Name.Local {
			case "v":
				inValue = false
			case "t":
				inText = false
			case "c":
				column := cellColumn(cellRef)
				if column < 0 || column >= maxColumns {
					return nil, ErrInvalid
				}
				for len(row) <= column {
					row = append(row, "")
				}
				if cellType == "s" {
					index, parseErr := strconv.Atoi(cellValue)
					if parseErr != nil || index < 0 || index >= len(shared) {
						return nil, ErrInvalid
					}
					cellValue = shared[index]
				}
				if utf8.RuneCountInString(cellValue) > maxCellRunes || strings.ContainsRune(cellValue, '\x00') {
					return nil, ErrInvalid
				}
				row[column] = cellValue
			case "row":
				rows = append(rows, row)
				if len(rows) > maxRows+1 {
					return nil, ErrInvalid
				}
			}
		}
	}
	return rows, nil
}

func attribute(attributes []xml.Attr, name string) string {
	for _, attribute := range attributes {
		if attribute.Name.Local == name {
			return attribute.Value
		}
	}
	return ""
}

func cellColumn(reference string) int {
	column := 0
	letters := 0
	for _, char := range reference {
		if char < 'A' || char > 'Z' {
			break
		}
		column = column*26 + int(char-'A'+1)
		letters++
	}
	if letters == 0 {
		return -1
	}
	return column - 1
}

func importRows(rows [][]string) ([]Contact, error) {
	if len(rows) < 2 || len(rows) > maxRows+1 {
		return nil, ErrInvalid
	}
	columns := map[string]int{}
	setColumn := func(name string, index int) error {
		if _, duplicate := columns[name]; duplicate {
			return ErrInvalid
		}
		columns[name] = index
		return nil
	}
	for index, header := range rows[0] {
		key := strings.ToLower(strings.TrimSpace(header))
		switch key {
		case "display_name", "name", "ชื่อ", "ชื่อที่แสดง":
			if err := setColumn("display_name", index); err != nil {
				return nil, err
			}
		case "contact_code", "รหัสคู่ค้า":
			if err := setColumn("contact_code", index); err != nil {
				return nil, err
			}
		case "country", "ประเทศ":
			if err := setColumn("country", index); err != nil {
				return nil, err
			}
		case "tax_id", "เลขประจำตัวผู้เสียภาษี":
			if err := setColumn("tax_id", index); err != nil {
				return nil, err
			}
		case "branch_code", "รหัสสาขา":
			if err := setColumn("branch_code", index); err != nil {
				return nil, err
			}
		}
	}
	if _, ok := columns["display_name"]; !ok {
		return nil, ErrInvalid
	}
	contacts := make([]Contact, 0, len(rows)-1)
	for _, row := range rows[1:] {
		value := func(key string) string {
			index, ok := columns[key]
			if !ok || index >= len(row) {
				return ""
			}
			return row[index]
		}
		contact, err := NormalizeVendor(VendorInput{DisplayName: value("display_name"), ContactCode: value("contact_code"), Country: value("country"), TaxID: value("tax_id"), BranchCode: value("branch_code")})
		if err != nil {
			return nil, ErrInvalid
		}
		contacts = append(contacts, contact)
	}
	return contacts, nil
}
