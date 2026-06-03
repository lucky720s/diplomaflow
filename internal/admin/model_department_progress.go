package admin

import "time"

// Domain models for the teacher-facing department-progress module.
// These are read-only aggregates assembled from existing admin-owned tables.

// Normalized admission decision (single source of truth — frontend only displays).
const (
	AdmissionNotDecided       = "not_decided"
	AdmissionAdmitted         = "admitted"
	AdmissionNotAdmitted      = "not_admitted"
	AdmissionRevisionRequired = "revision_required"
)

// ---- Summary ----

type DepartmentProgressStatsData struct {
	TotalStudents             int32
	TotalTeams                int32
	TotalProjects             int32
	CompletedProjects         int32
	PendingTopicRegistrations int32
	PendingSupervisorRequests int32
	PendingNormControl        int32
	PendingAntiplagiat        int32
	PendingPreDefenses        int32
	ScheduledPreDefenses      int32
	AdmittedCount             int32
	NotAdmittedCount          int32
	RevisionRequiredCount     int32
	AverageGrade              float32
	HasAverageGrade           bool
}

// ---- Team list ----

type DepartmentProgressTeamFilter struct {
	DepartmentID      int64
	Search            string
	SupervisorID      int64
	CurrentStage      string
	TopicStatus       string
	NormControlStatus string
	AntiplagiatStatus string
	PreDefenseStatus  string
	AdmissionStatus   string
	MinGrade          *int32
	MaxGrade          *int32
	Page              int32
	PageSize          int32
	SortBy            string
	SortOrder         string
}

// dpTeamRow is the flat row returned by the single paginated list query.
type dpTeamRow struct {
	TeamID    int64     `gorm:"column:team_id"`
	TeamName  string    `gorm:"column:team_name"`
	CreatedAt time.Time `gorm:"column:created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at"`

	ProjectID        int64  `gorm:"column:project_id"`
	ProjectTitle     string `gorm:"column:project_title"`
	CurrentStateName string `gorm:"column:current_state_name"`

	SupervisorID int64 `gorm:"column:supervisor_id"`

	TopicStatus        string     `gorm:"column:topic_status"`
	NormControlStatus  string     `gorm:"column:norm_control_status"`
	AntiplagiatStatus  string     `gorm:"column:antiplagiat_status"`
	AntiplagiatPercent *int32     `gorm:"column:antiplagiat_percent"`
	PreDefenseStatus   string     `gorm:"column:pre_defense_status"`
	PreDefenseGrade    *int32     `gorm:"column:pre_defense_grade"`
	FinalGrade         *int32     `gorm:"column:final_grade"`
	AdmissionStatus    string     `gorm:"column:admission_status"`
	ProgressPercentage float32    `gorm:"column:progress_percentage"`
	LastActivityAt     *time.Time `gorm:"column:last_activity_at"`
}

// DepartmentProgressTeam is the assembled list item (row + batched relations).
type DepartmentProgressTeam struct {
	Row        *dpTeamRow
	Supervisor *DPUser
	Members    []*DPMember
}

type DPUser struct {
	ID       int64
	FullName string
	Email    string
	Position string
}

type DPMember struct {
	UserID   int64
	FullName string
	Email    string
	Role     string
}

// ---- Team details ----

type DepartmentProgressTeamDetails struct {
	Team              *DPTeamInfo
	Project           *DPProjectInfo
	Supervisor        *DPUser
	Members           []*DPMember
	CurrentStateName  string
	Progress          float32
	Steps             []*DPStep
	TopicRegistration *DPTopicRegistration
	NormControl       *DPNormControl
	Antiplagiat       *DPAntiplagiat
	PreDefenses       []*DPPreDefense
	Grades            []*Grade
	History           []*UnifiedHistoryItem
}

type DPTeamInfo struct {
	TeamID    int64     `gorm:"column:team_id"`
	TeamName  string    `gorm:"column:team_name"`
	Status    string    `gorm:"column:status"`
	CreatedAt time.Time `gorm:"column:created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at"`

	ProjectID        int64  `gorm:"column:project_id"`
	ProjectTitle     string `gorm:"column:project_title"`
	ProjectDesc      string `gorm:"column:project_desc"`
	CurrentStateID   int64  `gorm:"column:current_state_id"`
	CurrentStateName string `gorm:"column:current_state_name"`
	WorkflowID       int64  `gorm:"column:workflow_id"`
	SupervisorID     int64  `gorm:"column:supervisor_id"`
}

