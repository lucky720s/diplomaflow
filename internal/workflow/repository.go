// internal/workflow/repository.go

package workflow

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"time"
)

var (
	ErrWorkflowNotFound   = errors.New("workflow not found")
	ErrStateNotFound      = errors.New("state not found")
	ErrTransitionNotFound = errors.New("transition not found")
)

type Repository interface {
	// Workflow CRUD
	CreateWorkflow(ctx context.Context, wf *Workflow) error
	GetWorkflow(ctx context.Context, id int64) (*Workflow, error)
	GetWorkflowByName(ctx context.Context, name string) (*Workflow, error)
	GetWorkflowFull(ctx context.Context, id int64) (*Workflow, error)
	GetActiveWorkflowByDepartment(ctx context.Context, departmentID int64) (*Workflow, error)
	ListWorkflows(ctx context.Context, departmentID int64) ([]*Workflow, error)
	UpdateWorkflow(ctx context.Context, wf *Workflow) error
	DeleteWorkflow(ctx context.Context, id int64) error
	SetActiveWorkflow(ctx context.Context, workflowID int64) error
	DeactivateWorkflowsForDepartment(ctx context.Context, departmentID int64) error

	// State CRUD
	CreateState(ctx context.Context, state *State) error
	GetState(ctx context.Context, id int64) (*State, error)
	GetInitialState(ctx context.Context, workflowID int64) (*State, error)
	ListStates(ctx context.Context, workflowID int64) ([]State, error)
	UpdateState(ctx context.Context, state *State) error
	DeleteState(ctx context.Context, id int64) error
	ReorderStates(ctx context.Context, workflowID int64, stateIDs []int64) error

	// Transition CRUD
	CreateTransition(ctx context.Context, tr *Transition) error
	GetTransition(ctx context.Context, id int64) (*Transition, error)
	GetTransitionsFromState(ctx context.Context, stateID int64) ([]Transition, error)
	GetTransitionByEvent(ctx context.Context, fromStateID int64, eventName string) (*Transition, error)
	ListTransitions(ctx context.Context, workflowID int64) ([]Transition, error)
	UpdateTransition(ctx context.Context, tr *Transition) error
	DeleteTransition(ctx context.Context, id int64) error

	// State Actions
	CreateStateAction(ctx context.Context, action *StateAction) error
	GetStateAction(ctx context.Context, id int64) (*StateAction, error)
	ListStateActions(ctx context.Context, stateID int64) ([]StateAction, error)
	GetStateActionsByTrigger(ctx context.Context, stateID int64, trigger string) ([]StateAction, error)
	UpdateStateAction(ctx context.Context, action *StateAction) error
	DeleteStateAction(ctx context.Context, id int64) error

	// Versioning & Cloning
	CloneWorkflow(ctx context.Context, sourceID, targetDepartmentID int64, newName string) (*Workflow, error)
	CreateNewVersion(ctx context.Context, workflowID int64) (*Workflow, error)

	// Validation
	ValidateWorkflow(ctx context.Context, workflowID int64) (*ValidationResult, error)

	// Department Config
	GetDepartmentConfig(ctx context.Context, departmentID int64, academicYear string) (*DepartmentWorkflowConfig, error)
	CreateDepartmentConfig(ctx context.Context, config *DepartmentWorkflowConfig) error
	UpdateDepartmentConfig(ctx context.Context, config *DepartmentWorkflowConfig) error

	// Transaction support
	Transaction(ctx context.Context, fn func(tx *gorm.DB) error) error
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	_ = db.AutoMigrate(
		&Workflow{},
		&State{},
		&Transition{},
		&StateAction{},
		&StateCondition{},
		&DepartmentWorkflowConfig{},
		&DepartmentCustomStep{},
		&ActionRegistry{},
	)
	return &repository{db: db}
}

// ==================== Workflow ====================

func (r *repository) CreateWorkflow(ctx context.Context, wf *Workflow) error {
	wf.CreatedAt = time.Now()
	wf.UpdatedAt = time.Now()
	return r.db.WithContext(ctx).Create(wf).Error
}

