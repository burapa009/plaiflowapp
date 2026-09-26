package worker

import (
	"context"
	"errors"
	"testing"

	"plaiflow/api/internal/inbound"
)

type testGroupLinker struct {
	linked bool
	err    error
	calls  int
	leaves int
}

func (l *testGroupLinker) ConsumeLINEGroupCode(context.Context, inbound.Event) (bool, error) {
	l.calls++
	return l.linked, l.err
}
func (l *testGroupLinker) DisconnectLINEGroupBySource(context.Context, inbound.Event) (bool, error) {
	l.leaves++
	return true, l.err
}

func TestLINEGroupProcessorHandlesCodesBeforeDocuments(t *testing.T) {
	linker := &testGroupLinker{linked: true}
	documents := 0
	process := NewLINEGroupProcessor(func(context.Context, inbound.Event) (inbound.Status, string, error) {
		documents++
		return inbound.Processed, "document", nil
	}, linker)
	code := inbound.Event{LinkCodeHash: make([]byte, 32)}
	for _, check := range []struct {
		linked bool
		err    error
		want   inbound.Status
	}{{true, nil, inbound.Processed}, {false, nil, inbound.Ignored}, {false, errors.New("db down"), inbound.Retryable}} {
		linker.linked, linker.err = check.linked, check.err
		status, _, err := process(context.Background(), code)
		if status != check.want || (err != nil) != (check.err != nil) {
			t.Fatalf("status=%s err=%v", status, err)
		}
	}
	if linker.calls != 3 || documents != 0 {
		t.Fatalf("linker=%d documents=%d", linker.calls, documents)
	}
	if _, _, err := process(context.Background(), inbound.Event{}); err != nil || documents != 1 {
		t.Fatalf("normal event was not passed to document processor: %v", err)
	}
	linker.err = nil
	if status, _, err := process(context.Background(), inbound.Event{Provider: "line", SourceType: "group", Type: "leave"}); err != nil || status != inbound.Processed || linker.leaves != 1 || documents != 1 {
		t.Fatalf("leave status=%s err=%v leaves=%d documents=%d", status, err, linker.leaves, documents)
	}
}
