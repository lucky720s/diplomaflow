package workflow

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
)

type Repository interface {
	// Workflow CRUD
	CreateWorkflow(ctx context.Context, wf *Workflow) error
	GetWorkflow(ctx context.Context, id int64) (*Workflow, error)
	GetWorkflowFull(ctx context.Context, id int64) (*Workflow, error)
	GetWorkflowByName(ctx context.Context, name string) (*Workflow, error)
	ListWorkflows(ctx context.Context, departmentID int64) ([]*Workflow, error)
	UpdateWorkflow(ctx context.Context, wf *Workflow) error
	DeleteWorkflow(ctx context.Context, id int64) error

	// State CRUD
	CreateState(ctx context.Context, st *State) error
	GetState(ctx context.Context, id int64) (*State, error)
	ListStates(ctx context.Context, workflowID int64) ([]State, error)
	UpdateState(ctx context.Context, st *State) error
	DeleteState(ctx context.Context, id int64) error
	GetInitialState(ctx context.Context, workflowID int64) (*State, error)

	// Transition CRUD
	CreateTransition(ctx context.Context, tr *Transition) error
	GetTransition(ctx context.Context, id int64) (*Transition, error)
	GetTransitionsByWorkflow(ctx context.Context, workflowID int64) ([]Transition, error)
	GetTransitionsFromState(ctx context.Context, stateID int64) ([]Transition, error)
	UpdateTransition(ctx context.Context, tr *Transition) error
	DeleteTransition(ctx context.Context, id int64) error
	GetNextState(ctx context.Context, currentStateID int64, eventName string) (*State, error)

	// State Actions
	CreateStateAction(ctx context.Context, sa *StateAction) error
	GetStateAction(ctx context.Context, id int64) (*StateAction, error)
	ListStateActions(ctx context.Context, stateID int64) ([]StateAction, error)
	GetStateActionsByTrigger(ctx context.Context, stateID int64, trigger string) ([]StateAction, error)
	UpdateStateAction(ctx context.Context, sa *StateAction) error
	DeleteStateAction(ctx context.Context, id int64) error

	// Workflow Management
	SetActiveWorkflow(ctx context.Context, workflowID int64) (*Workflow, error)
	GetActiveWorkflowByDepartment(ctx context.Context, departmentID int64) (*Workflow, error)

	// Templates
	ListTemplates(ctx context.Context) ([]*WorkflowTemplate, error)
	GetTemplate(ctx context.Context, id int64) (*WorkflowTemplate, error)
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
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
	if err := r.db.WithContext(ctx).First(&wf, id).Error; err != nil {
		return nil, err
	}
	return &wf, nil
}

func (r *repository) GetWorkflowFull(ctx context.Context, id int64) (*Workflow, error) {
	var wf Workflow
	if err := r.db.WithContext(ctx).
		Preload("States", func(db *gorm.DB) *gorm.DB {
			return db.Order("order_index ASC")
		}).
		Preload("States.Actions", func(db *gorm.DB) *gorm.DB {
			return db.Where("is_enabled = ?", true).Order("order_index ASC")
		}).
		Preload("Transitions").
		First(&wf, id).Error; err != nil {
		return nil, err
	}
	return &wf, nil
}

func (r *repository) GetWorkflowByName(ctx context.Context, name string) (*Workflow, error) {
	var wf Workflow
	if err := r.db.WithContext(ctx).
		Preload("States", func(db *gorm.DB) *gorm.DB {
			return db.Order("order_index ASC")
		}).
		Where("name = ?", name).
		First(&wf).Error; err != nil {
		return nil, err
	}
	return &wf, nil
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

// ==================== State ====================

func (r *repository) CreateState(ctx context.Context, st *State) error {
	if st.OrderIndex == 0 {
		var maxOrder int32
		r.db.WithContext(ctx).
			Model(&State{}).
			Where("workflow_id = ?", st.WorkflowID).
			Select("COALESCE(MAX(order_index), 0)").
			Scan(&maxOrder)
		st.OrderIndex = maxOrder + 1
	}
	st.CreatedAt = time.Now()
	st.UpdatedAt = time.Now()
	return r.db.WithContext(ctx).Create(st).Error
}

func (r *repository) GetState(ctx context.Context, id int64) (*State, error) {
	var st State
	if err := r.db.WithContext(ctx).
		Preload("Actions", "is_enabled = ?", true).
		First(&st, id).Error; err != nil {
		return nil, err
	}
	return &st, nil
}

func (r *repository) ListStates(ctx context.Context, workflowID int64) ([]State, error) {
	var states []State
	err := r.db.WithContext(ctx).
		Where("workflow_id = ?", workflowID).
		Order("order_index ASC").
		Find(&states).Error
	return states, err
}

func (r *repository) UpdateState(ctx context.Context, st *State) error {
	st.UpdatedAt = time.Now()
	return r.db.WithContext(ctx).Save(st).Error
}

func (r *repository) DeleteState(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&State{}, id).Error
}

