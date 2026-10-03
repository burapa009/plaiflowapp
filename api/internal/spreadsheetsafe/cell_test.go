package spreadsheetsafe

import "testing"

func TestCSVCellNeutralizesLeadingVariants(t *testing.T) {
	for _, value := range []string{
		"=HYPERLINK(1)", "\t+cmd", "\r\n@SUM(1)", "\ufeff =cmd", "\u200b=cmd", "\x01=cmd", " \uff1dHYPERLINK(1)", "\u2212cmd",
	} {
		if got := CSVCell(value); got != "'"+value {
			t.Errorf("CSVCell(%q) = %q", value, got)
		}
	}
	if got := CSVCell("ภาษาไทย"); got != "ภาษาไทย" {
		t.Fatalf("safe value changed: %q", got)
	}
}

func TestXMLTextRemovesInvalidRunes(t *testing.T) {
	if got := XMLText("ไทย\x00\uffff=SUM(1)"); got != "ไทย\ufffd\ufffd=SUM(1)" {
		t.Fatalf("XMLText = %q", got)
	}
}
