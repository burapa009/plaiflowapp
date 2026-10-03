package spreadsheetsafe

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// Formula reports whether a spreadsheet may interpret the cell as a formula
// after ignoring leading whitespace, control, or format characters.
func Formula(value string) bool {
	trimmed := strings.TrimLeftFunc(value, func(r rune) bool {
		return unicode.IsSpace(r) || unicode.IsControl(r) || unicode.Is(unicode.Cf, r) || r == '\ufeff'
	})
	first, _ := utf8.DecodeRuneInString(trimmed)
	return strings.ContainsRune("=+-@＝＋－＠−", first)
}

func CSVCell(value string) string {
	if Formula(value) {
		return "'" + value
	}
	return value
}

func CSVRow(row []string) []string {
	values := make([]string, len(row))
	for i, value := range row {
		values[i] = CSVCell(value)
	}
	return values
}

// XMLText replaces characters disallowed by XML 1.0, keeping XLSX readable.
func XMLText(value string) string {
	return strings.Map(func(r rune) rune {
		if r == '\t' || r == '\n' || r == '\r' || r >= 0x20 && r <= 0xd7ff || r >= 0xe000 && r <= 0xfffd || r >= 0x10000 && r <= 0x10ffff {
			return r
		}
		return '\ufffd'
	}, value)
}
