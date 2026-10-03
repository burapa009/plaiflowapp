package httpapi

import (
	"errors"
	"net/http"
	"regexp"
	"strings"
	"unicode/utf8"

	"plaiflow/api/internal/job"
	"plaiflow/api/internal/secretary"
	"plaiflow/api/internal/tenant"
)

func (s *server) registerSecretaryRoutes(mux *http.ServeMux) {
	pilot := func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			if !s.config.SecretaryPilotOrganizations[r.PathValue("organization")] {
				writeError(w, r, http.StatusNotFound, "not_found", "Resource was not found")
				return
			}
			next(w, r)
		}
	}
	mux.HandleFunc("GET /v1/o/{organization}/secretary-briefing", pilot(s.secretaryBriefing))
	mux.HandleFunc("POST /v1/o/{organization}/secretary-briefing/refresh", pilot(s.refreshSecretaryBriefing))
	mux.HandleFunc("GET /v1/o/{organization}/secretary-intents/{intent}", pilot(s.secretaryIntent))
	mux.HandleFunc("GET /v1/o/{organization}/tasks/{task}/secretary-source", pilot(s.getSecretaryTaskSource))
	mux.HandleFunc("POST /v1/o/{organization}/tasks/{task}/secretary-source", pilot(s.setSecretaryTaskSource))
	mux.HandleFunc("GET /v1/o/{organization}/secretary-preferences", pilot(s.getSecretaryPreferences))
	mux.HandleFunc("POST /v1/o/{organization}/secretary-preferences", pilot(s.setSecretaryPreferences))
	mux.HandleFunc("GET /v1/o/{organization}/routine-templates", pilot(s.listRoutineTemplates))
	mux.HandleFunc("POST /v1/o/{organization}/routine-templates", pilot(s.saveRoutineTemplate))
	mux.HandleFunc("POST /v1/o/{organization}/routine-templates/{template}", pilot(s.saveRoutineTemplate))
	mux.HandleFunc("GET /v1/o/{organization}/routine-suggestions", pilot(s.listRoutineSuggestions))
	mux.HandleFunc("POST /v1/o/{organization}/routine-suggestions/{suggestion}/{action}", pilot(s.resolveRoutineSuggestion))
}

func (s *server) listRoutineTemplates(w http.ResponseWriter, r *http.Request) {
	session, membership, ok := s.workContext(w, r, false)
	if !ok {
		return
	}
	templates, err := s.config.Secretary.ListRoutineTemplates(r.Context(), session.UserID, membership.OrganizationID)
	if err != nil {
		s.writeSecretarySourceError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"templates": templates})
}

func (s *server) saveRoutineTemplate(w http.ResponseWriter, r *http.Request) {
	session, membership, ok := s.workContext(w, r, true)
	if !ok {
		return
	}
	if membership.Role != tenant.Owner && membership.Role != tenant.Admin {
		writeError(w, r, http.StatusForbidden, "forbidden", "Action is not allowed")
		return
	}
	if err := r.ParseForm(); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_template", "Routine template is invalid")
		return
	}
	command := secretary.TemplateCommand{RoutineTemplate: secretary.RoutineTemplate{
		ID: r.PathValue("template"), Title: strings.TrimSpace(r.FormValue("title")),
		ResponsibleUserID: r.FormValue("responsible_user_id"), Cadence: r.FormValue("cadence"),
		FirstDueOn: r.FormValue("first_due_on"), Active: r.FormValue("active") != "false"},
		UserID: session.UserID, OrganizationID: membership.OrganizationID, Now: s.config.Now().UTC()}
	if utf8.RuneCountInString(command.Title) < 1 || utf8.RuneCountInString(command.Title) > 200 ||
		!secretaryUUID.MatchString(command.ResponsibleUserID) ||
		(command.Cadence != "Weekly" && command.Cadence != "Monthly" && command.Cadence != "Quarterly") ||
		!validDate(command.FirstDueOn) || command.FirstDueOn == "" ||
		(command.ID != "" && !secretaryUUID.MatchString(command.ID)) {
		writeError(w, r, http.StatusUnprocessableEntity, "invalid_template", "Routine template is invalid")
		return
	}
	result, err := s.config.Secretary.SaveRoutineTemplate(r.Context(), command)
	if err != nil {
		s.writeSecretarySourceError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *server) listRoutineSuggestions(w http.ResponseWriter, r *http.Request) {
	session, membership, ok := s.workContext(w, r, false)
	if !ok {
		return
	}
	items, err := s.config.Secretary.ListRoutineSuggestions(r.Context(), session.UserID, membership.OrganizationID, s.config.Now())
	if err != nil {
		s.writeSecretarySourceError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"suggestions": items})
}

