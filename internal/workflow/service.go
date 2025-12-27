package workflow

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"go.uber.org/zap"
	"gorm.io/datatypes"
)

type Service struct {
	repo   Repository
	logger *zap.Logger
}

func NewService(repo Repository, logger *zap.Logger) *Service {
	return &Service{
		repo:   repo,
		logger: logger,
	}
}

type CreateWorkflowInput struct {
	Name         string
	Description  string
	DepartmentID int64
	Settings     map[string]interface{}
}

func (s *Service) CreateWorkflow(ctx context.Context, input *CreateWorkflowInput) (*Workflow, error) {
	settingsJSON, _ := json.Marshal(input.Settings)

	wf := &Workflow{
		Name:         input.Name,
		Description:  input.Description,
		DepartmentID: input.DepartmentID,
		Version:      1,
		IsActive:     false,
		Settings:     datatypes.JSON(settingsJSON),
	}

	if err := s.repo.CreateWorkflow(ctx, wf); err != nil {
		return nil, fmt.Errorf("failed to create workflow: %w", err)
	}

	return wf, nil
}

func (s *Service) GetWorkflow(ctx context.Context, id int64) (*Workflow, error) {
	return s.repo.GetWorkflow(ctx, id)
}

func (s *Service) GetWorkflowFull(ctx context.Context, id int64) (*Workflow, error) {
	return s.repo.GetWorkflowFull(ctx, id)
}

func (s *Service) GetWorkflowByName(ctx context.Context, name string) (*Workflow, error) {
	return s.repo.GetWorkflowByName(ctx, name)
}

func (s *Service) ListWorkflows(ctx context.Context, departmentID int64) ([]*Workflow, error) {
	return s.repo.ListWorkflows(ctx, departmentID)
}

type UpdateWorkflowInput struct {
	Name        string
	Description string
	Settings    map[string]interface{}
}

func (s *Service) UpdateWorkflow(ctx context.Context, id int64, input *UpdateWorkflowInput) (*Workflow, error) {
	wf, err := s.repo.GetWorkflow(ctx, id)
	if err != nil {
		return nil, err
	}

	if input.Name != "" {
		wf.Name = input.Name
	}
	if input.Description != "" {
		wf.Description = input.Description
	}
	if input.Settings != nil {
		settingsJSON, _ := json.Marshal(input.Settings)
		wf.Settings = datatypes.JSON(settingsJSON)
	}

	if err := s.repo.UpdateWorkflow(ctx, wf); err != nil {
		return nil, err
	}

	return wf, nil
}

func (s *Service) DeleteWorkflow(ctx context.Context, id int64) error {
	return s.repo.DeleteWorkflow(ctx, id)
}

func (s *Service) SetActiveWorkflow(ctx context.Context, workflowID int64) (*Workflow, error) {
	return s.repo.SetActiveWorkflow(ctx, workflowID)
}

func (s *Service) GetActiveWorkflowByDepartment(ctx context.Context, departmentID int64) (*Workflow, error) {
	return s.repo.GetActiveWorkflowByDepartment(ctx, departmentID)
}

