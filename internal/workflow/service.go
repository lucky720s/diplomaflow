// internal/workflow/service.go

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

// ==================== Workflow CRUD ====================

func (s *Service) CreateWorkflow(ctx context.Context, input *CreateWorkflowInput) (*Workflow, error) {
	settingsJSON, _ := json.Marshal(input.Settings)

	wf := &Workflow{
		Name:         input.Name,
		Description:  input.Description,
		DepartmentID: input.DepartmentID,
		Version:      1,
		IsActive:     false,
		IsTemplate:   input.IsTemplate,
		Settings:     datatypes.JSON(settingsJSON),
	}

	if err := s.repo.CreateWorkflow(ctx, wf); err != nil {
		return nil, fmt.Errorf("failed to create workflow: %w", err)
	}

	s.logger.Info("Workflow created",
		zap.Int64("id", wf.ID),
		zap.String("name", wf.Name),
		zap.Int64("department_id", wf.DepartmentID))

	return wf, nil
}

func (s *Service) GetWorkflow(ctx context.Context, id int64) (*Workflow, error) {
	return s.repo.GetWorkflow(ctx, id)
}

func (s *Service) GetWorkflowByName(ctx context.Context, name string) (*Workflow, error) {
	return s.repo.GetWorkflowByName(ctx, name)
}

func (s *Service) GetWorkflowFull(ctx context.Context, id int64) (*Workflow, error) {
	return s.repo.GetWorkflowFull(ctx, id)
}

func (s *Service) GetActiveWorkflowByDepartment(ctx context.Context, departmentID int64) (*Workflow, error) {
	return s.repo.GetActiveWorkflowByDepartment(ctx, departmentID)
}

