package document

import (
	"bytes"
	"context"
	"encoding/binary"
	"io"
	"net"
	"testing"
)

func TestClamAVScanReportsCleanAndInfected(t *testing.T) {
	for _, test := range []struct {
		reply    string
		infected bool
	}{{"stream: OK\x00", false}, {"stream: Eicar-Test-Signature FOUND\x00", true}} {
		listener, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatal(err)
		}
		go func() {
			defer listener.Close()
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			defer conn.Close()
			command := make([]byte, len("zINSTREAM\x00"))
			if _, err := io.ReadFull(conn, command); err != nil || string(command) != "zINSTREAM\x00" {
				return
			}
			for {
				var size uint32
				if err := binary.Read(conn, binary.BigEndian, &size); err != nil {
					return
				}
				if size == 0 {
					break
				}
				if _, err := io.CopyN(io.Discard, conn, int64(size)); err != nil {
					return
				}
			}
			_, _ = conn.Write([]byte(test.reply))
		}()
		err = ClamAV{Address: listener.Addr().String()}.Scan(context.Background(), bytes.NewBufferString("%PDF-1.4\n%%EOF"))
		if (err == ErrMalware) != test.infected || (!test.infected && err != nil) {
			t.Fatalf("reply=%q err=%v", test.reply, err)
		}
	}
}
