package admin

import "time"

type AntiplagCheck struct {
	SubmissionID string `gorm:"primaryKey;column:submission_id;size:36"`

	ProjectID int64  `gorm:"column:project_id;index;not null"`
	TeamID    *int64 `gorm:"column:team_id;index"`
	StepID    *int64 `gorm:"column:step_id;index"`

	PrimaryFileID *string `gorm:"column:primary_file_id;size:36"`
	FileIDs       []byte  `gorm:"column:file_ids;type:jsonb;not null"`

	DocumentVersion int32 `gorm:"column:document_version;not null;default:1"`

	CheckerID *int64 `gorm:"column:checker_id;index"`

	PlagiarismScore *int32 `gorm:"column:plagiarism_score"`
	AIScore         *int32 `gorm:"column:ai_score"`

	Status         string `gorm:"column:status;size:50;not null;default:'submitted'"`
	OverallComment string `gorm:"column:overall_comment;type:text"`

	StartedAt  *time.Time `gorm:"column:started_at"`
	ReviewedAt *time.Time `gorm:"column:reviewed_at"`

	CreatedAt time.Time `gorm:"column:created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at"`
}

func (AntiplagCheck) TableName() string { return "antiplag_checks" }

type AntiplagComment struct {
	ID           string `gorm:"primaryKey;column:id;type:uuid"`
	SubmissionID string `gorm:"column:submission_id;size:36;index;not null"`

	Text       string `gorm:"column:text;type:text;not null"`
	PageNumber *int32 `gorm:"column:page_number"`

	CreatedAt time.Time `gorm:"column:created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at"`
}

func (AntiplagComment) TableName() string { return "antiplag_comments" }

type AntiplagHistory struct {
	ID           int64   `gorm:"primaryKey;column:id"`
	ProjectID    int64   `gorm:"column:project_id;index;not null"`
	SubmissionID *string `gorm:"column:submission_id;size:36;index"`

	Action  string `gorm:"column:action;size:100;not null"`
	ActorID *int64 `gorm:"column:actor_id;index"`
	Comment string `gorm:"column:comment;type:text"`

	CreatedAt time.Time `gorm:"column:created_at"`
}

func (AntiplagHistory) TableName() string { return "antiplag_history" }

const (
	AntiplagStatusSubmitted = "submitted"
	AntiplagStatusInReview  = "in_review"
	AntiplagStatusReturned  = "returned"
	AntiplagStatusApproved  = "approved"
)

const (
	AntiplagDefaultMinPlagiarism int32 = 70
	AntiplagDefaultMaxAI         int32 = 70
)
