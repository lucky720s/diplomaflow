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
	ErrTransitionNotFound   = errors.New("transition not found")
	ErrTransitionNotAllowed = errors.New("transition not allowed")
	ErrConditionFailed      = errors.New("transition condition failed")
)

type WorkflowEngine struct {
	repo   workflow.Repository
	logger *zap.Logger
}

func NewWorkflowEngine(repo workflow.Repository, logger *zap.Logger) *WorkflowEngine {
	return &WorkflowEngine{repo: repo, logger: logger}
}

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

		conditions, err := e.parseConditions(tr.Conditions)
		if err != nil {
			e.logger.Warn("failed to parse transition conditions", zap.Error(err))
			continue
		}

		for _, cond := range conditions {
			if !e.evaluateCondition(cond, userID, userRole, projectData) {
				at.CanExecute = false
				at.BlockedReason = fmt.Sprintf("condition '%s' not met", cond.Type)
				at.MissingRequirements = append(at.MissingRequirements, cond.Field)
			}
		}

		available = append(available, at)
	}

	return available, nil
}

func (e *WorkflowEngine) ExecuteTransition(ctx context.Context, req *ExecuteTransitionRequest) (*ExecuteTransitionResult, error) {
	transition, err := e.repo.GetTransition(ctx, req.TransitionID)
	if err != nil {
		return nil, ErrTransitionNotFound
	}

	if transition.FromStateID != req.CurrentStateID {
		return nil, ErrTransitionNotAllowed
	}

	conditions, _ := e.parseConditions(transition.Conditions)
	for _, cond := range conditions {
		if !e.evaluateCondition(cond, req.UserID, req.UserRole, req.ProjectData) {
			return nil, fmt.Errorf("%w: %s", ErrConditionFailed, cond.Type)
		}
	}

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
		EventName:    transition.EventName,
		NewStateID:   toState.ID,
		NewStateName: toState.Name,
		DataPatch:    map[string]interface{}{},
	}

	// Always write workflow-owned info under "wf"
	wfPatch := map[string]interface{}{
		"last_transition": map[string]interface{}{
			"event_name":    transition.EventName,
			"transition_id": transition.ID,
			"from_state":    fromState.Name,
			"to_state":      toState.Name,
			"performed_by":  req.UserID,
			"performed_at":  time.Now().UTC().Format(time.RFC3339),
		},
	}
	result.DataPatch["wf"] = wfPatch

	actionCtx := &plugins.ActionContext{
		ProjectID:     req.ProjectID,
		UserID:        req.UserID,
		DepartmentID:  req.DepartmentID,
		ProjectData:   req.ProjectData,
		PreviousState: fromState.Name,
		NewState:      toState.Name,
		TransitionID:  transition.ID,
		Payload:       req.Payload,
		Metadata:      map[string]interface{}{},
	}

	// EXIT actions: execute PRE, plan POST
	exitActions, _ := e.repo.GetStateActionsByTrigger(ctx, fromState.ID, plugins.TriggerOnExit)
	exitPost := []int64{}
	for _, action := range exitActions {
		actionCtx.StateID = fromState.ID
		actionCtx.Trigger = plugins.TriggerOnExit
		actionCtx.Config = e.parseConfig(action.Config)

		plugin, plugErr := plugins.Get(action.Type)
		if plugErr != nil {
			e.logger.Warn("action plugin not found", zap.String("type", action.Type), zap.Error(plugErr))
			continue
		}

		if isPreCommit(plugin) {
			ar := plugin.Execute(ctx, actionCtx)
			if !ar.Success && !action.IsOptional {
				return nil, fmt.Errorf("pre-exit action '%s' failed: %w", action.Name, ar.Error)
			}
			result.ExecutedActionNames = append(result.ExecutedActionNames, action.Name)
			e.mergeActionDataIntoPatch(result.DataPatch, action.Name, ar.Data)
		} else {
			exitPost = append(exitPost, action.ID)
			result.ExecutedActionNames = append(result.ExecutedActionNames, action.Name) // planned
		}
	}

	// ENTER actions: execute PRE, plan POST
	enterActions, _ := e.repo.GetStateActionsByTrigger(ctx, toState.ID, plugins.TriggerOnEnter)
	enterPost := []int64{}
	for _, action := range enterActions {
		actionCtx.StateID = toState.ID
		actionCtx.Trigger = plugins.TriggerOnEnter
		actionCtx.Config = e.parseConfig(action.Config)

		plugin, plugErr := plugins.Get(action.Type)
		if plugErr != nil {
			e.logger.Warn("action plugin not found", zap.String("type", action.Type), zap.Error(plugErr))
			continue
		}

		if isPreCommit(plugin) {
			ar := plugin.Execute(ctx, actionCtx)
			if !ar.Success && !action.IsOptional {
				return nil, fmt.Errorf("pre-enter action '%s' failed: %w", action.Name, ar.Error)
			}
			result.ExecutedActionNames = append(result.ExecutedActionNames, action.Name)
			e.mergeActionDataIntoPatch(result.DataPatch, action.Name, ar.Data)
		} else {
			enterPost = append(enterPost, action.ID)
			result.ExecutedActionNames = append(result.ExecutedActionNames, action.Name) // planned
		}
	}

	if len(exitPost) > 0 {
		result.PostCommitActions = append(result.PostCommitActions, PostCommitActionGroup{
			Trigger:   plugins.TriggerOnExit,
			ActionIDs: exitPost,
		})
	}
	if len(enterPost) > 0 {
		result.PostCommitActions = append(result.PostCommitActions, PostCommitActionGroup{
			Trigger:   plugins.TriggerOnEnter,
			ActionIDs: enterPost,
		})
	}

	// Deadline
	if toState.DurationDays > 0 {
		d := time.Now().AddDate(0, 0, int(toState.DurationDays))
		result.NewDeadline = &d
	} else if toState.FixedDeadline != nil {
		result.NewDeadline = toState.FixedDeadline
	}

	if toState.IsFinal {
		result.SetStatus = "completed"
	}

	return result, nil
}

