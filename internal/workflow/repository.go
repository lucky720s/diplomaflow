package workflow

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"
)

type Repository interface {
	CreateWorkflow(ctx context.Context, wf *Workflow) error
	GetWorkflow(ctx context.Context, id int64) (*Workflow, error)
	GetWorkflowByName(ctx context.Context, name string) (*Workflow, error)
	ListWorkflows(ctx context.Context, departmentID int64) ([]*Workflow, error)
	UpdateWorkflow(ctx context.Context, wf *Workflow) error
	DeleteWorkflow(ctx context.Context, id int64) error

	CreateState(ctx context.Context, st *State) error
	GetState(ctx context.Context, id int64) (*State, error)
	ListStates(ctx context.Context, workflowID int64) ([]*State, error)
	UpdateState(ctx context.Context, st *State) error
	DeleteState(ctx context.Context, id int64) error

	CreateTransition(ctx context.Context, tr *Transition) error
	DeleteTransition(ctx context.Context, id int64) error
	GetNextState(ctx context.Context, currentStateID int64, eventName string) (*State, error)

	CreateStateAction(ctx context.Context, sa *StateAction) error
	ListStateActions(ctx context.Context, stateID int64) ([]*StateAction, error)
	DeleteStateAction(ctx context.Context, id int64) error

	SetActiveWorkflow(ctx context.Context, workflowID int64) (*Workflow, error)
	GetActiveWorkflowByDepartment(ctx context.Context, departmentID int64) (*Workflow, error)
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	_ = db.AutoMigrate(&Workflow{}, &State{}, &Transition{}, &StateAction{})
	return &repository{db: db}
}

func (r *repository) CreateWorkflow(ctx context.Context, wf *Workflow) error {
	return r.db.WithContext(ctx).Create(wf).Error
}

func (r *repository) GetWorkflow(ctx context.Context, id int64) (*Workflow, error) {
	var wf Workflow
	if err := r.db.WithContext(ctx).Preload("Steps").First(&wf, id).Error; err != nil {
		return nil, err
	}
	return &wf, nil
}
func (r *repository) GetWorkflowByName(ctx context.Context, name string) (*Workflow, error) {
	var wf Workflow
	if err := r.db.WithContext(ctx).Preload("Steps").Where("name = ?", name).First(&wf).Error; err != nil {
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
	err := query.Find(&wfs).Error
	return wfs, err
}

func (r *repository) UpdateWorkflow(ctx context.Context, wf *Workflow) error {
	return r.db.WithContext(ctx).Save(wf).Error
}

func (r *repository) DeleteWorkflow(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&Workflow{}, id).Error
}

func (r *repository) CreateState(ctx context.Context, st *State) error {
	return r.db.WithContext(ctx).Create(st).Error
}

func (r *repository) GetState(ctx context.Context, id int64) (*State, error) {
	var st State
	if err := r.db.WithContext(ctx).First(&st, id).Error; err != nil {
		return nil, err
	}
	return &st, nil
}

func (r *repository) ListStates(ctx context.Context, workflowID int64) ([]*State, error) {
	var states []*State
	err := r.db.WithContext(ctx).Where("workflow_id = ?", workflowID).Find(&states).Error
	return states, err
}

func (r *repository) UpdateState(ctx context.Context, st *State) error {
	return r.db.WithContext(ctx).Save(st).Error
}

func (r *repository) DeleteState(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&State{}, id).Error
}

func (r *repository) CreateTransition(ctx context.Context, tr *Transition) error {
	return r.db.WithContext(ctx).Create(tr).Error
}

func (r *repository) DeleteTransition(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&Transition{}, id).Error
}

func (r *repository) GetNextState(ctx context.Context, currentStateID int64, eventName string) (*State, error) {
	var tr Transition
	if err := r.db.WithContext(ctx).Where("from_state_id = ? AND event_name = ?", currentStateID, eventName).First(&tr).Error; err != nil {
		return nil, err
	}
	return r.GetState(ctx, tr.ToStateID)
}

func (r *repository) CreateStateAction(ctx context.Context, sa *StateAction) error {
	return r.db.WithContext(ctx).Create(sa).Error
}

func (r *repository) ListStateActions(ctx context.Context, stateID int64) ([]*StateAction, error) {
	var actions []*StateAction
	err := r.db.WithContext(ctx).Where("state_id = ?", stateID).Find(&actions).Error
	return actions, err
}

func (r *repository) DeleteStateAction(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&StateAction{}, id).Error
}

func (r *repository) SetActiveWorkflow(ctx context.Context, workflowID int64) (*Workflow, error) {
	var targetWf Workflow

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.First(&targetWf, workflowID).Error; err != nil {
			return fmt.Errorf("workflow not found: %w", err)
		}

		if err := tx.Model(&Workflow{}).
			Where("department_id = ?", targetWf.DepartmentID).
			Update("is_active", false).Error; err != nil {
			return err
		}

		targetWf.IsActive = true
		if err := tx.Save(&targetWf).Error; err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		return nil, err
	}
	return &targetWf, nil
}

func (r *repository) GetActiveWorkflowByDepartment(ctx context.Context, departmentID int64) (*Workflow, error) {
	var wf Workflow
	if err := r.db.WithContext(ctx).
		Preload("Steps").
		Where("department_id = ? AND is_active = ?", departmentID, true).First(&wf).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("no active workflow found")
		}
		return nil, err
	}
	return &wf, nil
}