func (s *Service) ListWorkflows(ctx context.Context, departmentID int64) ([]*Workflow, error) {
	return s.repo.ListWorkflows(ctx, departmentID)
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

func (s *Service) SetActiveWorkflow(ctx context.Context, id int64) (*Workflow, error) {
	if err := s.repo.SetActiveWorkflow(ctx, id); err != nil {
		return nil, err
	}
	return s.repo.GetWorkflow(ctx, id)
}

// ==================== State Management ====================

func (s *Service) CreateState(ctx context.Context, input *CreateStateInput) (*State, error) {
	configJSON, _ := json.Marshal(input.Config)

	state := &State{
		WorkflowID:   input.WorkflowID,
		Name:         input.Name,
		DisplayName:  input.DisplayName,
		Description:  input.Description,
		OrderIndex:   input.OrderIndex,
		Type:         input.Type,
		IsInitial:    input.IsInitial,
		IsFinal:      input.IsFinal,
		IsOptional:   input.IsOptional,
		Config:       datatypes.JSON(configJSON),
		DurationDays: input.DurationDays,
		Color:        input.Color,
		Icon:         input.Icon,
	}

	if err := s.repo.CreateState(ctx, state); err != nil {
		return nil, fmt.Errorf("failed to create state: %w", err)
	}

	return state, nil
}

func (s *Service) GetState(ctx context.Context, id int64) (*State, error) {
	return s.repo.GetState(ctx, id)
}

func (s *Service) GetInitialState(ctx context.Context, workflowID int64) (*State, error) {
	return s.repo.GetInitialState(ctx, workflowID)
}

func (s *Service) ListStates(ctx context.Context, workflowID int64) ([]State, error) {
	return s.repo.ListStates(ctx, workflowID)
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

	if err := s.repo.UpdateState(ctx, state); err != nil {
		return nil, err
	}

	return state, nil
}

func (s *Service) DeleteState(ctx context.Context, id int64) error {
	return s.repo.DeleteState(ctx, id)
}

// ==================== Transition Management ====================

func (s *Service) CreateTransition(ctx context.Context, input *CreateTransitionInput) (*Transition, error) {
	conditionsJSON, _ := json.Marshal(input.Conditions)

	tr := &Transition{
		WorkflowID:  input.WorkflowID,
		EventName:   input.EventName,
		DisplayName: input.DisplayName,
		FromStateID: input.FromStateID,
		ToStateID:   input.ToStateID,
		Conditions:  datatypes.JSON(conditionsJSON),
		ButtonLabel: input.ButtonLabel,
		ButtonColor: input.ButtonColor,
		ConfirmText: input.ConfirmText,
	}

	if err := s.repo.CreateTransition(ctx, tr); err != nil {
		return nil, fmt.Errorf("failed to create transition: %w", err)
	}

	return tr, nil
}

func (s *Service) GetTransition(ctx context.Context, id int64) (*Transition, error) {
	return s.repo.GetTransition(ctx, id)
}

func (s *Service) GetNextState(ctx context.Context, currentStateID int64, eventName string) (*State, error) {
	tr, err := s.repo.GetTransitionByEvent(ctx, currentStateID, eventName)
	if err != nil {
		return nil, err
	}
	return s.repo.GetState(ctx, tr.ToStateID)
}

func (s *Service) ListTransitions(ctx context.Context, workflowID int64) ([]Transition, error) {
	return s.repo.ListTransitions(ctx, workflowID)
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

// ==================== State Actions ====================

func (s *Service) CreateStateAction(ctx context.Context, input *CreateStateActionInput) (*StateAction, error) {
	configJSON, _ := json.Marshal(input.Config)
	conditionsJSON, _ := json.Marshal(input.Conditions)

	action := &StateAction{
		StateID:    input.StateID,
		Name:       input.Name,
		Type:       input.Type,
		Trigger:    input.Trigger,
		OrderIndex: input.OrderIndex,
		Config:     datatypes.JSON(configJSON),
		IsEnabled:  input.IsEnabled,
		Conditions: datatypes.JSON(conditionsJSON),
	}

	if err := s.repo.CreateStateAction(ctx, action); err != nil {
		return nil, fmt.Errorf("failed to create state action: %w", err)
	}

	return action, nil
}

func (s *Service) ListStateActions(ctx context.Context, stateID int64) ([]StateAction, error) {
	return s.repo.ListStateActions(ctx, stateID)
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

// ==================== Configuration Retrieval ====================

// GetTeamConfiguration возвращает конфигурацию команды для workflow/state
func (s *Service) GetTeamConfiguration(ctx context.Context, departmentID, workflowID, stateID int64) (*TeamConfigurationResult, error) {
	// Получаем workflow
	var wf *Workflow
	var err error

	if workflowID > 0 {
		wf, err = s.repo.GetWorkflow(ctx, workflowID)
	} else {
		wf, err = s.repo.GetActiveWorkflowByDepartment(ctx, departmentID)
	}
	if err != nil {
		return nil, fmt.Errorf("workflow not found: %w", err)
	}

	// Получаем state
	var state *State
	if stateID > 0 {
		state, err = s.repo.GetState(ctx, stateID)
	} else {
		state, err = s.repo.GetInitialState(ctx, wf.ID)
	}
	if err != nil {
		s.logger.Warn("State not found, using workflow defaults", zap.Error(err))
	}

	// Парсим настройки workflow
	var wfSettings WorkflowSettings
	if err := json.Unmarshal(wf.Settings, &wfSettings); err != nil {
		s.logger.Warn("Failed to parse workflow settings", zap.Error(err))
	}

	result := &TeamConfigurationResult{
		WorkflowID:       wf.ID,
		TeamStepRequired: wfSettings.TeamRequired,
		TeamConfig: &TeamConfig{
			MinSize:   wfSettings.MinTeamSize,
			MaxSize:   wfSettings.MaxTeamSize,
			AllowSolo: wfSettings.AllowSoloProject,
		},
		InviteExpireDays: 3, // Default
	}

	// Если есть state с типом TEAM_FORMATION, берём его конфиг
	if state != nil {
		result.StateID = state.ID
		result.StateName = state.Name

		if state.Type == "TEAM_FORMATION" {
			var stateConfig StateConfig
			if err := json.Unmarshal(state.Config, &stateConfig); err == nil {
				if stateConfig.TeamConfig != nil {
					result.TeamConfig = stateConfig.TeamConfig
					if stateConfig.TeamConfig.InviteExpireDays > 0 {
						result.InviteExpireDays = stateConfig.TeamConfig.InviteExpireDays
					}
				}
			}
		}
	}

	// Проверяем department config (может переопределять)
	deptConfig, _ := s.repo.GetDepartmentConfig(ctx, departmentID, "")
	if deptConfig != nil {
		var ts TeamSettings
		if err := json.Unmarshal(deptConfig.TeamSettings, &ts); err == nil {
			result.TeamConfig.MinSize = ts.MinSize
			result.TeamConfig.MaxSize = ts.MaxSize
			result.TeamConfig.AllowSolo = ts.AllowSolo
			if ts.InviteExpireDays > 0 {
				result.InviteExpireDays = ts.InviteExpireDays
			}
		}
	}

	return result, nil
}

// GetStepConfiguration возвращает полную конфигурацию этапа
func (s *Service) GetStepConfiguration(ctx context.Context, stateID, projectID int64) (*StepConfigurationResult, error) {
	state, err := s.repo.GetState(ctx, stateID)
	if err != nil {
		return nil, err
	}

	var stateConfig StateConfig
	if err := json.Unmarshal(state.Config, &stateConfig); err != nil {
		s.logger.Warn("Failed to parse state config", zap.Error(err))
	}

	result := &StepConfigurationResult{
		StateID:      state.ID,
		StateName:    state.Name,
		StateType:    state.Type,
		DurationDays: state.DurationDays,
	}

	// Заполняем конфиги в зависимости от типа состояния
	switch state.Type {
	case "TEAM_FORMATION":
		result.TeamConfig = stateConfig.TeamConfig
	case "DOCUMENT_UPLOAD":
		result.FileConfig = stateConfig.FileConfig
	case "FORM_SUBMIT":
		result.FormConfig = stateConfig.FormConfig
	case "REVIEW", "APPROVAL":
		result.ReviewConfig = stateConfig.ReviewConfig
	}

	return result, nil
}

// GetDepartmentWorkflowConfiguration возвращает полную конфигурацию workflow для кафедры
func (s *Service) GetDepartmentWorkflowConfiguration(ctx context.Context, departmentID int64, academicYear string) (*DepartmentWorkflowConfiguration, error) {
	wf, err := s.repo.GetActiveWorkflowByDepartment(ctx, departmentID)
	if err != nil {
		return nil, err
	}

	// Парсим настройки
	var wfSettings WorkflowSettings
	_ = json.Unmarshal(wf.Settings, &wfSettings)

	result := &DepartmentWorkflowConfiguration{
		WorkflowID:   wf.ID,
		WorkflowName: wf.Name,
		TeamConfig: &TeamConfig{
			MinSize:   wfSettings.MinTeamSize,
			MaxSize:   wfSettings.MaxTeamSize,
			AllowSolo: wfSettings.AllowSoloProject,
		},
		Settings: wfSettings,
	}

	// Добавляем states с конфигурациями
	for _, state := range wf.States {
		sc := StateConfigResult{
			ID:           state.ID,
			Name:         state.Name,
			DisplayName:  state.DisplayName,
			Type:         state.Type,
			OrderIndex:   state.OrderIndex,
			DurationDays: state.DurationDays,
		}

		var config map[string]interface{}
		_ = json.Unmarshal(state.Config, &config)
		sc.Config = config

		if state.FixedDeadline != nil {
			sc.Deadline = state.FixedDeadline
		} else if state.DurationDays > 0 {
			// Рассчитываем от текущей даты (для preview)
			deadline := time.Now().AddDate(0, 0, int(state.DurationDays))
			sc.Deadline = &deadline
		}

		result.States = append(result.States, sc)
	}

	// Применяем department config overrides
	deptConfig, _ := s.repo.GetDepartmentConfig(ctx, departmentID, academicYear)
	if deptConfig != nil {
		var ts TeamSettings
		if err := json.Unmarshal(deptConfig.TeamSettings, &ts); err == nil {
			result.TeamConfig.MinSize = ts.MinSize
			result.TeamConfig.MaxSize = ts.MaxSize
			result.TeamConfig.AllowSolo = ts.AllowSolo
		}

		// Применяем deadline overrides
		var deadlines []DeadlineOverride
		_ = json.Unmarshal(deptConfig.DeadlineOverrides, &deadlines)
		for _, d := range deadlines {
			for i := range result.States {
				if result.States[i].ID == d.StateID {
					if d.FixedDate != nil {
						result.States[i].Deadline = d.FixedDate
					}
					break
				}
			}
		}
	}

	return result, nil
}

// ==================== Versioning & Validation ====================

func (s *Service) CloneWorkflow(ctx context.Context, sourceID, targetDepartmentID int64, newName string) (*Workflow, error) {
	return s.repo.CloneWorkflow(ctx, sourceID, targetDepartmentID, newName)
}

func (s *Service) CreateNewVersion(ctx context.Context, workflowID int64) (*Workflow, error) {
	return s.repo.CreateNewVersion(ctx, workflowID)
}

func (s *Service) ValidateWorkflow(ctx context.Context, workflowID int64) (*ValidationResult, error) {
	return s.repo.ValidateWorkflow(ctx, workflowID)
}

// ==================== Input/Output Types ====================

type CreateWorkflowInput struct {
	Name         string
	Description  string
	DepartmentID int64
	Settings     map[string]interface{}
	IsTemplate   bool
}

type UpdateWorkflowInput struct {
	Name        string
	Description string
	Settings    map[string]interface{}
}

type CreateStateInput struct {
	WorkflowID   int64
	Name         string
	DisplayName  string
	Description  string
	OrderIndex   int32
	Type         string
	IsInitial    bool
	IsFinal      bool
	IsOptional   bool
	Config       map[string]interface{}
	DurationDays int32
	Color        string
	Icon         string
}

type UpdateStateInput struct {
	Name         string
	DisplayName  string
	Description  string
	Config       map[string]interface{}
	DurationDays int32
	OrderIndex   int32
	IsOptional   bool
}

type CreateTransitionInput struct {
	WorkflowID  int64
	EventName   string
	DisplayName string
	FromStateID int64
	ToStateID   int64
	Conditions  []TransitionCondition
	ButtonLabel string
	ButtonColor string
	ConfirmText string
}

type UpdateTransitionInput struct {
	DisplayName string
	ButtonLabel string
	ButtonColor string
	ConfirmText string
}

type CreateStateActionInput struct {
	StateID    int64
	Name       string
	Type       string
	Trigger    string
	OrderIndex int32
	Config     map[string]interface{}
	IsEnabled  bool
	Conditions []TransitionCondition
}

type UpdateStateActionInput struct {
	Name       string
	Config     map[string]interface{}
	OrderIndex int32
	IsEnabled  bool
}

type TeamConfigurationResult struct {
	WorkflowID       int64
	StateID          int64
	StateName        string
	TeamConfig       *TeamConfig
	InviteExpireDays int32
	TeamStepRequired bool
}

type StepConfigurationResult struct {
	StateID      int64
	StateName    string
	StateType    string
	DurationDays int32
	TeamConfig   *TeamConfig
	FileConfig   *FileConfig
	FormConfig   *FormConfig
	ReviewConfig *ReviewConfig
}

type DepartmentWorkflowConfiguration struct {
	WorkflowID   int64
	WorkflowName string
	TeamConfig   *TeamConfig
	States       []StateConfigResult
	Settings     WorkflowSettings
}

type StateConfigResult struct {
	ID           int64
	Name         string
	DisplayName  string
	Type         string
	OrderIndex   int32
	Config       map[string]interface{}
	Deadline     *time.Time
	DurationDays int32
}

type TransitionCondition struct {
	Type     string      `json:"type"`
	Field    string      `json:"field"`
	Operator string      `json:"operator"`
	Value    interface{} `json:"value"`
}

func (s *Service) ReorderStates(ctx context.Context, workflowID int64, stateIDs []int64) error {
	return s.repo.ReorderStates(ctx, workflowID, stateIDs)
}
