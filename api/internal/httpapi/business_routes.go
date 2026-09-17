package httpapi

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"unicode/utf8"

	"plaiflow/api/internal/business"
	"plaiflow/api/internal/plan"
	"plaiflow/api/internal/tenant"
)

func (s *server) registerBusinessRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /v1/o/{organization}/plan", s.organizationPlan)
	mux.HandleFunc("GET /v1/o/{organization}/vendors", s.listVendors)
	mux.HandleFunc("POST /v1/o/{organization}/vendors", s.createVendor)
	mux.HandleFunc("GET /v1/o/{organization}/vendors/export.csv", s.exportVendors)
	mux.HandleFunc("GET /v1/o/{organization}/vendors/export.xlsx", s.exportVendors)
	mux.HandleFunc("POST /v1/o/{organization}/vendor-imports/preview", s.previewVendorImport)
	mux.HandleFunc("GET /v1/o/{organization}/vendor-imports/{preview}", s.getVendorImportPreview)
	mux.HandleFunc("POST /v1/o/{organization}/vendor-imports/{preview}/commit", s.commitVendorImport)
}

func (s *server) getVendorImportPreview(w http.ResponseWriter, r *http.Request) {
	session, membership, ok := s.workContext(w, r, false)
	if !ok {
		return
	}
	if membership.Role != tenant.Owner && membership.Role != tenant.Admin {
		writeError(w, r, http.StatusForbidden, "forbidden", "Action is not allowed")
		return
	}
	preview, err := s.config.Business.GetImportPreview(r.Context(), session.UserID, membership.OrganizationID, r.PathValue("preview"), s.config.Now().UTC())
	if err != nil {
		writeBusinessError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, preview)
}

func (s *server) listPlans(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"plans": plan.Catalog(), "billing_enabled": false})
}

func (s *server) organizationPlan(w http.ResponseWriter, r *http.Request) {
	_, membership, ok := s.workContext(w, r, false)
	if !ok {
		return
	}
	key, err := s.config.PlanStore.EffectivePlan(r.Context(), membership.OrganizationID)
	definition, found := plan.Lookup(key)
	if err != nil || !found {
		writeError(w, r, http.StatusServiceUnavailable, "plan_unavailable", "Plan is unavailable")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"effective_plan": definition})
}

