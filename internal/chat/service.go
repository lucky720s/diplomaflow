package chat

import (
	"context"
	"errors"
	"time"

	"github.com/lucky720s/diplomaflow/pkg/logger"
	"gorm.io/gorm"
)

var (
	ErrForbidden    = errors.New("not a participant of this conversation")
	ErrInvalidInput = errors.New("invalid input")
)

type Service struct {
	repo   Repository
	logger *logger.Logger
}

func NewService(repo Repository, log *logger.Logger) *Service {
	return &Service{repo: repo, logger: log}
}

// CreateConversation создаёт беседу. requester всегда включается в участники.
// Для direct-беседы (ровно 2 участника) переиспользуется существующая, если есть.
func (s *Service) CreateConversation(ctx context.Context, requesterID int64, cType, title string, participantIDs []int64) (*Conversation, []int64, error) {
	if requesterID <= 0 {
		return nil, nil, ErrInvalidInput
	}

	// Нормализуем множество участников: requester + переданные, без дублей.
	set := map[int64]struct{}{requesterID: {}}
	for _, id := range participantIDs {
		if id > 0 {
			set[id] = struct{}{}
		}
	}
	ids := make([]int64, 0, len(set))
	for id := range set {
		ids = append(ids, id)
	}
	if len(ids) < 2 {
		return nil, nil, ErrInvalidInput
	}

	if cType == "" {
		if len(ids) == 2 {
			cType = "direct"
		} else {
			cType = "group"
		}
	}

	// Reuse существующего direct-диалога.
	if cType == "direct" && len(ids) == 2 {
		if existing, err := s.repo.FindDirectConversation(ctx, ids[0], ids[1]); err == nil && existing != nil {
			return existing, ids, nil
		} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, err
		}
	}

	conv := &Conversation{Type: cType, Title: title, CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()}
	if err := s.repo.CreateConversation(ctx, conv, ids); err != nil {
		return nil, nil, err
	}
	return conv, ids, nil
}

func (s *Service) ListConversations(ctx context.Context, userID int64) ([]*ConversationView, error) {
	convs, err := s.repo.ListConversationsByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	views := make([]*ConversationView, 0, len(convs))
	for _, c := range convs {
		pids, _ := s.repo.ParticipantIDs(ctx, c.ID)
		last, _ := s.repo.LastMessage(ctx, c.ID)
		unread, _ := s.repo.UnreadCount(ctx, c.ID, userID)
		views = append(views, &ConversationView{
			Conversation:   c,
			ParticipantIDs: pids,
			LastMessage:    last,
			UnreadCount:    unread,
		})
	}
	return views, nil
}

func (s *Service) GetConversation(ctx context.Context, convID, requesterID int64) (*Conversation, []int64, error) {
	if err := s.ensureParticipant(ctx, convID, requesterID); err != nil {
		return nil, nil, err
	}
	conv, err := s.repo.GetConversation(ctx, convID)
	if err != nil {
		return nil, nil, err
	}
	pids, err := s.repo.ParticipantIDs(ctx, convID)
	if err != nil {
		return nil, nil, err
	}
	return conv, pids, nil
}

func (s *Service) SendMessage(ctx context.Context, convID, senderID int64, content string) (*Message, []int64, error) {
	if content == "" {
		return nil, nil, ErrInvalidInput
	}
	if err := s.ensureParticipant(ctx, convID, senderID); err != nil {
		return nil, nil, err
	}
	m := &Message{
		ConversationID: convID,
		SenderID:       senderID,
		Content:        content,
		CreatedAt:      time.Now().UTC(),
	}
	if err := s.repo.CreateMessage(ctx, m); err != nil {
		return nil, nil, err
	}
	pids, err := s.repo.ParticipantIDs(ctx, convID)
	if err != nil {
		return nil, nil, err
	}
	return m, pids, nil
}

func (s *Service) ListMessages(ctx context.Context, convID, requesterID int64, page, pageSize int32) ([]*Message, int64, error) {
	if err := s.ensureParticipant(ctx, convID, requesterID); err != nil {
		return nil, 0, err
	}
	if pageSize <= 0 {
		pageSize = 30
	}
	if page <= 0 {
		page = 1
	}
	offset := (page - 1) * pageSize
	return s.repo.ListMessages(ctx, convID, int(pageSize), int(offset))
}

func (s *Service) MarkRead(ctx context.Context, convID, userID, lastReadMessageID int64) error {
	if err := s.ensureParticipant(ctx, convID, userID); err != nil {
		return err
	}
	return s.repo.MarkRead(ctx, convID, userID, lastReadMessageID)
}

func (s *Service) ensureParticipant(ctx context.Context, convID, userID int64) error {
	ok, err := s.repo.IsParticipant(ctx, convID, userID)
	if err != nil {
		return err
	}
	if !ok {
		return ErrForbidden
	}
	return nil
}

// ConversationView — агрегат для списка бесед.
type ConversationView struct {
	Conversation   *Conversation
	ParticipantIDs []int64
	LastMessage    *Message
	UnreadCount    int64
}