func (s *Service) CloneWorkflow(ctx context.Context, sourceID, targetDepartmentID int64, newName string, asTemplate bool) (*Workflow, error) {
	source, err := s.repo.GetWorkflowFull(ctx, sourceID)
	if err != nil {
		return nil, fmt.Errorf("source workflow not found: %w", err)
	}

	newWorkflow := &Workflow{
		Name:         newName,
		Description:  source.Description + " (clone)",
		DepartmentID: targetDepartmentID,
		Version:      1,
		IsActive:     false,
		IsTemplate:   asTemplate,
		ParentID:     &sourceID,
		Settings:     source.Settings,
	}

	if err := s.repo.CreateWorkflow(ctx, newWorkflow); err != nil {
		return nil, err
	}

	stateIDMap := make(map[int64]int64)
	for _, state := range source.States {
		newState := &State{
			WorkflowID:   newWorkflow.ID,
			Name:         state.Name,
			DisplayName:  state.DisplayName,
			Description:  state.Description,
			OrderIndex:   state.OrderIndex,
			Type:         state.Type,
			IsInitial:    state.IsInitial,
			IsFinal:      state.IsFinal,
			IsOptional:   state.IsOptional,
			Config:       state.Config,
			DurationDays: state.DurationDays,
			DurationMode: state.DurationMode,
			Color:        state.Color,
			Icon:         state.Icon,
		}

		if err := s.repo.CreateState(ctx, newState); err != nil {
			if delErr := s.repo.DeleteWorkflow(ctx, newWorkflow.ID); delErr != nil {
				s.logger.Error("Failed to rollback workflow creation", zap.Error(delErr))
			}
			return nil, err
		}
		stateIDMap[state.ID] = newState.ID
		for _, action := range state.Actions {
			newAction := &StateAction{
				StateID:    newState.ID,
				Name:       action.Name,
				Type:       action.Type,
				Trigger:    action.Trigger,
				OrderIndex: action.OrderIndex,
				Config:     action.Config,
				IsEnabled:  action.IsEnabled,
				Conditions: action.Conditions,
			}
			if err := s.repo.CreateStateAction(ctx, newAction); err != nil {
				s.logger.Warn("Failed to create state action during clone", zap.Error(err))
			}
		}
	}

	for _, tr := range source.Transitions {
		newTransition := &Transition{
			WorkflowID:  newWorkflow.ID,
			EventName:   tr.EventName,
			DisplayName: tr.DisplayName,
			FromStateID: stateIDMap[tr.FromStateID],
			ToStateID:   stateIDMap[tr.ToStateID],
			Conditions:  tr.Conditions,
			ButtonLabel: tr.ButtonLabel,
			ButtonColor: tr.ButtonColor,
			ConfirmText: tr.ConfirmText,
			Priority:    tr.Priority,
		}
		if err := s.repo.CreateTransition(ctx, newTransition); err != nil {
			s.logger.Warn("Failed to create transition during clone", zap.Error(err))
		}
	}

	return s.repo.GetWorkflowFull(ctx, newWorkflow.ID)
}

func (s *Service) CreateNewVersion(ctx context.Context, workflowID int64, changelog string) (*Workflow, error) {
	current, err := s.repo.GetWorkflow(ctx, workflowID)
	if err != nil {
		return nil, err
	}

	current.IsActive = false
	err = s.repo.UpdateWorkflow(ctx, current)
	if err != nil {
		return nil, err
	}
	newVersion, err := s.CloneWorkflow(ctx, workflowID, current.DepartmentID, current.Name, false)
	if err != nil {
		return nil, err
	}

	newVersion.Version = current.Version + 1
	newVersion.ParentID = &workflowID
	if err := s.repo.UpdateWorkflow(ctx, newVersion); err != nil {
		s.logger.Error("Failed to update workflow", zap.Error(err))
		return nil, err
	}
	s.logger.Info("Created new workflow version",
		zap.Int64("workflow_id", newVersion.ID),
		zap.Int32("version", newVersion.Version),
		zap.String("changelog", changelog))

	return newVersion, nil
}

type ValidationResult struct {
	IsValid  bool
	Errors   []ValidationError
	Warnings []ValidationWarning
}

type ValidationError struct {
	Code    string
	Message string
	Path    string
}

type ValidationWarning struct {
	Code    string
	Message string
	Path    string
}

func (r *ValidationResult) addError(code, message, path string) {
	r.Errors = append(r.Errors, ValidationError{Code: code, Message: message, Path: path})
}

func (r *ValidationResult) addWarning(code, message, path string) {
	r.Warnings = append(r.Warnings, ValidationWarning{Code: code, Message: message, Path: path})
}

