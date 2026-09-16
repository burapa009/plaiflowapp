package business

import (
	"context"
	"errors"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

var (
	ErrInvalid   = errors.New("business contact is invalid")
	ErrNotFound  = errors.New("business contact was not found")
	ErrForbidden = errors.New("business contact action is forbidden")
	ErrDuplicate = errors.New("business contact already exists")
	ErrExpired   = errors.New("import preview expired")
)

type Contact struct {
	ID             string     `json:"id"`
	OrganizationID string     `json:"organization_id,omitempty"`
	DisplayName    string     `json:"display_name"`
	NormalizedName string     `json:"-"`
	ContactCode    string     `json:"contact_code,omitempty"`
	Country        string     `json:"country"`
	TaxID          string     `json:"tax_id,omitempty"`
	BranchCode     string     `json:"branch_code,omitempty"`
	Customer       bool       `json:"customer"`
	Vendor         bool       `json:"vendor"`
	ArchivedAt     *time.Time `json:"archived_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at,omitempty"`
}

type VendorInput struct {
	DisplayName string
	ContactCode string
	Country     string
	TaxID       string
	BranchCode  string
	Customer    bool
}

func NormalizeVendor(input VendorInput) (Contact, error) {
	displayName := strings.TrimSpace(input.DisplayName)
	if utf8.RuneCountInString(displayName) < 1 || utf8.RuneCountInString(displayName) > 240 {
		return Contact{}, ErrInvalid
	}
	country := strings.ToUpper(strings.TrimSpace(input.Country))
	if country == "" {
		country = "TH"
	}
	if len(country) != 2 {
		return Contact{}, ErrInvalid
	}
	taxID := compactTaxID(input.TaxID)
	branch := strings.TrimSpace(input.BranchCode)
	if country == "TH" {
		if taxID != "" && !validThaiTaxID(taxID) {
			return Contact{}, ErrInvalid
		}
		if taxID != "" && branch == "" {
			branch = "00000"
		}
		if branch != "" && (len(branch) != 5 || !digits(branch)) {
			return Contact{}, ErrInvalid
		}
	}
	if utf8.RuneCountInString(taxID) > 64 || utf8.RuneCountInString(branch) > 32 || utf8.RuneCountInString(input.ContactCode) > 64 {
		return Contact{}, ErrInvalid
	}
	return Contact{
		DisplayName: displayName, NormalizedName: strings.ToLower(displayName), ContactCode: strings.TrimSpace(input.ContactCode),
		Country: country, TaxID: taxID, BranchCode: branch, Customer: input.Customer, Vendor: true,
	}, nil
}

func compactTaxID(value string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) || r == '-' {
			return -1
		}
		return r
	}, strings.TrimSpace(value))
}

func validThaiTaxID(value string) bool {
	if len(value) != 13 || !digits(value) {
		return false
	}
	sum := 0
	for index := 0; index < 12; index++ {
		sum += int(value[index]-'0') * (13 - index)
	}
	return int(value[12]-'0') == (11-sum%11)%10
}

func digits(value string) bool {
	for _, char := range value {
		if char < '0' || char > '9' {
			return false
		}
	}
	return true
}

type Page struct {
	Vendors    []Contact `json:"vendors"`
	NextCursor string    `json:"next_cursor,omitempty"`
}

type PreviewRow struct {
	Row     int     `json:"row"`
	Status  string  `json:"status"`
	Reason  string  `json:"reason,omitempty"`
	Contact Contact `json:"contact"`
}

type Preview struct {
	ID             string       `json:"id"`
	OrganizationID string       `json:"organization_id"`
	Rows           []PreviewRow `json:"rows"`
	Ready          int          `json:"ready"`
	Duplicates     int          `json:"duplicates"`
	Warnings       int          `json:"warnings"`
	Errors         int          `json:"errors"`
	ExpiresAt      time.Time    `json:"expires_at"`
}

type ImportResult struct {
	Created int `json:"created"`
	Skipped int `json:"skipped"`
}

type Store interface {
	CreateVendor(context.Context, string, string, Contact, time.Time) (Contact, error)
	ListVendors(context.Context, string, string, string, string, int) (Page, error)
	ExportVendors(context.Context, string, string, int) ([]Contact, error)
	CreateImportPreview(context.Context, string, string, string, []Contact, time.Time) (Preview, error)
	GetImportPreview(context.Context, string, string, string, time.Time) (Preview, error)
	CommitImport(context.Context, string, string, string, time.Time) (ImportResult, error)
}
