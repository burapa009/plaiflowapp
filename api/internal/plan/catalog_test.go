package plan

import (
	"context"
	"testing"
)

type fixedPlanStore string

func (s fixedPlanStore) EffectivePlan(context.Context, string) (Key, error) { return Key(s), nil }

func TestCatalogOwnsPricesAndEntitlements(t *testing.T) {
	catalog := Catalog()
	if len(catalog) != 5 {
		t.Fatalf("plans=%d", len(catalog))
	}
	starter, ok := Lookup(Starter)
	if !ok || starter.Prices[Monthly].TotalSatang != 15000 || starter.Prices[SixMonths].TotalSatang != 85500 || starter.Prices[Yearly].TotalSatang != 153000 {
		t.Fatalf("starter=%+v", starter)
	}
	if !starter.Allows(ImportBusinessContacts) || !starter.Allows(ExportXLSX) || starter.Allows(ExportDrive) {
		t.Fatalf("starter entitlements=%+v", starter.Entitlements)
	}
	business, _ := Lookup(Business)
	if !business.Allows(ExportDrive) {
		t.Fatal("business must include Drive export")
	}
	growth, _ := Lookup(Growth)
	if growth.Prices[Monthly].TotalSatang != 50000 || growth.Prices[SixMonths].TotalSatang != 285000 || growth.Prices[Yearly].TotalSatang != 510000 || growth.Limits.DocumentsPerMonth != 2000 || !growth.Allows(ExportDrive) {
		t.Fatalf("growth=%+v", growth)
	}
}

func TestGateUsesStoredPlanNotSubmittedIdentifiers(t *testing.T) {
	gate := Gate{Store: fixedPlanStore(Free)}
	decision, err := gate.Check(context.Background(), "organization", ExportXLSX, 1)
	if err != nil || decision.Allowed || decision.Plan != string(Free) {
		t.Fatalf("decision=%+v err=%v", decision, err)
	}
}

func TestFirmPlanIsTrustedAndPubliclyPriced(t *testing.T) {
	firm, ok := Lookup(AccountingFirm)
	if !ok || !firm.Allows(ManageFirm) || !firm.Allows(FirmPortfolio) || !firm.Allows(FirmApprovedExport) {
		t.Fatalf("firm entitlements=%+v found=%v", firm, ok)
	}
	if firm.Prices[Monthly].TotalSatang != 100000 || firm.Prices[SixMonths].TotalSatang != 570000 || firm.Prices[Yearly].TotalSatang != 1020000 || firm.Limits.ClientRelationships != 20 || firm.Allows(ExportDrive) {
		t.Fatalf("firm price, limits or entitlement: %+v", firm)
	}
	decision, err := (Gate{Store: fixedPlanStore(AccountingFirm)}).Check(context.Background(), "firm", FirmPortfolio, 0)
	if err != nil || !decision.Allowed {
		t.Fatalf("decision=%+v err=%v", decision, err)
	}
}
