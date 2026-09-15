package work

import (
	"context"
	"encoding/csv"
	"errors"
	"io"
	"strconv"
	"strings"
	"time"
	"unicode"

	"plaiflow/api/internal/tenant"
)

var (
	ErrNotFound          = errors.New("work resource not found")
	ErrForbidden         = errors.New("work action forbidden")
	ErrConflict          = errors.New("work state conflict")
	ErrPermanentDelivery = errors.New("permanent delivery failure")
)

type Status string

const (
	Open       Status = "Open"
	InProgress Status = "InProgress"
	Done       Status = "Done"
	Cancelled  Status = "Cancelled"
)

type Priority string

const (
	Normal Priority = "Normal"
	High   Priority = "High"
	Urgent Priority = "Urgent"
)

type Task struct {
	ID, OrganizationID, Title, Description, CreatorUserID string
	AssigneeUserID                                        string
	AssigneeName                                          string
	WatcherNames                                          []string
	Status                                                Status
	Priority                                              Priority
	DueOn                                                 string
	IsOverdue                                             bool
	CreatedAt, UpdatedAt, StatusChangedAt                 time.Time
	CompletedAt                                           *time.Time
}

type CreateTask struct {
	ID, OrganizationID, ActorUserID, Title, Description, AssigneeUserID, DueOn string
	Priority                                                                   Priority
	Now                                                                        time.Time
}

type UpdateTask struct {
	ID, OrganizationID, ActorUserID, Title, Description, AssigneeUserID, DueOn string
	Status                                                                     Status
	Priority                                                                   Priority
	TitleSet, DescriptionSet, AssigneeSet, DueOnSet, StatusSet, PrioritySet    bool
	Now                                                                        time.Time
}

type TaskFilter struct {
	Status, Priority                       string
	AssigneeUserID, CreatorUserID          string
	DueFrom, DueTo, CreatedFrom, CreatedTo string
	Overdue                                *bool
	Cursor                                 string
	Limit                                  int
}

type TaskPage struct {
	Tasks      []Task `json:"tasks"`
	NextCursor string `json:"next_cursor,omitempty"`
}

type Notification struct {
	ID, OrganizationID, RecipientUserID, Category, Title, DeepLink string
	CreatedAt                                                      time.Time
	ReadAt                                                         *time.Time
}

type NotificationPage struct {
	Notifications []Notification `json:"notifications"`
	NextCursor    string         `json:"next_cursor,omitempty"`
}

type LINEDelivery struct {
	ID, OrganizationID, To, OrganizationName, DeepLink string
	AttemptCount                                       int
}

type Preference struct {
	OrganizationID, UserID, Category, WebMode, LineMode string
	UpdatedAt                                           time.Time
}

type Dashboard struct {
	Scope        string      `json:"scope"`
	Role         tenant.Role `json:"role"`
	MyFocus      []Task      `json:"my_focus"`
	Watching     []Task      `json:"watching"`
	TeamQueue    []Task      `json:"team_queue,omitempty"`
	OpenCount    int         `json:"open_count"`
	OverdueCount int         `json:"overdue_count"`
	UnreadCount  int         `json:"unread_count"`
}

type AssistantSummary struct {
	Scope       string    `json:"scope"`
	GeneratedAt time.Time `json:"generated_at"`
	Bullets     []string  `json:"bullets"`
}

func Summarize(d Dashboard, now time.Time) AssistantSummary {
	bullets := []string{
		"งานที่ต้องโฟกัสของคุณ " + strconv.Itoa(len(d.MyFocus)) + " รายการ",
		"งานเกินกำหนด " + strconv.Itoa(d.OverdueCount) + " รายการ",
		"การแจ้งเตือนที่ยังไม่อ่าน " + strconv.Itoa(d.UnreadCount) + " รายการ",
	}
	if d.Role == tenant.Owner || d.Role == tenant.Admin {
		bullets = append(bullets, "งานในคิวทีม "+strconv.Itoa(len(d.TeamQueue))+" รายการ")
	}
	return AssistantSummary{Scope: d.Scope, GeneratedAt: now.UTC(), Bullets: bullets}
}

type ExportRequest struct {
	ID, OrganizationID, ActorUserID, RequestID string
	Filters                                    TaskFilter
	Now                                        time.Time
}

type ExportJob struct {
	ID, OrganizationID, RequesterUserID, Status, Mode, FailureCode string
	RowCount, ByteCount                                            int64
	RequestedAt                                                    time.Time
	StartedAt, CompletedAt, ExpiresAt                              *time.Time
}

type ExportPage struct {
	Exports    []ExportJob `json:"exports"`
	NextCursor string      `json:"next_cursor,omitempty"`
}

type Capability string

