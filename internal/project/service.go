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

// internal/project/service.go — исправленный CreateProject

func (s *Service) CreateProject(ctx context.Context, req *projectv1.CreateProjectRequest) (*projectv1.CreateProjectResponse, error) {
	// ✅ Получаем полную конфигурацию workflow для кафедры
	wf, err := s.workflowClient.GetActiveWorkflowByDepartment(ctx, &workflowv1.GetActiveWorkflowByDepartmentRequest{
		DepartmentId: req.DepartmentId,
	})
	if err != nil {
		s.logger.Error("Failed to get active workflow", zap.Error(err))
		return nil, errors.New("no active workflow configured for this department")
	}

	if len(wf.States) == 0 {
		return nil, errors.New("workflow has no states configured")
	}

	// Находим initial state
	initialState := s.findInitialState(wf.States)
	if initialState == nil {
		return nil, errors.New("workflow has no initial state defined")
	}

	// ✅ Получаем конфигурацию начального этапа
	stepConfig, err := s.workflowClient.GetStepConfiguration(ctx, &workflowv1.GetStepConfigurationRequest{
		StateId:   initialState.Id,
		ProjectId: 0, // Проект ещё не создан
	})
	if err != nil {
		s.logger.Warn("Failed to get step configuration", zap.Error(err))
	}

	// Рассчитываем deadline
	var deadlineAt *time.Time
	if initialState.DurationDays > 0 {
		deadline := time.Now().AddDate(0, 0, int(initialState.DurationDays))
		deadlineAt = &deadline
	} else if initialState.FixedDeadline != nil {
		t := initialState.FixedDeadline.AsTime()
		deadlineAt = &t
	}

	// ✅ Проверяем требования к команде из конфигурации
	teamRequired := false
	if stepConfig != nil && stepConfig.TeamConfig != nil {
		teamRequired = stepConfig.TeamConfig.MinSize > 1 || !stepConfig.TeamConfig.AllowSolo
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
		"team_required":    teamRequired,
	}

	if stepConfig != nil && stepConfig.TeamConfig != nil {
		eventPayload["team_config"] = map[string]interface{}{
			"min_size":   stepConfig.TeamConfig.MinSize,
			"max_size":   stepConfig.TeamConfig.MaxSize,
			"allow_solo": stepConfig.TeamConfig.AllowSolo,
		}
	}

	if err := s.repo.CreateWithOutbox(ctx, project, "ProjectCreated", "project-events", eventPayload); err != nil {
		s.logger.Error("Failed to create project", zap.Error(err))
		return nil, err
	}

	// Выполняем ON_ENTER actions для начального состояния
	if s.actionExecutor != nil {
		if execErr := s.actionExecutor.ExecuteActions(ctx, initialState.Id, "ON_ENTER", project); execErr != nil {
			s.logger.Warn("Failed to execute ON_ENTER actions for initial state", zap.Error(execErr))
		}
	}

	s.logger.Info("Project created",
		zap.Uint("project_id", project.ID),
		zap.Int64("workflow_id", wf.Id),
		zap.String("initial_state", initialState.Name),
		zap.Bool("team_required", teamRequired))

	return &projectv1.CreateProjectResponse{
		ProjectId: int64(project.ID),
		Status:    "active",
	}, nil
}

