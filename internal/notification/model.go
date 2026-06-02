package notification

import (
	"time"

	"gorm.io/gorm"
)

type Notification struct {
	ID        int64  `gorm:"primaryKey"`
	UserID    int64  `gorm:"index;not null"`
	Title     string `gorm:"not null"`
	Message   string
	Link      string
	Type      string `gorm:"size:50"`
	IsRead    bool   `gorm:"default:false;index"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

// DeviceToken — push-токен устройства (FCM) для мобильного клиента.
type DeviceToken struct {
	ID        int64  `gorm:"primaryKey"`
	UserID    int64  `gorm:"index;not null"`
	Token     string `gorm:"uniqueIndex;size:512;not null"`
	Platform  string `gorm:"size:20;not null;default:android"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (DeviceToken) TableName() string { return "device_tokens" }