func (r *repository) GetInitialState(ctx context.Context, workflowID int64) (*State, error) {
	var st State
	if err := r.db.WithContext(ctx).
		Where("workflow_id = ? AND is_initial = ?", workflowID, true).
		First(&st).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Fallback: первый по order_index
			if createErr := r.db.WithContext(ctx).
				Where("workflow_id = ?", workflowID).
				Order("order_index ASC").
				First(&st).Error; createErr != nil {
				return nil, createErr
			}
			return &st, nil
		}
		return nil, err
	}
	return &st, nil
}

// ==================== Transition ====================

func (r *repository) CreateTransition(ctx context.Context, tr *Transition) error {
	tr.CreatedAt = time.Now()
	return r.db.WithContext(ctx).Create(tr).Error
}

func (r *repository) GetTransition(ctx context.Context, id int64) (*Transition, error) {
	var tr Transition
	if err := r.db.WithContext(ctx).First(&tr, id).Error; err != nil {
		return nil, err
	}
	return &tr, nil
}

func (r *repository) GetTransitionsByWorkflow(ctx context.Context, workflowID int64) ([]Transition, error) {
	var transitions []Transition
	err := r.db.WithContext(ctx).
		Where("workflow_id = ?", workflowID).
		Find(&transitions).Error
	return transitions, err
}

func (r *repository) GetTransitionsFromState(ctx context.Context, stateID int64) ([]Transition, error) {
	var transitions []Transition
	err := r.db.WithContext(ctx).
		Where("from_state_id = ?", stateID).
		Order("priority DESC").
		Find(&transitions).Error
	return transitions, err
}

func (r *repository) UpdateTransition(ctx context.Context, tr *Transition) error {
	return r.db.WithContext(ctx).Save(tr).Error
}

func (r *repository) DeleteTransition(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&Transition{}, id).Error
}

func (r *repository) GetNextState(ctx context.Context, currentStateID int64, eventName string) (*State, error) {
	var tr Transition
	if err := r.db.WithContext(ctx).
		Where("from_state_id = ? AND event_name = ?", currentStateID, eventName).
		First(&tr).Error; err != nil {
		return nil, fmt.Errorf("transition not found: %w", err)
	}
	return r.GetState(ctx, tr.ToStateID)
}

// ==================== State Actions ====================

func (r *repository) CreateStateAction(ctx context.Context, sa *StateAction) error {
	sa.CreatedAt = time.Now()
	sa.UpdatedAt = time.Now()
	return r.db.WithContext(ctx).Create(sa).Error
}

func (r *repository) GetStateAction(ctx context.Context, id int64) (*StateAction, error) {
	var sa StateAction
	if err := r.db.WithContext(ctx).First(&sa, id).Error; err != nil {
		return nil, err
	}
	return &sa, nil
}

func (r *repository) ListStateActions(ctx context.Context, stateID int64) ([]StateAction, error) {
	var actions []StateAction
	err := r.db.WithContext(ctx).
		Where("state_id = ?", stateID).
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

func (r *repository) UpdateStateAction(ctx context.Context, sa *StateAction) error {
	sa.UpdatedAt = time.Now()
	return r.db.WithContext(ctx).Save(sa).Error
}

func (r *repository) DeleteStateAction(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&StateAction{}, id).Error
}

// ==================== Workflow Management ====================

func (r *repository) SetActiveWorkflow(ctx context.Context, workflowID int64) (*Workflow, error) {
	var targetWf Workflow
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.First(&targetWf, workflowID).Error; err != nil {
			return fmt.Errorf("workflow not found: %w", err)
		}

		// Деактивируем все workflow этой кафедры
		if err := tx.Model(&Workflow{}).
			Where("department_id = ?", targetWf.DepartmentID).
			Update("is_active", false).Error; err != nil {
			return err
		}

		// Активируем выбранный
		targetWf.IsActive = true
		targetWf.UpdatedAt = time.Now()
		return tx.Save(&targetWf).Error
	})
	if err != nil {
		return nil, err
	}
	return &targetWf, nil
}

func (r *repository) GetActiveWorkflowByDepartment(ctx context.Context, departmentID int64) (*Workflow, error) {
	var wf Workflow
	if err := r.db.WithContext(ctx).
		Preload("States", func(db *gorm.DB) *gorm.DB {
			return db.Order("order_index ASC")
		}).
		Where("department_id = ? AND is_active = ?", departmentID, true).
		First(&wf).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("no active workflow found for department")
		}
		return nil, err
	}
	return &wf, nil
}

// ==================== Templates ====================

func (r *repository) ListTemplates(ctx context.Context) ([]*WorkflowTemplate, error) {
	var templates []*WorkflowTemplate
	err := r.db.WithContext(ctx).
		Order("name ASC").
		Find(&templates).Error
	return templates, err
}

func (r *repository) GetTemplate(ctx context.Context, id int64) (*WorkflowTemplate, error) {
	var tmpl WorkflowTemplate
	if err := r.db.WithContext(ctx).First(&tmpl, id).Error; err != nil {
		return nil, err
	}
	return &tmpl, nil
}
