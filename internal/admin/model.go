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
	UpdatedBy    int64
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

// ==================== PRE-DEFENSE MODELS ====================

// PreDefenseSubmission - заявление на предзащиту
type PreDefenseSubmission struct {
	ID           string `gorm:"primaryKey;size:36"`
	TeamID       int64  `gorm:"index;not null"`
	ProjectID    int64  `gorm:"index;not null"`
	SupervisorID int64  `gorm:"index"`
	SubmittedBy  int64  `gorm:"not null"`

	// Статус заявления: pending, scheduled, in_progress, completed, passed, failed, cancelled
	Status string `gorm:"size:30;default:'pending';index"`

	// Сообщение от команды при подаче
	Message string `gorm:"type:text"`

	// Предпочтительные даты (JSON array of strings)
	PreferredDates datatypes.JSON `gorm:"type:jsonb"`

	// Прикреплённые документы (JSON array of document IDs)
	DocumentIDs datatypes.JSON `gorm:"type:jsonb"`

	// Информация о назначенной предзащите
	ScheduledDate   *time.Time `gorm:"index"`
	ScheduledTime   string     `gorm:"size:10"` // HH:MM format
	Location        string     `gorm:"size:255"`
	MeetingLink     string     `gorm:"size:500"`
	DurationMinutes int32      `gorm:"default:30"`
	ScheduledBy     int64
	ScheduledAt     *time.Time

	// Оценивание
	Grade        int32  `gorm:"default:0"` // 0-100 баллов
	GradeComment string `gorm:"type:text"`
	GradedBy     int64
	GradedAt     *time.Time

	// Результат предзащиты: passed, failed, conditional
	Result            string         `gorm:"size:20"`
	ResultComment     string         `gorm:"type:text"`
	Recommendations   datatypes.JSON `gorm:"type:jsonb"` // JSON array of strings
	AllowResubmission bool           `gorm:"default:false"`

	// Завершение
	CompletedBy int64
	CompletedAt *time.Time

	// Номер попытки (для пересдач)
	AttemptNumber     int32  `gorm:"default:1"`
	PreviousAttemptID string `gorm:"size:36;index"`

	// Timestamps
	SubmittedAt time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   gorm.DeletedAt `gorm:"index"`

	// Relations (для preload)
	Commission []PreDefenseCommissionMember `gorm:"foreignKey:SubmissionID;references:ID"`
	Documents  []PreDefenseDocument         `gorm:"foreignKey:SubmissionID;references:ID"`
	History    []PreDefenseHistory          `gorm:"foreignKey:SubmissionID;references:ID"`
}

// PreDefenseCommissionMember - член комиссии предзащиты
type PreDefenseCommissionMember struct {
	ID           int64  `gorm:"primaryKey"`
	SubmissionID string `gorm:"index;not null;size:36"`
	UserID       int64  `gorm:"index;not null"`
	Role         string `gorm:"size:20;not null"` // chairman, member, secretary

	// Присутствие и оценка
	IsPresent       bool   `gorm:"default:false"`
	IndividualGrade int32  `gorm:"default:0"` // 0-100
	Comment         string `gorm:"type:text"`

	// Кто добавил/удалил
	AddedBy   int64
	RemovedBy *int64
	RemovedAt *time.Time

	CreatedAt time.Time
	UpdatedAt time.Time
}

// PreDefenseDocument - документ для предзащиты
type PreDefenseDocument struct {
	ID           int64  `gorm:"primaryKey"`
	SubmissionID string `gorm:"index;not null;size:36"`
	FileID       string `gorm:"size:36"` // ID файла в file_service (если есть)

	FileName    string `gorm:"size:255;not null"`
	FileType    string `gorm:"size:50;not null"` // presentation, report, abstract, review, other
	DisplayName string `gorm:"size:255"`
	MimeType    string `gorm:"size:100"`
	Size        int64  `gorm:"default:0"`
	DownloadURL string `gorm:"size:500"`

	UploadedBy int64 `gorm:"not null"`
	CreatedAt  time.Time
	DeletedAt  gorm.DeletedAt `gorm:"index"`
}

// PreDefenseHistory - история изменений предзащиты
type PreDefenseHistory struct {
	ID           int64  `gorm:"primaryKey"`
	SubmissionID string `gorm:"index;not null;size:36"`

	// submitted, scheduled, rescheduled, graded, completed, cancelled, commission_added, commission_removed
	Action   string `gorm:"size:30;not null"`
	ActorID  int64  `gorm:"not null"`
	OldValue string `gorm:"type:text"` // JSON или plain text
	NewValue string `gorm:"type:text"` // JSON или plain text
	Comment  string `gorm:"type:text"`

	CreatedAt time.Time
}

