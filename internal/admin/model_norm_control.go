package admin

import "time"

type NormControlCheck struct {
	SubmissionID string `gorm:"primaryKey;column:submission_id;size:36"`

	ProjectID int64  `gorm:"column:project_id;index;not null"`
	TeamID    *int64 `gorm:"column:team_id;index"`
	StepID    *int64 `gorm:"column:step_id;index"`

	PrimaryFileID *string `gorm:"column:primary_file_id;size:36"`
	FileIDs       []byte  `gorm:"column:file_ids;type:jsonb;not null"`

	DocumentVersion int32 `gorm:"column:document_version;not null;default:1"`

	CheckerID *int64 `gorm:"column:checker_id;index"`

	Status         string `gorm:"column:status;size:50;not null;default:'submitted'"`
	OverallComment string `gorm:"column:overall_comment;type:text"`

	StartedAt  *time.Time `gorm:"column:started_at"`
	ReviewedAt *time.Time `gorm:"column:reviewed_at"`

	CreatedAt time.Time `gorm:"column:created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at"`
}

func (NormControlCheck) TableName() string { return "norm_control_checks" }

type NormControlIssue struct {
	ID           string `gorm:"primaryKey;column:id;type:uuid"`
	SubmissionID string `gorm:"column:submission_id;size:36;index;not null"`

	Category    string `gorm:"column:category;size:100;not null"`
	Severity    string `gorm:"column:severity;size:50;not null"`
	Description string `gorm:"column:description;type:text;not null"`

	PageNumber *int32 `gorm:"column:page_number"`
	LineNumber *int32 `gorm:"column:line_number"`

	IsResolved bool       `gorm:"column:is_resolved;not null;default:false"`
	ResolvedAt *time.Time `gorm:"column:resolved_at"`

	CreatedAt time.Time `gorm:"column:created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at"`
}

func (NormControlIssue) TableName() string { return "norm_control_issues" }

type NormControlHistory struct {
	ID           int64   `gorm:"primaryKey;column:id"`
	ProjectID    int64   `gorm:"column:project_id;index;not null"`
	SubmissionID *string `gorm:"column:submission_id;size:36;index"`

	Action  string `gorm:"column:action;size:100;not null"`
	ActorID *int64 `gorm:"column:actor_id;index"`
	Comment string `gorm:"column:comment;type:text"`

	CreatedAt time.Time `gorm:"column:created_at"`
}

func (NormControlHistory) TableName() string { return "norm_control_history" }

type NormControlChecklist struct {
	ID       string `gorm:"primaryKey;column:id;type:uuid"`
	Name     string `gorm:"column:name;size:255;not null"`
	Category string `gorm:"column:category;size:100;not null"`

	Items []byte `gorm:"column:items;type:jsonb;not null"`

	CreatedBy *int64 `gorm:"column:created_by"`
	IsActive  bool   `gorm:"column:is_active;not null;default:true"`

	CreatedAt time.Time `gorm:"column:created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at"`
}

func (NormControlChecklist) TableName() string { return "norm_control_checklists" }

const (
	NormStatusSubmitted = "submitted"
	NormStatusInReview  = "in_review"
	NormStatusReturned  = "returned"
	NormStatusApproved  = "approved"
)
