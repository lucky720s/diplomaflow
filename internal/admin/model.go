package admin

import (
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// ==================== CORE (Grades / Submissions / Topic) ====================

// Grade - оценка за этап (state) дипломного проекта.
// В БД колонка называется state_id (см. миграцию 000004_admin.up.sql),
// но в коде/прото исторически используется step_id.
// Поэтому: StepID <-> column:state_id
type Grade struct {
	ID        int64 `gorm:"primaryKey"`
	ProjectID int64 `gorm:"index;not null"`
	StepID    int64 `gorm:"column:state_id;index"` // maps to admin_grades.state_id
	TeamID    int64 `gorm:"index"`

	Grade    int32  `gorm:"not null"` // 0-100
	Comment  string `gorm:"type:text"`
	GradedBy int64  `gorm:"not null"`

	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

type GradeHistory struct {
	ID        int64 `gorm:"primaryKey"`
	GradeID   int64 `gorm:"index;not null"`
	ProjectID int64 `gorm:"index;not null"`

	StepID int64 `gorm:"column:state_id;index"` // maps to admin_grade_history.state_id

	OldGrade  int32
	NewGrade  int32  `gorm:"not null"`
	ChangedBy int64  `gorm:"not null"`
	Reason    string `gorm:"type:text"`

	CreatedAt time.Time
}

type TopicRegistration struct {
	ID string `gorm:"primaryKey;size:36"`

	// В БД team_id после 000017 стал optional, поэтому not null не ставим
	TeamID    int64 `gorm:"index"`
	ProjectID int64 `gorm:"index;not null"`

	ProposedTopic    string `gorm:"not null"`
	TopicDescription string `gorm:"type:text"`

	SupervisorID int64 `gorm:"index;not null"`
	SubmittedBy  int64 `gorm:"not null"`

	Status          string `gorm:"size:30;default:'pending'"` // pending, approved, rejected, revision_requested
	RejectionReason string `gorm:"type:text"`
	Comment         string `gorm:"type:text"`

	ReviewerID *int64
	ReviewedAt *time.Time

	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

type TopicRegistrationReview struct {
	ID             int64  `gorm:"primaryKey"`
	RegistrationID string `gorm:"index;not null;size:36"`

	ReviewerID int64  `gorm:"not null"`
	Action     string `gorm:"size:30;not null"`
	Comment    string `gorm:"type:text"`

	CreatedAt time.Time
}

type Submission struct {
	ID string `gorm:"primaryKey;size:36"`

	ProjectID int64 `gorm:"index;not null"`
	TeamID    int64 `gorm:"index"`

	StepID int64 `gorm:"column:state_id;index"` // maps to admin_submissions.state_id

	SubmittedBy int64  `gorm:"not null"`
	Status      string `gorm:"size:20;default:'pending'"`

	Data  datatypes.JSON `gorm:"type:jsonb"`
	Files datatypes.JSON `gorm:"type:jsonb"`

	ReviewerID    *int64
	ReviewComment string `gorm:"type:text"`
	ReviewedAt    *time.Time

	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

type SubmissionReview struct {
	ID           int64  `gorm:"primaryKey"`
	SubmissionID string `gorm:"index;not null;size:36"`

	ReviewerID int64  `gorm:"not null"`
	Action     string `gorm:"size:30;not null"`
	Comment    string `gorm:"type:text"`
	Grade      *int32

	CreatedAt time.Time
}

type SupervisorAssignment struct {
	ID int64 `gorm:"primaryKey"`

	TeamID       int64 `gorm:"uniqueIndex;not null"`
	SupervisorID int64 `gorm:"index;not null"`
	AssignedBy   int64 `gorm:"not null"`

	CreatedAt time.Time
	UpdatedAt time.Time
}

type AdminActivity struct {
	ID           int64          `gorm:"primaryKey"`
	ActivityType string         `gorm:"size:50;not null;index"`
	Description  string         `gorm:"type:text"`
	ActorID      int64          `gorm:"index;not null"`
	TargetID     int64          `gorm:"index"`
	TargetType   string         `gorm:"size:50"`
	Metadata     datatypes.JSON `gorm:"type:jsonb"`

	CreatedAt time.Time
}

// ==================== DTOs for admin panel ====================

type StudentFullInfo struct {
	ID           int64
	Email        string
	FirstName    string
	LastName     string
	Role         string
	UniversityID int64
	DepartmentID int64

	TeamID   int64
	TeamName string

	ProjectID    int64
	ProjectTitle string
	CurrentStep  string

	CreatedAt time.Time
}

type TeamFullDetails struct {
	ID   int64
	Name string

	ProjectID    int64
	ProjectTitle string
	CurrentStep  string
	Status       string

	SupervisorID int64

	Members []*TeamMemberDetails

	CreatedAt time.Time
	UpdatedAt time.Time
}

type TeamMemberDetails struct {
	UserID   int64
	FullName string
	Email    string
	Role     string
	JoinedAt time.Time
}

type TeamAdminUpdateData struct {
	Name         *string
	SupervisorID *int64
	MemberIDs    []int64
	UpdatedBy    int64
}

type GradeHistoryFull struct {
	ID        int64
	ProjectID int64

	StepID   int64
	StepName string

	OldGrade    int32
	NewGrade    int32
	ChangedBy   int64
	ChangerName string
	Reason      string
	ChangedAt   time.Time
}

type SupervisorBasicInfo struct {
	ID       int64
	FullName string
	Email    string
	Position string
}

// ==================== Supervisor Request ====================

// SupervisorRequest - заявка команды на научного руководителя.
// Важно: project_id может быть NULL (team-first), проект создаётся при approve.
type SupervisorRequest struct {
	ID string `gorm:"primaryKey;size:36"`

	TeamID int64 `gorm:"index"` // optional by schema after 000017 (we still set it in code)

	ProjectID *int64 `gorm:"column:project_id;index"` // NULL allowed by 000020

	SupervisorID int64 `gorm:"index;not null"`
	RequestedBy  int64 `gorm:"not null"`

	Status        string `gorm:"size:30;default:'pending'"` // pending, approved, rejected, cancelled
	Message       string `gorm:"type:text"`
	ProposedTopic string `gorm:"size:500"`
	RejectReason  string `gorm:"type:text"`

	RespondedAt *time.Time
	ExpiresAt   *time.Time

	CreatedAt time.Time
	UpdatedAt time.Time
}

type SupervisorRequestHistory struct {
	ID int64 `gorm:"primaryKey"`

	RequestID string `gorm:"index;not null;size:36"`
	Action    string `gorm:"size:30;not null"` // created, approved, rejected, cancelled
	ActorID   int64  `gorm:"not null"`
	Comment   string `gorm:"type:text"`

	CreatedAt time.Time
}

type SupervisorRequestWithDetails struct {
	SupervisorRequest

	TeamName        string
	SupervisorName  string
	SupervisorEmail string
	RequesterName   string
}

// ==================== PRE-DEFENSE (align to migration 000010) ====================
// Сейчас pre-defense в handler/service не реализован полностью,
// но модели должны соответствовать БД, чтобы будущая реализация не ломалась.

type PreDefenseSubmission struct {
	ID string `gorm:"primaryKey;size:36"` // UUID as string

	TeamID    int64 `gorm:"index;not null"`
	ProjectID int64 `gorm:"index;not null"`

	SupervisorID *int64 `gorm:"index"`
	SubmittedBy  int64  `gorm:"not null"`

	Status string `gorm:"size:30;default:'pending';index"`

	ScheduledDate *time.Time `gorm:"index"`
	ScheduledTime string     `gorm:"size:10"`
	Location      string     `gorm:"size:255"`
	MeetingLink   string     `gorm:"size:500"`

	DurationMinutes int32 `gorm:"default:30"`

	Grade        *int32 `gorm:""`
	GradeComment string `gorm:"type:text"`
	GradedBy     *int64
	GradedAt     *time.Time

	Result        string `gorm:"size:30"`
	ResultComment string `gorm:"type:text"`

	// В миграции recommendations TEXT[].
	// Без дополнительной обвязки будем хранить как JSON в будущей миграции/реализации.
	// Сейчас не используем, поэтому не мапим на колонку.
	Recommendations datatypes.JSON `gorm:"-"`

	SubmittedAt time.Time
	CompletedAt *time.Time

	CreatedAt time.Time
	UpdatedAt time.Time

	// Relations (соответствуют миграции)
	Commission []PreDefenseCommissionMember `gorm:"foreignKey:SubmissionID;references:ID"`
	Documents  []PreDefenseDocument         `gorm:"foreignKey:SubmissionID;references:ID"`
	History    []PreDefenseHistory          `gorm:"foreignKey:SubmissionID;references:ID"`
}

type PreDefenseCommissionMember struct {
	ID           int64  `gorm:"primaryKey"`
	SubmissionID string `gorm:"index;not null;size:36"`
	UserID       int64  `gorm:"index;not null"`
	Role         string `gorm:"size:30;not null"` // chairman, member, secretary

	IsPresent       bool `gorm:"default:false"`
	IndividualGrade *int32
	Comment         string `gorm:"type:text"`

	CreatedAt time.Time
}

type PreDefenseDocument struct {
	ID string `gorm:"primaryKey;size:36"` // UUID string (migration 000010)

	SubmissionID string `gorm:"index;not null;size:36"`

	FileName    string `gorm:"size:255;not null"`
	FileType    string `gorm:"size:50"`
	DisplayName string `gorm:"size:255"`
	Size        int64
	DownloadURL string `gorm:"size:500"`

	UploadedBy *int64
	UploadedAt time.Time

	CreatedAt time.Time
}

type PreDefenseHistory struct {
	ID           int64  `gorm:"primaryKey"`
	SubmissionID string `gorm:"index;not null;size:36"`

	Action   string `gorm:"size:50;not null"`
	ActorID  int64  `gorm:"not null"`
	OldValue string `gorm:"type:text"`
	NewValue string `gorm:"type:text"`
	Comment  string `gorm:"type:text"`

	CreatedAt time.Time
}

// ==================== TableName overrides ====================

func (Grade) TableName() string                   { return "admin_grades" }
func (GradeHistory) TableName() string            { return "admin_grade_history" }
func (TopicRegistration) TableName() string       { return "admin_topic_registrations" }
func (TopicRegistrationReview) TableName() string { return "admin_topic_registration_reviews" }
func (Submission) TableName() string              { return "admin_submissions" }
func (SubmissionReview) TableName() string        { return "admin_submission_reviews" }
func (SupervisorAssignment) TableName() string    { return "admin_supervisor_assignments" }
func (AdminActivity) TableName() string           { return "admin_activities" }

func (SupervisorRequest) TableName() string        { return "admin_supervisor_requests" }
func (SupervisorRequestHistory) TableName() string { return "admin_supervisor_request_history" }

// pre-defense tables (migration 000010)
func (PreDefenseSubmission) TableName() string       { return "admin_pre_defense_submissions" }
func (PreDefenseCommissionMember) TableName() string { return "admin_pre_defense_commission" }
func (PreDefenseDocument) TableName() string         { return "admin_pre_defense_documents" }
func (PreDefenseHistory) TableName() string          { return "admin_pre_defense_history" }

// ==================== Status constants ====================

const (
	StatusPending           = "pending"
	StatusApproved          = "approved"
	StatusRejected          = "rejected"
	StatusRevisionRequested = "revision_requested"
)

const (
	SupervisorRequestStatusPending   = "pending"
	SupervisorRequestStatusApproved  = "approved"
	SupervisorRequestStatusRejected  = "rejected"
	SupervisorRequestStatusCancelled = "cancelled"
)

const (
	SupervisorRequestActionCreated   = "created"
	SupervisorRequestActionApproved  = "approved"
	SupervisorRequestActionRejected  = "rejected"
	SupervisorRequestActionCancelled = "cancelled"
)

const (
	ActivityTypeSubmission        = "SUBMISSION"
	ActivityTypeGrade             = "GRADE"
	ActivityTypeTeamUpdate        = "TEAM_UPDATE"
	ActivityTypeTeamDelete        = "TEAM_DELETE"
	ActivityTypeSupervisorAssign  = "SUPERVISOR_ASSIGN"
	ActivityTypeReview            = "REVIEW"
	ActivityTypeTopicRegistration = "TOPIC_REGISTRATION"
	ActivityTypeTopicApproval     = "TOPIC_APPROVAL"

	ActivityTypeSupervisorRequest         = "SUPERVISOR_REQUEST"
	ActivityTypeSupervisorRequestApproved = "SUPERVISOR_REQUEST_APPROVED"
	ActivityTypeSupervisorRequestRejected = "SUPERVISOR_REQUEST_REJECTED"
)
