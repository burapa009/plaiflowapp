package document

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"io"
	"testing"
	"time"
)

func TestIntakeRejectsMalformedAndAppendedPolyglots(t *testing.T) {
	imageData := image.NewRGBA(image.Rect(0, 0, 1, 1))
	imageData.Set(0, 0, color.White)
	var pngData, jpegData bytes.Buffer
	if err := png.Encode(&pngData, imageData); err != nil {
		t.Fatal(err)
	}
	if err := jpeg.Encode(&jpegData, imageData, nil); err != nil {
		t.Fatal(err)
	}
	pdfData := []byte("%PDF-1.4\n1 0 obj\n<<>>\nendobj\n%%EOF\n")
	storage := &memoryTemp{objects: map[string][]byte{}}
	scans := 0
	intake := Intake{Temporary: storage, Scanner: scanFunc(func(_ context.Context, body io.Reader) error {
		scans++
		_, err := io.Copy(io.Discard, body)
		return err
	})}
	for i, body := range [][]byte{pdfData, pngData.Bytes(), jpegData.Bytes()} {
		if _, err := intake.Prepare(context.Background(), "org", string(rune('a'+i)), bytes.NewReader(body)); err != nil {
			t.Fatalf("valid format %d: %v", i, err)
		}
	}
	if scans != 3 || len(storage.objects) != 3 {
		t.Fatalf("valid scans=%d staged=%d", scans, len(storage.objects))
	}
	for i, body := range [][]byte{
		[]byte("%PDF-fake\n%%EOF"),
		append(bytes.Clone(pdfData), []byte("<script>alert(1)</script>")...),
		append(bytes.Clone(pngData.Bytes()), []byte("PK\x03\x04zip")...),
		append(bytes.Clone(jpegData.Bytes()), []byte("<svg/onload=alert(1)>")...),
	} {
		if _, err := intake.Prepare(context.Background(), "org", string(rune('d'+i)), bytes.NewReader(body)); !errors.Is(err, ErrUnsupportedType) {
			t.Fatalf("malformed/polyglot %d: %v", i, err)
		}
	}
	if scans != 3 || len(storage.objects) != 3 {
		t.Fatalf("rejected data reached scanner or remained staged: scans=%d staged=%d", scans, len(storage.objects))
	}
	tooLarge := append(bytes.Clone(pdfData), bytes.Repeat([]byte("x"), int(MaxFileBytes))...)
	if _, err := intake.Prepare(context.Background(), "org", "oversize", bytes.NewReader(tooLarge)); !errors.Is(err, ErrTooLarge) || len(storage.objects) != 3 {
		t.Fatalf("oversize err=%v staged=%d", err, len(storage.objects))
	}
}

func TestScannerFailureLeavesNoAcceptedDocumentAndAllowsRetry(t *testing.T) {
	storage := &memoryTemp{objects: map[string][]byte{}}
	commits := 0
	failScan := true
	service := Service{Intake: Intake{Temporary: storage, Scanner: scanFunc(func(context.Context, io.Reader) error {
		if failScan {
			return errors.New("scanner offline")
		}
		return nil
	})}, Committer: commitFunc(func(_ context.Context, input CommitInput) (CommitResult, error) {
		commits++
		return CommitResult{Document: Document{ID: "doc", OrganizationID: input.OrganizationID}, Accepted: true}, nil
	})}
	input := AcceptInput{OrganizationID: "org", ActorUserID: "user", AttemptID: "first", OriginKey: "same-upload-key", Filename: "invoice.pdf", Channel: "Web", Now: time.Now()}
	if _, err := service.Accept(context.Background(), input, bytes.NewBufferString("%PDF-1.4\n%%EOF")); !errors.Is(err, ErrScanUnavailable) || commits != 0 || len(storage.objects) != 0 {
		t.Fatalf("failed scan err=%v commits=%d staged=%d", err, commits, len(storage.objects))
	}
	failScan = false
	input.AttemptID = "retry"
	result, err := service.Accept(context.Background(), input, bytes.NewBufferString("%PDF-1.4\n%%EOF"))
	if err != nil || !result.Accepted || commits != 1 || len(storage.objects) != 1 {
		t.Fatalf("retry result=%+v err=%v commits=%d staged=%d", result, err, commits, len(storage.objects))
	}
}
