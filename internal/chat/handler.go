package chat

import (
	"context"
	"errors"
	"strconv"
	"strings"

	chatv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/chat/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type ChatUseCase interface {
	CreateConversation(ctx context.Context, requesterID int64, cType, title string, participantIDs []int64) (*Conversation, []int64, error)
	ListConversations(ctx context.Context, userID int64) ([]*ConversationView, error)
	GetConversation(ctx context.Context, convID, requesterID int64) (*Conversation, []int64, error)
	SendMessage(ctx context.Context, convID, senderID int64, content string) (*Message, []int64, error)
	ListMessages(ctx context.Context, convID, requesterID int64, page, pageSize int32) ([]*Message, int64, error)
	MarkRead(ctx context.Context, convID, userID, lastReadMessageID int64) error
}

type Handler struct {
	chatv1.UnimplementedChatServiceServer
	service ChatUseCase
}

func NewHandler(service ChatUseCase) *Handler {
	return &Handler{service: service}
}

func userIDFromMD(ctx context.Context) (int64, bool) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return 0, false
	}
	vals := md.Get("x-user-id")
	if len(vals) == 0 || strings.TrimSpace(vals[0]) == "" {
		return 0, false
	}
	id, err := strconv.ParseInt(vals[0], 10, 64)
	if err != nil || id <= 0 {
		return 0, false
	}
	return id, true
}

func mapErr(err error) error {
	switch {
	case errors.Is(err, ErrForbidden):
		return status.Error(codes.PermissionDenied, err.Error())
	case errors.Is(err, ErrInvalidInput):
		return status.Error(codes.InvalidArgument, err.Error())
	default:
		return status.Errorf(codes.Internal, "%v", err)
	}
}

func toPBConversation(c *Conversation, pids []int64) *chatv1.Conversation {
	return &chatv1.Conversation{
		Id:             c.ID,
		Type:           c.Type,
		Title:          c.Title,
		ParticipantIds: pids,
		CreatedAt:      timestamppb.New(c.CreatedAt),
	}
}

func toPBMessage(m *Message) *chatv1.Message {
	if m == nil {
		return nil
	}
	return &chatv1.Message{
		Id:             m.ID,
		ConversationId: m.ConversationID,
		SenderId:       m.SenderID,
		Content:        m.Content,
		CreatedAt:      timestamppb.New(m.CreatedAt),
	}
}

func (h *Handler) CreateConversation(ctx context.Context, req *chatv1.CreateConversationRequest) (*chatv1.CreateConversationResponse, error) {
	userID, ok := userIDFromMD(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "missing x-user-id")
	}
	conv, pids, err := h.service.CreateConversation(ctx, userID, req.Type, req.Title, req.ParticipantIds)
	if err != nil {
		return nil, mapErr(err)
	}
	return &chatv1.CreateConversationResponse{Conversation: toPBConversation(conv, pids)}, nil
}

func (h *Handler) ListConversations(ctx context.Context, _ *chatv1.ListConversationsRequest) (*chatv1.ListConversationsResponse, error) {
	userID, ok := userIDFromMD(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "missing x-user-id")
	}
	views, err := h.service.ListConversations(ctx, userID)
	if err != nil {
		return nil, mapErr(err)
	}
	out := make([]*chatv1.ConversationPreview, 0, len(views))
	for _, v := range views {
		out = append(out, &chatv1.ConversationPreview{
			Conversation: toPBConversation(v.Conversation, v.ParticipantIDs),
			LastMessage:  toPBMessage(v.LastMessage),
			UnreadCount:  v.UnreadCount,
		})
	}
	return &chatv1.ListConversationsResponse{Conversations: out}, nil
}

func (h *Handler) GetConversation(ctx context.Context, req *chatv1.GetConversationRequest) (*chatv1.GetConversationResponse, error) {
	userID, ok := userIDFromMD(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "missing x-user-id")
	}
	conv, pids, err := h.service.GetConversation(ctx, req.ConversationId, userID)
	if err != nil {
		return nil, mapErr(err)
	}
	return &chatv1.GetConversationResponse{Conversation: toPBConversation(conv, pids)}, nil
}

func (h *Handler) SendMessage(ctx context.Context, req *chatv1.SendMessageRequest) (*chatv1.SendMessageResponse, error) {
	userID, ok := userIDFromMD(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "missing x-user-id")
	}
	m, pids, err := h.service.SendMessage(ctx, req.ConversationId, userID, req.Content)
	if err != nil {
		return nil, mapErr(err)
	}
	return &chatv1.SendMessageResponse{Message: toPBMessage(m), ParticipantIds: pids}, nil
}

func (h *Handler) ListMessages(ctx context.Context, req *chatv1.ListMessagesRequest) (*chatv1.ListMessagesResponse, error) {
	userID, ok := userIDFromMD(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "missing x-user-id")
	}
	msgs, total, err := h.service.ListMessages(ctx, req.ConversationId, userID, req.Page, req.PageSize)
	if err != nil {
		return nil, mapErr(err)
	}
	out := make([]*chatv1.Message, 0, len(msgs))
	for _, m := range msgs {
		out = append(out, toPBMessage(m))
	}
	return &chatv1.ListMessagesResponse{Messages: out, TotalCount: total}, nil
}

func (h *Handler) MarkRead(ctx context.Context, req *chatv1.MarkReadRequest) (*emptypb.Empty, error) {
	userID, ok := userIDFromMD(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "missing x-user-id")
	}
	if err := h.service.MarkRead(ctx, req.ConversationId, userID, req.LastReadMessageId); err != nil {
		return nil, mapErr(err)
	}
	return &emptypb.Empty{}, nil
}