// PreDefenseSubmissionWithDetails - заявление с полной информацией
type PreDefenseSubmissionWithDetails struct {
	PreDefenseSubmission

	// Enriched data (из других сервисов)
	TeamName       string
	ProjectTitle   string
	SupervisorName string
	SubmitterName  string
	GraderName     string
	SchedulerName  string
	CompleterName  string

	// Commission with user details
	CommissionMembers []*CommissionMemberWithDetails

	// Documents
	DocumentsList []*PreDefenseDocumentInfo
}

// CommissionMemberWithDetails - член комиссии с деталями пользователя
type CommissionMemberWithDetails struct {
	ID              int64
	UserID          int64
	FullName        string
	Email           string
	Role            string
	Position        string // Professor, Senior Lecturer, etc.
	IsPresent       bool
	IndividualGrade int32
	Comment         string
}

// PreDefenseDocumentInfo - информация о документе
type PreDefenseDocumentInfo struct {
	ID           int64
	FileID       string
	FileName     string
	FileType     string
	DisplayName  string
	Size         int64
	DownloadURL  string
	UploadedBy   int64
	UploaderName string
	UploadedAt   time.Time
}

// PreDefenseHistoryWithDetails - история с именами акторов
type PreDefenseHistoryWithDetails struct {
	ID           int64
	SubmissionID string
	Action       string
	ActorID      int64
	ActorName    string
	OldValue     string
	NewValue     string
	Comment      string
	CreatedAt    time.Time
}

// PreDefenseStats - статистика по предзащитам
type PreDefenseStats struct {
	PendingCount    int32
	ScheduledCount  int32
	InProgressCount int32
	CompletedCount  int32
	PassedCount     int32
	FailedCount     int32
	ThisWeekCount   int32
}

// PreDefenseScheduleItem - элемент расписания предзащит
type PreDefenseScheduleItem struct {
	SubmissionID          string
	TeamID                int64
	TeamName              string
	ProjectTitle          string
	SupervisorName        string
	ScheduledDate         time.Time
	ScheduledTime         string
	Location              string
	MeetingLink           string
	DurationMinutes       int32
	Status                string
	CommissionMemberNames []string
}

// ProjectProgressForPreDefense - прогресс проекта для отображения в предзащите
type ProjectProgressForPreDefense struct {
	ProjectID     int64
	Title         string
	CurrentState  string
	CurrentStepID int64
	Status        string
	Steps         []*StepStatusInfo
}

// StepStatusInfo - информация о статусе этапа
type StepStatusInfo struct {
	StepID      int64
	StepName    string
	Status      string // pending, in_progress, completed, skipped
	CompletedAt *time.Time
	Grade       *int32
}

// SubmitPreDefenseInput - входные данные для подачи заявления на предзащиту
type SubmitPreDefenseInput struct {
	TeamID         int64
	ProjectID      int64
	SubmittedBy    int64
	Message        string
	PreferredDates []string // ISO format dates
	DocumentIDs    []string
}

// SchedulePreDefenseInput - входные данные для назначения предзащиты
type SchedulePreDefenseInput struct {
	SubmissionID        string
	ScheduledBy         int64
	ScheduledDate       time.Time
	ScheduledTime       string // HH:MM
	Location            string
	MeetingLink         string
	DurationMinutes     int32
	CommissionMemberIDs []int64
	Comment             string
}

// GradePreDefenseInput - входные данные для выставления оценки
type GradePreDefenseInput struct {
	SubmissionID string
	GradedBy     int64
	Grade        int32 // 0-100
	Comment      string
	MemberGrades []CommissionMemberGradeInput
}

// CommissionMemberGradeInput - оценка от члена комиссии
type CommissionMemberGradeInput struct {
	MemberID int64
	Grade    int32
	Comment  string
}

// CompletePreDefenseInput - входные данные для завершения предзащиты
type CompletePreDefenseInput struct {
	SubmissionID      string
	CompletedBy       int64
	Result            string // passed, failed, conditional
	ResultComment     string
	Recommendations   []string
	AllowResubmission bool
}

