package httpapi

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"plaiflow/api/internal/auth"
	"plaiflow/api/internal/tenant"
	"plaiflow/api/internal/work"
)

func (s *server) registerWorkRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /v1/o/{organization}/tasks", s.listTasks)
	mux.HandleFunc("POST /v1/o/{organization}/tasks", s.createTask)
	mux.HandleFunc("GET /v1/o/{organization}/tasks/{task}", s.getTask)
	mux.HandleFunc("POST /v1/o/{organization}/tasks/{task}", s.updateTask)
	mux.HandleFunc("POST /v1/o/{organization}/tasks/{task}/watchers/{user}", s.addWatcher)
	mux.HandleFunc("POST /v1/o/{organization}/tasks/{task}/watchers/{user}/remove", s.removeWatcher)
	mux.HandleFunc("GET /v1/o/{organization}/notifications", s.listNotifications)
	mux.HandleFunc("POST /v1/o/{organization}/notifications/{notification}/read", s.markNotificationRead)
	mux.HandleFunc("POST /v1/o/{organization}/notifications/read-all", s.markAllNotificationsRead)
	mux.HandleFunc("GET /v1/o/{organization}/notification-preferences", s.listNotificationPreferences)
	mux.HandleFunc("POST /v1/o/{organization}/notification-preferences", s.setNotificationPreference)
	mux.HandleFunc("GET /v1/o/{organization}/work-dashboard", s.workDashboard)
	mux.HandleFunc("GET /v1/o/{organization}/assistant-summary", s.assistantSummary)
	mux.HandleFunc("POST /v1/o/{organization}/task-exports", s.exportTasks)
	mux.HandleFunc("GET /v1/o/{organization}/task-exports", s.listExports)
}

func (s *server) workContext(w http.ResponseWriter, r *http.Request, mutation bool) (auth.Session, tenant.Membership, bool) {
	var session auth.Session
	var ok bool
	if mutation {
		session, ok = s.formMutation(w, r)
	} else {
		session, ok = s.authenticated(w, r)
	}
	if !ok {
		return auth.Session{}, tenant.Membership{}, false
	}
	membership, err := s.config.Tenants.ResolveMembership(r.Context(), session.UserID, r.PathValue("organization"))
	if err != nil {
		writeError(w, r, http.StatusNotFound, "not_found", "Resource was not found")
		return auth.Session{}, tenant.Membership{}, false
	}
	return session, membership, true
}

func (s *server) createTask(w http.ResponseWriter, r *http.Request) {
	session, membership, ok := s.workContext(w, r, true)
	if !ok {
		return
	}
	decision, err := s.config.Gate.Check(r.Context(), membership.OrganizationID, work.CreateTasks, 1)
	if err != nil || !decision.Allowed {
		code := "feature_unavailable"
		if err == nil && decision.Reason == "limit_reached" {
			code = "usage_limit_reached"
		}
		writeError(w, r, http.StatusPaymentRequired, code, "Task creation is unavailable")
		return
	}
	title := strings.TrimSpace(r.FormValue("title"))
	description := r.FormValue("description")
	priority := work.Priority(r.FormValue("priority"))
	if priority == "" {
		priority = work.Normal
	}
	dueOn := r.FormValue("due_on")
	if utf8.RuneCountInString(title) < 1 || utf8.RuneCountInString(title) > 200 || utf8.RuneCountInString(description) > 5000 ||
		!validPriority(priority) || !validDate(dueOn) {
		writeError(w, r, http.StatusUnprocessableEntity, "invalid_task", "Task input is invalid")
		return
	}
	command := work.CreateTask{ID: newUUID(), OrganizationID: membership.OrganizationID, ActorUserID: session.UserID,
		Title: title, Description: description, AssigneeUserID: r.FormValue("assignee_user_id"), DueOn: dueOn, Priority: priority, Now: s.config.Now().UTC()}
	task, err := s.config.Work.CreateTask(r.Context(), command)
	if err != nil {
		writeWorkError(w, r, err)
		return
	}
	if err := s.config.Gate.Record(r.Context(), membership.OrganizationID, work.CreateTasks, 1, command.ID); err != nil {
		s.config.Logger.Warn("usage_record_failed", "request_id", requestID(r), "capability", work.CreateTasks)
	}
	writeJSON(w, http.StatusCreated, task)
}

