package line

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"plaiflow/api/internal/work"
)

func TestSenderUsesDirectRecipientAndGenericContent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v2/bot/message/push" || r.Header.Get("Authorization") != "Bearer token" {
			t.Fatalf("path=%s authorization=%q", r.URL.Path, r.Header.Get("Authorization"))
		}
		var body struct {
			To       string `json:"to"`
			Messages []struct {
				Text string `json:"text"`
			} `json:"messages"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body.To != "U-direct" || len(body.Messages) != 1 || strings.Contains(body.Messages[0].Text, "secret task") || !strings.Contains(body.Messages[0].Text, "Org") {
			t.Fatalf("body=%+v", body)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	sender := NewSender(server.Client(), server.URL, "token", "https://app.example")
	if err := sender.Send(t.Context(), work.LINEDelivery{To: "U-direct", OrganizationName: "Org", DeepLink: "/o/org/tasks/task"}); err != nil {
		t.Fatal(err)
	}
}

func TestSenderClassifiesPermanentAndRetryableResponses(t *testing.T) {
	for _, test := range []struct {
		status    int
		permanent bool
	}{{http.StatusBadRequest, true}, {http.StatusTooManyRequests, false}} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(test.status) }))
		sender := NewSender(server.Client(), server.URL, "token", "https://app.example")
		err := sender.Send(t.Context(), work.LINEDelivery{To: "U-direct", OrganizationName: "Org", DeepLink: "/o/org/tasks/task"})
		server.Close()
		if err == nil || errors.Is(err, work.ErrPermanentDelivery) != test.permanent {
			t.Fatalf("status=%d permanent=%v err=%v", test.status, errors.Is(err, work.ErrPermanentDelivery), err)
		}
	}
}
