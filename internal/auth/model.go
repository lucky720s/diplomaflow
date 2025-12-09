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
	DepartmentID int64
	CreatedAt    time.Time
	UpdatedAt    time.Time
	DeletedAt    gorm.DeletedAt `gorm:"index"`
}
type RefreshToken struct {
	ID        uint64    `gorm:"primaryKey"`
	UserID    int64     `gorm:"index;not null"`
	Token     string    `gorm:"uniqueIndex;not null"`
	UserAgent string    `gorm:"type:varchar(255)"`
	ClientIP  string    `gorm:"type:varchar(45)"`
	ExpiresAt time.Time `gorm:"not null"`
	Revoked   bool      `gorm:"default:false"`
	CreatedAt time.Time
}
