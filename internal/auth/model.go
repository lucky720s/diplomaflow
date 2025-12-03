package auth

import (
	"time"

	"gorm.io/gorm"
)

type User struct {
	ID           int64  `gorm:"primaryKey"`
	Email        string `gorm:"unique;not null"`
	Password     string `gorm:"not null"`
	FirstName    string
	LastName     string
	Role         string `gorm:"not null"`
	UniversityID int64
	CreatedAt    time.Time
	UpdatedAt    time.Time
	DeletedAt    gorm.DeletedAt `gorm:"index"`
}
