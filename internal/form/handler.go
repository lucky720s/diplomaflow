package form

import (
	"context"
	"encoding/json"

	"github.com/lucky720s/diplomaflow/pkg/logger"
	formv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/form/v1"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Handler struct {
	formv1.UnimplementedFormServiceServer
	service *Service
	logger  *logger.Logger
}

func NewHandler(service *Service, log *logger.Logger) *Handler {
	return &Handler{
		service: service,
		logger:  log,
	}
}

func (h *Handler) SubmitForm(ctx context.Context, req *formv1.SubmitFormRequest) (*formv1.SubmitFormResponse, error) {
	if req.Data == nil {
		return nil, status.Error(codes.InvalidArgument, "data is required")
	}

	dataMap := req.Data.AsMap()

	id, err := h.service.SubmitForm(ctx, req.ProjectId, req.StepId, req.UserId, dataMap)
	if err != nil {
		h.logger.Error("SubmitForm failed", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "failed to submit form: %v", err)
	}

	return &formv1.SubmitFormResponse{
		SubmissionId: id,
		Success:      true,
	}, nil
}

func (h *Handler) GetFormSubmission(ctx context.Context, req *formv1.GetFormSubmissionRequest) (*formv1.GetFormSubmissionResponse, error) {
	sub, err := h.service.GetFormSubmission(ctx, req.SubmissionId)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "submission not found: %v", err)
	}

	var dataMap map[string]interface{}
	if err := json.Unmarshal(sub.Data, &dataMap); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to unmarshal data: %v", err)
	}

	pbStruct, err := structpb.NewStruct(dataMap)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create proto struct: %v", err)
	}

	return &formv1.GetFormSubmissionResponse{
		SubmissionId: sub.ID,
		ProjectId:    sub.ProjectID,
		StepId:       sub.StepID,
		UserId:       sub.UserID,
		Data:         pbStruct,
		CreatedAt:    timestamppb.New(sub.CreatedAt),
	}, nil
}

func (h *Handler) ListProjectForms(ctx context.Context, req *formv1.ListProjectFormsRequest) (*formv1.ListProjectFormsResponse, error) {
	subs, err := h.service.ListProjectForms(ctx, req.ProjectId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list forms: %v", err)
	}

	var previews []*formv1.FormPreview
	for _, sub := range subs {
		previews = append(previews, &formv1.FormPreview{
			SubmissionId: sub.ID,
			StepId:       sub.StepID,
			CreatedAt:    timestamppb.New(sub.CreatedAt),
		})
	}

	return &formv1.ListProjectFormsResponse{Forms: previews}, nil
}
