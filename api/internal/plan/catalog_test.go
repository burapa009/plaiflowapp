package plan

import (
	"context"
	"testing"
)

type fixedPlanStore string

func (s fixedPlanStore) EffectivePlan(context.Context, string) (Key, error) { return Key(s), nil }

func TestCatalogOwnsPricesAndEntitlements(t *testing.T) {
	catalog := Catalog()
	if len(catalog) != 3 {
		t.Fatalf("plans=%d", len(catalog))
	}
	starter, ok := Lookup(Starter)
	if !ok || starter.Prices[SixMonths].TotalSatang != 109848 || starter.Prices[Yearly].TotalSatang != 199000 {
		t.Fatalf("starter=%+v", starter)
	}
	if !starter.Allows(ImportBusinessContacts) || !starter.Allows(ExportXLSX) || starter.Allows(ExportDrive) {
		t.Fatalf("starter entitlements=%+v", starter.Entitlements)
	}
	business, _ := Lookup(Business)
	if !business.Allows(ExportDrive) {
		t.Fatal("business must include Drive export")
	}
}

func TestGateUsesStoredPlanNotSubmittedIdentifiers(t *testing.T) {
	gate := Gate{Store: fixedPlanStore(Free)}
	decision, err := gate.Check(context.Background(), "organization", ExportXLSX, 1)
	if err != nil || decision.Allowed || decision.Plan != string(Free) {
		t.Fatalf("decision=%+v err=%v", decision, err)
	}
}