// PerformAction с проверкой конфигурации состояния
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

	// ✅ Получаем конфигурацию текущего состояния
	stepConfig, err := s.workflowClient.GetStepConfiguration(ctx, &workflowv1.GetStepConfigurationRequest{
		StateId:   currentStateID,
		ProjectId: projectID,
	})
	if err != nil {
		s.logger.Warn("Failed to get step configuration", zap.Error(err))
	}

	// ✅ Валидируем action в зависимости от типа состояния
	if stepConfig != nil {
		if validationErr := s.validateActionForState(ctx, stepConfig, actionName, payload, project); validationErr != nil {
			return nil, validationErr
		}
	}

	// Выполняем ON_EXIT actions для текущего состояния
	if s.actionExecutor != nil {
		if execErr := s.actionExecutor.ExecuteActions(ctx, currentStateID, "ON_EXIT", project); execErr != nil {
			s.logger.Warn("Failed to execute ON_EXIT actions", zap.Error(execErr))
		}
	}

	// Получаем обработчик для действия
	handler, err := s.registry.Get(actionName)
	if err != nil {
		return nil, fmt.Errorf("unknown action: %s", actionName)
	}

	// Парсим текущие данные проекта
	currentData, _ := JSONToMap(project.Data)

	// Конфигурация состояния для обработчика
	stateConfig := make(map[string]interface{})
	if stepConfig != nil {
		if stepConfig.TeamConfig != nil {
			stateConfig["team_config"] = map[string]interface{}{
				"min_size":   stepConfig.TeamConfig.MinSize,
				"max_size":   stepConfig.TeamConfig.MaxSize,
				"allow_solo": stepConfig.TeamConfig.AllowSolo,
			}
		}
		if stepConfig.FileConfig != nil {
			stateConfig["file_config"] = map[string]interface{}{
				"max_files":     stepConfig.FileConfig.MaxFiles,
				"max_size":      stepConfig.FileConfig.MaxSizeBytes,
				"allowed_types": stepConfig.FileConfig.AllowedExtensions,
			}
		}
		if stepConfig.ReviewConfig != nil {
			stateConfig["review_config"] = map[string]interface{}{
				"reviewer_roles":  stepConfig.ReviewConfig.ReviewerRoles,
				"min_reviewers":   stepConfig.ReviewConfig.MinReviewers,
				"require_comment": stepConfig.ReviewConfig.RequireComment,
			}
		}
	}

	// Выполняем действие
	newData, err := handler.Handle(ctx, currentData, payload, stateConfig)
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
	}

	// Записываем историю
	history := &StateHistory{
		ProjectID: project.ID,
		StateName: project.CurrentState,
		Status:    "completed",
		Comment:   fmt.Sprintf("Action: %s", actionName),
		CreatedAt: time.Now(),
	}
	_ = s.repo.AddHistory(ctx, history)

	// Сохраняем проект
	project.UpdatedAt = time.Now()
	if updateErr := s.repo.Update(ctx, project); updateErr != nil {
		return nil, fmt.Errorf("failed to update project: %w", updateErr)
	}

	return project, nil
}

// validateActionForState проверяет, можно ли выполнить action для данного состояния
func (s *Service) validateActionForState(ctx context.Context, config *workflowv1.StepConfiguration, actionName string, payload map[string]interface{}, project *Project) error {
	switch config.StateType {
	case workflowv1.StateType_TEAM_FORMATION:
		if actionName == "TEAM_FORMED" {
			// Проверяем, что команда соответствует требованиям
			if config.TeamConfig != nil {
				teamID, ok := payload["team_id"].(float64)
				if !ok {
					return errors.New("team_id is required")
				}
				// TODO: Проверить размер команды через Team Service
				_ = teamID
			}
		}

	case workflowv1.StateType_DOCUMENT_UPLOAD:
		if actionName == "DOCUMENTS_SUBMITTED" || actionName == "TASK_UPLOADED" {
			if config.FileConfig != nil {
				fileIDs, ok := payload["file_ids"].([]interface{})
				if !ok || len(fileIDs) == 0 {
					return errors.New("at least one file is required")
				}
				if int32(len(fileIDs)) > config.FileConfig.MaxFiles {
					return fmt.Errorf("too many files, maximum is %d", config.FileConfig.MaxFiles)
				}
			}
		}

	case workflowv1.StateType_REVIEW, workflowv1.StateType_APPROVAL:
		if config.ReviewConfig != nil && config.ReviewConfig.RequireComment {
			comment, ok := payload["comment"].(string)
			if !ok || comment == "" {
				return errors.New("comment is required for review")
			}
		}
	}

	return nil
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