func (s *server) listTasks(w http.ResponseWriter, r *http.Request) {
	session, membership, ok := s.workContext(w, r, false)
	if !ok {
		return
	}
	filter, ok := parseTaskFilter(w, r)
	if !ok {
		return
	}
	page, err := s.config.Work.ListTasks(r.Context(), session.UserID, membership.OrganizationID, filter)
	if err != nil {
		writeWorkError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, page)
}

func (s *server) getTask(w http.ResponseWriter, r *http.Request) {
	session, membership, ok := s.workContext(w, r, false)
	if !ok {
		return
	}
	task, err := s.config.Work.GetTask(r.Context(), session.UserID, membership.OrganizationID, r.PathValue("task"))
	if err != nil {
		writeWorkError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, task)
}

func (s *server) updateTask(w http.ResponseWriter, r *http.Request) {
	session, membership, ok := s.workContext(w, r, true)
	if !ok {
		return
	}
	if err := r.ParseForm(); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_task", "Task input is invalid")
		return
	}
	_, titleSet := r.Form["title"]
	_, descriptionSet := r.Form["description"]
	_, assigneeSet := r.Form["assignee_user_id"]
	_, dueOnSet := r.Form["due_on"]
	_, statusSet := r.Form["status"]
	_, prioritySet := r.Form["priority"]
	command := work.UpdateTask{ID: r.PathValue("task"), OrganizationID: membership.OrganizationID, ActorUserID: session.UserID,
		Title: strings.TrimSpace(r.FormValue("title")), Description: r.FormValue("description"), AssigneeUserID: r.FormValue("assignee_user_id"),
		DueOn: r.FormValue("due_on"), Status: work.Status(r.FormValue("status")), Priority: work.Priority(r.FormValue("priority")),
		TitleSet: titleSet, DescriptionSet: descriptionSet, AssigneeSet: assigneeSet, DueOnSet: dueOnSet, StatusSet: statusSet, PrioritySet: prioritySet,
		Now: s.config.Now().UTC()}
	if (command.TitleSet && (utf8.RuneCountInString(command.Title) < 1 || utf8.RuneCountInString(command.Title) > 200)) ||
		(command.DescriptionSet && utf8.RuneCountInString(command.Description) > 5000) ||
		(command.StatusSet && !validStatus(command.Status)) || (command.PrioritySet && !validPriority(command.Priority)) ||
		(command.DueOnSet && !validDate(command.DueOn)) {
		writeError(w, r, http.StatusUnprocessableEntity, "invalid_task", "Task input is invalid")
		return
	}
	task, err := s.config.Work.UpdateTask(r.Context(), command)
	if err != nil {
		writeWorkError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, task)
}

