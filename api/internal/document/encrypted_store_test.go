package document

import (
	"bytes"
	"context"
	"io"
	"testing"
)

type memoryBlob struct{ objects map[string][]byte }

func (m *memoryBlob) Put(_ context.Context, key string, body io.Reader, _ int64) error {
	data, err := io.ReadAll(body)
	if err == nil {
		m.objects[key] = data
	}
	return err
}
func (m *memoryBlob) Get(_ context.Context, key string) (io.ReadCloser, error) {
	return io.NopCloser(bytes.NewReader(m.objects[key])), nil
}
func (m *memoryBlob) Delete(_ context.Context, key string) error {
	delete(m.objects, key)
	return nil
}

func TestEncryptedStoreRoundTripAndTamper(t *testing.T) {
	ctx := context.Background()
	raw := &memoryBlob{objects: map[string][]byte{}}
	store, err := NewEncryptedStore(raw, bytes.Repeat([]byte{7}, 32))
	if err != nil {
		t.Fatal(err)
	}
	plain := bytes.Repeat([]byte("secret document"), 5000)
	if err := store.Put(ctx, "quarantine/one", bytes.NewReader(plain)); err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(raw.objects["quarantine/one"], plain[:100]) {
		t.Fatal("plaintext stored in object")
	}
	read, err := store.Open(ctx, "quarantine/one")
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := io.ReadAll(read)
	read.Close()
	if err != nil || !bytes.Equal(decoded, plain) {
		t.Fatalf("roundtrip err=%v equal=%v", err, bytes.Equal(decoded, plain))
	}
	raw.objects["quarantine/one"][30] ^= 1
	read, err = store.Open(ctx, "quarantine/one")
	if err != nil {
		t.Fatal(err)
	}
	_, err = io.ReadAll(read)
	read.Close()
	if err == nil {
		t.Fatal("tampered ciphertext was accepted")
	}
}
