package postcommit

import (
	"context"

	workflowv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/workflow/v1"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type GRPCServer struct {
	workflowv1.UnimplementedWorkflowActionsServiceServer
	worker *Worker
	logger *zap.Logger
}

func NewGRPCServer(worker *Worker, logger *zap.Logger) *GRPCServer {
	return &GRPCServer{worker: worker, logger: logger}
}

func (s *GRPCServer) ProcessWorkflowActionEvent(
	ctx context.Context,
	req *workflowv1.ProcessWorkflowActionEventRequest,
) (*workflowv1.ProcessWorkflowActionEventResponse, error) {
	if req.GetEventType() == "" {
		return nil, status.Error(codes.InvalidArgument, "event_type is required")
	}
	if len(req.GetPayloadJson()) == 0 {
		return nil, status.Error(codes.InvalidArgument, "payload_json is required")
	}
	// topic оставляем просто для трассировки/совместимости
	if req.GetTopic() != "" && req.GetTopic() != "workflow-actions" {
		return nil, status.Errorf(codes.InvalidArgument, "unexpected topic: %s", req.GetTopic())
	}

	if err := s.worker.Handle(ctx, req.GetEventType(), req.GetPayloadJson()); err != nil {
		return nil, err
	}
	return &workflowv1.ProcessWorkflowActionEventResponse{Accepted: true}, nil
}
