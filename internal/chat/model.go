package chat

import (
	"time"

	"gorm.io/gorm"
)

type Conversation struct {
	ID        int64  `gorm:"primaryKey"`
	Type      string `gorm:"size:20;not null;default:direct"`
	Title     string `gorm:"size:255;not null;default:''"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (Conversation) TableName() string { return "conversations" }

type Participant struct {
	ID                int64 `gorm:"primaryKey"`
	ConversationID    int64 `gorm:"index;not null;uniqueIndex:uq_conversation_participant"`
	UserID            int64 `gorm:"index;not null;uniqueIndex:uq_conversation_participant"`
	LastReadMessageID int64 `gorm:"not null;default:0"`
	CreatedAt         time.Time
}

func (Participant) TableName() string { return "conversation_participants" }

type Message struct {
	ID             int64  `gorm:"primaryKey"`
	ConversationID int64  `gorm:"index;not null"`
	SenderID       int64  `gorm:"not null"`
	Content        string `gorm:"not null"`
	CreatedAt      time.Time
	DeletedAt      gorm.DeletedAt `gorm:"index"`
}

func (Message) TableName() string { return "chat_messages" }
