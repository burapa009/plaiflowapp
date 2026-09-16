package plan

import (
	"context"
	"errors"

	"plaiflow/api/internal/work"
)

type Key string
type Interval string

const (
	Free     Key = "Free"
	Starter  Key = "Starter"
	Business Key = "Business"

	Monthly   Interval = "monthly"
	SixMonths Interval = "six_months"
	Yearly    Interval = "yearly"

	ManageBusinessContacts work.Capability = "business_contacts.manage"
	ImportBusinessContacts work.Capability = "business_contacts.import"
	ExportCSV              work.Capability = "business_contacts.export.csv"
	ExportXLSX             work.Capability = "business_contacts.export.xlsx"
	ExportDrive            work.Capability = "business_contacts.export.drive"
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
}

func (d Definition) Allows(capability work.Capability) bool { return d.Entitlements[capability] }

var catalog = []Definition{
	{Key: Free, Name: "Free", Prices: prices(0, 0, 0), Entitlements: entitlements(false, false, false)},
	{Key: Starter, Name: "Starter", Prices: prices(19900, 109848, 199000), Entitlements: entitlements(true, true, false)},
	{Key: Business, Name: "Business", Recommended: true, Prices: prices(49900, 275448, 499000), Entitlements: entitlements(true, true, true)},
}

func prices(monthly, sixMonths, yearly int64) map[Interval]Price {
	sixMonthSaving, yearlySaving := 0, 0
	if monthly > 0 {
		sixMonthSaving, yearlySaving = 8, 17
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
