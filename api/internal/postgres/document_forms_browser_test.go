package postgres

// Opt-in browser fixture: real API/auth/CSRF/PostgreSQL with synthetic documents
// and an in-memory artifact store. Binds loopback only and expires automatically.
import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"plaiflow/api/internal/auth"
	"plaiflow/api/internal/document"
	"plaiflow/api/internal/httpapi"
	"plaiflow/api/internal/job"
	"plaiflow/api/internal/ocr"
	"strings"
	"sync"
	"testing"
	"time"
)

type formBrowserBlobs struct {
	sync.Mutex
	objects map[string][]byte
	image   []byte
	result  []byte
}

func (b *formBrowserBlobs) Put(_ context.Context, key string, r io.Reader) error {
	raw, e := io.ReadAll(r)
	b.Lock()
	defer b.Unlock()
	b.objects[key] = raw
	return e
}
func (b *formBrowserBlobs) Open(_ context.Context, key string) (io.ReadCloser, error) {
	b.Lock()
	defer b.Unlock()
	raw, ok := b.objects[key]
	if !ok {
		if strings.HasPrefix(key, "test/result/") {
			raw = b.result
		} else if strings.HasPrefix(key, "test/") {
			raw = b.image
		} else {
			return nil, errors.New("missing fixture artifact")
		}
	}
	return io.NopCloser(bytes.NewReader(raw)), nil
}
func (b *formBrowserBlobs) Delete(_ context.Context, key string) error {
	b.Lock()
	defer b.Unlock()
	delete(b.objects, key)
	return nil
}
func serveDocumentFormBrowser(t *testing.T, s *Store, user, org, doc string) {
	t.Helper()
	if os.Getenv("FORM_BROWSER_TEST") != "1" {
		return
	}
	now := time.Now().UTC()
	token, csrf := sha256.Sum256([]byte("local-form-test-session")), sha256.Sum256([]byte("local-form-test-csrf"))
	if e := s.CreateSession(context.Background(), auth.SessionRecord{ID: postgresUUID(), UserID: user, TokenHash: token[:], CSRFHash: csrf[:], CreatedAt: now, AuthenticatedAt: now, LastSeenAt: now, IdleExpiresAt: now.Add(time.Hour), AbsoluteExpiresAt: now.Add(time.Hour), AuthenticatedProvider: "google"}); e != nil {
		t.Fatal(e)
	}
	service, e := auth.NewService(auth.ServiceConfig{WebOrigin: "http://localhost:3110"}, s)
	if e != nil {
		t.Fatal(e)
	}
	original, e := os.ReadFile("../../../web/public/review-preview.png")
	if e != nil {
		t.Fatal(e)
	}
	result, _ := json.Marshal(ocr.Result{SchemaVersion: 1, Pages: []ocr.Page{{Number: 1, Lines: []ocr.Line{{Text: "Synthetic test document INV-LOCAL", Confidence: .9}}}}})
	blobs := &formBrowserBlobs{objects: map[string][]byte{}, image: original, result: result}
	meta, e := s.CurrentReview(context.Background(), user, org, doc)
	if e != nil {
		t.Fatal(e)
	}
	form, e := s.GetDocumentForm(context.Background(), user, org, doc)
	if e != nil {
		t.Fatal(e)
	}
	meta.Canonical = &form.Data
	meta.DocumentType = form.Data.Type
	meta.Values = form.Data.LegacyValues()
	raw, _ := json.Marshal(meta)
	blobs.objects[meta.ObjectKey] = raw
	workerAuth, e := job.NewWorkerAuth(bytes.Repeat([]byte("x"), 32), "test", []string{"ocr"})
	if e != nil {
		t.Fatal(e)
	}
	artifactTokens, e := job.NewArtifactToken(bytes.Repeat([]byte("fixture-artifact-key"), 2))
	if e != nil {
		t.Fatal(e)
	}
	h := httpapi.New(httpapi.Config{Auth: service, Tenants: s, OCR: s, OCRAuth: workerAuth, OCRTokens: artifactTokens, OCRStorage: blobs, Documents: &document.Service{Committer: s, Reader: s, Intake: document.Intake{Temporary: blobs}}, Extraction: s, ExtractionEnabled: true, ReviewEnabled: true, OCRPilotOrganizations: map[string]bool{"*": true}, Now: time.Now}, s)
	mux := http.NewServeMux()
	mux.Handle("/", h)
	mux.HandleFunc("GET /fixture-login", func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, &http.Cookie{Name: auth.SessionCookieName, Value: "local-form-test-session", Path: "/", Secure: true, HttpOnly: true, SameSite: http.SameSiteLaxMode})
		http.SetCookie(w, &http.Cookie{Name: auth.CSRFCookieName, Value: "local-form-test-csrf", Path: "/", Secure: true, SameSite: http.SameSiteLaxMode})
		http.Redirect(w, r, "http://localhost:3110/o/"+org+"/documents/"+doc, http.StatusSeeOther)
	})
	listener, e := net.Listen("tcp", "127.0.0.1:55440")
	if e != nil {
		t.Fatal(e)
	}
	server := &http.Server{Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	defer server.Close()
	go server.Serve(listener)
	fmt.Printf("FORM_BROWSER_READY http://localhost:55440/fixture-login document=%s organization=%s\n", doc, org)
	stop := "../../../.form-browser-stop"
	os.Remove(stop)
	deadline := time.Now().Add(25 * time.Minute)
	if testDeadline, ok := t.Deadline(); ok && testDeadline.Before(deadline.Add(30*time.Second)) {
		deadline = testDeadline.Add(-30 * time.Second)
	}
	for time.Now().Before(deadline) {
		if _, e := os.Stat(stop); e == nil {
			os.Remove(stop)
			return
		}
		time.Sleep(time.Second)
	}
}
