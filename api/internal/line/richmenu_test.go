package line

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestRichMenuPublishesSixAuthorizedWebDestinations(t *testing.T) {
	var paths []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Fatal("missing bearer token")
		}
		paths = append(paths, r.URL.Path)
		if r.URL.Path == "/v2/bot/richmenu" {
			var body struct {
				Areas []struct {
					Action struct {
						URI string `json:"uri"`
					} `json:"action"`
				} `json:"areas"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil || len(body.Areas) != 6 || body.Areas[5].Action.URI != "https://staging.plaiflow.app/open/settings" {
				t.Fatalf("body=%+v err=%v", body, err)
			}
			_, _ = w.Write([]byte(`{"richMenuId":"rich-1"}`))
		}
	}))
	defer server.Close()

	publisher := RichMenuPublisher{Client: server.Client(), BaseURL: server.URL}
	id, err := publisher.Publish(context.Background(), "test-token", "https://staging.plaiflow.app", []byte("png"))
	if err != nil || id != "rich-1" {
		t.Fatalf("id=%q err=%v", id, err)
	}
	want := []string{"/v2/bot/richmenu", "/v2/bot/richmenu/rich-1/content", "/v2/bot/user/all/richmenu/rich-1"}
	if !reflect.DeepEqual(paths, want) {
		t.Fatalf("paths=%v", paths)
	}
}
