package university

import (
	"time"

	"gorm.io/gorm"
)

type University struct {
	ID        int64  `gorm:"primaryKey"`
	Name      string `gorm:"unique;not null"`
	ShortName string `gorm:"unique;not null"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

type Department struct {
	ID           int64  `gorm:"primaryKey"`
	Name         string `gorm:"not null"`
	UniversityID int64  `gorm:"index;not null"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
	DeletedAt    gorm.DeletedAt `gorm:"index"`
}