func (r *repository) GetWorkflow(ctx context.Context, id int64) (*Workflow, error) {
	var wf Workflow
	err := r.db.WithContext(ctx).First(&wf, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrWorkflowNotFound
	}
	return &wf, err
}

func (r *repository) GetWorkflowByName(ctx context.Context, name string) (*Workflow, error) {
	var wf Workflow
	err := r.db.WithContext(ctx).Where("name = ?", name).First(&wf).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrWorkflowNotFound
	}
	return &wf, err
}

func (r *repository) GetWorkflowFull(ctx context.Context, id int64) (*Workflow, error) {
	var wf Workflow
	err := r.db.WithContext(ctx).
		Preload("States", func(db *gorm.DB) *gorm.DB {
			return db.Order("order_index ASC")
		}).
		Preload("States.Actions", func(db *gorm.DB) *gorm.DB {
			return db.Where("is_enabled = ?", true).Order("order_index ASC")
		}).
		Preload("Transitions").
		First(&wf, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrWorkflowNotFound
	}
	return &wf, err
}

func (r *repository) GetActiveWorkflowByDepartment(ctx context.Context, departmentID int64) (*Workflow, error) {
	var wf Workflow
	err := r.db.WithContext(ctx).
		Preload("States", func(db *gorm.DB) *gorm.DB {
			return db.Order("order_index ASC")
		}).
		Preload("Transitions").
		Where("department_id = ? AND is_active = ?", departmentID, true).
		First(&wf).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrWorkflowNotFound
	}
	return &wf, err
}

func (r *repository) ListWorkflows(ctx context.Context, departmentID int64) ([]*Workflow, error) {
	var wfs []*Workflow
	query := r.db.WithContext(ctx)
	if departmentID > 0 {
		query = query.Where("department_id = ?", departmentID)
	}
	err := query.Order("created_at DESC").Find(&wfs).Error
	return wfs, err
}

func (r *repository) UpdateWorkflow(ctx context.Context, wf *Workflow) error {
	wf.UpdatedAt = time.Now()
	return r.db.WithContext(ctx).Save(wf).Error
}

func (r *repository) DeleteWorkflow(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&Workflow{}, id).Error
}

func (r *repository) SetActiveWorkflow(ctx context.Context, workflowID int64) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Получаем workflow
		var wf Workflow
		if err := tx.First(&wf, workflowID).Error; err != nil {
			return err
		}

		// Деактивируем все workflow для этой кафедры
		if err := tx.Model(&Workflow{}).
			Where("department_id = ?", wf.DepartmentID).
			Update("is_active", false).Error; err != nil {
			return err
		}

		// Активируем выбранный
		return tx.Model(&Workflow{}).
			Where("id = ?", workflowID).
			Update("is_active", true).Error
	})
}

func (r *repository) DeactivateWorkflowsForDepartment(ctx context.Context, departmentID int64) error {
	return r.db.WithContext(ctx).
		Model(&Workflow{}).
		Where("department_id = ?", departmentID).
		Update("is_active", false).Error
}

// ==================== State ====================

func (r *repository) CreateState(ctx context.Context, state *State) error {
	state.CreatedAt = time.Now()
	state.UpdatedAt = time.Now()
	return r.db.WithContext(ctx).Create(state).Error
}

func (r *repository) GetState(ctx context.Context, id int64) (*State, error) {
	var state State
	err := r.db.WithContext(ctx).
		Preload("Actions", func(db *gorm.DB) *gorm.DB {
			return db.Where("is_enabled = ?", true).Order("order_index ASC")
		}).
		First(&state, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrStateNotFound
	}
	return &state, err
}

func (r *repository) GetInitialState(ctx context.Context, workflowID int64) (*State, error) {
	var state State
	// Сначала ищем по флагу is_initial
	err := r.db.WithContext(ctx).
		Where("workflow_id = ? AND is_initial = ?", workflowID, true).
		First(&state).Error
	if err == nil {
		return &state, nil
	}

	// Fallback: берём с минимальным order_index
	err = r.db.WithContext(ctx).
		Where("workflow_id = ?", workflowID).
		Order("order_index ASC").
		First(&state).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrStateNotFound
	}
	return &state, err
}

