package admin

import (
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// Grade - оценка за этап дипломного проекта
type Grade struct {
	ID        int64  `gorm:"primaryKey"`
	ProjectID int64  `gorm:"index;not null"`
	StepID    int64  `gorm:"index;not null"`
	TeamID    int64  `gorm:"index"`
	Grade     int32  `gorm:"not null"` // 0-100 баллов
	Comment   string `gorm:"type:text"`
	GradedBy  int64  `gorm:"not null"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
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

// TopicRegistration - заявление на регистрацию дипломной темы
type TopicRegistration struct {
	ID               string `gorm:"primaryKey;size:36"`
	TeamID           int64  `gorm:"index;not null"`
	ProjectID        int64  `gorm:"index"`
	ProposedTopic    string `gorm:"not null"`
	TopicDescription string `gorm:"type:text"`
	SupervisorID     int64  `gorm:"index;not null"`
	SubmittedBy      int64  `gorm:"not null"`
	Status           string `gorm:"size:30;default:'pending'"` // pending, approved, rejected, revision_requested
	RejectionReason  string `gorm:"type:text"`
	Comment          string `gorm:"type:text"`
	ReviewerID       *int64
	ReviewedAt       *time.Time
	CreatedAt        time.Time
	UpdatedAt        time.Time
	DeletedAt        gorm.DeletedAt `gorm:"index"`
}

// TopicRegistrationReview - история проверок заявления на тему
type TopicRegistrationReview struct {
	ID             int64  `gorm:"primaryKey"`
	RegistrationID string `gorm:"index;not null;size:36"`
	ReviewerID     int64  `gorm:"not null"`
	Action         string `gorm:"size:30;not null"` // submitted, approved, rejected, revision_requested
	Comment        string `gorm:"type:text"`
	CreatedAt      time.Time
}

// Submission - заявление/отчет на проверку (для документов и отчетов)
type Submission struct {
	ID            string         `gorm:"primaryKey;size:36"`
	ProjectID     int64          `gorm:"index;not null"`
	TeamID        int64          `gorm:"index"`
	StepID        int64          `gorm:"index;not null"`
	SubmittedBy   int64          `gorm:"not null"`
	Status        string         `gorm:"size:20;default:'pending'"`
	Data          datatypes.JSON `gorm:"type:jsonb"`
	Files         datatypes.JSON `gorm:"type:jsonb"`
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
	Action       string `gorm:"size:30;not null"`
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

// StudentFullInfo - полная информация о студенте
type StudentFullInfo struct {
	ID           int64
	Email        string
	FirstName    string
	LastName     string
	Role         string
	UniversityID int64
	DepartmentID int64
	TeamID       int64
	TeamName     string
	ProjectID    int64
	ProjectTitle string
	CurrentStep  string
	CreatedAt    time.Time
}

// TeamFullDetails - полная информация о команде для админки
type TeamFullDetails struct {
	ID           int64
	Name         string
	ProjectID    int64
	ProjectTitle string
	CurrentStep  string
	Status       string
	SupervisorID int64
	Members      []*TeamMemberDetails
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// TeamMemberDetails - детальная информация об участнике команды
type TeamMemberDetails struct {
	UserID   int64
	FullName string
	Email    string
	Role     string
	JoinedAt time.Time
}

// TeamAdminUpdateData - данные для обновления команды администратором
type TeamAdminUpdateData struct {
	Name         *string
	SupervisorID *int64
	MemberIDs    []int64
}

// GradeHistoryFull - расширенная история оценок
type GradeHistoryFull struct {
	ID          int64
	ProjectID   int64
	StepID      int64
	StepName    string
	OldGrade    int32
	NewGrade    int32
	ChangedBy   int64
	ChangerName string
	Reason      string
	ChangedAt   time.Time
}

// SupervisorBasicInfo - базовая информация о супервайзере
type SupervisorBasicInfo struct {
	ID       int64
	FullName string
	Email    string
	Position string
}

// SupervisorRequest - заявка команды на научного руководителя
type SupervisorRequest struct {
	ID            string `gorm:"primaryKey;size:36"`
	TeamID        int64  `gorm:"index;not null"`
	ProjectID     int64  `gorm:"index"`
	SupervisorID  int64  `gorm:"index;not null"`
	RequestedBy   int64  `gorm:"not null"`
	Status        string `gorm:"size:30;default:'pending'"` // pending, approved, rejected, cancelled
	Message       string `gorm:"type:text"`
	ProposedTopic string `gorm:"size:500"`
	RejectReason  string `gorm:"type:text"`
	RespondedAt   *time.Time
	ExpiresAt     *time.Time
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// SupervisorRequestHistory - история изменений запроса
type SupervisorRequestHistory struct {
	ID        int64  `gorm:"primaryKey"`
	RequestID string `gorm:"index;not null;size:36"`
	Action    string `gorm:"size:30;not null"` // created, approved, rejected, cancelled
	ActorID   int64  `gorm:"not null"`
	Comment   string `gorm:"type:text"`
	CreatedAt time.Time
}

// SupervisorRequestWithDetails - запрос с дополнительной информацией
type SupervisorRequestWithDetails struct {
	SupervisorRequest
	TeamName        string
	SupervisorName  string
	SupervisorEmail string
	RequesterName   string
}

// TableName overrides
func (Grade) TableName() string                    { return "admin_grades" }
func (GradeHistory) TableName() string             { return "admin_grade_history" }
func (TopicRegistration) TableName() string        { return "admin_topic_registrations" }
func (TopicRegistrationReview) TableName() string  { return "admin_topic_registration_reviews" }
func (Submission) TableName() string               { return "admin_submissions" }
func (SubmissionReview) TableName() string         { return "admin_submission_reviews" }
func (SupervisorAssignment) TableName() string     { return "admin_supervisor_assignments" }
func (AdminActivity) TableName() string            { return "admin_activities" }
func (SupervisorRequest) TableName() string        { return "admin_supervisor_requests" }
func (SupervisorRequestHistory) TableName() string { return "admin_supervisor_request_history" }

// SubmissionStatus constants
const (
	StatusPending           = "pending"
	StatusApproved          = "approved"
	StatusRejected          = "rejected"
	StatusRevisionRequested = "revision_requested"
)

// ActivityType constants
const (
	ActivityTypeSubmission        = "SUBMISSION"
	ActivityTypeGrade             = "GRADE"
	ActivityTypeTeamUpdate        = "TEAM_UPDATE"
	ActivityTypeTeamDelete        = "TEAM_DELETE"
	ActivityTypeSupervisorAssign  = "SUPERVISOR_ASSIGN"
	ActivityTypeReview            = "REVIEW"
	ActivityTypeTopicRegistration = "TOPIC_REGISTRATION"
	ActivityTypeTopicApproval     = "TOPIC_APPROVAL"
)

// SupervisorRequest Status constants
const (
	SupervisorRequestStatusPending   = "pending"
	SupervisorRequestStatusApproved  = "approved"
	SupervisorRequestStatusRejected  = "rejected"
	SupervisorRequestStatusCancelled = "cancelled"
)

// SupervisorRequest Action constants
const (
	SupervisorRequestActionCreated   = "created"
	SupervisorRequestActionApproved  = "approved"
	SupervisorRequestActionRejected  = "rejected"
	SupervisorRequestActionCancelled = "cancelled"
)

// ActivityType для SupervisorRequest
const (
	ActivityTypeSupervisorRequest         = "SUPERVISOR_REQUEST"
	ActivityTypeSupervisorRequestApproved = "SUPERVISOR_REQUEST_APPROVED"
	ActivityTypeSupervisorRequestRejected = "SUPERVISOR_REQUEST_REJECTED"
)
