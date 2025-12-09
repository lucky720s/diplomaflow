package team

import (
	"time"

	"gorm.io/gorm"
)

type Team struct {
	ID        uint64       `gorm:"primaryKey"`
	Name      string       `gorm:"not null"`
	ProjectID *int64       `gorm:"uniqueIndex"`
	Members   []TeamMember `gorm:"foreignKey:TeamID;constraint:OnDelete:CASCADE"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

type TeamMember struct {
	ID        uint64 `gorm:"primaryKey"`
	TeamID    uint64 `gorm:"index"`
	UserID    int64  `gorm:"index"`
	Role      string
	CreatedAt time.Time
}
type TeamInvite struct {
	ID        uint64 `gorm:"primaryKey"`
	TeamID    uint64 `gorm:"index"`
	Team      Team   `gorm:"foreignKey:TeamID"`
	UserID    int64  `gorm:"index"`
	InviterID int64
	Status    string `gorm:"default:'PENDING'"`
	CreatedAt time.Time
	UpdatedAt time.Time
}
