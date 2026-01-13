package project

import (
	"time"

	"gorm.io/datatypes"
)

type Project struct {
	ID          int64  `gorm:"primaryKey;column:id"`
	Title       string `gorm:"column:title;not null"`
	Description string `gorm:"column:description"`

	StudentID    int64  `gorm:"column:student_id;not null;index"`
	UniversityID int64  `gorm:"column:university_id;not null"`
	DepartmentID int64  `gorm:"column:department_id;not null;index"`
	TeamID       *int64 `gorm:"column:team_id"`

	WorkflowID      int64  `gorm:"column:workflow_id;not null;index"`
	WorkflowVersion int32  `gorm:"column:workflow_version;default:1"`
	WorkflowName    string `gorm:"column:workflow_name"`

	CurrentStateID   int64  `gorm:"column:current_state_id;index"`
	CurrentStateName string `gorm:"column:current_state_name"`

	Status string         `gorm:"column:status;default:'active';index"`
	Data   datatypes.JSON `gorm:"column:data;type:jsonb;default:'{}'"`

	DeadlineAt        *time.Time `gorm:"column:deadline_at;index"`
	DeadlineProcessed bool       `gorm:"column:deadline_processed;default:false"`

	CreatedAt time.Time `gorm:"column:created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at"`

	History []StateHistory `gorm:"foreignKey:ProjectID"`
}

func (Project) TableName() string { return "projects" }

type StateHistory struct {
	ID int64 `gorm:"primaryKey;column:id"`

	ProjectID int64  `gorm:"column:project_id;index;not null"`
	EventName string `gorm:"column:event_name"`
	Status    string `gorm:"column:status"`
	ChangedBy *int64 `gorm:"column:changed_by"`
	Comment   string `gorm:"column:comment"`

	FromStateID   *int64 `gorm:"column:from_state_id"`
	ToStateID     *int64 `gorm:"column:to_state_id"`
	FromStateName string `gorm:"column:from_state_name"`
	ToStateName   string `gorm:"column:to_state_name"`

	Metadata  datatypes.JSON `gorm:"column:metadata;type:jsonb;default:'{}'"`
	CreatedAt time.Time      `gorm:"column:created_at"`
}

func (StateHistory) TableName() string { return "state_histories" }

type OutboxEvent struct {
	ID int64 `gorm:"primaryKey;column:id"`

	Topic     string         `gorm:"column:topic;not null"`
	EventType string         `gorm:"column:event_type;not null"`
	Payload   datatypes.JSON `gorm:"column:payload;type:jsonb;not null"`
	Status    string         `gorm:"column:status;default:'pending';index"`

	CreatedAt   time.Time  `gorm:"column:created_at"`
	ProcessedAt *time.Time `gorm:"column:processed_at"`
}

func (OutboxEvent) TableName() string { return "outbox_events" }