func (s *Service) ValidateWorkflow(ctx context.Context, workflowID int64) (*ValidationResult, error) {
	wf, err := s.repo.GetWorkflowFull(ctx, workflowID)
	if err != nil {
		return nil, err
	}

	result := &ValidationResult{IsValid: true}

	hasInitial := false
	hasFinal := false
	stateMap := make(map[int64]*State)

	for i := range wf.States {
		state := &wf.States[i]
		stateMap[state.ID] = state

		if state.IsInitial {
			if hasInitial {
				result.addError("MULTIPLE_INITIAL", "Multiple initial states found", fmt.Sprintf("states[%d]", i))
			}
			hasInitial = true
		}
		if state.IsFinal {
			hasFinal = true
		}
	}

	if !hasInitial {
		result.addError("NO_INITIAL_STATE", "Workflow must have an initial state", "workflow")
	}

	if !hasFinal {
		result.addWarning("NO_FINAL_STATE", "Workflow has no final state", "workflow")
	}

	transitionMap := make(map[int64][]int64)
	for _, tr := range wf.Transitions {
		if _, ok := stateMap[tr.FromStateID]; !ok {
			result.addError("INVALID_FROM_STATE",
				fmt.Sprintf("Transition references non-existent from_state: %d", tr.FromStateID),
				fmt.Sprintf("transitions[%d]", tr.ID))
		}
		if _, ok := stateMap[tr.ToStateID]; !ok {
			result.addError("INVALID_TO_STATE",
				fmt.Sprintf("Transition references non-existent to_state: %d", tr.ToStateID),
				fmt.Sprintf("transitions[%d]", tr.ID))
		}
		transitionMap[tr.FromStateID] = append(transitionMap[tr.FromStateID], tr.ToStateID)
	}

	if hasInitial {
		reachable := s.findReachableStates(wf.States, transitionMap)
		for _, state := range wf.States {
			if !reachable[state.ID] && !state.IsInitial {
				result.addWarning("UNREACHABLE_STATE",
					fmt.Sprintf("State '%s' is not reachable from initial state", state.Name),
					fmt.Sprintf("states[%d]", state.ID))
			}
		}
	}

	for i, state := range wf.States {
		if err := s.validateStateConfig(state); err != nil {
			result.addError("INVALID_STATE_CONFIG", err.Error(), fmt.Sprintf("states[%d].config", i))
		}
	}

	result.IsValid = len(result.Errors) == 0
	return result, nil
}

func (s *Service) findReachableStates(states []State, transitions map[int64][]int64) map[int64]bool {
	reachable := make(map[int64]bool)

	var initialID int64
	for _, state := range states {
		if state.IsInitial {
			initialID = state.ID
			break
		}
	}

	queue := []int64{initialID}
	reachable[initialID] = true

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		for _, next := range transitions[current] {
			if !reachable[next] {
				reachable[next] = true
				queue = append(queue, next)
			}
		}
	}

	return reachable
}

func (s *Service) validateStateConfig(state State) error {
	var config map[string]interface{}
	if err := json.Unmarshal(state.Config, &config); err != nil {
		return fmt.Errorf("invalid JSON config")
	}

	switch state.Type {
	case StateTypeTeamFormation:
		if tc, ok := config["team_config"].(map[string]interface{}); ok {
			minSize, _ := tc["min_size"].(float64)
			maxSize, _ := tc["max_size"].(float64)
			if minSize > maxSize {
				return fmt.Errorf("min_size cannot be greater than max_size")
			}
		}
	case StateTypeDocumentUpload:
		if fc, ok := config["file_config"].(map[string]interface{}); ok {
			maxSize, _ := fc["max_size_bytes"].(float64)
			if maxSize <= 0 {
				return fmt.Errorf("max_size_bytes must be positive")
			}
		}
	}

	return nil
}