// ReschedulePreDefenseInput - входные данные для переноса предзащиты
type ReschedulePreDefenseInput struct {
	SubmissionID   string
	RescheduledBy  int64
	NewDate        time.Time
	NewTime        string
	NewLocation    string
	NewMeetingLink string
	Reason         string
}

// StartPreDefenseInput - входные данные для начала предзащиты (вручную)
type StartPreDefenseInput struct {
	SubmissionID string
	StartedBy    int64
	Comment      string
}

// PreDefenseFilter - фильтр для списка предзащит
type PreDefenseFilter struct {
	DepartmentID int64
	SupervisorID int64
	TeamID       int64
	Status       string
	DateFrom     time.Time
	DateTo       time.Time
	Limit        int
	Offset       int
}

// ScheduledPreDefenseFilter - фильтр для расписания
type ScheduledPreDefenseFilter struct {
	DepartmentID       int64
	CommissionMemberID int64
	Location           string
	DateFrom           time.Time
	DateTo             time.Time
}

// TableName overrides
func (Grade) TableName() string                      { return "admin_grades" }
func (GradeHistory) TableName() string               { return "admin_grade_history" }
func (TopicRegistration) TableName() string          { return "admin_topic_registrations" }
func (TopicRegistrationReview) TableName() string    { return "admin_topic_registration_reviews" }
func (Submission) TableName() string                 { return "admin_submissions" }
func (SubmissionReview) TableName() string           { return "admin_submission_reviews" }
func (SupervisorAssignment) TableName() string       { return "admin_supervisor_assignments" }
func (AdminActivity) TableName() string              { return "admin_activities" }
func (SupervisorRequest) TableName() string          { return "admin_supervisor_requests" }
func (SupervisorRequestHistory) TableName() string   { return "admin_supervisor_request_history" }
func (PreDefenseSubmission) TableName() string       { return "admin_pre_defense_submissions" }
func (PreDefenseCommissionMember) TableName() string { return "admin_pre_defense_commission_members" }
func (PreDefenseDocument) TableName() string         { return "admin_pre_defense_documents" }
func (PreDefenseHistory) TableName() string          { return "admin_pre_defense_history" }

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

// ==================== PRE-DEFENSE STATUS CONSTANTS ====================

const (
	PreDefenseStatusPending    = "pending"
	PreDefenseStatusScheduled  = "scheduled"
	PreDefenseStatusInProgress = "in_progress"
	PreDefenseStatusCompleted  = "completed"
	PreDefenseStatusPassed     = "passed"
	PreDefenseStatusFailed     = "failed"
	PreDefenseStatusCancelled  = "cancelled"
)

// PreDefense Result constants
const (
	PreDefenseResultPassed      = "passed"
	PreDefenseResultFailed      = "failed"
	PreDefenseResultConditional = "conditional"
)

// PreDefense Commission Role constants
const (
	CommissionRoleChairman  = "chairman"
	CommissionRoleMember    = "member"
	CommissionRoleSecretary = "secretary"
)

// PreDefense Document Type constants
const (
	DocumentTypePresentation = "presentation"
	DocumentTypeReport       = "report"
	DocumentTypeAbstract     = "abstract"
	DocumentTypeReview       = "review"
	DocumentTypeOther        = "other"
)

// PreDefense History Action constants
const (
	PreDefenseActionSubmitted         = "submitted"
	PreDefenseActionScheduled         = "scheduled"
	PreDefenseActionRescheduled       = "rescheduled"
	PreDefenseActionGraded            = "graded"
	PreDefenseActionCompleted         = "completed"
	PreDefenseActionCancelled         = "cancelled"
	PreDefenseActionCommissionAdded   = "commission_added"
	PreDefenseActionCommissionRemoved = "commission_removed"
	PreDefenseActionStarted           = "started" // для IN_PROGRESS
)

// ==================== ACTIVITY TYPES FOR PRE-DEFENSE ====================

const (
	ActivityTypePreDefenseSubmitted   = "PRE_DEFENSE_SUBMITTED"
	ActivityTypePreDefenseScheduled   = "PRE_DEFENSE_SCHEDULED"
	ActivityTypePreDefenseStarted     = "PRE_DEFENSE_STARTED"
	ActivityTypePreDefenseGraded      = "PRE_DEFENSE_GRADED"
	ActivityTypePreDefenseCompleted   = "PRE_DEFENSE_COMPLETED"
	ActivityTypePreDefenseCancelled   = "PRE_DEFENSE_CANCELLED"
	ActivityTypePreDefenseRescheduled = "PRE_DEFENSE_RESCHEDULED"
)
