package engine

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/lucky720s/diplomaflow/internal/workflow"
	"github.com/lucky720s/diplomaflow/internal/workflow/plugins"
	teamv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/team/v1"
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
	repo       workflow.Repository
	teamClient teamv1.TeamServiceClient
	logger     *zap.Logger
}

func NewWorkflowEngine(repo workflow.Repository, teamClient teamv1.TeamServiceClient, logger *zap.Logger) *WorkflowEngine {
	return &WorkflowEngine{repo: repo, teamClient: teamClient, logger: logger}
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

	// ===== STRICT domain validation for TEAM_FORMED =====
	if strings.EqualFold(strings.TrimSpace(transition.EventName), "TEAM_FORMED") {
		if err := e.validateTeamFormed(ctx, req, fromState); err != nil {
			return nil, err
		}
	}

	result := &ExecuteTransitionResult{
		Success:             true,
		EventName:           transition.EventName,
		NewStateID:          toState.ID,
		NewStateName:        toState.Name,
		DataPatch:           map[string]interface{}{"wf": map[string]interface{}{}},
		ExecutedActionNames: []string{},
		PostActions:         []PostCommitActionGroup{},
		SetStatus:           "",
	}

	// workflow-owned patch (pure)
	result.DataPatch["wf"] = map[string]interface{}{
		"last_transition": map[string]interface{}{
			"event_name":    transition.EventName,
			"transition_id": transition.ID,
			"from_state":    fromState.Name,
			"to_state":      toState.Name,
			"performed_by":  req.UserID,
			"performed_at":  time.Now().UTC().Format(time.RFC3339),
		},
	}

	actx := &plugins.ActionContext{
		ProjectID:     req.ProjectID,
		UserID:        req.UserID,
		DepartmentID:  req.DepartmentID,
		UniversityID:  req.UniversityID,
		TeamID:        req.TeamID,
		ProjectData:   req.ProjectData,
		PreviousState: fromState.Name,
		NewState:      toState.Name,
		TransitionID:  transition.ID,
		Payload:       req.Payload,
		Metadata:      make(map[string]interface{}),
	}

	// 1) ON_EXIT: PRE actions выполняем сейчас, POST actions планируем
	exitActions, _ := e.repo.GetStateActionsByTrigger(ctx, fromState.ID, plugins.TriggerOnExit)
	if err := e.runPhase(ctx, result, actx, fromState.ID, plugins.TriggerOnExit, exitActions); err != nil {
		return nil, err
	}

	// 2) ON_ENTER
	enterActions, _ := e.repo.GetStateActionsByTrigger(ctx, toState.ID, plugins.TriggerOnEnter)
	if err := e.runPhase(ctx, result, actx, toState.ID, plugins.TriggerOnEnter, enterActions); err != nil {
		return nil, err
	}

	// 3) deadline calculation (pure)
	if toState.DurationDays > 0 {
		d := time.Now().UTC().AddDate(0, 0, int(toState.DurationDays))
		result.NewDeadline = &d
	} else if toState.FixedDeadline != nil {
		result.NewDeadline = toState.FixedDeadline
	}

	if toState.IsFinal {
		result.SetStatus = "completed"
	}

	return result, nil
}

func (e *WorkflowEngine) validateTeamFormed(ctx context.Context, req *ExecuteTransitionRequest, fromState *workflow.State) error {
	// применяем правила только если исходное состояние — этап формирования команды
	if fromState == nil || strings.ToUpper(strings.TrimSpace(fromState.Type)) != "TEAM_FORMATION" {
		return nil
	}
	if e.teamClient == nil {
		return errors.New("team validation required but team_client is not configured in workflow_service")
	}
	if req.TeamID <= 0 {
		return errors.New("team_id is required to perform TEAM_FORMED transition")
	}

	// парсим team_config из fromState.Config
	var sc workflow.StateConfig
	_ = json.Unmarshal(fromState.Config, &sc)

	tc := sc.TeamConfig
	if tc == nil {
		// если team_config не задан — не блокируем (но лучше задать в state.config)
		e.logger.Warn("TEAM_FORMATION state has no team_config; skipping validation", zap.Int64("state_id", fromState.ID))
		return nil
	}

	teamResp, err := e.teamClient.GetTeam(ctx, &teamv1.GetTeamRequest{TeamId: req.TeamID})
	if err != nil {
		return fmt.Errorf("failed to fetch team for validation: %w", err)
	}

	size := int32(len(teamResp.Members))

	// allow_solo logic
	if size == 1 && !tc.AllowSolo {
		return fmt.Errorf("team size=1 (solo) is not allowed by workflow config")
	}

	// min/max
	min := tc.MinSize
	max := tc.MaxSize

	// Если allow_solo=true, то size=1 допускаем даже если min=2
	if size != 1 || !tc.AllowSolo {
		if min > 0 && size < min {
			return fmt.Errorf("team size %d is below minimum %d", size, min)
		}
	}
	if max > 0 && size > max {
		return fmt.Errorf("team size %d exceeds maximum %d", size, max)
	}

	if tc.RequireLeader {
		hasLeader := false
		for _, m := range teamResp.Members {
			if strings.EqualFold(m.Role, "leader") {
				hasLeader = true
				break
			}
		}
		if !hasLeader {
			return errors.New("team has no leader but workflow requires leader")
		}
	}

	return nil
}

