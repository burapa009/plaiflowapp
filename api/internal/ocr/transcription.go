package ocr

import (
	"errors"
	"strings"
	"unicode/utf8"
)

// ValidateTranscription keeps generated suggestions tied to literal source evidence.
func (p Page) ValidateTranscription() error {
	invalid := errors.New("invalid OCR transcription")
	if !utf8.ValidString(p.Text) || strings.TrimSpace(p.Text) == "" || utf8.RuneCountInString(p.Text) > 16000 || len(p.Lines) != 0 || p.Rotation != 0 || p.AverageConfidence != 0 || p.LineCount != 0 || p.LowConfidenceCount != 0 || p.TranscriptionMS < 0 || p.ExtractionMS < 0 || p.TranscriptionMS > 600000 || p.ExtractionMS > 600000 || p.TranscriptionMS+p.ExtractionMS > p.DurationMS+2 || len(p.Proposals) > 11 {
		return invalid
	}
	lines := strings.Split(p.Text, "\n")
	if len(lines) > 5000 {
		return invalid
	}
	excluded := make(map[int]bool)
	inFigure := false
	for i, line := range lines {
		if len(line) > 16384 {
			return invalid
		}
		lower := strings.ToLower(line)
		if strings.Contains(lower, "<figure") {
			inFigure = true
		}
		if inFigure {
			excluded[i+1] = true
		}
		if strings.Contains(lower, "</figure") {
			inFigure = false
		}
	}
	seen := make(map[string]bool)
	for _, proposal := range p.Proposals {
		switch proposal.Field {
		case "document_number", "issue_date", "seller_name", "seller_tax_id", "seller_branch", "buyer_name", "buyer_tax_id", "currency", "subtotal", "vat_amount", "total_amount":
		default:
			return invalid
		}
		if seen[proposal.Field] || strings.TrimSpace(proposal.Raw) == "" || utf8.RuneCountInString(proposal.Raw) > 240 || proposal.Line < 1 || proposal.Line > len(lines) || excluded[proposal.Line] || !strings.Contains(lines[proposal.Line-1], proposal.Raw) {
			return invalid
		}
		seen[proposal.Field] = true
	}
	return nil
}
