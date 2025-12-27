// internal/workflow/model_department.go - РАСШИРЕННАЯ ВЕРСИЯ
package workflow

import (
	"time"

	"gorm.io/datatypes"
)

// DepartmentWorkflowConfig - конфигурация workflow для конкретной кафедры
type DepartmentWorkflowConfig struct {
	ID           int64  `gorm:"primaryKey"`
	DepartmentID int64  `gorm:"not null;index"`
	WorkflowID   int64  `gorm:"not null"`
	AcademicYear string `gorm:"type:varchar(20);not null"` // "2024-2025"
	IsActive     bool   `gorm:"default:false"`

	// Настройки команд
	TeamSettings datatypes.JSON `gorm:"type:jsonb"`

	// Переопределение дедлайнов
	DeadlineOverrides datatypes.JSON `gorm:"type:jsonb"`

	// Кастомные переопределения конфигов состояний
	ConfigOverrides datatypes.JSON `gorm:"type:jsonb"`

	CreatedAt time.Time
	UpdatedAt time.Time
}

// TeamSettings - настройки команды для кафедры
type TeamSettings struct {
	AllowSolo        bool `json:"allow_solo"`
	MinSize          int  `json:"min_size"`
	MaxSize          int  `json:"max_size"`
	RequireSameGroup bool `json:"require_same_group"`
	InviteExpireDays int  `json:"invite_expire_days"`
	RequireLeader    bool `json:"require_leader"`
	LeaderCanKick    bool `json:"leader_can_kick"`
}

// DeadlineOverride - переопределение дедлайна для конкретного этапа
type DeadlineOverride struct {
	StateID      int64      `json:"state_id"`
	StateName    string     `json:"state_name"`
	FixedDate    *time.Time `json:"fixed_date,omitempty"`    // Конкретная дата
	DurationDays *int       `json:"duration_days,omitempty"` // Или относительная
	StartFrom    string     `json:"start_from"`              // "workflow_start", "previous_step", "fixed_date"
}

// DepartmentCustomStep - кастомный этап для кафедры
type DepartmentCustomStep struct {
	ID                 int64  `gorm:"primaryKey"`
	DepartmentConfigID int64  `gorm:"not null;index"`
	Name               string `gorm:"type:varchar(255);not null"`
	DisplayName        string `gorm:"type:varchar(255);not null"`
	StepType           string `gorm:"type:varchar(50);not null"`
	InsertAfterStateID *int64
	Config             datatypes.JSON `gorm:"type:jsonb"`
	IsRequired         bool           `gorm:"default:true"`
	DurationDays       int32          `gorm:"default:7"`
	OrderIndex         int32
	CreatedAt          time.Time
}

func (DepartmentWorkflowConfig) TableName() string { return "department_workflow_configs" }
func (DepartmentCustomStep) TableName() string     { return "department_custom_steps" }
