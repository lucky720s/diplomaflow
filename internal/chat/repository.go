package chat

import (
	"context"

	"gorm.io/gorm"
)

type Repository interface {
	CreateConversation(ctx context.Context, conv *Conversation, participantIDs []int64) error
	GetConversation(ctx context.Context, id int64) (*Conversation, error)
	ParticipantIDs(ctx context.Context, convID int64) ([]int64, error)
	IsParticipant(ctx context.Context, convID, userID int64) (bool, error)
	// FindDirectConversation ищет существующий 1-на-1 диалог между двумя пользователями.
	FindDirectConversation(ctx context.Context, userA, userB int64) (*Conversation, error)

	ListConversationsByUser(ctx context.Context, userID int64) ([]*Conversation, error)
	LastMessage(ctx context.Context, convID int64) (*Message, error)
	UnreadCount(ctx context.Context, convID, userID int64) (int64, error)

	CreateMessage(ctx context.Context, m *Message) error
	ListMessages(ctx context.Context, convID int64, limit, offset int) ([]*Message, int64, error)
	MarkRead(ctx context.Context, convID, userID, lastReadMessageID int64) error
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

func (r *repository) CreateConversation(ctx context.Context, conv *Conversation, participantIDs []int64) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(conv).Error; err != nil {
			return err
		}
		for _, uid := range participantIDs {
			p := &Participant{ConversationID: conv.ID, UserID: uid}
			if err := tx.Create(p).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *repository) GetConversation(ctx context.Context, id int64) (*Conversation, error) {
	var c Conversation
	if err := r.db.WithContext(ctx).First(&c, id).Error; err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *repository) ParticipantIDs(ctx context.Context, convID int64) ([]int64, error) {
	var ids []int64
	err := r.db.WithContext(ctx).Model(&Participant{}).
		Where("conversation_id = ?", convID).
		Pluck("user_id", &ids).Error
	return ids, err
}

func (r *repository) IsParticipant(ctx context.Context, convID, userID int64) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&Participant{}).
		Where("conversation_id = ? AND user_id = ?", convID, userID).
		Count(&count).Error
	return count > 0, err
}

func (r *repository) FindDirectConversation(ctx context.Context, userA, userB int64) (*Conversation, error) {
	var conv Conversation
	// Беседа типа direct, в которой ровно два участника — userA и userB.
	err := r.db.WithContext(ctx).
		Model(&Conversation{}).
		Joins("JOIN conversation_participants p ON p.conversation_id = conversations.id").
		Where("conversations.type = ?", "direct").
		Where("p.user_id IN ?", []int64{userA, userB}).
		Group("conversations.id").
		Having("COUNT(DISTINCT p.user_id) = 2").
		First(&conv).Error
	if err != nil {
		return nil, err
	}
	return &conv, nil
}

func (r *repository) ListConversationsByUser(ctx context.Context, userID int64) ([]*Conversation, error) {
	var convs []*Conversation
	err := r.db.WithContext(ctx).
		Model(&Conversation{}).
		Joins("JOIN conversation_participants p ON p.conversation_id = conversations.id").
		Where("p.user_id = ?", userID).
		Order("conversations.updated_at DESC").
		Find(&convs).Error
	return convs, err
}

func (r *repository) LastMessage(ctx context.Context, convID int64) (*Message, error) {
	var m Message
	err := r.db.WithContext(ctx).
		Where("conversation_id = ?", convID).
		Order("id DESC").
		First(&m).Error
	if err != nil {
		return nil, err
	}
	return &m, nil
}

// UnreadCount — число сообщений беседы новее last_read_message_id пользователя,
// не считая его собственных.
func (r *repository) UnreadCount(ctx context.Context, convID, userID int64) (int64, error) {
	var lastRead int64
	if err := r.db.WithContext(ctx).Model(&Participant{}).
		Where("conversation_id = ? AND user_id = ?", convID, userID).
		Pluck("last_read_message_id", &lastRead).Error; err != nil {
		return 0, err
	}
	var count int64
	err := r.db.WithContext(ctx).Model(&Message{}).
		Where("conversation_id = ? AND id > ? AND sender_id <> ?", convID, lastRead, userID).
		Count(&count).Error
	return count, err
}

func (r *repository) CreateMessage(ctx context.Context, m *Message) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(m).Error; err != nil {
			return err
		}
		// Поднимаем updated_at беседы, чтобы список сортировался по свежести.
		return tx.Model(&Conversation{}).
			Where("id = ?", m.ConversationID).
			Update("updated_at", m.CreatedAt).Error
	})
}

func (r *repository) ListMessages(ctx context.Context, convID int64, limit, offset int) ([]*Message, int64, error) {
	var msgs []*Message
	var total int64
	q := r.db.WithContext(ctx).Model(&Message{}).Where("conversation_id = ?", convID)
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err := q.Order("id DESC").Limit(limit).Offset(offset).Find(&msgs).Error
	return msgs, total, err
}

func (r *repository) MarkRead(ctx context.Context, convID, userID, lastReadMessageID int64) error {
	return r.db.WithContext(ctx).Model(&Participant{}).
		Where("conversation_id = ? AND user_id = ?", convID, userID).
		Update("last_read_message_id", lastReadMessageID).Error
}
