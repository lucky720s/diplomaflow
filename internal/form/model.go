package form

import (
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type FormSubmission struct {
	ID        string         `gorm:"primaryKey"`
	ProjectID int64          `gorm:"index"`
	StepID    int64          `gorm:"index"`
	UserID    int64          `gorm:"index"`
	Data      datatypes.JSON `gorm:"type:jsonb"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}
