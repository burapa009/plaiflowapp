package work

import (
	"archive/zip"
	"bytes"
	"io"
	"strings"
	"testing"
)

func TestXLSXWritesUntrustedValuesAsText(t *testing.T) {
	var output bytes.Buffer
	writer, err := NewXLSXWriter(&output)
	if err != nil {
		t.Fatal(err)
	}
	if err = writer.Write(ExportRow{OrganizationID: "org-1", OrganizationName: "=IMPORTXML()", TaskID: "task-1", Title: "+SUM(1,1)"}); err != nil {
		t.Fatal(err)
	}
	if err = writer.Close(); err != nil {
		t.Fatal(err)
	}
	archive, err := zip.NewReader(bytes.NewReader(output.Bytes()), int64(output.Len()))
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range archive.File {
		if file.Name == "xl/worksheets/sheet1.xml" {
			body, _ := file.Open()
			data, _ := io.ReadAll(body)
			body.Close()
			text := string(data)
			if strings.Contains(text, "<f>") || !strings.Contains(text, "<t xml:space=\"preserve\">=IMPORTXML()</t>") || !strings.Contains(text, ">+SUM(1,1)</t>") {
				t.Fatalf("unsafe worksheet: %s", text)
			}
			return
		}
	}
	t.Fatal("worksheet missing")
}
