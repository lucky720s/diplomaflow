package project

import (
	"time"

	"gorm.io/datatypes"
)

type Project struct {
	ID             int64 `gorm:"primaryKey"`
	Title          string
	WorkflowID     int64
	CurrentStateID int64
	Status         string `gorm:"type:varchar(50)"`
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type ProjectStateData struct {
	ID         int64          `gorm:"primaryKey"`
	ProjectID  int64          `gorm:"not null;index"`
	StateID    int64          `gorm:"not null;index"`
	Status     string         `gorm:"type:varchar(50);not null"`
	Data       datatypes.JSON `gorm:"not null"`
	DeadlineAt *time.Time
	CreatedAt  time.Time
	UpdatedAt  time.Time
}
