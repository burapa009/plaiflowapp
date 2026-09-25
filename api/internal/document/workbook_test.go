package document

import (
	"archive/zip"
	"bytes"
	"io"
	"strings"
	"testing"
)

func TestWriteWorkbookStreamsSeparateClientSheetsAsText(t *testing.T) {
	var output bytes.Buffer
	err := WriteWorkbook(&output, []WorkbookSheet{
		{Name: "Summary", Header: []string{"client_id", "rows"}, Rows: func(write func([]string) error) error { return write([]string{"client-a", "1"}) }},
		{Name: "Client-01", Header: []string{"seller_name"}, Rows: func(write func([]string) error) error { return write([]string{"=สูตร ภาษาไทย"}) }},
		{Name: "Client-02", Header: []string{"seller_name"}, Rows: func(func([]string) error) error { return nil }},
	})
	if err != nil {
		t.Fatal(err)
	}
	archive, err := zip.NewReader(bytes.NewReader(output.Bytes()), int64(output.Len()))
	if err != nil {
		t.Fatal(err)
	}
	if len(archive.File) != 7 {
		t.Fatalf("parts=%d", len(archive.File))
	}
	for _, file := range archive.File {
		body, err := file.Open()
		if err != nil {
			t.Fatal(err)
		}
		data, err := io.ReadAll(body)
		body.Close()
		if err != nil {
			t.Fatal(err)
		}
		if file.Name == "xl/worksheets/sheet2.xml" && (!strings.Contains(string(data), `t="inlineStr"`) || !strings.Contains(string(data), "=สูตร ภาษาไทย") || strings.Contains(string(data), "<f>")) {
			t.Fatalf("client sheet=%q", data)
		}
		if file.Name == "xl/worksheets/sheet3.xml" && strings.Count(string(data), "<row ") != 1 {
			t.Fatalf("empty client sheet=%q", data)
		}
	}
}
