package workflow

import (
	"context"
	"fmt"
	"os"

	workflowv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/workflow/v1"
	rkpostgres "github.com/rookie-ninja/rk-db/postgres"
	"gorm.io/gorm"
)

type Repository interface {
	CreateWorkflow(ctx context.Context, req *workflowv1.CreateWorkflowRequest) (*Workflow, error)
	GetWorkflow(ctx context.Context, id int64) (*Workflow, error)
	ListWorkflows(ctx context.Context, departmentID int64) ([]*Workflow, error)
	UpdateWorkflow(ctx context.Context, req *workflowv1.UpdateWorkflowRequest) (*Workflow, error)
	DeleteWorkflow(ctx context.Context, id int64) error

	CreateState(ctx context.Context, req *workflowv1.CreateStateRequest) (*State, error)
	GetState(ctx context.Context, id int64) (*State, error)
	ListStates(ctx context.Context, workflowID int64) ([]*State, error)
	UpdateState(ctx context.Context, req *workflowv1.UpdateStateRequest) (*State, error)
	DeleteState(ctx context.Context, id int64) error

	CreateTransition(ctx context.Context, req *workflowv1.CreateTransitionRequest) (*Transition, error)
	DeleteTransition(ctx context.Context, id int64) error

	CreateStateAction(ctx context.Context, req *workflowv1.CreateStateActionRequest) (*StateAction, error)
	ListStateActions(ctx context.Context, stateID int64) ([]*StateAction, error)
	DeleteStateAction(ctx context.Context, id int64) error

	GetNextState(ctx context.Context, currentStateID int64, eventName string) (*State, error)
}

type repository struct {
	db *gorm.DB
}

func (Workflow) TableName() string    { return "workflow_schema.workflows" }
func (State) TableName() string       { return "workflow_schema.states" }
func (Transition) TableName() string  { return "workflow_schema.transitions" }
func (StateAction) TableName() string { return "workflow_schema.state_actions" }

func NewRepository() (Repository, error) {
	pgEntry := rkpostgres.GetPostgresEntry("workflow-conn")
	dbName := os.Getenv("MAIN_POSTGRES_DB_NAME")
	db := pgEntry.GetDB(dbName)
	if db == nil {
		return nil, fmt.Errorf("Database '%s' not found", dbName)
	}
	if err := db.AutoMigrate(&Workflow{}, &State{}, &Transition{}, &StateAction{}); err != nil {
		return nil, fmt.Errorf("AutoMigrate error: %v", err)
	}
	return &repository{db: db}, nil
}

func (r *repository) CreateWorkflow(ctx context.Context, req *workflowv1.CreateWorkflowRequest) (*Workflow, error) {
	wf := &Workflow{Name: req.GetName(), DepartmentID: req.GetDepartmentId()}
	if err := r.db.WithContext(ctx).Create(wf).Error; err != nil {
		return nil, err
	}
	return wf, nil
}
func (r *repository) GetWorkflow(ctx context.Context, id int64) (*Workflow, error) {
	var wf Workflow
	err := r.db.WithContext(ctx).First(&wf, id).Error
	return &wf, err
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
func (r *repository) UpdateWorkflow(ctx context.Context, req *workflowv1.UpdateWorkflowRequest) (*Workflow, error) {
	var wf Workflow
	if err := r.db.WithContext(ctx).First(&wf, req.GetId()).Error; err != nil {
		return nil, err
	}
	wf.Name = req.GetName()
	if err := r.db.WithContext(ctx).Save(&wf).Error; err != nil {
		return nil, err
	}
	return &wf, nil
}
func (r *repository) DeleteWorkflow(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&Workflow{}, id).Error
}

func (r *repository) CreateState(ctx context.Context, req *workflowv1.CreateStateRequest) (*State, error) {
	configBytes, _ := req.GetConfig().MarshalJSON()
	st := &State{
		WorkflowID:   req.GetWorkflowId(),
		Name:         req.GetName(),
		Description:  req.GetDescription(),
		Type:         req.GetType().String(),
		Config:       configBytes,
		DurationDays: req.GetDurationDays(),
	}
	if err := r.db.WithContext(ctx).Create(st).Error; err != nil {
		return nil, err
	}
	return st, nil
}
func (r *repository) GetState(ctx context.Context, id int64) (*State, error) {
	var state State
	err := r.db.WithContext(ctx).First(&state, id).Error
	return &state, err
}
func (r *repository) ListStates(ctx context.Context, workflowID int64) ([]*State, error) {
	var states []*State
	err := r.db.WithContext(ctx).Where("workflow_id = ?", workflowID).Find(&states).Error
	return states, err
}
func (r *repository) UpdateState(ctx context.Context, req *workflowv1.UpdateStateRequest) (*State, error) {
	var st State
	if err := r.db.WithContext(ctx).First(&st, req.GetId()).Error; err != nil {
		return nil, err
	}
	configBytes, _ := req.GetConfig().MarshalJSON()
	st.Name = req.GetName()
	st.Description = req.GetDescription()
	st.Config = configBytes
	st.DurationDays = req.GetDurationDays()
	if err := r.db.WithContext(ctx).Save(&st).Error; err != nil {
		return nil, err
	}
	return &st, nil
}
func (r *repository) DeleteState(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&State{}, id).Error
}

func (r *repository) CreateTransition(ctx context.Context, req *workflowv1.CreateTransitionRequest) (*Transition, error) {
	tr := &Transition{
		WorkflowID:  req.GetWorkflowId(),
		EventName:   req.GetEventName(),
		FromStateID: req.GetFromStateId(),
		ToStateID:   req.GetToStateId(),
	}
	if err := r.db.WithContext(ctx).Create(tr).Error; err != nil {
		return nil, err
	}
	return tr, nil
}
func (r *repository) DeleteTransition(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&Transition{}, id).Error
}
func (r *repository) CreateStateAction(ctx context.Context, req *workflowv1.CreateStateActionRequest) (*StateAction, error) {
	configBytes, _ := req.GetConfig().MarshalJSON()
	sa := &StateAction{
		StateID: req.GetStateId(),
		Type:    req.GetType().String(),
		Trigger: req.GetTrigger().String(),
		Config:  configBytes,
	}
	if err := r.db.WithContext(ctx).Create(sa).Error; err != nil {
		return nil, err
	}
	return sa, nil
}
func (r *repository) ListStateActions(ctx context.Context, stateID int64) ([]*StateAction, error) {
	var actions []*StateAction
	err := r.db.WithContext(ctx).Where("state_id = ?", stateID).Find(&actions).Error
	return actions, err
}
func (r *repository) DeleteStateAction(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&StateAction{}, id).Error
}
func (r *repository) GetNextState(ctx context.Context, currentStateID int64, eventName string) (*State, error) {
	var tr Transition
	if err := r.db.WithContext(ctx).Where("from_state_id = ? AND event_name = ?", currentStateID, eventName).First(&tr).Error; err != nil {
		return nil, err
	}
	return r.GetState(ctx, tr.ToStateID)
}
