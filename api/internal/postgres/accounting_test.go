package postgres

import (
	"errors"
	"testing"
	"time"

	"plaiflow/api/internal/accounting"
	"plaiflow/api/internal/business"
)

func TestAccountingCategoriesAndRulesStayInTheirOrganization(t *testing.T) {
	store, ctx := isolatedTestStore(t, 10)
	userA, userB := postgresUUID(), postgresUUID()
	orgA, orgB := postgresUUID(), postgresUUID()
	for _, user := range []string{userA, userB} {
		if _, err := store.pool.Exec(ctx, `INSERT INTO users(id) VALUES($1)`, user); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := store.CreateOrganization(ctx, userA, orgA, "Accounting A", time.Now()); err != nil {
		t.Fatal(err)
	}
	if _, err := store.CreateOrganization(ctx, userB, orgB, "Accounting B", time.Now()); err != nil {
		t.Fatal(err)
	}
	categoriesA, err := store.ListCategories(ctx, userA, orgA)
	if err != nil || len(categoriesA) != 3 {
		t.Fatalf("expected three editable starter categories: %+v %v", categoriesA, err)
	}
	categoryB, err := store.CreateCategory(ctx, userB, orgB, accounting.Category{ID: postgresUUID(), Name: "เฉพาะ B"}, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	categoriesA, err = store.ListCategories(ctx, userA, orgA)
	if err != nil || len(categoriesA) != 3 {
		t.Fatalf("another Organization's category leaked: %+v %v", categoriesA, err)
	}
	vendor, err := business.NormalizeVendor(business.VendorInput{DisplayName: "ร้าน A", Country: "TH"})
	if err != nil {
		t.Fatal(err)
	}
	vendor.ID = postgresUUID()
	if _, err := store.CreateVendor(ctx, userA, orgA, vendor, time.Now()); err != nil {
		t.Fatal(err)
	}
	_, err = store.CreateRule(ctx, accounting.RuleInput{ID: postgresUUID(), ActorID: userA, OrganizationID: orgA,
		VendorID: vendor.ID, CategoryID: categoryB.ID, ApprovedAt: time.Now()})
	if !errors.Is(err, accounting.ErrInvalid) {
		t.Fatalf("cross-Organization category entered a rule: %v", err)
	}
}
