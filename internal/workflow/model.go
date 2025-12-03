package workflow

import (
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type Workflow struct {
	ID           int64  `gorm:"primaryKey"`
	Name         string `gorm:"not null"`
	DepartmentID int64  `gorm:"index;not null"`
	IsActive     bool   `gorm:"default:false;index"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
	DeletedAt    gorm.DeletedAt `gorm:"index"`
	Steps        []State        `gorm:"foreignKey:WorkflowID"`
}

type State struct {
	ID           int64  `gorm:"primaryKey"`
	WorkflowID   int64  `gorm:"not null;index"`
	Name         string `gorm:"type:varchar(255);not null"`
	Description  string
	Type         string         `gorm:"type:varchar(50);not null"`
	Config       datatypes.JSON `gorm:"not null"`
	DurationDays int32
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type Transition struct {
	ID          int64  `gorm:"primaryKey"`
	WorkflowID  int64  `gorm:"not null;index"`
	EventName   string `gorm:"type:varchar(100);not null;index:idx_from_event"`
	FromStateID int64  `gorm:"not null;index:idx_from_event"`
	ToStateID   int64  `gorm:"not null"`
}

type StateAction struct {
	ID      int64          `gorm:"primaryKey"`
	StateID int64          `gorm:"not null;index"`
	Type    string         `gorm:"type:varchar(50);not null"`
	Trigger string         `gorm:"type:varchar(50);not null"`
	Config  datatypes.JSON `gorm:"not null"`
}
