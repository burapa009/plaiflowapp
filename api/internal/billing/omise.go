package billing

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

var ErrProvider = errors.New("payment provider unavailable")
var ErrInvalidWebhook = errors.New("invalid payment webhook")

type Charge struct {
	ID       string `json:"id"`
	Status   string `json:"status"`
	Currency string `json:"currency"`
	Amount   int64  `json:"amount"`
	Paid     bool   `json:"paid"`
	Livemode bool   `json:"livemode"`
	Source   struct {
		Type          string `json:"type"`
		ScannableCode struct {
			Image struct {
				DownloadURI string `json:"download_uri"`
			} `json:"image"`
		} `json:"scannable_code"`
	} `json:"source"`
	Metadata  map[string]string `json:"metadata"`
	ExpiresAt time.Time         `json:"expires_at"`
	PaidAt    *time.Time        `json:"paid_at"`
}

func (c Charge) Successful(intentID string, amount int64, live bool) bool {
	return c.Matches(intentID, amount, live) && c.Status == "successful" && c.Paid
}

func (c Charge) Matches(intentID string, amount int64, live bool) bool {
	return c.ID != "" && c.Currency == "THB" && c.Amount == amount && c.Livemode == live &&
		c.Source.Type == "promptpay" && c.Metadata["intent_id"] == intentID
}

type Omise struct {
	SecretKey string
	Endpoint  string
	Client    *http.Client
}

func (o Omise) endpoint() string {
	if o.Endpoint != "" {
		return o.Endpoint
	}
	return "https://api.omise.co"
}

func (o Omise) client() *http.Client {
	if o.Client != nil {
		return o.Client
	}
	return &http.Client{Timeout: 10 * time.Second}
}

func (o Omise) CreateCharge(ctx context.Context, intentID string, amount int64, expiresAt time.Time) (Charge, error) {
	if intentID == "" || amount <= 0 || o.SecretKey == "" {
		return Charge{}, ErrProvider
	}
	form := url.Values{
		"amount": {strconv.FormatInt(amount, 10)}, "currency": {"THB"},
		"source[type]": {"promptpay"}, "metadata[intent_id]": {intentID},
		"description": {"PlaiFlow subscription " + intentID},
		"expires_at":  {expiresAt.UTC().Format(time.RFC3339)},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, o.endpoint()+"/charges", strings.NewReader(form.Encode()))
	if err != nil {
		return Charge{}, ErrProvider
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return o.request(req)
}

func (o Omise) RetrieveCharge(ctx context.Context, id string) (Charge, error) {
	if !strings.HasPrefix(id, "chrg_") || o.SecretKey == "" {
		return Charge{}, ErrProvider
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, o.endpoint()+"/charges/"+url.PathEscape(id), nil)
	if err != nil {
		return Charge{}, ErrProvider
	}
	return o.request(req)
}

func (o Omise) request(req *http.Request) (Charge, error) {
	req.SetBasicAuth(o.SecretKey, "")
	response, err := o.client().Do(req)
	if err != nil {
		return Charge{}, ErrProvider
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return Charge{}, ErrProvider
	}
	var charge Charge
	if err := json.NewDecoder(io.LimitReader(response.Body, 1<<20)).Decode(&charge); err != nil || !strings.HasPrefix(charge.ID, "chrg_") {
		return Charge{}, ErrProvider
	}
	return charge, nil
}

// VerifyWebhook checks Omise's base64-secret HMAC over timestamp + "." + raw body.
func VerifyWebhook(body []byte, timestamp, signature, encodedSecret string, now time.Time) error {
	seconds, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil || seconds <= 0 || now.Sub(time.Unix(seconds, 0)) > 5*time.Minute || time.Unix(seconds, 0).Sub(now) > 5*time.Minute {
		return ErrInvalidWebhook
	}
	secret, err := base64.StdEncoding.DecodeString(encodedSecret)
	if err != nil || len(secret) < 16 {
		return ErrInvalidWebhook
	}
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(timestamp))
	mac.Write([]byte("."))
	mac.Write(body)
	expected := mac.Sum(nil)
	for _, value := range strings.Split(signature, ",") {
		candidate, err := hex.DecodeString(strings.TrimSpace(value))
		if err == nil && hmac.Equal(candidate, expected) {
			return nil
		}
	}
	return ErrInvalidWebhook
}