func (s *server) createVendor(w http.ResponseWriter, r *http.Request) {
	session, membership, ok := s.workContext(w, r, true)
	if !ok {
		return
	}
	if membership.Role != tenant.Owner && membership.Role != tenant.Admin {
		writeError(w, r, http.StatusForbidden, "forbidden", "Action is not allowed")
		return
	}
	decision, err := s.config.Gate.Check(r.Context(), membership.OrganizationID, plan.ManageBusinessContacts, 1)
	if err != nil || !decision.Allowed {
		writeError(w, r, http.StatusPaymentRequired, "feature_unavailable", "Business contacts are unavailable")
		return
	}
	contact, err := business.NormalizeVendor(business.VendorInput{
		DisplayName: r.FormValue("display_name"), ContactCode: r.FormValue("contact_code"), Country: r.FormValue("country"),
		TaxID: r.FormValue("tax_id"), BranchCode: r.FormValue("branch_code"), Customer: r.FormValue("customer") == "true",
	})
	if err != nil {
		writeError(w, r, http.StatusUnprocessableEntity, "invalid_vendor", "Vendor input is invalid")
		return
	}
	contact.ID = newUUID()
	created, err := s.config.Business.CreateVendor(r.Context(), session.UserID, membership.OrganizationID, contact, s.config.Now().UTC())
	if err != nil {
		writeBusinessError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func (s *server) listVendors(w http.ResponseWriter, r *http.Request) {
	session, membership, ok := s.workContext(w, r, false)
	if !ok {
		return
	}
	limit, valid := pageLimit(r.URL.Query().Get("limit"))
	if !valid {
		writeError(w, r, http.StatusBadRequest, "invalid_cursor", "Pagination is invalid")
		return
	}
	search := strings.TrimSpace(r.URL.Query().Get("q"))
	if utf8.RuneCountInString(search) > 240 {
		writeError(w, r, http.StatusBadRequest, "invalid_search", "Search is invalid")
		return
	}
	page, err := s.config.Business.ListVendors(r.Context(), session.UserID, membership.OrganizationID, search, r.URL.Query().Get("cursor"), limit)
	if err != nil {
		writeBusinessError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, page)
}

func (s *server) exportVendors(w http.ResponseWriter, r *http.Request) {
	session, membership, ok := s.workContext(w, r, false)
	if !ok {
		return
	}
	if membership.Role != tenant.Owner && membership.Role != tenant.Admin {
		writeError(w, r, http.StatusForbidden, "forbidden", "Action is not allowed")
		return
	}
	xlsx := strings.HasSuffix(r.URL.Path, ".xlsx")
	capability := plan.ExportCSV
	if xlsx {
		capability = plan.ExportXLSX
	}
	decision, err := s.config.Gate.Check(r.Context(), membership.OrganizationID, capability, 1)
	if err != nil || !decision.Allowed {
		writeError(w, r, http.StatusPaymentRequired, "feature_unavailable", "Export is unavailable")
		return
	}
	contacts, err := s.config.Business.ExportVendors(r.Context(), session.UserID, membership.OrganizationID, 5000)
	if err != nil {
		writeBusinessError(w, r, err)
		return
	}
	extension, contentType := "csv", "text/csv; charset=utf-8"
	write := business.WriteCSV
	if xlsx {
		extension, contentType, write = "xlsx", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", business.WriteXLSX
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="vendors-%s.%s"`, s.config.Now().UTC().Format("20060102"), extension))
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Cache-Control", "private, no-store")
	if err := write(w, contacts); err != nil {
		s.config.Logger.Error("vendor_export_failed", "request_id", requestID(r))
		return
	}
	_ = s.config.Gate.Record(r.Context(), membership.OrganizationID, capability, int64(len(contacts)), requestID(r))
}

func (s *server) previewVendorImport(w http.ResponseWriter, r *http.Request) {
	if !strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data;") {
		writeError(w, r, http.StatusUnsupportedMediaType, "invalid_content_type", "Request content type is invalid")
		return
	}
	session, ok := s.authenticatedMutation(w, r)
	if !ok {
		return
	}
	membership, err := s.config.Tenants.ResolveMembership(r.Context(), session.UserID, r.PathValue("organization"))
	if err != nil {
		writeError(w, r, http.StatusNotFound, "not_found", "Resource was not found")
		return
	}
	if membership.Role != tenant.Owner && membership.Role != tenant.Admin {
		writeError(w, r, http.StatusForbidden, "forbidden", "Action is not allowed")
		return
	}
	decision, err := s.config.Gate.Check(r.Context(), membership.OrganizationID, plan.ImportBusinessContacts, 1)
	if err != nil || !decision.Allowed {
		writeError(w, r, http.StatusPaymentRequired, "feature_unavailable", "Import is unavailable")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 11<<20)
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		writeError(w, r, http.StatusRequestEntityTooLarge, "invalid_import", "Import file is invalid")
		return
	}
	if r.MultipartForm != nil {
		defer r.MultipartForm.RemoveAll()
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		writeError(w, r, http.StatusUnprocessableEntity, "invalid_import", "Import file is invalid")
		return
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, (10<<20)+1))
	if err != nil || len(data) > 10<<20 {
		writeError(w, r, http.StatusRequestEntityTooLarge, "invalid_import", "Import file is invalid")
		return
	}
	contacts, err := business.ParseImport(header.Filename, header.Header.Get("Content-Type"), data)
	if err != nil {
		writeError(w, r, http.StatusUnprocessableEntity, "invalid_import", "Import file is invalid")
		return
	}
	preview, err := s.config.Business.CreateImportPreview(r.Context(), session.UserID, membership.OrganizationID, newUUID(), contacts, s.config.Now().UTC())
	if err != nil {
		writeBusinessError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, preview)
}

func (s *server) commitVendorImport(w http.ResponseWriter, r *http.Request) {
	session, membership, ok := s.workContext(w, r, true)
	if !ok {
		return
	}
	if membership.Role != tenant.Owner && membership.Role != tenant.Admin {
		writeError(w, r, http.StatusForbidden, "forbidden", "Action is not allowed")
		return
	}
	decision, err := s.config.Gate.Check(r.Context(), membership.OrganizationID, plan.ImportBusinessContacts, 1)
	if err != nil || !decision.Allowed {
		writeError(w, r, http.StatusPaymentRequired, "feature_unavailable", "Import is unavailable")
		return
	}
	result, err := s.config.Business.CommitImport(r.Context(), session.UserID, membership.OrganizationID, r.PathValue("preview"), s.config.Now().UTC())
	if err != nil {
		writeBusinessError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, result)
}

func writeBusinessError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, business.ErrInvalid):
		writeError(w, r, http.StatusUnprocessableEntity, "invalid_vendor", "Vendor input is invalid")
	case errors.Is(err, business.ErrDuplicate):
		writeError(w, r, http.StatusConflict, "duplicate_vendor", "Vendor already exists")
	case errors.Is(err, business.ErrForbidden):
		writeError(w, r, http.StatusForbidden, "forbidden", "Action is not allowed")
	case errors.Is(err, business.ErrNotFound), errors.Is(err, business.ErrExpired):
		writeError(w, r, http.StatusNotFound, "not_found", "Resource was not found")
	default:
		writeError(w, r, http.StatusServiceUnavailable, "business_unavailable", "Business data is unavailable")
	}
}
