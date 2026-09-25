package billing

import (
	"context"
	"errors"
	"net/url"
	"strings"
	"time"

	"plaiflow/api/internal/plan"
)

var ErrForbidden = errors.New("billing forbidden")
var ErrConflict = errors.New("billing payment already pending")
var ErrInvalidPlan = errors.New("invalid billing plan or interval")
var ErrUnknownCharge = errors.New("unknown billing charge")

type Intent struct {
	ID             string        `json:"id"`
	OrganizationID string        `json:"-"`
	ActorUserID    string        `json:"-"`
	Plan           plan.Key      `json:"plan"`
	Interval       plan.Interval `json:"interval"`
	AmountSatang   int64         `json:"amount_satang"`
	Status         string        `json:"status"`
	ChargeID       string        `json:"-"`
	QRImageURL     string        `json:"qr_image_url,omitempty"`
	ExpiresAt      time.Time     `json:"expires_at"`
}

type Event struct{ ID, ChargeID, Type string }

type Summary struct {
	Plan                plan.Key      `json:"plan"`
	Interval            plan.Interval `json:"interval,omitempty"`
	Status              string        `json:"status"`
	PaidThrough         *time.Time    `json:"paid_through,omitempty"`
	GraceUntil          *time.Time    `json:"grace_until,omitempty"`
	CancelAtPeriodEnd   bool          `json:"cancel_at_period_end"`
	Pending             *Intent       `json:"pending,omitempty"`
	PendingExpired      bool          `json:"pending_expired,omitempty"`
	LatestPaymentStatus string        `json:"latest_payment_status,omitempty"`
}

type Store interface {
	CreateIntent(context.Context, Intent, time.Time) (Intent, error)
	AttachCharge(context.Context, Intent, Charge) (Intent, error)
	IntentForCharge(context.Context, string) (Intent, error)
	Activate(context.Context, Intent, Charge, time.Time) error
	MarkUnpaid(context.Context, Intent, string, time.Time) error
	QueueEvent(context.Context, Event) error
	ClaimEvents(context.Context, int) ([]Event, error)
	FinishEvent(context.Context, string, bool) error
	PendingCharges(context.Context, int) ([]string, error)
	Summary(context.Context, string, string, time.Time) (Summary, error)
	SetCancellation(context.Context, string, string, bool, time.Time) error
}

type Provider interface {
	CreateCharge(context.Context, string, int64, time.Time) (Charge, error)
	RetrieveCharge(context.Context, string) (Charge, error)
}

type Service struct {
	Store    Store
	Provider Provider
	Live     bool
	Now      func() time.Time
}

func (s Service) clock() time.Time {
	if s.Now != nil {
		return s.Now().UTC()
	}
	return time.Now().UTC()
}

func (s Service) Start(ctx context.Context, actorUserID, organizationID, intentID string, key plan.Key, interval plan.Interval) (Intent, error) {
	definition, ok := plan.Lookup(key)
	if !ok || (key != plan.Starter && key != plan.Business) {
		return Intent{}, ErrInvalidPlan
	}
	price, ok := definition.Prices[interval]
	if !ok || price.TotalSatang <= 0 {
		return Intent{}, ErrInvalidPlan
	}
	now := s.clock()
	intent, err := s.Store.CreateIntent(ctx, Intent{ID: intentID, OrganizationID: organizationID, ActorUserID: actorUserID,
		Plan: key, Interval: interval, AmountSatang: price.TotalSatang, Status: "creating", ExpiresAt: now.Add(30 * time.Minute)}, now)
	if err != nil {
		return Intent{}, err
	}
	if intent.Status != "creating" {
		return intent, nil
	}
	// A transport failure may have created a charge. Keep this intent unavailable for retry until expiry.
	charge, err := s.Provider.CreateCharge(ctx, intent.ID, intent.AmountSatang, intent.ExpiresAt)
	if err != nil {
		return Intent{}, ErrProvider
	}
	if charge.Amount != intent.AmountSatang || charge.Currency != "THB" || charge.Source.Type != "promptpay" ||
		charge.Metadata["intent_id"] != intent.ID || charge.Livemode != s.Live || charge.ID == "" ||
		charge.ExpiresAt.IsZero() || charge.ExpiresAt.After(intent.ExpiresAt.Add(time.Minute)) || charge.ExpiresAt.Before(now) {
		return Intent{}, ErrProvider
	}
	qr, err := url.Parse(charge.Source.ScannableCode.Image.DownloadURI)
	if err != nil || qr.Scheme != "https" || qr.Host != "api.omise.co" || !strings.HasPrefix(qr.Path, "/charges/"+charge.ID+"/documents/") {
		return Intent{}, ErrProvider
	}
	return s.Store.AttachCharge(ctx, intent, charge)
}

func (s Service) ProcessCharge(ctx context.Context, chargeID string) error {
	intent, err := s.Store.IntentForCharge(ctx, chargeID)
	if err != nil {
		if errors.Is(err, ErrUnknownCharge) {
			return nil
		}
		return err
	}
	charge, err := s.Provider.RetrieveCharge(ctx, chargeID)
	if err != nil {
		return err
	}
	if charge.ID != intent.ChargeID || !charge.Matches(intent.ID, intent.AmountSatang, s.Live) {
		return ErrProvider
	}
	if charge.Status == "failed" || charge.Status == "expired" {
		return s.Store.MarkUnpaid(ctx, intent, charge.Status, s.clock())
	}
	if charge.Status != "successful" {
		return nil
	}
	if !charge.Successful(intent.ID, intent.AmountSatang, s.Live) || charge.PaidAt == nil {
		return ErrProvider
	}
	return s.Store.Activate(ctx, intent, charge, s.clock())
}

func (s Service) ProcessDue(ctx context.Context) (int, error) {
	events, err := s.Store.ClaimEvents(ctx, 20)
	if err != nil {
		return 0, err
	}
	processed := 0
	for _, event := range events {
		ok := s.ProcessCharge(ctx, event.ChargeID) == nil
		if err := s.Store.FinishEvent(ctx, event.ID, ok); err != nil {
			return processed, err
		}
		if ok {
			processed++
		}
	}
	return processed, nil
}

func (s Service) ReconcilePending(ctx context.Context) (int, error) {
	charges, err := s.Store.PendingCharges(ctx, 100)
	if err != nil {
		return 0, err
	}
	repaired := 0
	for _, id := range charges {
		if s.ProcessCharge(ctx, id) == nil {
			repaired++
		}
	}
	return repaired, nil
}

// AddMonthsClamped preserves the original renewal day across short months.
func AddMonthsClamped(start time.Time, months, anchorDay int) time.Time {
	first := time.Date(start.Year(), start.Month()+time.Month(months), 1, start.Hour(), start.Minute(), start.Second(), start.Nanosecond(), start.Location())
	lastDay := first.AddDate(0, 1, -1).Day()
	day := anchorDay
	if day > lastDay {
		day = lastDay
	}
	return first.AddDate(0, 0, day-1)
}
