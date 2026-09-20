package drive

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGoogleDownloadSelectedRevision(t *testing.T) {
	mediaRequests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer access-token" {
			t.Fatalf("missing server-held access token")
		}
		switch r.URL.Path {
		case "/files/file-1":
			_, _ = io.WriteString(w, `{"id":"file-1","name":"invoice.pdf","mimeType":"application/pdf","headRevisionId":"rev-2","size":"8","capabilities":{"canDownload":true}}`)
		case "/files/file-1/revisions/rev-2":
			mediaRequests++
			if r.URL.Query().Get("alt") != "media" {
				t.Fatalf("content query=%q", r.URL.RawQuery)
			}
			_, _ = io.WriteString(w, "%PDF-1.7")
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	provider, err := NewGoogleProvider(GoogleConfig{ClientID: "id", ClientSecret: "secret", RedirectURI: "https://example.test/callback", DriveFilesEndpoint: server.URL + "/files", HTTPClient: server.Client()})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := provider.DownloadFile(context.Background(), "access-token", "file-1", "rev-1"); !errors.Is(err, ErrFileChanged) || mediaRequests != 0 {
		t.Fatalf("stale revision err=%v media=%d", err, mediaRequests)
	}
	file, err := provider.DownloadFile(context.Background(), "access-token", "file-1", "rev-2")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Body.Close()
	content, err := io.ReadAll(file.Body)
	if err != nil || string(content) != "%PDF-1.7" || file.Size != 8 || mediaRequests != 1 {
		t.Fatalf("file=%+v content=%q media=%d err=%v", file, content, mediaRequests, err)
	}
}
