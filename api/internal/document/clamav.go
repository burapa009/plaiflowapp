package document

import (
	"bufio"
	"context"
	"encoding/binary"
	"errors"
	"io"
	"net"
	"strings"
	"time"
)

// ClamAV uses INSTREAM over a private-only clamd TCP endpoint.
type ClamAV struct{ Address string }

func (c ClamAV) Scan(ctx context.Context, body io.Reader) error {
	if c.Address == "" {
		return ErrScanUnavailable
	}
	conn, err := (&net.Dialer{Timeout: 3 * time.Second}).DialContext(ctx, "tcp", c.Address)
	if err != nil {
		return ErrScanUnavailable
	}
	defer conn.Close()
	deadline := time.Now().Add(30 * time.Second)
	if when, ok := ctx.Deadline(); ok && when.Before(deadline) {
		deadline = when
	}
	if err := conn.SetDeadline(deadline); err != nil {
		return ErrScanUnavailable
	}
	if _, err := conn.Write([]byte("zINSTREAM\x00")); err != nil {
		return ErrScanUnavailable
	}
	buffer := make([]byte, 32<<10)
	var total int64
	for {
		n, readErr := body.Read(buffer)
		if n > 0 {
			total += int64(n)
			if total > MaxFileBytes {
				return ErrTooLarge
			}
			if err := binary.Write(conn, binary.BigEndian, uint32(n)); err != nil {
				return ErrScanUnavailable
			}
			if _, err := conn.Write(buffer[:n]); err != nil {
				return ErrScanUnavailable
			}
		}
		if errors.Is(readErr, io.EOF) {
			break
		}
		if readErr != nil {
			return ErrScanUnavailable
		}
	}
	if err := binary.Write(conn, binary.BigEndian, uint32(0)); err != nil {
		return ErrScanUnavailable
	}
	reply, err := bufio.NewReader(io.LimitReader(conn, 1024)).ReadString(0)
	if err != nil {
		return ErrScanUnavailable
	}
	if strings.HasSuffix(reply, " FOUND\x00") {
		return ErrMalware
	}
	if reply == "stream: OK\x00" {
		return nil
	}
	return ErrScanUnavailable
}
