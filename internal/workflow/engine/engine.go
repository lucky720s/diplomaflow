package engine

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/lucky720s/diplomaflow/internal/workflow"
	"github.com/lucky720s/diplomaflow/internal/workflow/plugins"
	"go.uber.org/zap"
)

var (
	ErrWorkflowNotFound     = errors.New("workflow not found")
	ErrStateNotFound        = errors.New("state not found")
	ErrTransitionNotFound   = errors.New("transition not found")
	ErrTransitionNotAllowed = errors.New("transition not allowed")
	ErrConditionFailed      = errors.New("transition condition failed")
)

type WorkflowEngine struct {
	repo   workflow.Repository
	logger *zap.Logger
}

func NewWorkflowEngine(repo workflow.Repository, logger *zap.Logger) *WorkflowEngine {
	return &WorkflowEngine{
		repo:   repo,
		logger: logger,
	}
}

// GetAvailableTransitions возвращает доступные переходы для проекта
func (e *WorkflowEngine) GetAvailableTransitions(
	ctx context.Context,
	projectID, currentStateID, userID int64,
	userRole string,
	projectData map[string]interface{},
) ([]*AvailableTransition, error) {
	transitions, err := e.repo.GetTransitionsFromState(ctx, currentStateID)
	if err != nil {
		return nil, err
	}

	var available []*AvailableTransition
	for _, tr := range transitions {
		at := &AvailableTransition{
			Transition: &tr,
			CanExecute: true,
		}

		// Проверяем условия
		conditions, err := e.parseConditions(tr.Conditions)
		if err != nil {
			e.logger.Warn("Failed to parse conditions", zap.Error(err))
			continue
		}

		for _, cond := range conditions {
			if !e.evaluateCondition(cond, userID, userRole, projectData) {
				at.CanExecute = false
				at.BlockedReason = fmt.Sprintf("Condition '%s' not met", cond.Type)
				at.MissingRequirements = append(at.MissingRequirements, cond.Field)
			}
		}

		available = append(available, at)
	}

	return available, nil
}

func (e *WorkflowEngine) ExecuteTransition(ctx context.Context, req *ExecuteTransitionRequest) (*ExecuteTransitionResult, error) {
	// 1. Получаем transition
	transition, err := e.repo.GetTransition(ctx, req.TransitionID)
	if err != nil {
		return nil, ErrTransitionNotFound
	}

	// 2. Проверяем, что текущее состояние соответствует from_state
	if transition.FromStateID != req.CurrentStateID {
		return nil, ErrTransitionNotAllowed
	}

	// 3. Проверяем условия
	conditions, _ := e.parseConditions(transition.Conditions)
	for _, cond := range conditions {
		if !e.evaluateCondition(cond, req.UserID, req.UserRole, req.ProjectData) {
			return nil, fmt.Errorf("%w: %s", ErrConditionFailed, cond.Type)
		}
	}

	// 4. Получаем from и to states
	fromState, err := e.repo.GetState(ctx, transition.FromStateID)
	if err != nil {
		return nil, err
	}

	toState, err := e.repo.GetState(ctx, transition.ToStateID)
	if err != nil {
		return nil, err
	}

	result := &ExecuteTransitionResult{
		Success:      true,
		NewStateID:   toState.ID,
		NewStateName: toState.Name,
	}

	// 5. Создаём контекст для actions
	actionCtx := &plugins.ActionContext{
		ProjectID:     req.ProjectID,
		UserID:        req.UserID,
		DepartmentID:  req.DepartmentID,
		ProjectData:   req.ProjectData,
		PreviousState: fromState.Name,
		NewState:      toState.Name,
		TransitionID:  transition.ID,
		Payload:       req.Payload,
		Metadata:      make(map[string]interface{}),
	}

	// 6. Выполняем ON_EXIT actions для текущего состояния
	exitActions, _ := e.repo.GetStateActionsByTrigger(ctx, fromState.ID, plugins.TriggerOnExit)
	for _, action := range exitActions {
		actionCtx.StateID = fromState.ID
		actionCtx.Trigger = plugins.TriggerOnExit
		actionCtx.Config = e.parseConfig(action.Config)

		actionResult := e.executeAction(ctx, &action, actionCtx)
		// ✅ ИСПРАВЛЕНО: используем поле action.IsOptional вместо метода
		if !actionResult.Success && !action.IsOptional {
			return nil, fmt.Errorf("exit action '%s' failed: %w", action.Name, actionResult.Error)
		}
		result.ExecutedActions = append(result.ExecutedActions, action.Name)
	}

	// 7. Выполняем ON_ENTER actions для нового состояния
	enterActions, _ := e.repo.GetStateActionsByTrigger(ctx, toState.ID, plugins.TriggerOnEnter)
	for _, action := range enterActions {
		actionCtx.StateID = toState.ID
		actionCtx.Trigger = plugins.TriggerOnEnter
		actionCtx.Config = e.parseConfig(action.Config)

		actionResult := e.executeAction(ctx, &action, actionCtx)
		// ✅ ИСПРАВЛЕНО: используем поле action.IsOptional вместо метода
		if !actionResult.Success && !action.IsOptional {
			e.logger.Error("Enter action failed",
				zap.Int64("action_id", action.ID),
				zap.String("action_name", action.Name),
				zap.Error(actionResult.Error))
			// Не возвращаем ошибку, переход уже произошёл
		}
		result.ExecutedActions = append(result.ExecutedActions, action.Name)
	}

	// 8. Рассчитываем дедлайн для нового состояния
	if toState.DurationDays > 0 {
		deadline := time.Now().AddDate(0, 0, int(toState.DurationDays))
		result.NewDeadline = &deadline
	} else if toState.FixedDeadline != nil {
		result.NewDeadline = toState.FixedDeadline
	}

	e.logger.Info("Transition executed",
		zap.Int64("project_id", req.ProjectID),
		zap.Int64("transition_id", req.TransitionID),
		zap.String("from_state", fromState.Name),
		zap.String("to_state", toState.Name),
		zap.Int("actions_executed", len(result.ExecutedActions)))

	return result, nil
}