func (s *server) addWatcher(w http.ResponseWriter, r *http.Request)    { s.setWatcher(w, r, true) }
func (s *server) removeWatcher(w http.ResponseWriter, r *http.Request) { s.setWatcher(w, r, false) }
func (s *server) setWatcher(w http.ResponseWriter, r *http.Request, add bool) {
	session, membership, ok := s.workContext(w, r, true)
	if !ok {
		return
	}
	if err := s.config.Work.SetWatcher(r.Context(), membership.OrganizationID, session.UserID, r.PathValue("task"), r.PathValue("user"), add, s.config.Now().UTC()); err != nil {
		writeWorkError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *server) listNotifications(w http.ResponseWriter, r *http.Request) {
	session, membership, ok := s.workContext(w, r, false)
	if !ok {
		return
	}
	limit, valid := pageLimit(r.URL.Query().Get("limit"))
	if !valid {
		writeError(w, r, http.StatusBadRequest, "invalid_cursor", "Pagination is invalid")
		return
	}
	page, err := s.config.Work.ListNotifications(r.Context(), session.UserID, membership.OrganizationID, r.URL.Query().Get("cursor"), limit)
	if err != nil {
		writeWorkError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, page)
}

func (s *server) markNotificationRead(w http.ResponseWriter, r *http.Request) {
	session, membership, ok := s.workContext(w, r, true)
	if !ok {
		return
	}
	if err := s.config.Work.MarkNotificationRead(r.Context(), session.UserID, membership.OrganizationID, r.PathValue("notification"), s.config.Now().UTC()); err != nil {
		writeWorkError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *server) markAllNotificationsRead(w http.ResponseWriter, r *http.Request) {
	session, membership, ok := s.workContext(w, r, true)
	if !ok {
		return
	}
	if err := s.config.Work.MarkAllNotificationsRead(r.Context(), session.UserID, membership.OrganizationID, s.config.Now().UTC()); err != nil {
		writeWorkError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *server) listNotificationPreferences(w http.ResponseWriter, r *http.Request) {
	session, membership, ok := s.workContext(w, r, false)
	if !ok {
		return
	}
	preferences, err := s.config.Work.ListPreferences(r.Context(), session.UserID, membership.OrganizationID)
	if err != nil {
		writeWorkError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"preferences": preferences})
}

func (s *server) setNotificationPreference(w http.ResponseWriter, r *http.Request) {
	session, membership, ok := s.workContext(w, r, true)
	if !ok {
		return
	}
	preference := work.Preference{OrganizationID: membership.OrganizationID, UserID: session.UserID, Category: r.FormValue("category"), WebMode: r.FormValue("web_mode"), LineMode: r.FormValue("line_mode"), UpdatedAt: s.config.Now().UTC()}
	if !validCategory(preference.Category) || !validMode(preference.WebMode) || !validMode(preference.LineMode) {
		writeError(w, r, http.StatusUnprocessableEntity, "invalid_preference", "Notification preference is invalid")
		return
	}
	if err := s.config.Work.SetPreference(r.Context(), preference); err != nil {
		writeWorkError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *server) workDashboard(w http.ResponseWriter, r *http.Request) {
	session, membership, ok := s.workContext(w, r, false)
	if !ok {
		return
	}
	dashboard, err := s.config.Work.Dashboard(r.Context(), session.UserID, membership.OrganizationID)
	if err != nil {
		writeWorkError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, dashboard)
}

func (s *server) assistantSummary(w http.ResponseWriter, r *http.Request) {
	session, membership, ok := s.workContext(w, r, false)
	if !ok {
		return
	}
	decision, err := s.config.Gate.Check(r.Context(), membership.OrganizationID, work.UseAssistant, 1)
	if err != nil || !decision.Allowed {
		writeError(w, r, http.StatusPaymentRequired, "feature_unavailable", "Assistant summary is unavailable")
		return
	}
	dashboard, err := s.config.Work.Dashboard(r.Context(), session.UserID, membership.OrganizationID)
	if err != nil {
		writeWorkError(w, r, err)
		return
	}
	summary := work.Summarize(dashboard, s.config.Now())
	if err := s.config.Gate.Record(r.Context(), membership.OrganizationID, work.UseAssistant, 1, requestID(r)); err != nil {
		s.config.Logger.Warn("usage_record_failed", "request_id", requestID(r), "capability", work.UseAssistant)
	}
	writeJSON(w, http.StatusOK, summary)
}

func (s *server) exportTasks(w http.ResponseWriter, r *http.Request) {
	session, membership, ok := s.workContext(w, r, true)
	if !ok {
		return
	}
	if membership.Role != tenant.Owner && membership.Role != tenant.Admin {
		writeError(w, r, http.StatusForbidden, "forbidden", "Action is not allowed")
		return
	}
	filter, ok := parseTaskFilter(w, r)
	if !ok {
		return
	}
	decision, err := s.config.Gate.Check(r.Context(), membership.OrganizationID, work.ExportTasks, 1)
	if err != nil || !decision.Allowed {
		writeError(w, r, http.StatusPaymentRequired, "feature_unavailable", "CSV export is unavailable")
		return
	}
	request := work.ExportRequest{ID: newUUID(), OrganizationID: membership.OrganizationID, ActorUserID: session.UserID, RequestID: requestID(r), Filters: filter, Now: s.config.Now().UTC()}
	if err := s.config.Work.AuditExport(r.Context(), request, "requested", 0, 0, ""); err != nil {
		writeError(w, r, http.StatusServiceUnavailable, "export_unavailable", "CSV export is unavailable")
		return
	}
	count, err := s.config.Work.CountExport(r.Context(), session.UserID, membership.OrganizationID, filter)
	if err != nil {
		_ = s.config.Work.AuditExport(r.Context(), request, "failed", 0, 0, "count_failed")
		writeWorkError(w, r, err)
		return
	}
	if count > 100000 {
		_ = s.config.Work.AuditExport(r.Context(), request, "rejected", count, 0, "export_too_large")
		writeError(w, r, http.StatusRequestEntityTooLarge, "export_too_large", "Narrow the filters and try again")
		return
	}
	if count > 5000 {
		decision, err = s.config.Gate.Check(r.Context(), membership.OrganizationID, work.ExportTasksAsync, 1)
		if err != nil || !decision.Allowed {
			_ = s.config.Work.AuditExport(r.Context(), request, "rejected", count, 0, "feature_unavailable")
			writeError(w, r, http.StatusPaymentRequired, "feature_unavailable", "Background CSV export is unavailable")
			return
		}
		job, err := s.config.Work.QueueExport(r.Context(), request, count)
		if err != nil {
			writeWorkError(w, r, err)
			return
		}
		_ = s.config.Gate.Record(r.Context(), membership.OrganizationID, work.ExportTasksAsync, 1, request.ID)
		writeJSON(w, http.StatusAccepted, job)
		return
	}
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="tasks-`+s.config.Now().UTC().Format("20060102")+`-`+request.ID+`.csv"`)
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Cache-Control", "private, no-store")
	counter := &countingWriter{ResponseWriter: w}
	csvWriter, err := work.NewCSVWriter(counter)
	if err == nil {
		err = s.config.Work.ExportRows(r.Context(), session.UserID, membership.OrganizationID, filter, csvWriter.Write)
	}
	if closeErr := csvWriterClose(csvWriter); err == nil {
		err = closeErr
	}
	if err != nil {
		_ = s.config.Work.AuditExport(r.Context(), request, "failed", count, counter.bytes, "stream_failed")
		return
	}
	_ = s.config.Work.AuditExport(r.Context(), request, "completed", count, counter.bytes, "")
	_ = s.config.Gate.Record(r.Context(), membership.OrganizationID, work.ExportTasks, count, request.ID)
}

func (s *server) listExports(w http.ResponseWriter, r *http.Request) {
	session, membership, ok := s.workContext(w, r, false)
	if !ok {
		return
	}
	if membership.Role != tenant.Owner && membership.Role != tenant.Admin {
		writeError(w, r, http.StatusForbidden, "forbidden", "Action is not allowed")
		return
	}
	limit, valid := pageLimit(r.URL.Query().Get("limit"))
	if !valid {
		writeError(w, r, http.StatusBadRequest, "invalid_cursor", "Pagination is invalid")
		return
	}
	page, err := s.config.Work.ListExports(r.Context(), session.UserID, membership.OrganizationID, r.URL.Query().Get("cursor"), limit)
	if err != nil {
		writeWorkError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, page)
}

type countingWriter struct {
	http.ResponseWriter
	bytes int64
}

func (w *countingWriter) Write(value []byte) (int, error) {
	n, err := w.ResponseWriter.Write(value)
	w.bytes += int64(n)
	return n, err
}

func csvWriterClose(w *work.CSVWriter) error {
	if w == nil {
		return nil
	}
	return w.Close()
}

func parseTaskFilter(w http.ResponseWriter, r *http.Request) (work.TaskFilter, bool) {
	allowed := map[string]bool{"status": true, "priority": true, "assignee_user_id": true, "creator_user_id": true, "due_from": true, "due_to": true, "created_from": true, "created_to": true, "overdue": true, "cursor": true, "limit": true, "csrf_token": true}
	for key := range r.URL.Query() {
		if !allowed[key] {
			writeError(w, r, http.StatusBadRequest, "invalid_filter", "Task filter is invalid")
			return work.TaskFilter{}, false
		}
	}
	limit, ok := pageLimit(r.URL.Query().Get("limit"))
	if !ok {
		writeError(w, r, http.StatusBadRequest, "invalid_cursor", "Pagination is invalid")
		return work.TaskFilter{}, false
	}
	filter := work.TaskFilter{Status: r.FormValue("status"), Priority: r.FormValue("priority"), AssigneeUserID: r.FormValue("assignee_user_id"), CreatorUserID: r.FormValue("creator_user_id"),
		DueFrom: r.FormValue("due_from"), DueTo: r.FormValue("due_to"), CreatedFrom: r.FormValue("created_from"), CreatedTo: r.FormValue("created_to"), Cursor: r.FormValue("cursor"), Limit: limit}
	if value := r.FormValue("overdue"); value != "" {
		parsed, err := strconv.ParseBool(value)
		if err != nil {
			writeError(w, r, http.StatusBadRequest, "invalid_filter", "Task filter is invalid")
			return work.TaskFilter{}, false
		}
		filter.Overdue = &parsed
	}
	if filter.Status != "" && !validStatus(work.Status(filter.Status)) || filter.Priority != "" && !validPriority(work.Priority(filter.Priority)) ||
		!validDate(filter.DueFrom) || !validDate(filter.DueTo) || !validDate(filter.CreatedFrom) || !validDate(filter.CreatedTo) {
		writeError(w, r, http.StatusBadRequest, "invalid_filter", "Task filter is invalid")
		return work.TaskFilter{}, false
	}
	return filter, true
}

func pageLimit(value string) (int, bool) {
	if value == "" {
		return 50, true
	}
	limit, err := strconv.Atoi(value)
	return limit, err == nil && limit > 0 && limit <= 100
}

func validStatus(value work.Status) bool {
	return value == work.Open || value == work.InProgress || value == work.Done || value == work.Cancelled
}
func validPriority(value work.Priority) bool {
	return value == work.Normal || value == work.High || value == work.Urgent
}
func validDate(value string) bool {
	if value == "" {
		return true
	}
	_, err := time.Parse("2006-01-02", value)
	return err == nil
}
func validCategory(value string) bool {
	return value == "Assignment" || value == "TaskChange" || value == "DueReminder"
}
func validMode(value string) bool { return value == "Immediate" || value == "Digest" || value == "Off" }

func writeWorkError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, work.ErrForbidden):
		writeError(w, r, http.StatusForbidden, "forbidden", "Action is not allowed")
	case errors.Is(err, work.ErrConflict):
		writeError(w, r, http.StatusConflict, "conflict", "Action conflicts with current state")
	case errors.Is(err, work.ErrNotFound):
		writeError(w, r, http.StatusNotFound, "not_found", "Resource was not found")
	default:
		writeError(w, r, http.StatusServiceUnavailable, "work_unavailable", "Work data is unavailable")
	}
}