func (r *repository) ListStates(ctx context.Context, workflowID int64) ([]State, error) {
	var states []State
	err := r.db.WithContext(ctx).
		Where("workflow_id = ?", workflowID).
		Order("order_index ASC").
		Find(&states).Error
	return states, err
}

func (r *repository) UpdateState(ctx context.Context, state *State) error {
	state.UpdatedAt = time.Now()
	return r.db.WithContext(ctx).Save(state).Error
}

func (r *repository) DeleteState(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Удаляем связанные actions
		if err := tx.Where("state_id = ?", id).Delete(&StateAction{}).Error; err != nil {
			return err
		}
		// Удаляем связанные transitions
		if err := tx.Where("from_state_id = ? OR to_state_id = ?", id, id).Delete(&Transition{}).Error; err != nil {
			return err
		}
		// Удаляем state
		return tx.Delete(&State{}, id).Error
	})
}

func (r *repository) ReorderStates(ctx context.Context, workflowID int64, stateIDs []int64) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for i, id := range stateIDs {
			if err := tx.Model(&State{}).
				Where("id = ? AND workflow_id = ?", id, workflowID).
				Update("order_index", i+1).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// ==================== Transition ====================

func (r *repository) CreateTransition(ctx context.Context, tr *Transition) error {
	tr.CreatedAt = time.Now()
	return r.db.WithContext(ctx).Create(tr).Error
}

func (r *repository) GetTransition(ctx context.Context, id int64) (*Transition, error) {
	var tr Transition
	err := r.db.WithContext(ctx).First(&tr, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrTransitionNotFound
	}
	return &tr, err
}

func (r *repository) GetTransitionsFromState(ctx context.Context, stateID int64) ([]Transition, error) {
	var transitions []Transition
	err := r.db.WithContext(ctx).
		Where("from_state_id = ?", stateID).
		Order("priority DESC").
		Find(&transitions).Error
	return transitions, err
}

func (r *repository) GetTransitionByEvent(ctx context.Context, fromStateID int64, eventName string) (*Transition, error) {
	var tr Transition
	err := r.db.WithContext(ctx).
		Where("from_state_id = ? AND event_name = ?", fromStateID, eventName).
		First(&tr).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrTransitionNotFound
	}
	return &tr, err
}

func (r *repository) ListTransitions(ctx context.Context, workflowID int64) ([]Transition, error) {
	var transitions []Transition
	err := r.db.WithContext(ctx).
		Where("workflow_id = ?", workflowID).
		Find(&transitions).Error
	return transitions, err
}

func (r *repository) UpdateTransition(ctx context.Context, tr *Transition) error {
	return r.db.WithContext(ctx).Save(tr).Error
}

func (r *repository) DeleteTransition(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&Transition{}, id).Error
}

// ==================== State Actions ====================

func (r *repository) CreateStateAction(ctx context.Context, action *StateAction) error {
	action.CreatedAt = time.Now()
	action.UpdatedAt = time.Now()
	return r.db.WithContext(ctx).Create(action).Error
}

func (r *repository) GetStateAction(ctx context.Context, id int64) (*StateAction, error) {
	var action StateAction
	err := r.db.WithContext(ctx).First(&action, id).Error
	return &action, err
}

func (r *repository) ListStateActions(ctx context.Context, stateID int64) ([]StateAction, error) {
	var actions []StateAction
	err := r.db.WithContext(ctx).
		Where("state_id = ? AND is_enabled = ?", stateID, true).
		Order("order_index ASC").
		Find(&actions).Error
	return actions, err
}

func (r *repository) GetStateActionsByTrigger(ctx context.Context, stateID int64, trigger string) ([]StateAction, error) {
	var actions []StateAction
	err := r.db.WithContext(ctx).
		Where("state_id = ? AND trigger = ? AND is_enabled = ?", stateID, trigger, true).
		Order("order_index ASC").
		Find(&actions).Error
	return actions, err
}

func (r *repository) UpdateStateAction(ctx context.Context, action *StateAction) error {
	action.UpdatedAt = time.Now()
	return r.db.WithContext(ctx).Save(action).Error
}

