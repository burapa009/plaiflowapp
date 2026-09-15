package work

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func TestWriteCSVProtectsSpreadsheetCells(t *testing.T) {
	var output bytes.Buffer
	rows := []ExportRow{{
		OrganizationID: "11111111-1111-4111-8111-111111111111", OrganizationName: "บริษัท ทดสอบ",
		TaskID: "22222222-2222-4222-8222-222222222222", Title: " =HYPERLINK(\"https://bad.example\")",
		Description: "\t=cmd", Status: Open, Priority: Urgent, DueDate: "2026-09-15",
		CreatorName: "+ผู้สร้าง", AssigneeName: "@ผู้รับ", WatcherNames: "-ผู้ติดตาม",
		CreatedAt: time.Date(2026, 9, 15, 3, 0, 0, 0, time.UTC), UpdatedAt: time.Date(2026, 9, 15, 4, 0, 0, 0, time.UTC),
	}}
	if err := WriteCSV(&output, rows); err != nil {
		t.Fatal(err)
	}
	written := output.String()
	if !strings.HasPrefix(written, "\ufefforganization_id,") {
		t.Fatalf("missing utf-8 BOM/header: %q", written)
	}
	for _, protected := range []string{"' =HYPERLINK", "'\t=cmd", "'+ผู้สร้าง", "'@ผู้รับ", "'-ผู้ติดตาม"} {
		if !strings.Contains(written, protected) {
			t.Fatalf("missing protected cell %q in %q", protected, written)
		}
	}
}
