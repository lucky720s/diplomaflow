package project

import (
	"time"

	"gorm.io/gorm"
)

type Project struct {
	ID           uint64 `gorm:"primaryKey"`
	Title        string `gorm:"not null"`
	Description  string
	StudentID    int64 `gorm:"not null;index"`
	TeamID       int64
	WorkflowName string `gorm:"not null"`
	CurrentState string
	Status       string
	History      []StateHistory `gorm:"foreignKey:ProjectID;constraint:OnDelete:CASCADE"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
	DeletedAt    gorm.DeletedAt `gorm:"index"`
}

type StateHistory struct {
	ID        uint64 `gorm:"primaryKey"`
	ProjectID uint64 `gorm:"index"`
	StateName string
	Status    string
	ChangedBy int64
	Comment   string
	CreatedAt time.Time
}
