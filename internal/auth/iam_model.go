package auth

import (
	"time"

	"gorm.io/gorm"
)

type DepartmentRole struct {
	ID           int64  `gorm:"primaryKey"`
	Slug         string `gorm:"column:name;not null"`
	DepartmentID int64  `gorm:"not null;index"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
	DeletedAt    gorm.DeletedAt `gorm:"index"`
}

func (DepartmentRole) TableName() string { return "roles" }

type UserRoleAssignment struct {
	ID int64 `gorm:"primaryKey"`

	UserID       int64  `gorm:"not null;index"`
	RoleID       int64  `gorm:"not null;index"`
	DepartmentID *int64 `gorm:"index"`
	WorkflowID   *int64 `gorm:"index"`

	AssignedBy *int64 `gorm:"index"`
	AssignedAt time.Time
	RevokedAt  *time.Time
	Comment    string `gorm:"type:text"`
}

func (UserRoleAssignment) TableName() string { return "user_role_assignments" }
