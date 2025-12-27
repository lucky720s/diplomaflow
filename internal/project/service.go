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
	// Получаем активный workflow для кафедры
	wf, err := s.workflowClient.GetActiveWorkflowByDepartment(ctx, &workflowv1.GetActiveWorkflowByDepartmentRequest{
		DepartmentId: req.DepartmentId,
	})
	if err != nil {
		s.logger.Error("Failed to get active workflow", zap.Error(err))
		return nil, errors.New("failed to fetch workflow configuration")
	}

	// ✅ ИСПРАВЛЕНО: используем States вместо Steps
	if len(wf.States) == 0 {
		return nil, errors.New("workflow has no states configured")
	}

	// ✅ Находим initial state (начальное состояние)
	initialState := s.findInitialState(wf.States)
	if initialState == nil {
		return nil, errors.New("workflow has no initial state defined")
	}

	// Рассчитываем deadline если у состояния есть duration
	var deadlineAt *time.Time
	if initialState.DurationDays > 0 {
		deadline := time.Now().AddDate(0, 0, int(initialState.DurationDays))
		deadlineAt = &deadline
	}

	project := &Project{
		Title:         req.Title,
		Description:   req.Description,
		StudentID:     req.StudentId,
		UniversityID:  req.UniversityId,
		DepartmentID:  req.DepartmentId,
		WorkflowID:    uint(wf.Id),
		WorkflowName:  wf.Name,
		CurrentStepID: strconv.FormatInt(initialState.Id, 10),
		CurrentState:  initialState.Name,
		Status:        "active",
		DeadlineAt:    deadlineAt,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	// Формируем payload для события
	eventPayload := map[string]interface{}{
		"student_id":       req.StudentId,
		"university_id":    req.UniversityId,
		"department_id":    req.DepartmentId,
		"workflow_id":      wf.Id,
		"initial_state_id": initialState.Id,
		"initial_state":    initialState.Name,
		"title":            req.Title,
	}

	if err := s.repo.CreateWithOutbox(ctx, project, "ProjectCreated", "project-events", eventPayload); err != nil {
		s.logger.Error("Failed to create project", zap.Error(err))
		return nil, err
	}

	// Выполняем ON_ENTER actions для начального состояния
	if s.actionExecutor != nil {
		if execErr := s.actionExecutor.ExecuteActions(ctx, initialState.Id, "ON_ENTER", project); execErr != nil {
			s.logger.Warn("Failed to execute ON_ENTER actions for initial state", zap.Error(execErr))
			// Не возвращаем ошибку - проект уже создан
		}
	}

	s.logger.Info("Project created",
		zap.Uint("project_id", project.ID),
		zap.Int64("workflow_id", wf.Id),
		zap.String("initial_state", initialState.Name))

	return &projectv1.CreateProjectResponse{
		ProjectId: int64(project.ID),
		Status:    "active",
	}, nil
}

