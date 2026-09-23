package httpapi

import (
	"testing"

	"plaiflow/api/internal/tenant"
)

func TestValidBusinessProfile(t *testing.T) {
	p := tenant.BusinessProfile{BusinessType: "บริษัทมหาชนจำกัด", VATStatus: "registered", BranchType: "head", NameTH: "บูเทพ", Phone: "0635167015", TaxID: "0123456789012", PostalCode: "10110"}
	if !validBusinessProfile(p) {
		t.Fatal("valid profile rejected")
	}
	p.Phone = "063516701x"
	if validBusinessProfile(p) {
		t.Fatal("invalid phone accepted")
	}
	p.Phone = "0635167015"
	p.BusinessType = "unexpected"
	if validBusinessProfile(p) {
		t.Fatal("unsupported business type accepted")
	}
}
