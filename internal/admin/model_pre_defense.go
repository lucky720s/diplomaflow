package admin

import (
	"time"
)

// PreDefenseSubmission - заявка на предзащиту
type PreDefenseSubmission struct {
	ID           string `gorm:"primaryKey;type:varchar(36)"`
	TeamID       int64  `gorm:"not null;index"`
	ProjectID    int64  `gorm:"not null;index"`
	SupervisorID *int64 `gorm:"index"`
	SubmittedBy  int64  `gorm:"not null"`
	Status       string `gorm:"not null;default:'pending'"`

	ScheduledDate   *time.Time `gorm:"type:date"`
	ScheduledTime   string     `gorm:"type:varchar(10)"`
	Location        string     `gorm:"type:varchar(255)"`
	MeetingLink     string     `gorm:"type:varchar(500)"`
	DurationMinutes int32      `gorm:"default:30"`
	Grade           *int32     `gorm:"default:null"`
	GradeComment    string     `gorm:"type:text"`
	GradedBy        *int64
	GradedAt        *time.Time
	Result          string `gorm:"type:varchar(50)"`
	ResultComment   string `gorm:"type:text"`
	SubmittedAt     time.Time
	CompletedAt     *time.Time
	CreatedAt       time.Time
	UpdatedAt       time.Time

	// Runtime-only (не сохраняются в БД, заполняются в service)
	TeamName       string `gorm:"-"`
	ProjectTitle   string `gorm:"-"`
	SubmitterName  string `gorm:"-"`
	SupervisorName string `gorm:"-"`
	GraderName     string `gorm:"-"`

	Commission []PreDefenseCommissionMember `gorm:"-"`
	Documents  []PreDefenseDocument         `gorm:"-"`
}

func (PreDefenseSubmission) TableName() string { return "admin_pre_defense_submissions" }

// PreDefenseCommissionMember - член комиссии предзащиты
type PreDefenseCommissionMember struct {
	ID           int64  `gorm:"primaryKey"`
	SubmissionID string `gorm:"not null;index;type:varchar(36)"`
	UserID       int64  `gorm:"not null"`
	Role         string `gorm:"not null;type:varchar(50)"`
	Position     string `gorm:"type:varchar(255)"`

	IsPresent       bool   `gorm:"default:false"`
	IndividualGrade int32  `gorm:"default:0"`
	Comment         string `gorm:"type:text"`
	CreatedAt       time.Time
}

func (PreDefenseCommissionMember) TableName() string { return "admin_pre_defense_commission" }

// PreDefenseDocument - документы предзащиты
type PreDefenseDocument struct {
	ID           string `gorm:"primaryKey;type:varchar(36)"`
	SubmissionID string `gorm:"not null;index;type:varchar(36)"`
	FileName     string `gorm:"type:varchar(255)"`
	FileType     string `gorm:"type:varchar(50)"`
	UploadedBy   *int64
	UploadedAt   time.Time
	CreatedAt    time.Time
}

func (PreDefenseDocument) TableName() string { return "admin_pre_defense_documents" }

type PreDefenseHistory struct {
	ID           int64  `gorm:"primaryKey"`
	SubmissionID string `gorm:"not null;index;type:varchar(36)"`
	Action       string `gorm:"not null;type:varchar(50)"`
	ActorID      int64  `gorm:"not null"`
	OldValue     string `gorm:"type:varchar(255)"`
	NewValue     string `gorm:"type:varchar(255)"`
	Comment      string `gorm:"type:text"`
	CreatedAt    time.Time
}

func (PreDefenseHistory) TableName() string { return "admin_pre_defense_history" }
