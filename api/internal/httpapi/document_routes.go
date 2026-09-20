package httpapi

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"path"
	"strings"
	"time"
	"unicode/utf8"

	"plaiflow/api/internal/document"
	"plaiflow/api/internal/drive"
	"plaiflow/api/internal/plan"
	"plaiflow/api/internal/tenant"
)

func (s *server) registerDocumentRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /v1/o/{organization}/documents", s.uploadDocument)
	mux.HandleFunc("GET /v1/o/{organization}/documents", s.listDocuments)
	mux.HandleFunc("GET /v1/o/{organization}/documents/summary", s.documentSummary)
	mux.HandleFunc("GET /v1/o/{organization}/documents/export.csv", s.exportDocuments)
	mux.HandleFunc("GET /v1/o/{organization}/documents/export.xlsx", s.exportDocuments)
	mux.HandleFunc("POST /v1/o/{organization}/documents/drive", s.importDriveDocument)
	mux.HandleFunc("GET /v1/o/{organization}/documents/{document}/original", s.openDocument)
}

func (s *server) documentSummary(w http.ResponseWriter, r *http.Request) {
	session, membership, ok := s.workContext(w, r, false)
	if !ok {
		return
	}
	summary, err := s.config.Documents.Summary(r.Context(), session.UserID, membership.OrganizationID)
	if err != nil {
		writeError(w, r, http.StatusServiceUnavailable, "document_unavailable", "Document summary is unavailable")
		return
	}
	writeJSON(w, http.StatusOK, summary)
}

func (s *server) importDriveDocument(w http.ResponseWriter, r *http.Request) {
	session, membership, ok := s.workContext(w, r, true)
	if !ok {
		return
	}
	if membership.Role != tenant.Owner && membership.Role != tenant.Admin {
		writeError(w, r, http.StatusForbidden, "forbidden", "Only an organization owner or admin can import Drive files")
		return
	}
	if s.config.Drive == nil {
		writeError(w, r, http.StatusServiceUnavailable, "drive_not_configured", "Google Drive is unavailable")
		return
	}
	var request struct {
		FileID   string `json:"file_id"`
		Revision string `json:"revision"`
	}
	r.Body = http.MaxBytesReader(w, r.Body, 16<<10)
	if json.NewDecoder(r.Body).Decode(&request) != nil || len(request.FileID) > 256 || len(request.Revision) > 256 || request.FileID == "" || request.Revision == "" {
		writeError(w, r, http.StatusBadRequest, "invalid_drive_file", "Drive file selection is invalid")
		return
	}
	file, err := s.config.Drive.DownloadSelected(r.Context(), session.UserID, membership.OrganizationID, request.FileID, request.Revision)
	if err != nil {
		switch {
		case errors.Is(err, drive.ErrReconnectRequired):
			writeError(w, r, http.StatusConflict, "drive_reconnect_required", "Google Drive ต้องเชื่อมต่อใหม่ก่อนนำเข้า")
		case errors.Is(err, drive.ErrFileChanged):
			writeError(w, r, http.StatusConflict, "drive_file_changed", "ไฟล์ Drive เปลี่ยนแปลงหรือเข้าถึงไม่ได้ กรุณาเลือกใหม่")
		case errors.Is(err, drive.ErrNotConnected):
			writeError(w, r, http.StatusConflict, "drive_not_connected", "Google Drive ยังไม่ได้เชื่อมต่อ")
		default:
			writeError(w, r, http.StatusServiceUnavailable, "drive_unavailable", "Google Drive ไม่พร้อมใช้งาน")
		}
		return
	}
	defer file.Body.Close()
	filename := path.Base(strings.ReplaceAll(file.Name, "\\", "/"))
	if filename == "." || filename == "/" || filename == "" || utf8.RuneCountInString(filename) > 240 {
		filename = "drive-" + file.ID
	}
	result, err := s.config.Documents.Accept(r.Context(), document.AcceptInput{
		OrganizationID: membership.OrganizationID, ActorUserID: session.UserID, AttemptID: newUUID(),
		OriginKey: "drive:" + file.ID + ":" + file.Revision, Filename: filename, Channel: "Drive", Now: s.config.Now().UTC(),
		DriveConnectionID: membership.OrganizationID, DriveFileID: file.ID, DriveRevision: file.Revision,
	}, file.Body)
	if err != nil {
		if errors.Is(err, document.ErrQuota) {
			writeError(w, r, http.StatusConflict, "document_quota_exhausted", "Document allowance is exhausted")
		} else if errors.Is(err, document.ErrTrashed) {
			writeError(w, r, http.StatusConflict, "document_in_trash", "Matching document is in trash")
		} else {
			writeError(w, r, http.StatusUnprocessableEntity, "drive_file_rejected", "Drive file did not pass document checks")
		}
		return
	}
	status := http.StatusCreated
	if result.Duplicate {
		status = http.StatusOK
	}
	writeJSON(w, status, result)
}

