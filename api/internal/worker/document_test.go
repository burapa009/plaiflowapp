package worker

import (
	"bytes"
	"context"
	"io"
	"sync"
	"testing"
	"time"

	"plaiflow/api/internal/document"
	"plaiflow/api/internal/inbound"
	lineadapter "plaiflow/api/internal/line"
)

type lineMemory struct {
	mu      sync.Mutex
	objects map[string][]byte
}

func (m *lineMemory) Put(_ context.Context, key string, body io.Reader) error {
	data, err := io.ReadAll(body)
	if err == nil {
		m.mu.Lock()
		m.objects[key] = data
		m.mu.Unlock()
	}
	return err
}
func (m *lineMemory) Open(_ context.Context, key string) (io.ReadCloser, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return io.NopCloser(bytes.NewReader(m.objects[key])), nil
}
func (m *lineMemory) Delete(_ context.Context, key string) error {
	m.mu.Lock()
	delete(m.objects, key)
	m.mu.Unlock()
	return nil
}

type lineResolve struct{}

func (lineResolve) ResolveLINEDocument(context.Context, inbound.Event) (string, string, error) {
	return "org-1", "user-1", nil
}

type lineDownload struct{}

func (lineDownload) Download(context.Context, string) (io.ReadCloser, error) {
	return io.NopCloser(bytes.NewReader([]byte("%PDF-1.7"))), nil
}

type expiredLineDownload struct{}

func (expiredLineDownload) Download(context.Context, string) (io.ReadCloser, error) {
	return nil, lineadapter.ErrContentUnavailable
}

func TestLINEDocumentProcessorEndsExpiredMessage(t *testing.T) {
	process := NewLINEDocumentProcessor(&document.Service{}, lineResolve{}, expiredLineDownload{})
	status, _, err := process(context.Background(), inbound.Event{Provider: "line", Channel: "channel", ProviderEventID: "event-1", SourceType: "user", SourceUserID: "line-user", Payload: []byte(`{"type":"message","message":{"type":"file","id":"message-1","fileName":"invoice.pdf"}}`)})
	if err != nil || status != inbound.Ignored {
		t.Fatalf("status=%s err=%v", status, err)
	}
}

func TestLINEDocumentProcessorUsesSharedDocumentPipeline(t *testing.T) {
	storage := &lineMemory{objects: map[string][]byte{}}
	service := &document.Service{Intake: document.Intake{Temporary: storage, Scanner: scannerFunc(func(context.Context, io.Reader) error { return nil })}, Committer: commitFunc(func(_ context.Context, input document.CommitInput) (document.CommitResult, error) {
		if input.Channel != "LINE" || input.OriginKey != "line:line:channel:message-1" || input.OrganizationID != "org-1" {
			t.Fatalf("input=%+v", input)
		}
		return document.CommitResult{Accepted: true}, nil
	})}
	process := NewLINEDocumentProcessor(service, lineResolve{}, lineDownload{})
	status, _, err := process(context.Background(), inbound.Event{Provider: "line", Channel: "channel", ProviderEventID: "event-1", Type: "message", SourceType: "user", SourceUserID: "line-user", OccurredAt: time.Now(), Payload: []byte(`{"type":"message","message":{"type":"file","id":"message-1","fileName":"invoice.pdf"}}`)})
	if err != nil || status != inbound.Processed || len(storage.objects) != 1 {
		t.Fatalf("status=%s err=%v objects=%d", status, err, len(storage.objects))
	}
}

func TestLINEDocumentProcessorMarksGroupSource(t *testing.T) {
	storage := &lineMemory{objects: map[string][]byte{}}
	service := &document.Service{Intake: document.Intake{Temporary: storage, Scanner: scannerFunc(func(context.Context, io.Reader) error { return nil })}, Committer: commitFunc(func(_ context.Context, input document.CommitInput) (document.CommitResult, error) {
		if !input.LINEGroup || input.Channel != "LINE" {
			t.Fatalf("group source lost: %+v", input)
		}
		return document.CommitResult{Accepted: true}, nil
	})}
	process := NewLINEDocumentProcessor(service, lineResolve{}, lineDownload{})
	status, _, err := process(context.Background(), inbound.Event{Provider: "line", Channel: "channel", SourceType: "group", SourceGroupID: "group-1", SourceUserID: "line-user", Payload: []byte(`{"type":"message","message":{"type":"file","id":"message-2","fileName":"invoice.pdf"}}`)})
	if err != nil || status != inbound.Processed {
		t.Fatalf("status=%s err=%v", status, err)
	}
}

type scannerFunc func(context.Context, io.Reader) error

func (f scannerFunc) Scan(ctx context.Context, body io.Reader) error { return f(ctx, body) }

type commitFunc func(context.Context, document.CommitInput) (document.CommitResult, error)

func (f commitFunc) CommitPrepared(ctx context.Context, input document.CommitInput) (document.CommitResult, error) {
	return f(ctx, input)
}
