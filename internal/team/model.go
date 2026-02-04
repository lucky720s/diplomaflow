package team

import (
	"time"

	"gorm.io/gorm"
)

const (
	RoleLeader = "leader"
	RoleMember = "member"
)

const (
	InviteStatusPending  = "PENDING"
	InviteStatusAccepted = "ACCEPTED"
	InviteStatusDeclined = "DECLINED"
	InviteStatusExpired  = "EXPIRED"
)

type Team struct {
	ID        int64  `gorm:"primaryKey"`
	Name      string `gorm:"not null"`
	ProjectID int64  `gorm:"uniqueIndex"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

type TeamMember struct {
	ID        int64  `gorm:"primaryKey"`
	TeamID    int64  `gorm:"index;not null"`
	UserID    int64  `gorm:"index;not null"`
	Role      string `gorm:"type:varchar(50);default:'member'"`
	CreatedAt time.Time
}

type TeamInvite struct {
	ID        int64  `gorm:"primaryKey"`
	TeamID    int64  `gorm:"index;not null"`
	UserID    int64  `gorm:"index;not null"`
	InviterID int64  `gorm:"not null"`
	Status    string `gorm:"type:varchar(50);default:'PENDING'"`
	ExpiresAt time.Time
	CreatedAt time.Time
	UpdatedAt time.Time
}