func (s *server) resolveRoutineSuggestion(w http.ResponseWriter, r *http.Request) {
	session, membership, ok := s.workContext(w, r, true)
	if !ok {
		return
	}
	action := r.PathValue("action")
	if action != "confirm" && action != "skip" || !secretaryUUID.MatchString(r.PathValue("suggestion")) {
		writeError(w, r, http.StatusNotFound, "not_found", "Suggestion was not found")
		return
	}
	result, err := s.config.Secretary.ResolveRoutineSuggestion(r.Context(), session.UserID, membership.OrganizationID,
		r.PathValue("suggestion"), action, s.config.Now().UTC())
	if err != nil {
		s.writeSecretarySourceError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

var secretaryUUID = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

func (s *server) getSecretaryPreferences(w http.ResponseWriter, r *http.Request) {
	session, membership, ok := s.workContext(w, r, false)
	if !ok {
		return
	}
	prefs, err := s.config.Secretary.GetPreferences(r.Context(), session.UserID, membership.OrganizationID)
	if err != nil {
		s.writeSecretarySourceError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, prefs)
}

func (s *server) setSecretaryPreferences(w http.ResponseWriter, r *http.Request) {
	session, membership, ok := s.workContext(w, r, true)
	if !ok {
		return
	}
	if err := r.ParseForm(); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_preferences", "Preferences are invalid")
		return
	}
	prefs := secretary.Preferences{HiddenCategories: r.Form["hidden_category"], PinnedTaskIDs: r.Form["pinned_task_id"]}
	if len(prefs.HiddenCategories) > 4 || len(prefs.PinnedTaskIDs) > 20 {
		writeError(w, r, http.StatusUnprocessableEntity, "invalid_preferences", "Preferences are invalid")
		return
	}
	seen := map[string]bool{}
	for _, category := range prefs.HiddenCategories {
		if seen[category] || category != "Task" && category != "FollowUp" && category != "MeetingAction" && category != "Routine" {
			writeError(w, r, http.StatusUnprocessableEntity, "invalid_preferences", "Preferences are invalid")
			return
		}
		seen[category] = true
	}
	seen = map[string]bool{}
	for _, id := range prefs.PinnedTaskIDs {
		if seen[id] || !secretaryUUID.MatchString(id) {
			writeError(w, r, http.StatusUnprocessableEntity, "invalid_preferences", "Preferences are invalid")
			return
		}
		seen[id] = true
	}
	updated, err := s.config.Secretary.SetPreferences(r.Context(), session.UserID, membership.OrganizationID, prefs, s.config.Now())
	if err != nil {
		s.writeSecretarySourceError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func (s *server) secretaryIntent(w http.ResponseWriter, r *http.Request) {
	session, membership, ok := s.workContext(w, r, false)
	if !ok {
		return
	}
	intent := r.PathValue("intent")
	switch intent {
	case "today", "why-first", "follow-up", "meeting-actions", "missing-documents":
	default:
		writeError(w, r, http.StatusNotFound, "not_found", "Intent was not found")
		return
	}
	answer, err := s.config.Secretary.Intent(r.Context(), session.UserID, membership.OrganizationID, intent, s.config.Now())
	if err != nil {
		writeError(w, r, http.StatusServiceUnavailable, "briefing_unavailable", "Intent is unavailable")
		return
	}
	writeJSON(w, http.StatusOK, answer)
}

func (s *server) getSecretaryTaskSource(w http.ResponseWriter, r *http.Request) {
	session, membership, ok := s.workContext(w, r, false)
	if !ok {
		return
	}
	source, err := s.config.Secretary.GetTaskSource(r.Context(), session.UserID, membership.OrganizationID, r.PathValue("task"))
	if err != nil {
		s.writeSecretarySourceError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, source)
}

func (s *server) setSecretaryTaskSource(w http.ResponseWriter, r *http.Request) {
	session, membership, ok := s.workContext(w, r, true)
	if !ok {
		return
	}
	if err := r.ParseForm(); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_source", "Task source is invalid")
		return
	}
	command := secretary.SourceCommand{TaskSource: secretary.TaskSource{TaskID: r.PathValue("task"),
		Category: r.FormValue("category"), FollowUpWith: strings.TrimSpace(r.FormValue("follow_up_with")),
		MeetingName: strings.TrimSpace(r.FormValue("meeting_name")), MeetingOn: r.FormValue("meeting_on")},
		UserID: session.UserID, OrganizationID: membership.OrganizationID, Now: s.config.Now().UTC()}
	valid := command.Category == "Task" && command.FollowUpWith == "" && command.MeetingName == "" && command.MeetingOn == "" ||
		command.Category == "FollowUp" && utf8.RuneCountInString(command.FollowUpWith) > 0 && utf8.RuneCountInString(command.FollowUpWith) <= 200 && command.MeetingName == "" && command.MeetingOn == "" ||
		command.Category == "MeetingAction" && utf8.RuneCountInString(command.MeetingName) > 0 && utf8.RuneCountInString(command.MeetingName) <= 200 && validDate(command.MeetingOn) && command.MeetingOn != "" && command.FollowUpWith == ""
	if !valid {
		writeError(w, r, http.StatusUnprocessableEntity, "invalid_source", "Task source is invalid")
		return
	}
	source, err := s.config.Secretary.SetTaskSource(r.Context(), command)
	if err != nil {
		s.writeSecretarySourceError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, source)
}

func (s *server) writeSecretarySourceError(w http.ResponseWriter, r *http.Request, err error) {
	if errors.Is(err, tenant.ErrNotFound) {
		writeError(w, r, http.StatusNotFound, "not_found", "Source was not found")
		return
	}
	if errors.Is(err, tenant.ErrForbidden) {
		writeError(w, r, http.StatusForbidden, "forbidden", "Action is not allowed")
		return
	}
	writeError(w, r, http.StatusServiceUnavailable, "source_unavailable", "Task source is unavailable")
}

func (s *server) refreshSecretaryBriefing(w http.ResponseWriter, r *http.Request) {
	session, membership, ok := s.workContext(w, r, true)
	if !ok {
		return
	}
	briefing, err := s.config.Secretary.Refresh(r.Context(), session.UserID, membership.OrganizationID, s.config.Now())
	if errors.Is(err, secretary.ErrTooSoon) {
		writeError(w, r, http.StatusTooManyRequests, "refresh_too_soon", "Briefing was refreshed recently")
		return
	}
	if errors.Is(err, tenant.ErrNotFound) {
		writeError(w, r, http.StatusNotFound, "not_found", "Resource was not found")
		return
	}
	if err != nil {
		writeError(w, r, http.StatusServiceUnavailable, "briefing_unavailable", "Briefing is unavailable")
		return
	}
	writeJSON(w, http.StatusOK, briefing)
}

func (s *server) generateSecretaryBriefing(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.workerAuthorized(w, r, "jobs:generate"); !ok {
		return
	}
	err := s.config.Secretary.Generate(r.Context(), s.leaseCommand(r))
	if err != nil {
		s.writeJobError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": string(job.Completed)})
}

func (s *server) secretaryBriefing(w http.ResponseWriter, r *http.Request) {
	session, membership, ok := s.workContext(w, r, false)
	if !ok {
		return
	}
	briefing, err := s.config.Secretary.Open(r.Context(), session.UserID, membership.OrganizationID, s.config.Now())
	if err == tenant.ErrNotFound {
		writeError(w, r, http.StatusNotFound, "not_found", "Resource was not found")
		return
	}
	if err != nil {
		writeError(w, r, http.StatusServiceUnavailable, "briefing_unavailable", "Briefing is unavailable")
		return
	}
	writeJSON(w, http.StatusOK, briefing)
}
