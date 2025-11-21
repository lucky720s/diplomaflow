package workflow

import (
	"context"
	"errors"

	workflowv1 "github.com/lucky720s/diplomaflow/protobuf/workflow/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

type Handler struct {
	workflowv1.UnimplementedWorkflowServiceServer
	repo Repository
}

func NewHandler(repo Repository) *Handler {
	return &Handler{repo: repo}
}
func (h *Handler) CreateWorkflow(ctx context.Context, req *workflowv1.CreateWorkflowRequest) (*workflowv1.CreateWorkflowResponse, error) {
	workflow, err := h.repo.CreateWorkflow(ctx, req.GetName(), req.GetDepartmentId())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "could not create workflow: %v", err)
	}
	return &workflowv1.CreateWorkflowResponse{
		Workflow: &workflowv1.Workflow{
			Id:           workflow.ID,
			Name:         workflow.Name,
			DepartmentId: workflow.DepartmentID,
		}}, nil
}
func (h *Handler) GetWorkflow(ctx context.Context, req *workflowv1.GetWorkflowRequest) (*workflowv1.GetWorkflowResponse, error) {
	workflow, stages, err := h.repo.GetWorkflowByDepartmentID(ctx, req.GetDepartmentId())
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Errorf(codes.NotFound, "workflow not found")
		}
		return nil, status.Errorf(codes.Internal, "could not get workflow: %v", err)
	}
	resStages := make([]*workflowv1.Stage, len(stages))
	for i, stage := range stages {
		resStages[i] = &workflowv1.Stage{
			Id:                stage.ID,
			Name:              stage.Name,
			WorkflowId:        workflow.ID,
			Order:             stage.Order,
			ResponsibleRoleId: stage.ResponsibleRoleID,
			DeadlineDays:      stage.DeadlineDays}
	}
	resWorkflow := &workflowv1.Workflow{
		Id:           workflow.ID,
		Name:         workflow.Name,
		DepartmentId: workflow.DepartmentID}
	return &workflowv1.GetWorkflowResponse{Workflow: resWorkflow, Stages: resStages}, nil
}

func (h *Handler) CreateStage(ctx context.Context, req *workflowv1.CreateStageRequest) (*workflowv1.CreateStageResponse, error) {
	stage, err := h.repo.CreateStage(ctx, req.GetName(), req.GetWorkflowId(), req.GetOrder(), req.GetResponsibleRoleId(), req.GetDeadlineDays())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "could not create stage: %v", err)
	}
	return &workflowv1.CreateStageResponse{
		Stage: &workflowv1.Stage{
			Id:                stage.ID,
			Name:              stage.Name,
			WorkflowId:        stage.WorkflowID,
			Order:             stage.Order,
			ResponsibleRoleId: stage.ResponsibleRoleID,
			DeadlineDays:      stage.DeadlineDays}}, nil
}
func (h *Handler) GetStage(ctx context.Context, req *workflowv1.GetStageRequest) (*workflowv1.GetStageResponse, error) {
	stage, err := h.repo.GetStageByID(ctx, req.GetStageId())
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Errorf(codes.NotFound, "stage not found")
		}
		return nil, status.Errorf(codes.Internal, "could not get stage: %v", err)
	}
	return &workflowv1.GetStageResponse{
		Stage: &workflowv1.Stage{
			Id:                stage.ID,
			Name:              stage.Name,
			WorkflowId:        stage.WorkflowID,
			Order:             stage.Order,
			ResponsibleRoleId: stage.ResponsibleRoleID,
			DeadlineDays:      stage.DeadlineDays}}, nil
}
func (h *Handler) GetNextStage(ctx context.Context, req *workflowv1.GetNextStageRequest) (*workflowv1.GetNextStageResponse, error) {
	currentStage, err := h.repo.GetStageByID(ctx, req.GetCurrentStageId())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "could not get next stage: %v", err)
	}
	nextStage, err := h.repo.GetNextStage(ctx, currentStage.WorkflowID, currentStage.Order)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Errorf(codes.NotFound, "next stage not found")
		}
		return nil, status.Errorf(codes.Internal, "could not get next stage: %v", err)
	}
	return &workflowv1.GetNextStageResponse{
		NextStageId: nextStage.ID}, nil
}
