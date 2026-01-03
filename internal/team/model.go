// internal/team/model.go

package team

import (
	"time"

	"gorm.io/gorm"
)

// Team — модель команды
type Team struct {
	ID        uint         `gorm:"primaryKey"`
	Name      string       `gorm:"not null"`
	ProjectID *int64       `gorm:"uniqueIndex"`
	Members   []TeamMember `gorm:"foreignKey:TeamID"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

// GetProjectID безопасно возвращает project_id
func (t *Team) GetProjectID() int64 {
	if t.ProjectID != nil {
		return *t.ProjectID
	}
	return 0
}

// TeamMember — участник команды
type TeamMember struct {
	ID        uint  `gorm:"primaryKey"`
	TeamID    uint  `gorm:"index;not null"`
	UserID    int64 `gorm:"index;not null"`
	Role      string
	CreatedAt time.Time
}

// TeamInvite — приглашение в команду
type TeamInvite struct {
	ID        int64     `gorm:"primaryKey"`
	TeamID    int64     `gorm:"index;not null"`
	Team      Team      `gorm:"foreignKey:TeamID"` // ← ДОБАВИТЬ ЭТУ СТРОКУ
	TeamName  string    `gorm:"-"`                 // Заполняется при запросе через JOIN
	UserID    int64     `gorm:"index;not null"`
	InviterID int64     `gorm:"not null"`
	Status    string    `gorm:"default:'PENDING'"`
	ExpiresAt time.Time `gorm:"not null"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

// StudentPreview — превью студента для списка доступных студентов
type StudentPreview struct {
	ID       int64
	FullName string
	Email    string
	// Добавляем для совместимости с конвертацией
	FirstName string
	LastName  string
}

// TeamInfo — информация о команде пользователя
type TeamInfo struct {
	TeamID              int64
	Name                string
	ProjectID           int64
	Role                string
	Members             []TeamMember
	MemberCount         int32
	PendingInvitesCount int32
}

// TableName overrides
func (Team) TableName() string       { return "teams" }
func (TeamMember) TableName() string { return "team_members" }
func (TeamInvite) TableName() string { return "team_invites" }
