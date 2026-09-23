package document

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"time"
)

const MaxFileBytes int64 = 20 << 20

var (
	ErrUnsupportedType = errors.New("unsupported document type")
	ErrTooLarge        = errors.New("document exceeds size limit")
	ErrScanUnavailable = errors.New("document scanner unavailable")
	ErrMalware         = errors.New("malware detected")
)

// Temporary keeps quarantine objects private and encrypted until acceptance or deletion.
type Temporary interface {
	Put(context.Context, string, io.Reader) error
	Open(context.Context, string) (io.ReadCloser, error)
	Delete(context.Context, string) error
}

type Scanner interface {
	Scan(context.Context, io.Reader) error
}

type Intake struct {
	Temporary Temporary
	Scanner   Scanner
	SkipScan  bool
}

type Prepared struct {
	TemporaryKey string
	SHA256       string
	MIME         string
	Size         int64
}

// Prepare bounds and validates original bytes for every intake channel.
func (i Intake) Prepare(ctx context.Context, organizationID, attemptID string, source io.Reader) (result Prepared, err error) {
	if i.Temporary == nil || (!i.SkipScan && i.Scanner == nil) || organizationID == "" || attemptID == "" || source == nil {
		return Prepared{}, errors.New("document intake is not configured")
	}
	keyDigest := sha256.Sum256([]byte(organizationID + ":" + attemptID))
	key := "quarantine/" + hex.EncodeToString(keyDigest[:])
	keep := false
	defer func() {
		if !keep {
			cleanup, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
			defer cancel()
			_ = i.Temporary.Delete(cleanup, key)
		}
	}()
	if err = i.Temporary.Put(ctx, key, io.LimitReader(source, MaxFileBytes+1)); err != nil {
		return Prepared{}, err
	}
	body, err := i.Temporary.Open(ctx, key)
	if err != nil {
		return Prepared{}, err
	}
	var prefix [512]byte
	prefixLen, readErr := io.ReadFull(body, prefix[:])
	if readErr != nil && !errors.Is(readErr, io.EOF) && !errors.Is(readErr, io.ErrUnexpectedEOF) {
		body.Close()
		return Prepared{}, readErr
	}
	mime := sniff(prefix[:prefixLen])
	hash := sha256.New()
	_, _ = hash.Write(prefix[:prefixLen])
	rest, err := io.Copy(hash, body)
	closeErr := body.Close()
	if err != nil {
		return Prepared{}, err
	}
	if closeErr != nil {
		return Prepared{}, closeErr
	}
	size := int64(prefixLen) + rest
	if size > MaxFileBytes {
		return Prepared{}, ErrTooLarge
	}
	if mime == "" || size == 0 {
		return Prepared{}, ErrUnsupportedType
	}
	if !i.SkipScan {
		body, err = i.Temporary.Open(ctx, key)
		if err != nil {
			return Prepared{}, err
		}
		scanErr := i.Scanner.Scan(ctx, body)
		closeErr = body.Close()
		if errors.Is(scanErr, ErrMalware) {
			return Prepared{}, ErrMalware
		}
		if scanErr != nil || closeErr != nil {
			return Prepared{}, ErrScanUnavailable
		}
	}
	keep = true
	return Prepared{TemporaryKey: key, SHA256: hex.EncodeToString(hash.Sum(nil)), MIME: mime, Size: size}, nil
}

func sniff(prefix []byte) string {
	if len(prefix) >= 5 && string(prefix[:5]) == "%PDF-" {
		return "application/pdf"
	}
	if len(prefix) >= 8 && string(prefix[:8]) == "\x89PNG\r\n\x1a\n" {
		return "image/png"
	}
	if len(prefix) >= 3 && prefix[0] == 0xff && prefix[1] == 0xd8 && prefix[2] == 0xff {
		return "image/jpeg"
	}
	return ""
}