type DPProjectInfo struct {
	ProjectID     int64
	TitleKZ       string
	TitleRU       string
	TitleEN       string
	TitleDisplay  string
	Description   string
	CurrentState  string
	CurrentStepID int64
}

// DPStep is one workflow step with its resolved status/grade/admission.
type DPStep struct {
	StepID          int64      `gorm:"column:step_id"`
	StepName        string     `gorm:"column:step_name"`
	DisplayName     string     `gorm:"column:display_name"`
	OrderIndex      int32      `gorm:"column:order_index"`
	StepType        string     `gorm:"column:step_type"`
	GradeType       string     `gorm:"column:grade_type"`    // from config.review_config
	PassingScore    int32      `gorm:"column:passing_score"` // from config.review_config
	SubmissionState string     `gorm:"column:submission_status"`
	Grade           *int32     `gorm:"column:grade"`
	ReviewerName    string     `gorm:"column:reviewer_name"`
	ReviewedAt      *time.Time `gorm:"column:reviewed_at"`
	Comment         string     `gorm:"column:comment"`

	// computed
	Status          string
	AdmissionStatus string
}

type DPTopicRegistration struct {
	Status          string     `gorm:"column:status"`
	ProposedTopicKZ string     `gorm:"column:proposed_topic_kz"`
	ProposedTopicRU string     `gorm:"column:proposed_topic_ru"`
	ProposedTopicEN string     `gorm:"column:proposed_topic_en"`
	ReviewerName    string     `gorm:"column:reviewer_name"`
	ReviewedAt      *time.Time `gorm:"column:reviewed_at"`
	Comment         string     `gorm:"column:comment"`
	RejectionReason string     `gorm:"column:rejection_reason"`
}

type DPNormControl struct {
	Status              string     `gorm:"column:status"`
	SubmissionID        string     `gorm:"column:submission_id"`
	ReviewerName        string     `gorm:"column:reviewer_name"`
	ReviewedAt          *time.Time `gorm:"column:reviewed_at"`
	IssuesCount         int32      `gorm:"column:issues_count"`
	CriticalIssuesCount int32      `gorm:"column:critical_issues_count"`
}

type DPAntiplagiat struct {
	Status            string     `gorm:"column:status"`
	SimilarityPercent *int32     `gorm:"column:similarity_percent"`
	AIPercent         *int32     `gorm:"column:ai_percent"`
	CheckedAt         *time.Time `gorm:"column:checked_at"`
	ReviewerName      string     `gorm:"column:reviewer_name"`
}

type DPPreDefense struct {
	ID            string     `gorm:"column:id"`
	Status        string     `gorm:"column:status"`
	ScheduledDate *time.Time `gorm:"column:scheduled_date"`
	Location      string     `gorm:"column:location"`
	Grade         *int32     `gorm:"column:grade"`
	Result        string     `gorm:"column:result"`
	Comment       string     `gorm:"column:comment"`

	// computed
	AdmissionStatus string
	Commission      []*PreDefenseCommissionMemberData
}

// PreDefenseCommissionMemberData mirrors the commission row (reused name resolution).
type PreDefenseCommissionMemberData struct {
	ID        int64  `gorm:"column:id"`
	UserID    int64  `gorm:"column:user_id"`
	FullName  string `gorm:"column:full_name"`
	Email     string `gorm:"column:email"`
	Role      string `gorm:"column:role"`
	IsPresent bool   `gorm:"column:is_present"`
	Comment   string `gorm:"column:comment"`
}

type UnifiedHistoryItem struct {
	ID        string    `gorm:"column:id"`
	Source    string    `gorm:"column:source"`
	Action    string    `gorm:"column:action"`
	ActorID   int64     `gorm:"column:actor_id"`
	ActorName string    `gorm:"column:actor_name"`
	OldValue  string    `gorm:"column:old_value"`
	NewValue  string    `gorm:"column:new_value"`
	Comment   string    `gorm:"column:comment"`
	CreatedAt time.Time `gorm:"column:created_at"`
}