func (r *repository) DeleteStateAction(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&StateAction{}, id).Error
}

// ==================== Versioning & Cloning ====================

func (r *repository) CloneWorkflow(ctx context.Context, sourceID, targetDepartmentID int64, newName string) (*Workflow, error) {
	var newWf *Workflow

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Получаем исходный workflow с relations
		var source Workflow
		if err := tx.Preload("States.Actions").Preload("Transitions").First(&source, sourceID).Error; err != nil {
			return err
		}

		// Создаём новый workflow
		newWf = &Workflow{
			Name:         newName,
			Description:  source.Description,
			DepartmentID: targetDepartmentID,
			Version:      1,
			IsActive:     false,
			IsTemplate:   false,
			ParentID:     &sourceID,
			Settings:     source.Settings,
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}
		if err := tx.Create(newWf).Error; err != nil {
			return err
		}

		// Mapping старых ID на новые для states
		stateIDMap := make(map[int64]int64)

		// Клонируем states
		for _, state := range source.States {
			newState := State{
				WorkflowID:    newWf.ID,
				Name:          state.Name,
				DisplayName:   state.DisplayName,
				Description:   state.Description,
				OrderIndex:    state.OrderIndex,
				Type:          state.Type,
				IsInitial:     state.IsInitial,
				IsFinal:       state.IsFinal,
				IsOptional:    state.IsOptional,
				Config:        state.Config,
				DurationDays:  state.DurationDays,
				DurationMode:  state.DurationMode,
				FixedDeadline: state.FixedDeadline,
				Color:         state.Color,
				Icon:          state.Icon,
				CreatedAt:     time.Now(),
				UpdatedAt:     time.Now(),
			}
			if err := tx.Create(&newState).Error; err != nil {
				return err
			}
			stateIDMap[state.ID] = newState.ID

			// Клонируем actions для state
			for _, action := range state.Actions {
				newAction := StateAction{
					StateID:           newState.ID,
					Name:              action.Name,
					Type:              action.Type,
					Trigger:           action.Trigger,
					OrderIndex:        action.OrderIndex,
					Config:            action.Config,
					IsEnabled:         action.IsEnabled,
					Conditions:        action.Conditions,
					MaxRetries:        action.MaxRetries,
					RetryDelaySeconds: action.RetryDelaySeconds,
					CreatedAt:         time.Now(),
					UpdatedAt:         time.Now(),
				}
				if err := tx.Create(&newAction).Error; err != nil {
					return err
				}
			}
		}

		// Клонируем transitions с обновлёнными state IDs
		for _, tr := range source.Transitions {
			newTr := Transition{
				WorkflowID:  newWf.ID,
				EventName:   tr.EventName,
				DisplayName: tr.DisplayName,
				FromStateID: stateIDMap[tr.FromStateID],
				ToStateID:   stateIDMap[tr.ToStateID],
				Conditions:  tr.Conditions,
				ButtonLabel: tr.ButtonLabel,
				ButtonColor: tr.ButtonColor,
				ConfirmText: tr.ConfirmText,
				Priority:    tr.Priority,
				CreatedAt:   time.Now(),
			}
			if err := tx.Create(&newTr).Error; err != nil {
				return err
			}
		}

		return nil
	})

	return newWf, err
}

func (r *repository) CreateNewVersion(ctx context.Context, workflowID int64) (*Workflow, error) {
	wf, err := r.GetWorkflow(ctx, workflowID)
	if err != nil {
		return nil, err
	}

	newName := wf.Name // Имя остаётся тем же
	newVersion := wf.Version + 1

	// Клонируем в ту же кафедру
	newWf, err := r.CloneWorkflow(ctx, workflowID, wf.DepartmentID, newName)
	if err != nil {
		return nil, err
	}

	// Обновляем версию
	newWf.Version = newVersion
	if err := r.UpdateWorkflow(ctx, newWf); err != nil {
		return nil, err
	}

	return newWf, nil
}

// ==================== Validation ====================

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

