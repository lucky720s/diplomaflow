package project

import (
	"time"

	"gorm.io/datatypes"
)

type Project struct {
	ID                uint   `gorm:"primaryKey"`
	Title             string `gorm:"not null"`
	Description       string
	StudentID         int64 `gorm:"not null"`
	UniversityID      int64 `gorm:"not null"`
	DepartmentID      int64 `gorm:"not null;index"`
	TeamID            int64
	WorkflowID        uint `gorm:"not null"`
	WorkflowName      string
	CurrentStepID     string `gorm:"not null"`
	CurrentState      string
	Status            string         `gorm:"default:'active'"`
	Data              datatypes.JSON `gorm:"type:jsonb"`
	DeadlineAt        *time.Time     `gorm:"index"`
	DeadlineProcessed bool           `gorm:"default:false"`
	History           []StateHistory `gorm:"foreignKey:ProjectID"`
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

type StateHistory struct {
	ID        uint `gorm:"primaryKey"`
	ProjectID uint
	StateName string
	Status    string
	ChangedBy int64
	Comment   string
	CreatedAt time.Time
}

type OutboxEvent struct {
	ID        uint           `gorm:"primaryKey"`
	Topic     string         `gorm:"not null"`
	EventType string         `gorm:"not null"`
	Payload   datatypes.JSON `gorm:"not null"`
	Status    string         `gorm:"default:'pending'"`
	CreatedAt time.Time
}
