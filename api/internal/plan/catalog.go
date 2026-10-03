package plan

import (
	"context"
	"errors"

	"plaiflow/api/internal/work"
)

type Key string
type Interval string

const (
	Free           Key = "Free"
	Starter        Key = "Starter"
	Business       Key = "Business"
	Growth         Key = "Growth"
	AccountingFirm Key = "AccountingFirm"

	Monthly   Interval = "monthly"
	SixMonths Interval = "six_months"
	Yearly    Interval = "yearly"

	ManageBusinessContacts work.Capability = "business_contacts.manage"
	ImportBusinessContacts work.Capability = "business_contacts.import"
	ExportCSV              work.Capability = "business_contacts.export.csv"
	ExportXLSX             work.Capability = "business_contacts.export.xlsx"
	ExportDrive            work.Capability = "business_contacts.export.drive"
	ManageFirm             work.Capability = "firm.manage"
	FirmPortfolio          work.Capability = "firm.portfolio"
	FirmReview             work.Capability = "firm.review"
	FirmApprovedExport     work.Capability = "firm.export.approved"
)

type Price struct {
	Interval             Interval `json:"interval"`
	Months               int      `json:"months"`
	TotalSatang          int64    `json:"total_satang"`
	EffectiveMonthSatang int64    `json:"effective_month_satang"`
	SavingPercent        int      `json:"saving_percent"`
}

type Definition struct {
	Key          Key                      `json:"key"`
	Name         string                   `json:"name"`
	Recommended  bool                     `json:"recommended,omitempty"`
	Prices       map[Interval]Price       `json:"prices"`
	Entitlements map[work.Capability]bool `json:"entitlements"`
	Limits       Limits                   `json:"limits"`
}

type Limits struct {
	Members             int `json:"members"`
	LINEGroups          int `json:"line_groups"`
	DocumentsPerMonth   int `json:"documents_per_month"`
	ClientRelationships int `json:"client_relationships,omitempty"`
}

func (d Definition) Allows(capability work.Capability) bool { return d.Entitlements[capability] }

var catalog = []Definition{
	{Key: Free, Name: "Free", Prices: prices(0, 0, 0), Entitlements: entitlements(false, false, false), Limits: Limits{Members: 1, LINEGroups: 1, DocumentsPerMonth: 30}},
	{Key: Starter, Name: "Starter", Prices: prices(15000, 85500, 153000), Entitlements: entitlements(true, true, false), Limits: Limits{Members: 3, LINEGroups: 2, DocumentsPerMonth: 300}},
	{Key: Business, Name: "Business", Recommended: true, Prices: prices(25000, 142500, 255000), Entitlements: entitlements(true, true, true), Limits: Limits{Members: 10, LINEGroups: 5, DocumentsPerMonth: 1000}},
	{Key: Growth, Name: "Growth", Prices: prices(50000, 285000, 510000), Entitlements: entitlements(true, true, true), Limits: Limits{Members: 20, LINEGroups: 10, DocumentsPerMonth: 2000}},
	{Key: AccountingFirm, Name: "Accounting Firm", Prices: prices(100000, 570000, 1020000), Entitlements: map[work.Capability]bool{
		ManageFirm: true, FirmPortfolio: true, FirmReview: true, FirmApprovedExport: true,
		work.CreateTasks: true, work.UseAssistant: true,
	}, Limits: Limits{Members: 5, LINEGroups: 5, DocumentsPerMonth: 1000, ClientRelationships: 20}},
}

func prices(monthly, sixMonths, yearly int64) map[Interval]Price {
	sixMonthSaving, yearlySaving := 0, 0
	if monthly > 0 {
		sixMonthSaving, yearlySaving = 5, 15
	}
	return map[Interval]Price{
		Monthly:   {Interval: Monthly, Months: 1, TotalSatang: monthly, EffectiveMonthSatang: monthly},
		SixMonths: {Interval: SixMonths, Months: 6, TotalSatang: sixMonths, EffectiveMonthSatang: sixMonths / 6, SavingPercent: sixMonthSaving},
		Yearly:    {Interval: Yearly, Months: 12, TotalSatang: yearly, EffectiveMonthSatang: yearly / 12, SavingPercent: yearlySaving},
	}
}

func entitlements(importContacts, exportXLSX, exportDrive bool) map[work.Capability]bool {
	return map[work.Capability]bool{
		work.CreateTasks:       true,
		work.DeliverLINE:       true,
		work.UseAssistant:      true,
		work.ExportTasks:       true,
		ManageBusinessContacts: true,
		ImportBusinessContacts: importContacts,
		ExportCSV:              true,
		ExportXLSX:             exportXLSX,
		ExportDrive:            exportDrive,
		work.ExportTasksAsync:  exportDrive,
	}
}

func Catalog() []Definition { return append([]Definition(nil), catalog...) }

func Lookup(key Key) (Definition, bool) {
	for _, definition := range catalog {
		if definition.Key == key {
			return definition, true
		}
	}
	return Definition{}, false
}

type Store interface {
	EffectivePlan(context.Context, string) (Key, error)
}

type Gate struct{ Store Store }

func (g Gate) Check(ctx context.Context, organizationID string, capability work.Capability, _ int64) (work.EntitlementDecision, error) {
	if g.Store == nil {
		return work.EntitlementDecision{}, errors.New("plan store is required")
	}
	key, err := g.Store.EffectivePlan(ctx, organizationID)
	if err != nil {
		return work.EntitlementDecision{}, err
	}
	definition, ok := Lookup(key)
	if !ok {
		return work.EntitlementDecision{Reason: "unknown_plan", Plan: string(key)}, nil
	}
	if !definition.Allows(capability) {
		return work.EntitlementDecision{Reason: "feature_unavailable", Plan: string(key)}, nil
	}
	return work.EntitlementDecision{Allowed: true, Reason: "included", Plan: string(key)}, nil
}

func (Gate) Record(context.Context, string, work.Capability, int64, string) error { return nil }
