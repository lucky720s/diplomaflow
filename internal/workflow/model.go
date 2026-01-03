package workflow

import (
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// Workflow представляет шаблон бизнес-процесса
type Workflow struct {
	ID           int64  `gorm:"primaryKey"`
	Name         string `gorm:"not null"`
	Description  string
	DepartmentID int64          `gorm:"index;not null"`
	Version      int32          `gorm:"default:1"`
	IsActive     bool           `gorm:"default:false"`
	IsTemplate   bool           `gorm:"default:false"`
	ParentID     *int64         `gorm:"index"`
	Settings     datatypes.JSON `gorm:"type:jsonb;default:'{}'"`
	States       []State        `gorm:"foreignKey:WorkflowID"`
	Transitions  []Transition   `gorm:"foreignKey:WorkflowID"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
	DeletedAt    gorm.DeletedAt `gorm:"index"`
}

// State представляет состояние в workflow
type State struct {
	ID            int64  `gorm:"primaryKey"`
	WorkflowID    int64  `gorm:"index;not null"`
	Name          string `gorm:"not null"`
	DisplayName   string
	Description   string
	OrderIndex    int32          `gorm:"default:0"`
	Type          string         `gorm:"not null"`
	IsInitial     bool           `gorm:"default:false"`
	IsFinal       bool           `gorm:"default:false"`
	IsOptional    bool           `gorm:"default:false"`
	Config        datatypes.JSON `gorm:"type:jsonb;default:'{}'"`
	DurationDays  int32          `gorm:"default:0"`
	DurationMode  string         `gorm:"default:'relative'"`
	FixedDeadline *time.Time
	Color         string
	Icon          string
	Actions       []StateAction `gorm:"foreignKey:StateID"`
	CreatedAt     time.Time
	UpdatedAt     time.Time
	DeletedAt     gorm.DeletedAt `gorm:"index"`
}

// Transition представляет переход между состояниями
type Transition struct {
	ID          int64  `gorm:"primaryKey"`
	WorkflowID  int64  `gorm:"index;not null"`
	EventName   string `gorm:"not null"`
	DisplayName string
	FromStateID int64          `gorm:"index;not null"`
	ToStateID   int64          `gorm:"index;not null"`
	Conditions  datatypes.JSON `gorm:"type:jsonb;default:'[]'"`
	ButtonLabel string
	ButtonColor string `gorm:"default:'primary'"`
	ConfirmText string
	Priority    int32 `gorm:"default:0"`
	CreatedAt   time.Time
}

// StateAction представляет автоматическое действие при переходе
type StateAction struct {
	ID                int64 `gorm:"primaryKey"`
	StateID           int64 `gorm:"index;not null"`
	Name              string
	Type              string         `gorm:"not null"`
	Trigger           string         `gorm:"not null"`
	OrderIndex        int32          `gorm:"default:0"`
	Config            datatypes.JSON `gorm:"type:jsonb;default:'{}'"`
	IsEnabled         bool           `gorm:"default:true"`
	IsOptional        bool           `gorm:"default:false"`
	Conditions        datatypes.JSON `gorm:"type:jsonb;default:'[]'"`
	MaxRetries        int32          `gorm:"default:3"`
	RetryDelaySeconds int32          `gorm:"default:60"`
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

// StateCondition для сложных правил
type StateCondition struct {
	ID           int64 `gorm:"primaryKey"`
	StateID      int64 `gorm:"index;not null"`
	Name         string
	Type         string `gorm:"not null"`
	Expression   string
	Config       datatypes.JSON `gorm:"type:jsonb;default:'{}'"`
	ErrorMessage string
	CreatedAt    time.Time
}

// DepartmentWorkflowConfig — конфигурация workflow для кафедры
type DepartmentWorkflowConfig struct {
	ID                int64          `gorm:"primaryKey"`
	DepartmentID      int64          `gorm:"index;not null"`
	WorkflowID        int64          `gorm:"index;not null"`
	AcademicYear      string         `gorm:"not null"`
	IsActive          bool           `gorm:"default:false"`
	ConfigOverrides   datatypes.JSON `gorm:"type:jsonb;default:'{}'"`
	TeamSettings      datatypes.JSON `gorm:"type:jsonb"`
	DeadlineOverrides datatypes.JSON `gorm:"type:jsonb;default:'{}'"`
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

// DepartmentCustomStep — кастомные этапы для кафедры
type DepartmentCustomStep struct {
	ID                 int64  `gorm:"primaryKey"`
	DepartmentConfigID int64  `gorm:"index;not null"`
	Name               string `gorm:"not null"`
	DisplayName        string `gorm:"not null"`
	StepType           string `gorm:"not null"`
	InsertAfterStateID *int64
	Config             datatypes.JSON `gorm:"type:jsonb;default:'{}'"`
	IsRequired         bool           `gorm:"default:true"`
	DurationDays       int32          `gorm:"default:7"`
	OrderIndex         int32          `gorm:"default:0"`
	CreatedAt          time.Time
}

// ActionRegistry — реестр доступных действий
type ActionRegistry struct {
	ID           string `gorm:"primaryKey"`
	Name         string `gorm:"not null"`
	Description  string
	Category     string
	ConfigSchema datatypes.JSON `gorm:"type:jsonb;not null"`
	IsSystem     bool           `gorm:"default:false"`
	IsEnabled    bool           `gorm:"default:true"`
	CreatedAt    time.Time
}

// ============ Config Structs (для JSON parsing) ============

// WorkflowSettings — настройки workflow из JSON
type WorkflowSettings struct {
	TeamRequired        bool     `json:"team_required"`
	MinTeamSize         int32    `json:"min_team_size"`
	MaxTeamSize         int32    `json:"max_team_size"`
	AllowSoloProject    bool     `json:"allow_solo_project"`
	AcademicYear        string   `json:"academic_year"`
	StartDate           string   `json:"start_date"`
	EndDate             string   `json:"end_date"`
	Timezone            string   `json:"timezone"`
	DeadlineWarningDays []int    `json:"deadline_warning_days"`
	RequiredRoles       []string `json:"required_roles"`
	AllowedFileTypes    []string `json:"allowed_file_types"`
}

// TeamConfig — конфигурация команды
type TeamConfig struct {
	MinSize          int32 `json:"min_size"`
	MaxSize          int32 `json:"max_size"`
	AllowSolo        bool  `json:"allow_solo"`
	RequireLeader    bool  `json:"require_leader"`
	InviteExpireDays int32 `json:"invite_expire_days"`
}

// FileConfig — конфигурация файлов
type FileConfig struct {
	MaxFiles          int32    `json:"max_files"`
	MaxSizeBytes      int64    `json:"max_size_bytes"`
	AllowedExtensions []string `json:"allowed_extensions"`
	RequiredFiles     []struct {
		Type        string   `json:"type"`
		DisplayName string   `json:"display_name"`
		Extensions  []string `json:"extensions"`
		TemplateURL string   `json:"template_url,omitempty"`
	} `json:"required_files,omitempty"`
}

// FormConfig — конфигурация формы
type FormConfig struct {
	FormID string                 `json:"form_id"`
	Schema map[string]interface{} `json:"json_schema"`
}

// ReviewConfig — конфигурация проверки
type ReviewConfig struct {
	ReviewerRoles  []string `json:"reviewer_roles"`
	MinReviewers   int32    `json:"min_reviewers"`
	RequireComment bool     `json:"require_comment"`
	AllowGrade     bool     `json:"allow_grade"`
}

// StateConfig — полная конфигурация состояния
type StateConfig struct {
	TeamConfig   *TeamConfig   `json:"team_config,omitempty"`
	FileConfig   *FileConfig   `json:"file_config,omitempty"`
	FormConfig   *FormConfig   `json:"form_config,omitempty"`
	ReviewConfig *ReviewConfig `json:"review_config,omitempty"`
}

// TeamSettings — настройки команды для department config
type TeamSettings struct {
	AllowSolo        bool  `json:"allow_solo"`
	MinSize          int32 `json:"min_size"`
	MaxSize          int32 `json:"max_size"`
	RequireSameGroup bool  `json:"require_same_group"`
	InviteExpireDays int32 `json:"invite_expire_days"`
}

// DeadlineOverride — переопределение дедлайнов
type DeadlineOverride struct {
	StateID      int64      `json:"state_id"`
	StateName    string     `json:"state_name"`
	FixedDate    *time.Time `json:"fixed_date,omitempty"`
	DurationDays *int32     `json:"duration_days,omitempty"`
	StartFrom    string     `json:"start_from,omitempty"`
}

// TableName overrides
func (Workflow) TableName() string                 { return "workflows" }
func (State) TableName() string                    { return "states" }
func (Transition) TableName() string               { return "transitions" }
func (StateAction) TableName() string              { return "state_actions" }
func (StateCondition) TableName() string           { return "state_conditions" }
func (DepartmentWorkflowConfig) TableName() string { return "department_workflow_configs" }
func (DepartmentCustomStep) TableName() string     { return "department_custom_steps" }
func (ActionRegistry) TableName() string           { return "action_registry" }