func (e *WorkflowEngine) executeAction(ctx context.Context, action *workflow.StateAction, actx *plugins.ActionContext) *plugins.ActionResult {
	// Получаем плагин из реестра
	plugin, err := plugins.Get(action.Type)
	if err != nil {
		e.logger.Warn("Action plugin not found", zap.String("type", action.Type))
		return &plugins.ActionResult{Success: false, Error: err}
	}

	// Выполняем действие
	result := plugin.Execute(ctx, actx)

	if !result.Success {
		e.logger.Warn("Action execution failed",
			zap.String("action_type", action.Type),
			zap.String("action_name", action.Name),
			zap.Error(result.Error))
	}

	return result
}

func (e *WorkflowEngine) parseConditions(data []byte) ([]TransitionCondition, error) {
	var conditions []TransitionCondition
	if len(data) == 0 {
		return conditions, nil
	}
	err := json.Unmarshal(data, &conditions)
	return conditions, err
}

func (e *WorkflowEngine) evaluateCondition(cond TransitionCondition, userID int64, userRole string, data map[string]interface{}) bool {
	switch cond.Type {
	case "role":
		allowedRoles, ok := cond.Value.([]interface{})
		if !ok {
			if role, ok := cond.Value.(string); ok {
				return role == userRole
			}
			return false
		}
		for _, r := range allowedRoles {
			roleStr, ok := r.(string)
			if !ok {
				continue
			}
			if roleStr == userRole {
				return true
			}
		}
		return false

	case "field":
		fieldValue, ok := data[cond.Field]
		if !ok {
			return cond.Operator == "not_exists" || cond.Operator == "empty"
		}
		return e.compareValues(fieldValue, cond.Operator, cond.Value)

	case "user":
		if cond.Field == "id" {
			return e.compareValues(float64(userID), cond.Operator, cond.Value)
		}
		return true

	default:
		return true
	}
}

func (e *WorkflowEngine) compareValues(actual interface{}, operator string, expected interface{}) bool {
	switch operator {
	case "eq", "equals", "==":
		return actual == expected
	case "ne", "not_equals", "!=":
		return actual != expected
	case "gt", ">":
		return toFloat(actual) > toFloat(expected)
	case "gte", ">=":
		return toFloat(actual) >= toFloat(expected)
	case "lt", "<":
		return toFloat(actual) < toFloat(expected)
	case "lte", "<=":
		return toFloat(actual) <= toFloat(expected)
	case "in":
		arr, ok := expected.([]interface{})
		if !ok {
			return false
		}
		for _, v := range arr {
			if actual == v {
				return true
			}
		}
		return false
	case "not_in":
		arr, ok := expected.([]interface{})
		if !ok {
			return true
		}
		for _, v := range arr {
			if actual == v {
				return false
			}
		}
		return true
	case "exists", "not_empty":
		return actual != nil && actual != ""
	case "not_exists", "empty":
		return actual == nil || actual == ""
	default:
		return true
	}
}

func toFloat(v interface{}) float64 {
	switch val := v.(type) {
	case float64:
		return val
	case float32:
		return float64(val)
	case int:
		return float64(val)
	case int64:
		return float64(val)
	case int32:
		return float64(val)
	default:
		return 0
	}
}

func (e *WorkflowEngine) parseConfig(data []byte) map[string]interface{} {
	var config map[string]interface{}
	if err := json.Unmarshal(data, &config); err != nil || config == nil {
		config = make(map[string]interface{})
	}
	return config
}

// ==================== Types ===================

type AvailableTransition struct {
	Transition          *workflow.Transition
	CanExecute          bool
	BlockedReason       string
	MissingRequirements []string
}

type ExecuteTransitionRequest struct {
	ProjectID      int64
	TransitionID   int64
	CurrentStateID int64
	UserID         int64
	UserRole       string
	DepartmentID   int64
	ProjectData    map[string]interface{}
	Payload        map[string]interface{}
}

type ExecuteTransitionResult struct {
	Success         bool
	NewStateID      int64
	NewStateName    string
	NewDeadline     *time.Time
	ExecutedActions []string
	Error           string
}

type TransitionCondition struct {
	Type     string      `json:"type"`
	Field    string      `json:"field"`
	Operator string      `json:"operator"`
	Value    interface{} `json:"value"`
}