// findInitialState находит начальное состояние workflow
func (s *Service) findInitialState(states []*workflowv1.State) *workflowv1.State {
	// Сначала ищем state с флагом IsInitial
	for _, state := range states {
		if state.IsInitial {
			return state
		}
	}

	// Если нет явного initial state, берём с минимальным OrderIndex
	if len(states) == 0 {
		return nil
	}

	minState := states[0]
	for _, state := range states[1:] {
		if state.OrderIndex < minState.OrderIndex {
			minState = state
		}
	}
	return minState
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

	if project.Status != "active" {
		return nil, errors.New("cannot perform action on inactive project")
	}

	currentStateID, err := strconv.ParseInt(project.CurrentStepID, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid current state ID: %w", err)
	}

	// Выполняем ON_EXIT actions для текущего состояния
	if s.actionExecutor != nil {
		if execErr := s.actionExecutor.ExecuteActions(ctx, currentStateID, "ON_EXIT", project); execErr != nil {
			s.logger.Warn("Failed to execute ON_EXIT actions", zap.Error(execErr))
		}
	}

	// Получаем информацию о текущем состоянии
	stateInfo, err := s.workflowClient.GetState(ctx, &workflowv1.GetStateRequest{StateId: currentStateID})
	if err != nil {
		return nil, fmt.Errorf("failed to get current state info: %w", err)
	}

	// Получаем обработчик для действия
	handler, err := s.registry.Get(actionName)
	if err != nil {
		return nil, fmt.Errorf("unknown action: %s", actionName)
	}

	// Парсим текущие данные проекта
	currentData, _ := JSONToMap(project.Data)

	// Парсим конфигурацию состояния
	stepConfig := make(map[string]interface{})
	if stateInfo.Config != nil {
		bytes, _ := stateInfo.Config.MarshalJSON()
		if unmarshalErr := json.Unmarshal(bytes, &stepConfig); unmarshalErr != nil {
			s.logger.Warn("Failed to unmarshal state config", zap.Error(unmarshalErr))
		}
	}

	// Выполняем действие
	newData, err := handler.Handle(ctx, currentData, payload, stepConfig)
	if err != nil {
		return nil, fmt.Errorf("action '%s' failed: %w", actionName, err)
	}

	// Сохраняем обновлённые данные
	jsonBytes, _ := json.Marshal(newData)
	project.Data = datatypes.JSON(jsonBytes)

	// Получаем следующее состояние
	nextState, err := s.workflowClient.GetNextState(ctx, &workflowv1.GetNextStateRequest{
		CurrentStateId: currentStateID,
		EventName:      actionName,
	})

	if err == nil && nextState != nil {
		// Обновляем состояние проекта
		project.CurrentStepID = strconv.FormatInt(nextState.Id, 10)
		project.CurrentState = nextState.Name

		// Рассчитываем новый deadline
		if nextState.DurationDays > 0 {
			deadline := time.Now().AddDate(0, 0, int(nextState.DurationDays))
			project.DeadlineAt = &deadline
			project.DeadlineProcessed = false
		} else if nextState.FixedDeadline != nil {
			deadline := nextState.FixedDeadline.AsTime()
			project.DeadlineAt = &deadline
			project.DeadlineProcessed = false
		}

		// Проверяем, является ли новое состояние финальным
		if nextState.IsFinal {
			project.Status = "completed"
		}

		// Выполняем ON_ENTER actions для нового состояния
		if s.actionExecutor != nil {
			if execErr := s.actionExecutor.ExecuteActions(ctx, nextState.Id, "ON_ENTER", project); execErr != nil {
				s.logger.Warn("Failed to execute ON_ENTER actions", zap.Error(execErr))
			}
		}

		s.logger.Info("Project transitioned",
			zap.Int64("project_id", projectID),
			zap.String("from_state", stateInfo.Name),
			zap.String("to_state", nextState.Name),
			zap.String("action", actionName))
	} else {
		s.logger.Debug("No transition found for action",
			zap.Int64("project_id", projectID),
			zap.String("action", actionName),
			zap.Error(err))
	}

	// Записываем историю
	history := &StateHistory{
		ProjectID: project.ID,
		StateName: project.CurrentState,
		Status:    "completed",
		Comment:   fmt.Sprintf("Action: %s", actionName),
		CreatedAt: time.Now(),
	}
	if err := s.repo.AddHistory(ctx, history); err != nil {
		s.logger.Warn("Failed to add history", zap.Error(err))
	}

	// Сохраняем проект
	project.UpdatedAt = time.Now()
	if updateErr := s.repo.Update(ctx, project); updateErr != nil {
		return nil, fmt.Errorf("failed to update project: %w", updateErr)
	}

	return project, nil
}

// GetAvailableActions возвращает доступные действия для проекта
func (s *Service) GetAvailableActions(ctx context.Context, projectID int64, userID int64, userRole string) ([]*AvailableAction, error) {
	project, err := s.repo.GetByID(ctx, uint64(projectID))
	if err != nil {
		return nil, err
	}

	currentStateID, _ := strconv.ParseInt(project.CurrentStepID, 10, 64)

	// Получаем доступные переходы через workflow service
	resp, err := s.workflowClient.GetAvailableTransitions(ctx, &workflowv1.GetAvailableTransitionsRequest{
		ProjectId:      projectID,
		CurrentStateId: currentStateID,
		UserId:         userID,
		UserRole:       userRole,
	})
	if err != nil {
		return nil, err
	}

	var actions []*AvailableAction
	for _, t := range resp.Transitions {
		actions = append(actions, &AvailableAction{
			Name:        t.Transition.EventName,
			DisplayName: t.Transition.DisplayName,
			ButtonLabel: t.Transition.ButtonLabel,
			ButtonColor: t.Transition.ButtonColor,
			CanExecute:  t.CanExecute,
			Reason:      t.BlockedReason,
		})
	}

	return actions, nil
}

// AvailableAction описывает доступное действие
type AvailableAction struct {
	Name        string
	DisplayName string
	ButtonLabel string
	ButtonColor string
	CanExecute  bool
	Reason      string
}
