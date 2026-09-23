package httpapi

import (
	"errors"
	"io"
	"net/http"
	"regexp"
	"strings"
	"unicode/utf8"

	"plaiflow/api/internal/tenant"
)

var businessDigits = regexp.MustCompile(`^[0-9]+$`)

func (s *server) getBusinessProfile(w http.ResponseWriter, r *http.Request) {
	session, ok := s.authenticated(w, r)
	if !ok {
		return
	}
	p, err := s.config.Tenants.GetBusinessProfile(r.Context(), session.UserID, r.PathValue("organization"))
	if err != nil {
		if errors.Is(err, tenant.ErrNotFound) {
			writeError(w, r, http.StatusNotFound, "not_found", "Resource was not found")
		} else {
			writeError(w, r, http.StatusServiceUnavailable, "business_unavailable", "Business profile is unavailable")
		}
		return
	}
	writeJSON(w, http.StatusOK, p)
}

func (s *server) businessLogo(w http.ResponseWriter, r *http.Request) {
	session, ok := s.authenticated(w, r)
	if !ok {
		return
	}
	p, err := s.config.Tenants.GetBusinessProfile(r.Context(), session.UserID, r.PathValue("organization"))
	if err != nil || !p.HasLogo {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", p.LogoType)
	w.Header().Set("Cache-Control", "private, no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Write(p.Logo)
}

func (s *server) updateBusinessProfile(w http.ResponseWriter, r *http.Request) {
	if !strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data;") {
		writeError(w, r, http.StatusUnsupportedMediaType, "invalid_content_type", "Request content type is invalid")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 600<<10)
	session, ok := s.authenticatedMutation(w, r)
	if !ok {
		return
	}
	membership, err := s.config.Tenants.ResolveMembership(r.Context(), session.UserID, r.PathValue("organization"))
	if err != nil {
		writeError(w, r, http.StatusNotFound, "not_found", "Resource was not found")
		return
	}
	if !membership.Role.Allows(tenant.EditOrganization) {
		writeError(w, r, http.StatusForbidden, "forbidden", "Action is not allowed")
		return
	}
	if err = r.ParseMultipartForm(512 << 10); err != nil {
		writeError(w, r, http.StatusRequestEntityTooLarge, "invalid_business", "Business profile is invalid")
		return
	}
	if r.MultipartForm != nil {
		defer r.MultipartForm.RemoveAll()
	}
	p := tenant.BusinessProfile{
		BusinessType: strings.TrimSpace(r.FormValue("business_type")), VATStatus: strings.TrimSpace(r.FormValue("vat_status")),
		BranchType: strings.TrimSpace(r.FormValue("branch_type")), NameTH: strings.TrimSpace(r.FormValue("name_th")),
		NameEN: strings.TrimSpace(r.FormValue("name_en")), TaxID: strings.TrimSpace(r.FormValue("tax_id")),
		Address1: strings.TrimSpace(r.FormValue("address_1")), Address2: strings.TrimSpace(r.FormValue("address_2")),
		District: strings.TrimSpace(r.FormValue("district")), Province: strings.TrimSpace(r.FormValue("province")),
		PostalCode: strings.TrimSpace(r.FormValue("postal_code")), Phone: strings.TrimSpace(r.FormValue("phone")),
	}
	if !validBusinessProfile(p) {
		writeError(w, r, http.StatusUnprocessableEntity, "invalid_business", "Business profile is invalid")
		return
	}
	file, _, err := r.FormFile("logo")
	if err == nil {
		defer file.Close()
		p.Logo, err = io.ReadAll(io.LimitReader(file, (512<<10)+1))
		if err != nil || len(p.Logo) > 512<<10 {
			writeError(w, r, http.StatusRequestEntityTooLarge, "invalid_logo", "Logo is too large")
			return
		}
		if len(p.Logo) > 0 {
			p.LogoType = http.DetectContentType(p.Logo)
			if p.LogoType != "image/png" && p.LogoType != "image/jpeg" && p.LogoType != "image/webp" {
				writeError(w, r, http.StatusUnprocessableEntity, "invalid_logo", "Logo must be PNG, JPEG, or WebP")
				return
			}
		} else {
			p.Logo = nil
		}
	} else if !errors.Is(err, http.ErrMissingFile) {
		writeError(w, r, http.StatusUnprocessableEntity, "invalid_logo", "Logo is invalid")
		return
	}
	if err = s.config.Tenants.UpdateBusinessProfile(r.Context(), session.UserID, membership.OrganizationID, p, s.config.Now().UTC()); err != nil {
		writeError(w, r, http.StatusServiceUnavailable, "business_not_saved", "Business profile could not be saved")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func validBusinessProfile(p tenant.BusinessProfile) bool {
	oneOf := func(value string, allowed ...string) bool {
		for _, item := range allowed {
			if value == item {
				return true
			}
		}
		return false
	}
	within := func(value string, max int) bool { return utf8.RuneCountInString(value) <= max }
	return oneOf(p.BusinessType, "บริษัทจำกัด", "ห้างหุ้นส่วนจำกัด", "ห้างหุ้นส่วนสามัญ", "ห้างหุ้นส่วนสามัญนิติบุคคล", "บริษัทมหาชนจำกัด", "ร้านค้า/กิจการเจ้าของคนเดียว", "คณะบุคคล", "มูลนิธิ", "สมาคม", "สหกรณ์", "บุคคลธรรมดา/ฟรีแลนซ์") &&
		oneOf(p.VATStatus, "registered", "unregistered") && oneOf(p.BranchType, "head", "branch", "none") &&
		p.NameTH != "" && within(p.NameTH, 160) && within(p.NameEN, 160) && within(p.Address1, 240) && within(p.Address2, 240) &&
		within(p.District, 120) && within(p.Province, 120) &&
		(p.TaxID == "" || len(p.TaxID) == 13 && businessDigits.MatchString(p.TaxID)) &&
		(p.PostalCode == "" || len(p.PostalCode) == 5 && businessDigits.MatchString(p.PostalCode)) &&
		len(p.Phone) >= 9 && len(p.Phone) <= 10 && businessDigits.MatchString(p.Phone)
}