func (s *server) exportDocuments(w http.ResponseWriter, r *http.Request) {
	session, membership, ok := s.workContext(w, r, false)
	if !ok {
		return
	}
	if membership.Role != tenant.Owner && membership.Role != tenant.Admin {
		writeError(w, r, http.StatusForbidden, "forbidden", "Only an organization owner or admin can export documents")
		return
	}
	filter, err := documentFilter(r)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_document_filter", "Document filters are invalid")
		return
	}
	format := "csv"
	if strings.HasSuffix(r.URL.Path, ".xlsx") {
		format = "xlsx"
	}
	capability := plan.ExportCSV
	if format == "xlsx" {
		capability = plan.ExportXLSX
	}
	decision, gateErr := s.config.Gate.Check(r.Context(), membership.OrganizationID, capability, 0)
	if gateErr != nil {
		writeError(w, r, http.StatusServiceUnavailable, "plan_unavailable", "Export entitlement is unavailable")
		return
	}
	if !decision.Allowed {
		writeError(w, r, http.StatusForbidden, "feature_unavailable", "This export format is not included in the current plan")
		return
	}
	var output bytes.Buffer
	_, err = s.config.Documents.Export(r.Context(), session.UserID, membership.OrganizationID, filter, format, &output)
	if err != nil {
		if strings.Contains(err.Error(), "requires asynchronous") {
			writeError(w, r, http.StatusRequestEntityTooLarge, "document_export_too_large", "กรุณาใช้ตัวกรองให้เหลือไม่เกิน 5,000 แถว")
			return
		}
		s.config.Logger.Error("document_export_failed", "request_id", requestID(r))
		writeError(w, r, http.StatusServiceUnavailable, "document_export_failed", "Document export is unavailable")
		return
	}
	w.Header().Set("Content-Type", map[string]string{"csv": "text/csv; charset=utf-8", "xlsx": "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"}[format])
	w.Header().Set("Content-Disposition", `attachment; filename="documents-`+s.config.Now().UTC().Format("20060102")+`.`+format+`"`)
	_, _ = w.Write(output.Bytes())
}

func documentFilter(r *http.Request) (document.Filter, error) {
	query := r.URL.Query()
	filter := document.Filter{Status: strings.TrimSpace(query.Get("status")), Channel: strings.TrimSpace(query.Get("source")), Filename: strings.TrimSpace(query.Get("filename")), Submitter: strings.TrimSpace(query.Get("submitter")), Assignee: strings.TrimSpace(query.Get("assignee"))}
	var err error
	if len(filter.Status) > 24 || len(filter.Channel) > 24 || len(filter.Filename) > 240 || len(filter.Submitter) > 64 || len(filter.Assignee) > 64 {
		return document.Filter{}, errors.New("filter too long")
	}
	parse := func(value string, end bool) (*time.Time, error) {
		if value == "" {
			return nil, nil
		}
		parsed, err := time.Parse(time.RFC3339, value)
		if err != nil {
			parsed, err = time.Parse("2006-01-02", value)
			if err != nil {
				return nil, err
			}
			if end {
				parsed = parsed.Add(24 * time.Hour)
			}
		}
		parsed = parsed.UTC()
		return &parsed, nil
	}
	if filter.From, err = parse(query.Get("from"), false); err != nil {
		return document.Filter{}, err
	}
	if filter.To, err = parse(query.Get("to"), true); err != nil {
		return document.Filter{}, err
	}
	if filter.From != nil && filter.To != nil && !filter.From.Before(*filter.To) {
		return document.Filter{}, errors.New("invalid date range")
	}
	return filter, nil
}

func (s *server) listDocuments(w http.ResponseWriter, r *http.Request) {
	session, membership, ok := s.workContext(w, r, false)
	if !ok {
		return
	}
	limit, valid := pageLimit(r.URL.Query().Get("limit"))
	if !valid || limit > 50 {
		writeError(w, r, http.StatusBadRequest, "invalid_cursor", "Pagination is invalid")
		return
	}
	filter, err := documentFilter(r)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_document_filter", "Document filters are invalid")
		return
	}
	page, err := s.config.Documents.ListFiltered(r.Context(), session.UserID, membership.OrganizationID, r.URL.Query().Get("cursor"), limit, filter)
	if errors.Is(err, document.ErrInvalidCursor) {
		writeError(w, r, http.StatusBadRequest, "invalid_cursor", "Pagination is invalid")
		return
	}
	if err != nil {
		writeError(w, r, http.StatusServiceUnavailable, "document_unavailable", "Documents are unavailable")
		return
	}
	writeJSON(w, http.StatusOK, page)
}