func isPreCommit(p plugins.ActionPlugin) bool {
	// Senior rule: only validation/grading can run before commit.
	switch p.Category() {
	case plugins.CategoryValidation, plugins.CategoryGrading:
		return true
	default:
		return false
	}
}

func (e *WorkflowEngine) mergeActionDataIntoPatch(patch map[string]interface{}, actionName string, data map[string]interface{}) {
	if len(data) == 0 {
		return
	}
	wf, ok := patch["wf"].(map[string]interface{})
	if !ok {
		wf = map[string]interface{}{}
		patch["wf"] = wf
	}
	results, ok := wf["action_results"].(map[string]interface{})
	if !ok {
		results = map[string]interface{}{}
		wf["action_results"] = results
	}
	results[actionName] = data
}

func (e *WorkflowEngine) parseConditions(data []byte) ([]TransitionCondition, error) {
	var conditions []TransitionCondition
	if len(data) == 0 {
		return conditions, nil
	}
	return conditions, json.Unmarshal(data, &conditions)
}

func (e *WorkflowEngine) parseConfig(data []byte) map[string]interface{} {
	var config map[string]interface{}
	if err := json.Unmarshal(data, &config); err != nil || config == nil {
		config = make(map[string]interface{})
	}
	return config
}

// --- condition evaluation (оставляем твой подход) ---

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
			if rs, ok := r.(string); ok && rs == userRole {
				return true
			}
		}
		return false
	case "field":
		fieldValue, ok := data[cond.Field]
		if !ok {
			return cond.Operator == "not_exists" || cond.Operator == "empty"
		}
		return compareValues(fieldValue, cond.Operator, cond.Value)
	default:
		return true
	}
}

func compareValues(actual interface{}, operator string, expected interface{}) bool {
	toFloat := func(v interface{}) float64 {
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
	case "exists", "not_empty":
		return actual != nil && actual != ""
	case "not_exists", "empty":
		return actual == nil || actual == ""
	default:
		return true
	}
}

// ================= Types =================

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

type PostCommitActionGroup struct {
	Trigger   string
	ActionIDs []int64
}

type ExecuteTransitionResult struct {
	Success             bool
	EventName           string
	NewStateID          int64
	NewStateName        string
	NewDeadline         *time.Time
	SetStatus           string
	ExecutedActionNames []string

	// Patch to merge into projects.data
	DataPatch map[string]interface{}

	// Post-commit actions to run async (idempotent consumer)
	PostCommitActions []PostCommitActionGroup
}

type TransitionCondition struct {
	Type     string      `json:"type"`
	Field    string      `json:"field"`
	Operator string      `json:"operator"`
	Value    interface{} `json:"value"`
}
