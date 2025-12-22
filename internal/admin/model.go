package admin

import (
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// Grade - оценка за этап дипломного проекта
type Grade struct {
	ID          int64  `gorm:"primaryKey"`
	ProjectID   int64  `gorm:"index;not null"`
	StepID      int64  `gorm:"index;not null"`
	TeamID      int64  `gorm:"index"`
	Grade       int32  `gorm:"not null"` // 0-100
	LetterGrade string `gorm:"size:2"`   // A, B, C, D, F
	Comment     string `gorm:"type:text"`
	GradedBy    int64  `gorm:"not null"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   gorm.DeletedAt `gorm:"index"`
}

// GradeHistory - история изменений оценок
type GradeHistory struct {
	ID        int64 `gorm:"primaryKey"`
	GradeID   int64 `gorm:"index;not null"`
	ProjectID int64 `gorm:"index;not null"`
	StepID    int64 `gorm:"index"`
	OldGrade  int32
	NewGrade  int32  `gorm:"not null"`
	ChangedBy int64  `gorm:"not null"`
	Reason    string `gorm:"type:text"`
	CreatedAt time.Time
}

// Submission - заявление/отчет на проверку
type Submission struct {
	ID            string         `gorm:"primaryKey;size:36"`
	ProjectID     int64          `gorm:"index;not null"`
	TeamID        int64          `gorm:"index"`
	StepID        int64          `gorm:"index;not null"`
	SubmittedBy   int64          `gorm:"not null"`
	Status        string         `gorm:"size:20;default:'pending'"` // pending, approved, rejected, revision_requested
	Data          datatypes.JSON `gorm:"type:jsonb"`
	Files         datatypes.JSON `gorm:"type:jsonb"` // array of file IDs
	ReviewerID    *int64
	ReviewComment string `gorm:"type:text"`
	ReviewedAt    *time.Time
	CreatedAt     time.Time
	UpdatedAt     time.Time
	DeletedAt     gorm.DeletedAt `gorm:"index"`
}

// SubmissionReview - история проверок submission
type SubmissionReview struct {
	ID           int64  `gorm:"primaryKey"`
	SubmissionID string `gorm:"index;not null;size:36"`
	ReviewerID   int64  `gorm:"not null"`
	Action       string `gorm:"size:30;not null"` // submitted, approved, rejected, revision_requested
	Comment      string `gorm:"type:text"`
	Grade        *int32
	CreatedAt    time.Time
}

// SupervisorAssignment - назначение руководителя команде
type SupervisorAssignment struct {
	ID           int64 `gorm:"primaryKey"`
	TeamID       int64 `gorm:"uniqueIndex;not null"`
	SupervisorID int64 `gorm:"index;not null"`
	AssignedBy   int64 `gorm:"not null"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// AdminActivity - лог активности для audit trail
type AdminActivity struct {
	ID           int64          `gorm:"primaryKey"`
	ActivityType string         `gorm:"size:50;not null;index"`
	Description  string         `gorm:"type:text"`
	ActorID      int64          `gorm:"index;not null"`
	TargetID     int64          `gorm:"index"`
	TargetType   string         `gorm:"size:50"`
	Metadata     datatypes.JSON `gorm:"type:jsonb"`
	CreatedAt    time.Time
}

// TableName overrides
func (Grade) TableName() string                { return "admin_grades" }
func (GradeHistory) TableName() string         { return "admin_grade_history" }
func (Submission) TableName() string           { return "admin_submissions" }
func (SubmissionReview) TableName() string     { return "admin_submission_reviews" }
func (SupervisorAssignment) TableName() string { return "admin_supervisor_assignments" }
func (AdminActivity) TableName() string        { return "admin_activities" }

// Helper functions
func CalculateLetterGrade(grade int32) string {
	switch {
	case grade >= 90:
		return "A"
	case grade >= 80:
		return "B"
	case grade >= 70:
		return "C"
	case grade >= 60:
		return "D"
	default:
		return "F"
	}
}

// SubmissionStatus constants
const (
	SubmissionStatusPending           = "pending"
	SubmissionStatusApproved          = "approved"
	SubmissionStatusRejected          = "rejected"
	SubmissionStatusRevisionRequested = "revision_requested"
)

// ActivityType constants
const (
	ActivityTypeSubmission       = "SUBMISSION"
	ActivityTypeGrade            = "GRADE"
	ActivityTypeTeamUpdate       = "TEAM_UPDATE"
	ActivityTypeTeamDelete       = "TEAM_DELETE"
	ActivityTypeSupervisorAssign = "SUPERVISOR_ASSIGN"
	ActivityTypeReview           = "REVIEW"
)