func (s *server) openDocument(w http.ResponseWriter, r *http.Request) {
	session, membership, ok := s.workContext(w, r, false)
	if !ok {
		return
	}
	doc, body, err := s.config.Documents.Open(r.Context(), session.UserID, membership.OrganizationID, r.PathValue("document"))
	if errors.Is(err, tenant.ErrNotFound) {
		writeError(w, r, http.StatusNotFound, "not_found", "Resource was not found")
		return
	}
	if err != nil {
		writeError(w, r, http.StatusServiceUnavailable, "document_unavailable", "Original is unavailable")
		return
	}
	defer body.Close()
	w.Header().Set("Content-Type", doc.MIME)
	w.Header().Set("Content-Disposition", "attachment; filename=\"document\"")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Cache-Control", "private, no-store")
	w.Header().Set("Content-Security-Policy", "sandbox")
	if _, err := io.Copy(w, body); err != nil {
		s.config.Logger.Error("document_stream_failed", "request_id", requestID(r))
	}
}

func (s *server) uploadDocument(w http.ResponseWriter, r *http.Request) {
	contentType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || contentType != "multipart/form-data" {
		writeError(w, r, http.StatusUnsupportedMediaType, "invalid_content_type", "File upload must be multipart")
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
	r.Body = http.MaxBytesReader(w, r.Body, document.MaxFileBytes+(1<<20))
	reader, err := r.MultipartReader()
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_upload", "Upload is invalid")
		return
	}
	part, err := reader.NextPart()
	if err != nil || part.FormName() != "file" {
		writeError(w, r, http.StatusBadRequest, "invalid_upload", "One file is required")
		return
	}
	defer part.Close()
	filename := path.Base(strings.ReplaceAll(part.FileName(), "\\", "/"))
	if filename == "." || filename == "/" || utf8.RuneCountInString(filename) < 1 || utf8.RuneCountInString(filename) > 240 {
		writeError(w, r, http.StatusUnprocessableEntity, "invalid_filename", "File name is invalid")
		return
	}
	attemptID := newUUID()
	origin := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	if origin == "" {
		origin = attemptID
	}
	if len(origin) > 128 || len(origin) < 8 {
		writeError(w, r, http.StatusBadRequest, "invalid_idempotency_key", "Upload key is invalid")
		return
	}
	digest := sha256.Sum256([]byte(session.UserID + ":" + origin))
	result, err := s.config.Documents.Accept(r.Context(), document.AcceptInput{
		OrganizationID: membership.OrganizationID, ActorUserID: session.UserID, AttemptID: attemptID,
		OriginKey: hex.EncodeToString(digest[:]), Filename: filename, Channel: "Web", Now: s.config.Now().UTC(),
	}, part)
	if err != nil {
		switch {
		case errors.Is(err, document.ErrUnsupportedType):
			writeError(w, r, http.StatusUnprocessableEntity, "unsupported_file", "File type is unsupported")
		case errors.Is(err, document.ErrTooLarge):
			writeError(w, r, http.StatusRequestEntityTooLarge, "file_too_large", "File exceeds 20 MiB")
		case errors.Is(err, document.ErrMalware):
			writeError(w, r, http.StatusUnprocessableEntity, "unsafe_file", "File cannot be accepted")
		case errors.Is(err, document.ErrScanUnavailable):
			writeError(w, r, http.StatusServiceUnavailable, "scanner_unavailable", "File checking is temporarily unavailable")
		case errors.Is(err, document.ErrQuota):
			writeError(w, r, http.StatusConflict, "document_quota_exhausted", "Document allowance is exhausted")
		case errors.Is(err, document.ErrTrashed):
			writeError(w, r, http.StatusConflict, "document_in_trash", "Matching document must be restored by an administrator")
		default:
			s.config.Logger.Error("document_upload_failed", "request_id", requestID(r))
			writeError(w, r, http.StatusServiceUnavailable, "document_unavailable", "Document could not be accepted")
		}
		return
	}
	status := http.StatusCreated
	if result.Duplicate {
		status = http.StatusOK
	}
	writeJSON(w, status, result)
}