// runPhase:
// - PRE actions: выполняем синхронно (должны быть pure/validation/grading)
// - POST actions: НЕ выполняем здесь, только возвращаем их IDs в result.PostActions
func (e *WorkflowEngine) runPhase(
	ctx context.Context,
	result *ExecuteTransitionResult,
	actx *plugins.ActionContext,
	stateID int64,
	trigger string,
	actions []workflow.StateAction,
) error {
	var postIDs []int64

	for _, action := range actions {
		actx.StateID = stateID
		actx.Trigger = trigger
		actx.Config = e.parseConfig(action.Config)

		if e.isPostActionType(action.Type) {
			postIDs = append(postIDs, action.ID)
			continue
		}

		ar := e.executeAction(ctx, &action, actx)
		if ar != nil && ar.Data != nil {
			e.mergeActionData(result.DataPatch, action.Name, ar.Data)
		}

		if ar == nil || ar.Success {
			result.ExecutedActionNames = append(result.ExecutedActionNames, action.Name)
			continue
		}

		if action.IsOptional {
			e.logger.Warn("PRE action failed but optional",
				zap.Int64("action_id", action.ID),
				zap.String("action_name", action.Name),
				zap.String("action_type", action.Type),
				zap.Error(ar.Error),
			)
			result.ExecutedActionNames = append(result.ExecutedActionNames, action.Name)
			continue
		}

		return fmt.Errorf("pre action '%s' failed: %w", action.Name, ar.Error)
	}

	if len(postIDs) > 0 {
		result.PostActions = append(result.PostActions, PostCommitActionGroup{
			Trigger:   trigger,
			ActionIDs: postIDs,
		})
	}

	return nil
}

func (e *WorkflowEngine) isPostActionType(actionType string) bool {
	t := strings.ToUpper(strings.TrimSpace(actionType))

	if t == "CALCULATE_GRADE" || t == "VALIDATE_DATA" || strings.HasPrefix(t, "VALIDATE_") || t == "UPDATE_PROJECT" {
		return false
	}
	return true
}

func (e *WorkflowEngine) mergeActionData(patch map[string]interface{}, actionName string, data map[string]interface{}) {
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

func (e *WorkflowEngine) executeAction(ctx context.Context, action *workflow.StateAction, actx *plugins.ActionContext) *plugins.ActionResult {
	pl, err := plugins.Get(action.Type)
	if err != nil {
		e.logger.Warn("Action plugin not found", zap.String("type", action.Type))
		return &plugins.ActionResult{Success: false, Error: err}
	}

	res := pl.Execute(ctx, actx)
	if res != nil && !res.Success {
		e.logger.Warn("Action execution failed",
			zap.String("action_type", action.Type),
			zap.String("action_name", action.Name),
			zap.Error(res.Error),
		)
	}
	return res
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
			if ok && roleStr == userRole {
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
	UniversityID   int64
	TeamID         int64
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
	DataPatch           map[string]interface{}
	ExecutedActionNames []string
	PostActions         []PostCommitActionGroup
	Error               string
}

type TransitionCondition struct {
	Type     string      `json:"type"`
	Field    string      `json:"field"`
	Operator string      `json:"operator"`
	Value    interface{} `json:"value"`
}
