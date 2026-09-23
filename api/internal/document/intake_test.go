package document

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"
)

type memoryTemp struct{ objects map[string][]byte }

type contextCheckingTemp struct{ *memoryTemp }

func (m contextCheckingTemp) Delete(ctx context.Context, key string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return m.memoryTemp.Delete(ctx, key)
}

func (m *memoryTemp) Put(_ context.Context, key string, body io.Reader) error {
	data, err := io.ReadAll(body)
	if err == nil {
		m.objects[key] = data
	}
	return err
}
func (m *memoryTemp) Open(_ context.Context, key string) (io.ReadCloser, error) {
	data, ok := m.objects[key]
	if !ok {
		return nil, errors.New("missing object")
	}
	return io.NopCloser(bytes.NewReader(data)), nil
}
func (m *memoryTemp) Delete(_ context.Context, key string) error {
	delete(m.objects, key)
	return nil
}

type scanFunc func(context.Context, io.Reader) error

func (f scanFunc) Scan(ctx context.Context, body io.Reader) error { return f(ctx, body) }

func TestPrepareAcceptsOneValidPDFAndRejectsInvalidBytes(t *testing.T) {
	ctx := context.Background()
	storage := &memoryTemp{objects: map[string][]byte{}}
	scanner := scanFunc(func(_ context.Context, body io.Reader) error {
		_, err := io.Copy(io.Discard, body)
		return err
	})
	service := Intake{Temporary: storage, Scanner: scanner}
	accepted, err := service.Prepare(ctx, "org-1", "attempt-1", bytes.NewBufferString("%PDF-1.4\n%%EOF"))
	if err != nil || accepted.MIME != "application/pdf" || accepted.Size != 14 || len(accepted.SHA256) != 64 {
		t.Fatalf("accepted=%+v err=%v", accepted, err)
	}
	if len(storage.objects) != 1 {
		t.Fatalf("accepted file not staged: %d", len(storage.objects))
	}
	_, err = service.Prepare(ctx, "org-1", "attempt-2", bytes.NewBufferString("not a pdf"))
	if !errors.Is(err, ErrUnsupportedType) || len(storage.objects) != 1 {
		t.Fatalf("invalid file err=%v staged=%d", err, len(storage.objects))
	}
}

func TestPrepareFailsClosedWhenScannerFails(t *testing.T) {
	storage := &memoryTemp{objects: map[string][]byte{}}
	service := Intake{Temporary: storage, Scanner: scanFunc(func(context.Context, io.Reader) error {
		return errors.New("scanner offline")
	})}
	_, err := service.Prepare(context.Background(), "org-1", "attempt-1", bytes.NewBufferString("%PDF-1.4\n%%EOF"))
	if !errors.Is(err, ErrScanUnavailable) || len(storage.objects) != 0 {
		t.Fatalf("scanner failure err=%v staged=%d", err, len(storage.objects))
	}
}

func TestPrepareCanSkipScannerForValidatedFile(t *testing.T) {
	storage := &memoryTemp{objects: map[string][]byte{}}
	intake := Intake{Temporary: storage, SkipScan: true}
	got, err := intake.Prepare(context.Background(), "org-1", "attempt-1", bytes.NewBufferString("%PDF-1.4\n%%EOF"))
	if err != nil || got.MIME != "application/pdf" || len(storage.objects) != 1 {
		t.Fatalf("got=%+v staged=%d err=%v", got, len(storage.objects), err)
	}
	_, err = intake.Prepare(context.Background(), "org-1", "attempt-2", bytes.NewBufferString("invalid bytes"))
	if !errors.Is(err, ErrUnsupportedType) || len(storage.objects) != 1 {
		t.Fatalf("invalid file err=%v staged=%d", err, len(storage.objects))
	}
}

func TestPrepareDeletesQuarantineAfterCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	storage := contextCheckingTemp{&memoryTemp{objects: map[string][]byte{}}}
	service := Intake{Temporary: storage, Scanner: scanFunc(func(context.Context, io.Reader) error {
		cancel()
		return errors.New("scanner canceled")
	})}
	_, err := service.Prepare(ctx, "org-1", "attempt-1", bytes.NewBufferString("%PDF-1.4\n%%EOF"))
	if !errors.Is(err, ErrScanUnavailable) || len(storage.objects) != 0 {
		t.Fatalf("scanner failure err=%v staged=%d", err, len(storage.objects))
	}
}
