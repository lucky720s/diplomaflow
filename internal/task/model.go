package task

import (
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// ==================== Board ====================

// Board - Kanban доска проекта (team_id хранится для join’ов/проверок)
type Board struct {
	ID          int64 `gorm:"primaryKey"`
	TeamID      int64 `gorm:"index;not null"`
	ProjectID   int64 `gorm:"uniqueIndex;not null"`
	Name        string
	Description string         `gorm:"type:text"`
	Settings    datatypes.JSON `gorm:"type:jsonb;default:'{}'"`
	CreatedBy   int64          `gorm:"not null"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   gorm.DeletedAt `gorm:"index"`

	// Relations (not stored, loaded separately)
	Columns []Column `gorm:"-"`
}

func (Board) TableName() string {
	return "task_boards"
}

// BoardSettings - настройки доски (JSON)
type BoardSettings struct {
	DefaultColumn      string   `json:"default_column"`
	AllowCustomColumns bool     `json:"allow_custom_columns"`
	ShowCompleted      bool     `json:"show_completed"`
	Labels             []string `json:"labels"`
}

// ==================== Column ====================

// Column - колонка на доске
type Column struct {
	ID           int64  `gorm:"primaryKey"`
	BoardID      int64  `gorm:"index;not null"`
	Name         string `gorm:"size:100;not null"`
	Slug         string `gorm:"size:50;not null"`
	Description  string `gorm:"type:text"`
	Color        string `gorm:"size:20;default:'#6B7280'"`
	Icon         string `gorm:"size:50"`
	OrderIndex   int32  `gorm:"not null;default:0"`
	WIPLimit     int32  `gorm:"default:0"`
	IsDefault    bool   `gorm:"default:false"`
	IsDoneColumn bool   `gorm:"default:false"`
	CreatedAt    time.Time
	UpdatedAt    time.Time

	// Computed field (not stored)
	TaskCount int32 `gorm:"-"`
}

func (Column) TableName() string {
	return "task_columns"
}

// ==================== Task ====================

// Task - задача
type Task struct {
	ID          int64 `gorm:"primaryKey"`
	BoardID     int64 `gorm:"index;not null"`
	ColumnID    int64 `gorm:"index;not null"`
	Title       string
	Description string `gorm:"type:text"`

	Status   string `gorm:"size:30;not null;default:'todo'"`
	Priority string `gorm:"size:20;not null;default:'medium'"`

	AssigneeID *int64 `gorm:"index"`
	CreatedBy  int64  `gorm:"index;not null"`

	DueDate     *time.Time `gorm:"type:date"`
	DueTime     *string    `gorm:"type:time"`
	StartedAt   *time.Time
	CompletedAt *time.Time

	EstimatedMinutes int32          `gorm:"default:0"`
	ActualMinutes    int32          `gorm:"default:0"`
	Position         int32          `gorm:"not null;default:0"`
	WorkflowStepID   *int64         `gorm:"index"`
	Labels           datatypes.JSON `gorm:"type:jsonb;default:'[]'"`
	CustomFields     datatypes.JSON `gorm:"type:jsonb;default:'{}'"`

	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`

	// Computed fields (not stored)
	CommentsCount    int32 `gorm:"-"`
	AttachmentsCount int32 `gorm:"-"`
	WatchersCount    int32 `gorm:"-"`
	IsOverdue        bool  `gorm:"-"`
}

func (Task) TableName() string {
	return "tasks"
}

// ==================== Comment ====================

// Comment - комментарий к задаче
type Comment struct {
	ID        int64 `gorm:"primaryKey"`
	TaskID    int64 `gorm:"index;not null"`
	AuthorID  int64 `gorm:"index;not null"`
	Content   string
	Mentions  datatypes.JSON `gorm:"type:jsonb;default:'[]'"`
	EditedAt  *time.Time
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

func (Comment) TableName() string {
	return "task_comments"
}

// UserMention - упоминание пользователя в комментарии
type UserMention struct {
	UserID   int64  `json:"user_id"`
	Username string `json:"username"`
	Position int32  `json:"position"`
}

// ==================== Attachment ====================

// Attachment - вложение к задаче
type Attachment struct {
	ID         int64  `gorm:"primaryKey"`
	TaskID     int64  `gorm:"index;not null"`
	FileID     string `gorm:"size:36"`
	FileName   string `gorm:"size:255;not null"`
	FileType   string `gorm:"size:100"`
	FileSize   int64  `gorm:"default:0"`
	UploadedBy int64  `gorm:"not null"`
	CreatedAt  time.Time
}

func (Attachment) TableName() string {
	return "task_attachments"
}

// ==================== Activity Log ====================

// ActivityLog - запись в логе активности
type ActivityLog struct {
	ID        int64 `gorm:"primaryKey"`
	TaskID    int64 `gorm:"index;not null"`
	ActorID   int64 `gorm:"index;not null"`
	Action    string
	FieldName string
	OldValue  string
	NewValue  string
	Metadata  datatypes.JSON `gorm:"type:jsonb;default:'{}'"`
	CreatedAt time.Time
}

func (ActivityLog) TableName() string {
	return "task_activity_log"
}

// ==================== Watcher ====================

// Watcher - наблюдатель задачи
type Watcher struct {
	ID        int64 `gorm:"primaryKey"`
	TaskID    int64 `gorm:"uniqueIndex:idx_task_user;not null"`
	UserID    int64 `gorm:"uniqueIndex:idx_task_user;not null"`
	CreatedAt time.Time
}

func (Watcher) TableName() string {
	return "task_watchers"
}

// ==================== Enums ====================

const (
	TaskStatusTodo       = "todo"
	TaskStatusInProgress = "in_progress"
	TaskStatusReview     = "review"
	TaskStatusDone       = "done"
)

const (
	TaskPriorityLow    = "low"
	TaskPriorityMedium = "medium"
	TaskPriorityHigh   = "high"
	TaskPriorityUrgent = "urgent"
)

const (
	ActionCreated      = "created"
	ActionUpdated      = "updated"
	ActionMoved        = "moved"
	ActionAssigned     = "assigned"
	ActionUnassigned   = "unassigned"
	ActionCommented    = "commented"
	ActionStatusChange = "status_changed"
	ActionDeleted      = "deleted"
)

// ==================== DTOs ====================

type UserPreview struct {
	ID        int64
	FullName  string
	Email     string
	AvatarURL string
}

type BoardStats struct {
	TotalTasks           int32
	CompletedTasks       int32
	OverdueTasks         int32
	TasksWithoutAssignee int32
	TasksByStatus        map[string]int32
	TasksByPriority      map[string]int32
}

type MemberStats struct {
	User            *UserPreview
	AssignedTasks   int32
	CompletedTasks  int32
	OverdueTasks    int32
	InProgressTasks int32
}

type DailyStats struct {
	Date      string
	Created   int32
	Completed int32
	Moved     int32
}

// Default columns for new boards (на случай восстановления/ремонта)
var DefaultColumns = []Column{
	{Name: "К выполнению", Slug: "todo", Color: "#6B7280", OrderIndex: 0, IsDefault: true, IsDoneColumn: false},
	{Name: "В работе", Slug: "in_progress", Color: "#3B82F6", OrderIndex: 1, IsDefault: false, IsDoneColumn: false},
	{Name: "На проверке", Slug: "review", Color: "#F59E0B", OrderIndex: 2, IsDefault: false, IsDoneColumn: false},
	{Name: "Готово", Slug: "done", Color: "#10B981", OrderIndex: 3, IsDefault: false, IsDoneColumn: true},
}

// DeadlineNotificationRun — запись о том, что уведомление по дедлайну уже отправляли (дедуп).
type DeadlineNotificationRun struct {
	ID       int64     `gorm:"primaryKey"`
	DedupKey string    `gorm:"uniqueIndex;not null"`
	Kind     string    `gorm:"index;not null"` // due_today | due_tomorrow | overdue
	TaskID   int64     `gorm:"index;not null"`
	UserID   int64     `gorm:"index;not null"` // assignee
	DueDate  time.Time `gorm:"type:date;not null"`
	SentAt   time.Time `gorm:"not null"`
}

func (DeadlineNotificationRun) TableName() string { return "task_deadline_notification_runs" }