type CreateStateInput struct {
	WorkflowID   int64
	Name         string
	DisplayName  string
	Description  string
	Type         string
	Config       map[string]interface{}
	DurationDays int32
	IsInitial    bool
	IsFinal      bool
	IsOptional   bool
	OrderIndex   int32
}

func (s *Service) CreateState(ctx context.Context, input *CreateStateInput) (*State, error) {
	configJSON, _ := json.Marshal(input.Config)

	state := &State{
		WorkflowID:   input.WorkflowID,
		Name:         input.Name,
		DisplayName:  input.DisplayName,
		Description:  input.Description,
		Type:         input.Type,
		Config:       datatypes.JSON(configJSON),
		DurationDays: input.DurationDays,
		DurationMode: "relative",
		IsInitial:    input.IsInitial,
		IsFinal:      input.IsFinal,
		IsOptional:   input.IsOptional,
		OrderIndex:   input.OrderIndex,
	}

	if err := s.repo.CreateState(ctx, state); err != nil {
		return nil, err
	}

	return state, nil
}

func (s *Service) GetState(ctx context.Context, id int64) (*State, error) {
	return s.repo.GetState(ctx, id)
}

func (s *Service) ListStates(ctx context.Context, workflowID int64) ([]State, error) {
	return s.repo.ListStates(ctx, workflowID)
}

type UpdateStateInput struct {
	Name         string
	DisplayName  string
	Description  string
	Config       map[string]interface{}
	DurationDays int32
	IsOptional   bool
	OrderIndex   int32
}

func (s *Service) UpdateState(ctx context.Context, id int64, input *UpdateStateInput) (*State, error) {
	state, err := s.repo.GetState(ctx, id)
	if err != nil {
		return nil, err
	}

	if input.Name != "" {
		state.Name = input.Name
	}
	if input.DisplayName != "" {
		state.DisplayName = input.DisplayName
	}
	if input.Description != "" {
		state.Description = input.Description
	}
	if input.Config != nil {
		configJSON, _ := json.Marshal(input.Config)
		state.Config = datatypes.JSON(configJSON)
	}
	if input.DurationDays > 0 {
		state.DurationDays = input.DurationDays
	}
	state.IsOptional = input.IsOptional
	if input.OrderIndex > 0 {
		state.OrderIndex = input.OrderIndex
	}

	if err := s.repo.UpdateState(ctx, state); err != nil {
		return nil, err
	}

	return state, nil
}

func (s *Service) DeleteState(ctx context.Context, id int64) error {
	return s.repo.DeleteState(ctx, id)
}