func (r *repository) ValidateWorkflow(ctx context.Context, workflowID int64) (*ValidationResult, error) {
	wf, err := r.GetWorkflowFull(ctx, workflowID)
	if err != nil {
		return nil, err
	}

	result := &ValidationResult{IsValid: true}

	// 1. Проверяем наличие initial state
	hasInitial := false
	for _, state := range wf.States {
		if state.IsInitial {
			hasInitial = true
			break
		}
	}
	if !hasInitial && len(wf.States) > 0 {
		result.Warnings = append(result.Warnings, ValidationWarning{
			Code:    "NO_INITIAL_STATE",
			Message: "No state is marked as initial, first state will be used",
			Path:    "states",
		})
	}

	// 2. Проверяем наличие final state
	hasFinal := false
	for _, state := range wf.States {
		if state.IsFinal {
			hasFinal = true
			break
		}
	}
	if !hasFinal {
		result.Warnings = append(result.Warnings, ValidationWarning{
			Code:    "NO_FINAL_STATE",
			Message: "No state is marked as final",
			Path:    "states",
		})
	}

	// 3. Проверяем что все states достижимы
	stateIDs := make(map[int64]bool)
	for _, state := range wf.States {
		stateIDs[state.ID] = false
	}
	// Initial state достижим
	for _, state := range wf.States {
		if state.IsInitial {
			stateIDs[state.ID] = true
		}
	}
	// Проходим по transitions
	for changed := true; changed; {
		changed = false
		for _, tr := range wf.Transitions {
			if stateIDs[tr.FromStateID] && !stateIDs[tr.ToStateID] {
				stateIDs[tr.ToStateID] = true
				changed = true
			}
		}
	}
	for id, reachable := range stateIDs {
		if !reachable {
			for _, state := range wf.States {
				if state.ID == id {
					result.Errors = append(result.Errors, ValidationError{
						Code:    "UNREACHABLE_STATE",
						Message: "State is not reachable from initial state",
						Path:    "states[" + state.Name + "]",
					})
					result.IsValid = false
					break
				}
			}
		}
	}

	// 4. Проверяем transitions
	for _, tr := range wf.Transitions {
		// Проверяем что from_state существует
		fromExists := false
		toExists := false
		for _, state := range wf.States {
			if state.ID == tr.FromStateID {
				fromExists = true
			}
			if state.ID == tr.ToStateID {
				toExists = true
			}
		}
		if !fromExists {
			result.Errors = append(result.Errors, ValidationError{
				Code:    "INVALID_FROM_STATE",
				Message: "Transition references non-existent from_state",
				Path:    "transitions[" + tr.EventName + "]",
			})
			result.IsValid = false
		}
		if !toExists {
			result.Errors = append(result.Errors, ValidationError{
				Code:    "INVALID_TO_STATE",
				Message: "Transition references non-existent to_state",
				Path:    "transitions[" + tr.EventName + "]",
			})
			result.IsValid = false
		}
	}

	return result, nil
}

// ==================== Department Config ====================

func (r *repository) GetDepartmentConfig(ctx context.Context, departmentID int64, academicYear string) (*DepartmentWorkflowConfig, error) {
	var config DepartmentWorkflowConfig
	query := r.db.WithContext(ctx).Where("department_id = ? AND is_active = ?", departmentID, true)
	if academicYear != "" {
		query = query.Where("academic_year = ?", academicYear)
	}
	err := query.First(&config).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil // Нет конфига — используем defaults
	}
	return &config, err
}

func (r *repository) CreateDepartmentConfig(ctx context.Context, config *DepartmentWorkflowConfig) error {
	config.CreatedAt = time.Now()
	config.UpdatedAt = time.Now()
	return r.db.WithContext(ctx).Create(config).Error
}

func (r *repository) UpdateDepartmentConfig(ctx context.Context, config *DepartmentWorkflowConfig) error {
	config.UpdatedAt = time.Now()
	return r.db.WithContext(ctx).Save(config).Error
}

func (r *repository) Transaction(ctx context.Context, fn func(tx *gorm.DB) error) error {
	return r.db.WithContext(ctx).Transaction(fn)
}