const (
	CreateTasks      Capability = "task.create"
	DeliverLINE      Capability = "notification.line"
	UseAssistant     Capability = "assistant.summary"
	ExportTasks      Capability = "task.export.csv"
	ExportTasksAsync Capability = "task.export.csv.background"
)

type EntitlementDecision struct {
	Allowed   bool       `json:"allowed"`
	Reason    string     `json:"reason"`
	Limit     int64      `json:"limit,omitempty"`
	Used      int64      `json:"used,omitempty"`
	Remaining int64      `json:"remaining,omitempty"`
	ResetAt   *time.Time `json:"reset_at,omitempty"`
}

type Gate interface {
	Check(context.Context, string, Capability, int64) (EntitlementDecision, error)
	Record(context.Context, string, Capability, int64, string) error
}

type UnlimitedGate struct{}

func (UnlimitedGate) Check(context.Context, string, Capability, int64) (EntitlementDecision, error) {
	return EntitlementDecision{Allowed: true, Reason: "unlimited"}, nil
}
func (UnlimitedGate) Record(context.Context, string, Capability, int64, string) error { return nil }

type Store interface {
	CreateTask(context.Context, CreateTask) (Task, error)
	ListTasks(context.Context, string, string, TaskFilter) (TaskPage, error)
	GetTask(context.Context, string, string, string) (Task, error)
	UpdateTask(context.Context, UpdateTask) (Task, error)
	SetWatcher(context.Context, string, string, string, string, bool, time.Time) error
	ListNotifications(context.Context, string, string, string, int) (NotificationPage, error)
	MarkNotificationRead(context.Context, string, string, string, time.Time) error
	MarkAllNotificationsRead(context.Context, string, string, time.Time) error
	ListPreferences(context.Context, string, string) ([]Preference, error)
	SetPreference(context.Context, Preference) error
	Dashboard(context.Context, string, string) (Dashboard, error)
	CountExport(context.Context, string, string, TaskFilter) (int64, error)
	ExportRows(context.Context, string, string, TaskFilter, func(ExportRow) error) error
	QueueExport(context.Context, ExportRequest, int64) (ExportJob, error)
	ListExports(context.Context, string, string, string, int) (ExportPage, error)
	AuditExport(context.Context, ExportRequest, string, int64, int64, string) error
}

var exportHeader = []string{
	"organization_id", "organization_name", "task_id", "title", "description", "status", "priority", "due_date",
	"is_overdue", "creator_display_name", "assignee_display_name", "watcher_display_names", "created_at", "updated_at",
	"status_changed_at", "completed_at",
}

type ExportRow struct {
	OrganizationID, OrganizationName, TaskID, Title, Description string
	Status                                                       Status
	Priority                                                     Priority
	DueDate                                                      string
	IsOverdue                                                    bool
	CreatorName, AssigneeName, WatcherNames                      string
	CreatedAt, UpdatedAt                                         time.Time
	StatusChangedAt, CompletedAt                                 *time.Time
}

func WriteCSV(output io.Writer, rows []ExportRow) error {
	w, err := NewCSVWriter(output)
	if err != nil {
		return err
	}
	for _, row := range rows {
		if err := w.Write(row); err != nil {
			return err
		}
	}
	return w.Close()
}

type CSVWriter struct{ writer *csv.Writer }

func NewCSVWriter(output io.Writer) (*CSVWriter, error) {
	if _, err := io.WriteString(output, "\ufeff"); err != nil {
		return nil, err
	}
	w := csv.NewWriter(output)
	w.UseCRLF = true
	if err := w.Write(exportHeader); err != nil {
		return nil, err
	}
	return &CSVWriter{writer: w}, nil
}

func (w *CSVWriter) Write(row ExportRow) error {
	return w.writer.Write([]string{
		row.OrganizationID, safeCSVCell(row.OrganizationName), row.TaskID, safeCSVCell(row.Title), safeCSVCell(row.Description),
		string(row.Status), string(row.Priority), row.DueDate, strconv.FormatBool(row.IsOverdue), safeCSVCell(row.CreatorName),
		safeCSVCell(row.AssigneeName), safeCSVCell(row.WatcherNames), timestamp(row.CreatedAt), timestamp(row.UpdatedAt),
		nullableTimestamp(row.StatusChangedAt), nullableTimestamp(row.CompletedAt),
	})
}

func (w *CSVWriter) Close() error {
	w.writer.Flush()
	return w.writer.Error()
}

func safeCSVCell(value string) string {
	trimmed := strings.TrimLeftFunc(value, unicode.IsSpace)
	if value != "" && (value[0] == '\t' || value[0] == '\r' || value[0] == '\n') ||
		trimmed != "" && strings.ContainsRune("=+-@", rune(trimmed[0])) {
		return "'" + value
	}
	return value
}

func timestamp(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC3339)
}

func nullableTimestamp(value *time.Time) string {
	if value == nil {
		return ""
	}
	return timestamp(*value)
}
