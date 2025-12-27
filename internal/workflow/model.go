package workflow

import (
	"encoding/json"
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// ================== WORKFLOW ==================
type Workflow struct {
	ID           int64          `gorm:"primaryKey"`
	Name         string         `gorm:"not null"`
	Description  string         `gorm:"type:text"`
	DepartmentID int64          `gorm:"index;not null"`
	Version      int32          `gorm:"default:1"`
	IsActive     bool           `gorm:"default:false;index"`
	IsTemplate   bool           `gorm:"default:false"`
	ParentID     *int64         `gorm:"index"`
	Settings     datatypes.JSON `gorm:"type:jsonb;default:'{}'"`

	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`

	// Relations
	States      []State      `gorm:"foreignKey:WorkflowID"`
	Transitions []Transition `gorm:"foreignKey:WorkflowID"`
}

// WorkflowSettings - глобальные настройки workflow
type WorkflowSettings struct {
	TeamRequired        bool     `json:"team_required"`
	MinTeamSize         int      `json:"min_team_size"`
	MaxTeamSize         int      `json:"max_team_size"`
	AllowSoloProject    bool     `json:"allow_solo_project"`
	AcademicYear        string   `json:"academic_year"`
	StartDate           string   `json:"start_date"`
	EndDate             string   `json:"end_date"`
	Timezone            string   `json:"timezone"`
	DeadlineWarningDays []int    `json:"deadline_warning_days"`
	RequiredRoles       []string `json:"required_roles"`
	AllowedFileTypes    []string `json:"allowed_file_types"`
}

// ================== STATE ==================
type State struct {
	ID            int64          `gorm:"primaryKey"`
	WorkflowID    int64          `gorm:"not null;index"`
	Name          string         `gorm:"type:varchar(255);not null"`
	DisplayName   string         `gorm:"type:varchar(255)"`
	Description   string         `gorm:"type:text"`
	OrderIndex    int32          `gorm:"not null;default:0"`
	Type          string         `gorm:"type:varchar(50);not null"`
	IsInitial     bool           `gorm:"default:false"`
	IsFinal       bool           `gorm:"default:false"`
	IsOptional    bool           `gorm:"default:false"`
	Config        datatypes.JSON `gorm:"type:jsonb;not null;default:'{}'"`
	DurationDays  int32          `gorm:"default:0"`
	DurationMode  string         `gorm:"default:'relative'"`
	FixedDeadline *time.Time     `gorm:"index"`
	Color         string         `gorm:"type:varchar(20)"`
	Icon          string         `gorm:"type:varchar(50)"`

	CreatedAt time.Time
	UpdatedAt time.Time

	// Relations
	Actions    []StateAction    `gorm:"foreignKey:StateID"`
	Conditions []StateCondition `gorm:"foreignKey:StateID"`
}

// StateType constants
const (
	StateTypeTeamFormation    = "TEAM_FORMATION"
	StateTypeSupervisorSelect = "SUPERVISOR_SELECTION"
	StateTypeTopicApproval    = "TOPIC_APPROVAL"
	StateTypeDocumentUpload   = "DOCUMENT_UPLOAD"
	StateTypeFormSubmit       = "FORM_SUBMIT"
	StateTypeExternalCheck    = "EXTERNAL_CHECK"
	StateTypeReview           = "REVIEW"
	StateTypeApproval         = "APPROVAL"
	StateTypeDefense          = "DEFENSE"
	StateTypeMilestone        = "MILESTONE"
	StateTypeGrading          = "GRADING"
	StateTypeCompleted        = "COMPLETED"
)

// ================== TRANSITION ==================
type Transition struct {
	ID          int64          `gorm:"primaryKey"`
	WorkflowID  int64          `gorm:"not null;index"`
	EventName   string         `gorm:"type:varchar(100);not null"`
	DisplayName string         `gorm:"type:varchar(255)"`
	FromStateID int64          `gorm:"not null;index:idx_from_event"`
	ToStateID   int64          `gorm:"not null"`
	Conditions  datatypes.JSON `gorm:"type:jsonb;default:'[]'"`
	ButtonLabel string         `gorm:"type:varchar(100)"`
	ButtonColor string         `gorm:"type:varchar(20)"`
	ConfirmText string         `gorm:"type:text"`
	Priority    int32          `gorm:"default:0"` // ДОБАВЛЕНО: для сортировки

	CreatedAt time.Time
}

// ================== STATE ACTION ==================
type StateAction struct {
	ID         int64          `gorm:"primaryKey"`
	StateID    int64          `gorm:"not null;index"`
	Name       string         `gorm:"type:varchar(100)"`
	Type       string         `gorm:"type:varchar(50);not null"`
	Trigger    string         `gorm:"type:varchar(50);not null"`
	OrderIndex int32          `gorm:"default:0"`
	Config     datatypes.JSON `gorm:"type:jsonb;not null;default:'{}'"`
	IsEnabled  bool           `gorm:"default:true"`
	Conditions datatypes.JSON `gorm:"type:jsonb;default:'[]'"`

	CreatedAt time.Time
	UpdatedAt time.Time // ДОБАВЛЕНО: было пропущено
}

func (sa *StateAction) IsOptional() bool {
	var config map[string]interface{}
	if err := json.Unmarshal(sa.Config, &config); err != nil {
		return false
	}
	if optional, ok := config["optional"].(bool); ok {
		return optional
	}
	return false
}

// ActionTrigger constants
const (
	TriggerOnEnter     = "ON_ENTER"
	TriggerOnExit      = "ON_EXIT"
	TriggerOnDeadline  = "ON_DEADLINE"
	TriggerOnUpdate    = "ON_UPDATE"
	TriggerOnCondition = "ON_CONDITION"
	TriggerScheduled   = "SCHEDULED"
)

// ActionType constants
const (
	ActionTypeSendNotification = "SEND_NOTIFICATION"
	ActionTypeSendEmail        = "SEND_EMAIL"
	ActionTypeAssignTask       = "ASSIGN_TASK"
	ActionTypeCallWebhook      = "CALL_WEBHOOK"
	ActionTypeCreateSubmission = "CREATE_SUBMISSION"
	ActionTypeUpdateProject    = "UPDATE_PROJECT"
	ActionTypeScheduleReminder = "SCHEDULE_REMINDER"
	ActionTypeValidateData     = "VALIDATE_DATA"
	ActionTypeCalculateGrade   = "CALCULATE_GRADE"
	ActionTypeGenerateDocument = "GENERATE_DOCUMENT"
	ActionTypeExternalAPI      = "EXTERNAL_API"
)

// ================== STATE CONDITION ==================
type StateCondition struct {
	ID           int64          `gorm:"primaryKey"`
	StateID      int64          `gorm:"not null;index"`
	Name         string         `gorm:"type:varchar(100)"`
	Type         string         `gorm:"type:varchar(50);not null"`
	Expression   string         `gorm:"type:text"`
	Config       datatypes.JSON `gorm:"type:jsonb"`
	ErrorMessage string         `gorm:"type:text"`
}

// ================== WORKFLOW TEMPLATE ==================
type WorkflowTemplate struct {
	ID           int64  `gorm:"primaryKey"`
	Name         string `gorm:"unique;not null"`
	Description  string
	Category     string
	TemplateData datatypes.JSON `gorm:"type:jsonb;not null"`
	IsSystem     bool           `gorm:"default:false"`
	CreatedBy    *int64
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func (WorkflowTemplate) TableName() string { return "workflow_templates" }

// ================== CONFIG TYPES ==================
type StateConfig struct {
	TeamConfig          *TeamFormationConfig       `json:"team_config,omitempty"`
	SupervisorConfig    *SupervisorSelectionConfig `json:"supervisor_config,omitempty"`
	FileConfig          *FileUploadConfig          `json:"file_config,omitempty"`
	FormConfig          *FormSubmitConfig          `json:"form_config,omitempty"`
	ExternalCheckConfig *ExternalCheckConfig       `json:"external_check_config,omitempty"`
	ReviewConfig        *ReviewConfig              `json:"review_config,omitempty"`
	GradingConfig       *GradingConfig             `json:"grading_config,omitempty"`
	AllowedRoles        []string                   `json:"allowed_roles,omitempty"`
	RequiredFields      []string                   `json:"required_fields,omitempty"`
	ValidationRules     []ValidationRule           `json:"validation_rules,omitempty"`
}

type TeamFormationConfig struct {
	MinSize          int  `json:"min_size"`
	MaxSize          int  `json:"max_size"`
	AllowSolo        bool `json:"allow_solo"`
	RequireLeader    bool `json:"require_leader"`
	AllowCrossGroup  bool `json:"allow_cross_group"`
	InviteExpireDays int  `json:"invite_expire_days"`
}

type SupervisorSelectionConfig struct {
	AllowedSupervisorRoles   []string `json:"allowed_roles"`
	MaxStudentsPerSupervisor int      `json:"max_students"`
	RequireTopicProposal     bool     `json:"require_topic"`
	AutoAssign               bool     `json:"auto_assign"`
}

type FileUploadConfig struct {
	MaxFiles          int            `json:"max_files"`
	MaxSizeBytes      int64          `json:"max_size_bytes"`
	AllowedExtensions []string       `json:"allowed_extensions"`
	RequiredFiles     []RequiredFile `json:"required_files"`
	NamingTemplate    string         `json:"naming_template"`
	VirusScan         bool           `json:"virus_scan"`
}

type RequiredFile struct {
	Type        string   `json:"type"`
	DisplayName string   `json:"display_name"`
	Extensions  []string `json:"extensions"`
	MaxSize     int64    `json:"max_size"`
	Template    string   `json:"template_url"`
}

type FormSubmitConfig struct {
	FormID        string          `json:"form_id"`
	JSONSchema    json.RawMessage `json:"json_schema"`
	UISchema      json.RawMessage `json:"ui_schema"`
	AllowEdit     bool            `json:"allow_edit"`
	AllowMultiple bool            `json:"allow_multiple"`
}

type ExternalCheckConfig struct {
	ServiceType    string          `json:"service_type"`
	ServiceURL     string          `json:"service_url"`
	APICredentials string          `json:"api_credentials"`
	MinScore       float64         `json:"min_score"`
	AutoReject     bool            `json:"auto_reject"`
	RetryCount     int             `json:"retry_count"`
	Webhooks       []WebhookConfig `json:"webhooks"`
}

type ReviewConfig struct {
	ReviewerRoles    []string `json:"reviewer_roles"`
	MinReviewers     int      `json:"min_reviewers"`
	RequireComment   bool     `json:"require_comment"`
	AllowGrade       bool     `json:"allow_grade"`
	GradeScale       string   `json:"grade_scale"`
	AutoApproveAfter int      `json:"auto_approve_days"`
}

type GradingConfig struct {
	GradeScale      string           `json:"grade_scale"`
	PassingScore    float64          `json:"passing_score"`
	Components      []GradeComponent `json:"components"`
	WeightedAverage bool             `json:"weighted_average"`
}

type GradeComponent struct {
	Name       string  `json:"name"`
	Weight     float64 `json:"weight"`
	MaxScore   float64 `json:"max_score"`
	GraderRole string  `json:"grader_role"`
}

type ValidationRule struct {
	Field   string      `json:"field"`
	Rule    string      `json:"rule"`
	Value   interface{} `json:"value"`
	Message string      `json:"message"`
}

type WebhookConfig struct {
	URL        string            `json:"url"`
	Method     string            `json:"method"`
	Headers    map[string]string `json:"headers"`
	Body       json.RawMessage   `json:"body"`
	RetryCount int               `json:"retry_count"`
	Timeout    int               `json:"timeout_seconds"`
}

// TransitionCondition - условия для перехода
type TransitionCondition struct {
	Type     string      `json:"type"`
	Field    string      `json:"field"`
	Operator string      `json:"operator"`
	Value    interface{} `json:"value"`
}
