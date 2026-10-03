package secretary

import (
	"context"
	"errors"
	"time"

	"plaiflow/api/internal/job"
)

var ErrTooSoon = errors.New("briefing was refreshed recently")

// Item is a source-backed, read-only action in a user's briefing.
type Item struct {
	SourceID string `json:"source_id"`
	Title    string `json:"title"`
	DueOn    string `json:"due_on,omitempty"`
	Priority string `json:"priority,omitempty"`
	Reason   string `json:"reason,omitempty"`
	URL      string `json:"url,omitempty"`
	Scope    string `json:"scope,omitempty"`
	Category string `json:"category,omitempty"`
	Pinned   bool   `json:"-"`
}

type Briefing struct {
	Status        string     `json:"status"`
	LocalDay      string     `json:"local_day"`
	GeneratedAt   *time.Time `json:"generated_at,omitempty"`
	PossiblyStale bool       `json:"possibly_stale"`
	Items         []Item     `json:"items"`
	Remaining     int        `json:"remaining_count"`
	JobID         string     `json:"job_id,omitempty"`
}

type IntentAnswer struct {
	Intent      string `json:"intent"`
	State       string `json:"state"`
	Explanation string `json:"explanation,omitempty"`
	Items       []Item `json:"items"`
}

type TaskSource struct {
	TaskID       string `json:"task_id"`
	Category     string `json:"category"`
	FollowUpWith string `json:"follow_up_with,omitempty"`
	MeetingName  string `json:"meeting_name,omitempty"`
	MeetingOn    string `json:"meeting_on,omitempty"`
}

type SourceCommand struct {
	TaskSource
	UserID, OrganizationID string
	Now                    time.Time
}

type Preferences struct {
	HiddenCategories []string `json:"hidden_categories"`
	PinnedTaskIDs    []string `json:"pinned_task_ids"`
}

type RoutineTemplate struct {
	ID                string `json:"id"`
	Title             string `json:"title"`
	ResponsibleUserID string `json:"responsible_user_id"`
	Cadence           string `json:"cadence"`
	FirstDueOn        string `json:"first_due_on"`
	Active            bool   `json:"active"`
}

type TemplateCommand struct {
	RoutineTemplate
	UserID, OrganizationID string
	Now                    time.Time
}

type RoutineSuggestion struct {
	ID                string `json:"id"`
	TemplateID        string `json:"template_id"`
	Title             string `json:"title"`
	ResponsibleUserID string `json:"responsible_user_id"`
	DueOn             string `json:"due_on"`
	Status            string `json:"status"`
	TaskID            string `json:"task_id,omitempty"`
}

type Store interface {
	Open(context.Context, string, string, time.Time) (Briefing, error)
	Refresh(context.Context, string, string, time.Time) (Briefing, error)
	Generate(context.Context, job.LeaseCommand) error
	Intent(context.Context, string, string, string, time.Time) (IntentAnswer, error)
	GetTaskSource(context.Context, string, string, string) (TaskSource, error)
	SetTaskSource(context.Context, SourceCommand) (TaskSource, error)
	GetPreferences(context.Context, string, string) (Preferences, error)
	SetPreferences(context.Context, string, string, Preferences, time.Time) (Preferences, error)
	ListRoutineTemplates(context.Context, string, string) ([]RoutineTemplate, error)
	SaveRoutineTemplate(context.Context, TemplateCommand) (RoutineTemplate, error)
	ListRoutineSuggestions(context.Context, string, string, time.Time) ([]RoutineSuggestion, error)
	ResolveRoutineSuggestion(context.Context, string, string, string, string, time.Time) (RoutineSuggestion, error)
}
