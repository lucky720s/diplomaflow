package project

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	projectv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/project/v1"
	workflowv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/workflow/v1"
	"go.uber.org/zap"
	"gorm.io/datatypes"
)

type Service struct {
	repo           Repository
	workflowClient workflowv1.WorkflowServiceClient
	registry       *ProcessorRegistry
	actionExecutor *StateActionExecutor
	logger         *zap.Logger
}

func NewService(
	repo Repository,
	wfClient workflowv1.WorkflowServiceClient,
	registry *ProcessorRegistry,
	actionExecutor *StateActionExecutor,
	logger *zap.Logger,
) *Service {
	return &Service{
		repo:           repo,
		workflowClient: wfClient,
		registry:       registry,
		actionExecutor: actionExecutor,
		logger:         logger,
	}
}

func (s *Service) CreateProject(ctx context.Context, req *projectv1.CreateProjectRequest) (*projectv1.CreateProjectResponse, error) {
	wf, err := s.workflowClient.GetActiveWorkflowByDepartment(ctx, &workflowv1.GetActiveWorkflowByDepartmentRequest{
		DepartmentId: req.DepartmentId,
	})
	if err != nil {
		s.logger.Error("Failed to get active workflow", zap.Error(err))
		return nil, errors.New("failed to fetch workflow configuration")
	}
	if len(wf.Steps) == 0 {
		return nil, errors.New("workflow has no steps")
	}
	initialStep := wf.Steps[0]
	project := &Project{
		Title:         req.Title,
		Description:   req.Description,
		StudentID:     req.StudentId,
		UniversityID:  req.UniversityId,
		DepartmentID:  req.DepartmentId,
		WorkflowID:    uint(wf.Id),
		WorkflowName:  wf.Name,
		CurrentStepID: strconv.FormatInt(initialStep.Id, 10),
		CurrentState:  initialStep.Name,
		Status:        "active",
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
	eventPayload := map[string]interface{}{
		"student_id":    req.StudentId,
		"university_id": req.UniversityId,
		"title":         req.Title,
	}
	if err := s.repo.CreateWithOutbox(ctx, project, "ProjectCreated", "project-events", eventPayload); err != nil {
		s.logger.Error("Failed to create project", zap.Error(err))
		return nil, err
	}
	return &projectv1.CreateProjectResponse{ProjectId: int64(project.ID), Status: "active"}, nil
}

func (s *Service) GetProject(ctx context.Context, id uint64) (*Project, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *Service) GetStudentProjects(ctx context.Context, studentID int64) ([]*Project, error) {
	return s.repo.ListByStudent(ctx, studentID)
}

func (s *Service) PerformAction(ctx context.Context, projectID int64, actionName string, payload map[string]interface{}) (*Project, error) {
	project, err := s.repo.GetByID(ctx, uint64(projectID))
	if err != nil {
		return nil, fmt.Errorf("project not found: %w", err)
	}
	currentStateID, _ := strconv.ParseInt(project.CurrentStepID, 10, 64)
	if s.actionExecutor != nil {
		if execErr := s.actionExecutor.ExecuteActions(ctx, currentStateID, "ON_EXIT", project); execErr != nil {
			s.logger.Warn("Failed to execute ON_EXIT actions", zap.Error(execErr))
		}
	}
	stateInfo, err := s.workflowClient.GetState(ctx, &workflowv1.GetStateRequest{StateId: currentStateID})
	if err != nil {
		return nil, fmt.Errorf("failed to get state info: %w", err)
	}
	handler, err := s.registry.Get(actionName)
	if err != nil {
		return nil, err
	}
	currentData, _ := JSONToMap(project.Data)
	stepConfig := make(map[string]interface{})
	if stateInfo.Config != nil {
		bytes, _ := stateInfo.Config.MarshalJSON()
		if unmarshalErr := json.Unmarshal(bytes, &stepConfig); unmarshalErr != nil {
			s.logger.Error("failed to unmarshal step config", zap.Error(unmarshalErr))
		}
	}
	newData, err := handler.Handle(ctx, currentData, payload, stepConfig)
	if err != nil {
		return nil, fmt.Errorf("action failed: %w", err)
	}
	jsonBytes, _ := json.Marshal(newData)
	project.Data = datatypes.JSON(jsonBytes)
	nextState, err := s.workflowClient.GetNextState(ctx, &workflowv1.GetNextStateRequest{
		CurrentStateId: currentStateID,
		EventName:      actionName,
	})
	if err == nil && nextState != nil {
		project.CurrentStepID = strconv.FormatInt(nextState.Id, 10)
		project.CurrentState = nextState.Name
		if nextState.DurationDays > 0 {
			deadline := time.Now().AddDate(0, 0, int(nextState.DurationDays))
			project.DeadlineAt = &deadline
			project.DeadlineProcessed = false
		}
		if s.actionExecutor != nil {
			if execErr := s.actionExecutor.ExecuteActions(ctx, nextState.Id, "ON_ENTER", project); execErr != nil {
				s.logger.Warn("Failed to execute ON_ENTER actions", zap.Error(execErr))
			}
		}
	}
	history := &StateHistory{
		ProjectID: project.ID,
		StateName: project.CurrentState,
		Status:    "completed",
		CreatedAt: time.Now(),
	}
	_ = s.repo.AddHistory(ctx, history)
	if updateErr := s.repo.Update(ctx, project); updateErr != nil {
		return nil, updateErr
	}
	return project, nil
}