func (s *Service) ReorderStates(ctx context.Context, workflowID int64, stateIDs []int64) error {
	for i, stateID := range stateIDs {
		state, err := s.repo.GetState(ctx, stateID)
		if err != nil {
			return err
		}
		if state.WorkflowID != workflowID {
			return fmt.Errorf("state %d does not belong to workflow %d", stateID, workflowID)
		}
		state.OrderIndex = int32(i + 1)
		if err := s.repo.UpdateState(ctx, state); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) GetNextState(ctx context.Context, currentStateID int64, eventName string) (*State, error) {
	return s.repo.GetNextState(ctx, currentStateID, eventName)
}

type CreateTransitionInput struct {
	WorkflowID  int64
	EventName   string
	DisplayName string
	FromStateID int64
	ToStateID   int64
	ButtonLabel string
	ButtonColor string
	ConfirmText string
}

func (s *Service) CreateTransition(ctx context.Context, input *CreateTransitionInput) (*Transition, error) {
	tr := &Transition{
		WorkflowID:  input.WorkflowID,
		EventName:   input.EventName,
		DisplayName: input.DisplayName,
		FromStateID: input.FromStateID,
		ToStateID:   input.ToStateID,
		ButtonLabel: input.ButtonLabel,
		ButtonColor: input.ButtonColor,
		ConfirmText: input.ConfirmText,
	}

	if err := s.repo.CreateTransition(ctx, tr); err != nil {
		return nil, err
	}

	return tr, nil
}

type UpdateTransitionInput struct {
	DisplayName string
	ButtonLabel string
	ButtonColor string
	ConfirmText string
}

func (s *Service) UpdateTransition(ctx context.Context, id int64, input *UpdateTransitionInput) (*Transition, error) {
	tr, err := s.repo.GetTransition(ctx, id)
	if err != nil {
		return nil, err
	}

	if input.DisplayName != "" {
		tr.DisplayName = input.DisplayName
	}
	if input.ButtonLabel != "" {
		tr.ButtonLabel = input.ButtonLabel
	}
	if input.ButtonColor != "" {
		tr.ButtonColor = input.ButtonColor
	}
	if input.ConfirmText != "" {
		tr.ConfirmText = input.ConfirmText
	}

	if err := s.repo.UpdateTransition(ctx, tr); err != nil {
		return nil, err
	}

	return tr, nil
}

func (s *Service) DeleteTransition(ctx context.Context, id int64) error {
	return s.repo.DeleteTransition(ctx, id)
}

func (s *Service) ListTransitions(ctx context.Context, workflowID int64) ([]Transition, error) {
	return s.repo.GetTransitionsByWorkflow(ctx, workflowID)
}

type CreateStateActionInput struct {
	StateID    int64
	Name       string
	Type       string
	Trigger    string
	OrderIndex int32
	Config     map[string]interface{}
	IsEnabled  bool
}

func (s *Service) CreateStateAction(ctx context.Context, input *CreateStateActionInput) (*StateAction, error) {
	configJSON, _ := json.Marshal(input.Config)

	action := &StateAction{
		StateID:    input.StateID,
		Name:       input.Name,
		Type:       input.Type,
		Trigger:    input.Trigger,
		OrderIndex: input.OrderIndex,
		Config:     datatypes.JSON(configJSON),
		IsEnabled:  input.IsEnabled,
	}

	if err := s.repo.CreateStateAction(ctx, action); err != nil {
		return nil, err
	}

	return action, nil
}

type UpdateStateActionInput struct {
	Name       string
	Config     map[string]interface{}
	OrderIndex int32
	IsEnabled  bool
}

func (s *Service) UpdateStateAction(ctx context.Context, id int64, input *UpdateStateActionInput) (*StateAction, error) {
	action, err := s.repo.GetStateAction(ctx, id)
	if err != nil {
		return nil, err
	}

	if input.Name != "" {
		action.Name = input.Name
	}
	if input.Config != nil {
		configJSON, _ := json.Marshal(input.Config)
		action.Config = datatypes.JSON(configJSON)
	}
	if input.OrderIndex > 0 {
		action.OrderIndex = input.OrderIndex
	}
	action.IsEnabled = input.IsEnabled

	if err := s.repo.UpdateStateAction(ctx, action); err != nil {
		return nil, err
	}

	return action, nil
}

func (s *Service) DeleteStateAction(ctx context.Context, id int64) error {
	return s.repo.DeleteStateAction(ctx, id)
}

func (s *Service) ListStateActions(ctx context.Context, stateID int64) ([]StateAction, error) {
	return s.repo.ListStateActions(ctx, stateID)
}

type AvailableTransition struct {
	Transition          *Transition
	CanExecute          bool
	BlockedReason       string
	MissingRequirements []string
}

func (s *Service) GetAvailableTransitions(ctx context.Context, projectID, currentStateID, userID int64, userRole string) ([]*AvailableTransition, error) {
	transitions, err := s.repo.GetTransitionsFromState(ctx, currentStateID)
	if err != nil {
		return nil, err
	}

	var result []*AvailableTransition
	for i := range transitions {
		tr := &transitions[i]
		at := &AvailableTransition{
			Transition: tr,
			CanExecute: true,
		}
		// TODO: Evaluate conditions based on userRole and projectData
		result = append(result, at)
	}

	return result, nil
}

type ExecuteTransitionInput struct {
	ProjectID    int64
	TransitionID int64
	UserID       int64
	Payload      map[string]interface{}
}

type ExecuteTransitionResult struct {
	NewStateID      int64
	NewStateName    string
	NewDeadline     *time.Time
	ExecutedActions []string
}

func (s *Service) ExecuteTransition(ctx context.Context, input *ExecuteTransitionInput) (*ExecuteTransitionResult, error) {
	tr, err := s.repo.GetTransition(ctx, input.TransitionID)
	if err != nil {
		return nil, fmt.Errorf("transition not found: %w", err)
	}

	toState, err := s.repo.GetState(ctx, tr.ToStateID)
	if err != nil {
		return nil, fmt.Errorf("target state not found: %w", err)
	}

	result := &ExecuteTransitionResult{
		NewStateID:   toState.ID,
		NewStateName: toState.Name,
	}

	if toState.DurationDays > 0 {
		deadline := time.Now().AddDate(0, 0, int(toState.DurationDays))
		result.NewDeadline = &deadline
	} else if toState.FixedDeadline != nil {
		result.NewDeadline = toState.FixedDeadline
	}

	s.logger.Info("Executed transition",
		zap.Int64("project_id", input.ProjectID),
		zap.Int64("transition_id", input.TransitionID),
		zap.Int64("new_state_id", toState.ID))

	return result, nil
}

func (s *Service) ListTemplates(ctx context.Context) ([]*WorkflowTemplate, error) {
	return s.repo.ListTemplates(ctx)
}

func (s *Service) CreateFromTemplate(ctx context.Context, templateID, departmentID int64, name string) (*Workflow, error) {
	template, err := s.repo.GetTemplate(ctx, templateID)
	if err != nil {
		return nil, err
	}

	var templateData struct {
		Settings    map[string]interface{} `json:"settings"`
		States      []State                `json:"states"`
		Transitions []Transition           `json:"transitions"`
	}

	if unmarshalErr := json.Unmarshal(template.TemplateData, &templateData); unmarshalErr != nil {
		return nil, fmt.Errorf("failed to unmarshal template data: %w", unmarshalErr)
	}

	wf, err := s.CreateWorkflow(ctx, &CreateWorkflowInput{
		Name:         name,
		Description:  template.Description,
		DepartmentID: departmentID,
		Settings:     templateData.Settings,
	})
	if err != nil {
		return nil, err
	}

	stateIDMap := make(map[int64]int64)
	for _, state := range templateData.States {
		newState := &State{
			WorkflowID:   wf.ID,
			Name:         state.Name,
			DisplayName:  state.DisplayName,
			Description:  state.Description,
			OrderIndex:   state.OrderIndex,
			Type:         state.Type,
			IsInitial:    state.IsInitial,
			IsFinal:      state.IsFinal,
			Config:       state.Config,
			DurationDays: state.DurationDays,
		}
		if err := s.repo.CreateState(ctx, newState); err != nil {
			return nil, err
		}
		stateIDMap[state.ID] = newState.ID
	}

	for _, tr := range templateData.Transitions {
		newTr := &Transition{
			WorkflowID:  wf.ID,
			EventName:   tr.EventName,
			DisplayName: tr.DisplayName,
			FromStateID: stateIDMap[tr.FromStateID],
			ToStateID:   stateIDMap[tr.ToStateID],
			Conditions:  tr.Conditions,
			ButtonLabel: tr.ButtonLabel,
			ButtonColor: tr.ButtonColor,
		}
		if err := s.repo.CreateTransition(ctx, newTr); err != nil {
			s.logger.Warn("Failed to create transition from template", zap.Error(err))
		}
	}

	return s.repo.GetWorkflowFull(ctx, wf.ID)
}
