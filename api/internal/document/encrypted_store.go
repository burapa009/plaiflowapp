package document

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"io"
	"os"
)

const encryptedChunk = 64 << 10

var encryptedMagic = [4]byte{'P', 'F', 'D', '1'}

// Blob is a private object-store boundary; keys are opaque and never client supplied.
type Blob interface {
	Put(context.Context, string, io.Reader, int64) error
	Get(context.Context, string) (io.ReadCloser, error)
	Delete(context.Context, string) error
}

type EncryptedStore struct {
	raw  Blob
	aead cipher.AEAD
}

func NewEncryptedStore(raw Blob, key []byte) (*EncryptedStore, error) {
	if raw == nil || len(key) != 32 {
		return nil, errors.New("invalid document storage configuration")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return &EncryptedStore{raw: raw, aead: aead}, nil
}

func (s *EncryptedStore) Put(ctx context.Context, key string, body io.Reader) error {
	if key == "" || body == nil {
		return errors.New("invalid object")
	}
	file, err := os.CreateTemp("", "plaiflow-encrypted-*")
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())
	defer file.Close()
	var noncePrefix [8]byte
	if _, err := rand.Read(noncePrefix[:]); err != nil {
		return err
	}
	if _, err := file.Write(encryptedMagic[:]); err != nil {
		return err
	}
	if _, err := file.Write(noncePrefix[:]); err != nil {
		return err
	}
	buffer := make([]byte, encryptedChunk)
	var counter uint32
	for {
		n, readErr := io.ReadFull(body, buffer)
		if readErr != nil && !errors.Is(readErr, io.EOF) && !errors.Is(readErr, io.ErrUnexpectedEOF) {
			return readErr
		}
		if n > 0 {
			if err := writeEncryptedChunk(file, s.aead, noncePrefix, counter, buffer[:n]); err != nil {
				return err
			}
			counter++
		}
		if readErr != nil {
			break
		}
	}
	if err := writeEncryptedChunk(file, s.aead, noncePrefix, counter, nil); err != nil {
		return err
	}
	info, err := file.Stat()
	if err != nil {
		return err
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return err
	}
	return s.raw.Put(ctx, key, file, info.Size())
}

func writeEncryptedChunk(w io.Writer, aead cipher.AEAD, prefix [8]byte, counter uint32, plain []byte) error {
	var nonce [12]byte
	copy(nonce[:8], prefix[:])
	binary.BigEndian.PutUint32(nonce[8:], counter)
	if err := binary.Write(w, binary.BigEndian, uint32(len(plain))); err != nil {
		return err
	}
	_, err := w.Write(aead.Seal(nil, nonce[:], plain, nil))
	return err
}

func (s *EncryptedStore) Open(ctx context.Context, key string) (io.ReadCloser, error) {
	raw, err := s.raw.Get(ctx, key)
	if err != nil {
		return nil, err
	}
	var header [12]byte
	if _, err := io.ReadFull(raw, header[:]); err != nil || string(header[:4]) != string(encryptedMagic[:]) {
		raw.Close()
		return nil, errors.New("encrypted document header is invalid")
	}
	var prefix [8]byte
	copy(prefix[:], header[4:])
	return &decryptReader{raw: raw, aead: s.aead, prefix: prefix}, nil
}

func (s *EncryptedStore) Delete(ctx context.Context, key string) error { return s.raw.Delete(ctx, key) }

type decryptReader struct {
	raw     io.ReadCloser
	aead    cipher.AEAD
	prefix  [8]byte
	counter uint32
	chunk   []byte
	done    bool
}

func (d *decryptReader) Read(out []byte) (int, error) {
	if len(out) == 0 {
		return 0, nil
	}
	for len(d.chunk) == 0 && !d.done {
		var size uint32
		if err := binary.Read(d.raw, binary.BigEndian, &size); err != nil {
			return 0, err
		}
		if size > encryptedChunk {
			return 0, errors.New("encrypted document chunk is invalid")
		}
		sealed := make([]byte, int(size)+d.aead.Overhead())
		if _, err := io.ReadFull(d.raw, sealed); err != nil {
			return 0, err
		}
		var nonce [12]byte
		copy(nonce[:8], d.prefix[:])
		binary.BigEndian.PutUint32(nonce[8:], d.counter)
		plain, err := d.aead.Open(nil, nonce[:], sealed, nil)
		if err != nil {
			return 0, err
		}
		d.counter++
		if size == 0 {
			var extra [1]byte
			if _, err := d.raw.Read(extra[:]); !errors.Is(err, io.EOF) {
				return 0, errors.New("encrypted document has trailing data")
			}
			d.done = true
			break
		}
		d.chunk = plain
	}
	if d.done {
		return 0, io.EOF
	}
	n := copy(out, d.chunk)
	d.chunk = d.chunk[n:]
	return n, nil
}

func (d *decryptReader) Close() error { return d.raw.Close() }
