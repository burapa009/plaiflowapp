package document

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"
	"time"
)

type commitFunc func(context.Context, CommitInput) (CommitResult, error)

func (f commitFunc) CommitPrepared(ctx context.Context, input CommitInput) (CommitResult, error) {
	return f(ctx, input)
}

func TestServiceAcceptKeepsOnlyAcceptedOriginal(t *testing.T) {
	storage := &memoryTemp{objects: map[string][]byte{}}
	service := Service{Intake: Intake{Temporary: storage, Scanner: scanFunc(func(context.Context, io.Reader) error { return nil })},
		Committer: commitFunc(func(_ context.Context, input CommitInput) (CommitResult, error) {
			return CommitResult{Document: Document{ID: "doc-1", OrganizationID: input.OrganizationID, StorageKey: input.TemporaryKey}, Accepted: true}, nil
		})}
	result, err := service.Accept(context.Background(), AcceptInput{OrganizationID: "org-1", ActorUserID: "user-1", AttemptID: "attempt-1", OriginKey: "origin-1", Filename: "invoice.pdf", Channel: "Web", Now: time.Now()}, bytes.NewBufferString("%PDF-1.4\n%%EOF"))
	if err != nil || result.Document.ID != "doc-1" || len(storage.objects) != 1 {
		t.Fatalf("result=%+v err=%v objects=%d", result, err, len(storage.objects))
	}
	service.Committer = commitFunc(func(context.Context, CommitInput) (CommitResult, error) { return CommitResult{}, ErrQuota })
	_, err = service.Accept(context.Background(), AcceptInput{OrganizationID: "org-1", ActorUserID: "user-1", AttemptID: "attempt-2", OriginKey: "origin-2", Filename: "other.pdf", Channel: "Web", Now: time.Now()}, bytes.NewBufferString("%PDF-1.4\n%%EOF"))
	if !errors.Is(err, ErrQuota) || len(storage.objects) != 1 {
		t.Fatalf("quota err=%v objects=%d", err, len(storage.objects))
	}
}
