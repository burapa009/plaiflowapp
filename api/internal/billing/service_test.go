package billing

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"plaiflow/api/internal/plan"
)

type testStore struct {
	intent Intent
	grants int
}

func (s *testStore) CreateIntent(_ context.Context, i Intent, _ time.Time) (Intent, error) {
	s.intent = i
	return i, nil
}
func (s *testStore) AttachCharge(_ context.Context, i Intent, c Charge) (Intent, error) {
	i.ChargeID = c.ID
	i.QRImageURL = c.Source.ScannableCode.Image.DownloadURI
	i.Status = "awaiting_payment"
	s.intent = i
	return i, nil
}
func (s *testStore) IntentForCharge(_ context.Context, id string) (Intent, error) {
	if id != s.intent.ChargeID {
		return Intent{}, errors.New("unknown charge")
	}
	return s.intent, nil
}
func (s *testStore) Activate(_ context.Context, _ Intent, _ Charge, _ time.Time) error {
	if s.intent.Status != "paid" {
		s.grants++
		s.intent.Status = "paid"
	}
	return nil
}
func (s *testStore) MarkUnpaid(_ context.Context, _ Intent, status string, _ time.Time) error {
	if s.intent.Status == "awaiting_payment" {
		s.intent.Status = status
	}
	return nil
}
func (s *testStore) QueueEvent(context.Context, Event) error               { return nil }
func (s *testStore) ClaimEvents(context.Context, int) ([]Event, error)     { return nil, nil }
func (s *testStore) FinishEvent(context.Context, string, bool) error       { return nil }
func (s *testStore) PendingCharges(context.Context, int) ([]string, error) { return nil, nil }
func (s *testStore) Summary(context.Context, string, string, time.Time) (Summary, error) {
	return Summary{}, nil
}
func (s *testStore) SetCancellation(context.Context, string, string, bool, time.Time) error {
	return nil
}

type testProvider struct {
	charge     Charge
	lateExpiry bool
}

func (p *testProvider) CreateCharge(_ context.Context, id string, amount int64, expiresAt time.Time) (Charge, error) {
	p.charge = Charge{ID: "chrg_test_1", Status: "pending", Currency: "THB", Amount: amount, ExpiresAt: expiresAt, Metadata: map[string]string{"intent_id": id}}
	p.charge.Source.Type = "promptpay"
	p.charge.Source.ScannableCode.Image.DownloadURI = "https://api.omise.co/charges/chrg_test_1/documents/qr/downloads/safe"
	if p.lateExpiry {
		p.charge.ExpiresAt = expiresAt.Add(24 * time.Hour)
	}
	return p.charge, nil
}
func (p *testProvider) RetrieveCharge(context.Context, string) (Charge, error) { return p.charge, nil }

func TestEachIntervalRequiresVerifiedPromptPayAndGrantsOnce(t *testing.T) {
	for _, tc := range []struct {
		interval plan.Interval
		amount   int64
	}{{plan.Monthly, 15000}, {plan.SixMonths, 85500}, {plan.Yearly, 153000}} {
		t.Run(string(tc.interval), func(t *testing.T) {
			now := time.Date(2026, 9, 25, 0, 0, 0, 0, time.UTC)
			store := &testStore{}
			provider := &testProvider{}
			service := Service{Store: store, Provider: provider, Now: func() time.Time { return now }}
			intent, err := service.Start(context.Background(), "owner", "org", fmt.Sprint(tc.interval), plan.Starter, tc.interval)
			if err != nil || intent.AmountSatang != tc.amount || intent.Status != "awaiting_payment" || store.grants != 0 {
				t.Fatalf("intent=%+v grants=%d err=%v", intent, store.grants, err)
			}
			if err := service.ProcessCharge(context.Background(), provider.charge.ID); err != nil || store.grants != 0 {
				t.Fatalf("pending unlocked rights: %v %d", err, store.grants)
			}
			provider.charge.Status = "successful"
			provider.charge.Paid = true
			provider.charge.PaidAt = &now
			if err := service.ProcessCharge(context.Background(), provider.charge.ID); err != nil {
				t.Fatal(err)
			}
			if err := service.ProcessCharge(context.Background(), provider.charge.ID); err != nil || store.grants != 1 {
				t.Fatalf("duplicate grant: %v %d", err, store.grants)
			}
		})
	}
}

func TestTamperedAmountAndPlanDoNotGrant(t *testing.T) {
	store := &testStore{}
	provider := &testProvider{}
	now := time.Now().UTC()
	service := Service{Store: store, Provider: provider, Now: func() time.Time { return now }}
	if _, err := service.Start(context.Background(), "owner", "org", "bad", plan.Free, plan.Monthly); !errors.Is(err, ErrInvalidPlan) {
		t.Fatalf("free checkout: %v", err)
	}
	if _, err := service.Start(context.Background(), "owner", "org", "good", plan.Business, plan.Monthly); err != nil {
		t.Fatal(err)
	}
	provider.charge.Status = "successful"
	provider.charge.Paid = true
	provider.charge.PaidAt = &now
	provider.charge.Amount++
	if err := service.ProcessCharge(context.Background(), provider.charge.ID); !errors.Is(err, ErrProvider) || store.grants != 0 {
		t.Fatalf("tampered amount granted: %v %d", err, store.grants)
	}
}

func TestFailedPromptPayClosesIntentWithoutGrant(t *testing.T) {
	store, provider := &testStore{}, &testProvider{}
	service := Service{Store: store, Provider: provider}
	if _, err := service.Start(context.Background(), "owner", "org", "failed", plan.Starter, plan.Monthly); err != nil {
		t.Fatal(err)
	}
	provider.charge.Status = "failed"
	if err := service.ProcessCharge(context.Background(), provider.charge.ID); err != nil {
		t.Fatal(err)
	}
	if err := service.ProcessCharge(context.Background(), provider.charge.ID); err != nil || store.intent.Status != "failed" || store.grants != 0 {
		t.Fatalf("failed payment changed rights: %s %d %v", store.intent.Status, store.grants, err)
	}
}

func TestQRWithLongerProviderExpiryIsNotDisplayed(t *testing.T) {
	provider := &testProvider{lateExpiry: true}
	service := Service{Store: &testStore{}, Provider: provider}
	if _, err := service.Start(context.Background(), "owner", "org", "late", plan.Starter, plan.Monthly); !errors.Is(err, ErrProvider) {
		t.Fatalf("unsafe QR accepted: %v", err)
	}
}

func TestAddMonthsClampsMonthEnd(t *testing.T) {
	start := time.Date(2027, time.January, 31, 9, 30, 0, 0, time.UTC)
	feb := AddMonthsClamped(start, 1, 31)
	if !feb.Equal(time.Date(2027, time.February, 28, 9, 30, 0, 0, time.UTC)) {
		t.Fatalf("monthly end=%s", feb)
	}
	if got := AddMonthsClamped(feb, 1, 31); !got.Equal(time.Date(2027, time.March, 31, 9, 30, 0, 0, time.UTC)) {
		t.Fatalf("following end=%s", got)
	}
	if got := AddMonthsClamped(start, 6, 31); !got.Equal(time.Date(2027, time.July, 31, 9, 30, 0, 0, time.UTC)) {
		t.Fatalf("six-month end=%s", got)
	}
}
