package workflow

import (
	"context"
	"fmt"
	"os"

	rkpostgres "github.com/rookie-ninja/rk-db/postgres"
	"gorm.io/gorm"
)

type Workflow struct {
	ID           int64 `gorm:"primaryKey"`
	Name         string
	DepartmentID int64 `gorm:"uniqueIndex"`
}
type Stage struct {
	ID                int64 `gorm:"primaryKey"`
	Name              string
	WorkflowID        int64 `gorm:"index:idx_workflow_order"`
	Order             int64 `gorm:"index:idx_workflow_order"`
	ResponsibleRoleID int64
	DeadlineDays      int64
}
type Repository interface {
	CreateWorkflow(ctx context.Context, name string, departmentID int64) (*Workflow, error)
	GetWorkflowByDepartmentID(ctx context.Context, departmentID int64) (*Workflow, []Stage, error)
	CreateStage(ctx context.Context, name string, workflowID int64, order int64, roleID int64, deadline int64) (*Stage, error)
	GetStageByID(ctx context.Context, workflowID int64) (*Stage, error)
	GetNextStage(ctx context.Context, workflowID int64, currentOrder int64) (*Stage, error)
}
type repository struct {
	db *gorm.DB
}

func NewRepository() (Repository, error) {
	pgEntry := rkpostgres.GetPostgresEntry("workflow-conn")
	dbName := os.Getenv("WORKFLOW_DB_NAME")
	db := pgEntry.GetDB(dbName)
	if db == nil {
		panic("Database not found")
	}
	if err := db.AutoMigrate(&Workflow{}, &Stage{}); err != nil {
		return nil, fmt.Errorf("auto migrate workflow err: %w", err)
	}
	return &repository{db: db}, nil
}

func (r *repository) CreateWorkflow(ctx context.Context, name string, departmentID int64) (*Workflow, error) {
	workflow := &Workflow{
		Name:         name,
		DepartmentID: departmentID,
	}
	res := r.db.WithContext(ctx).Create(workflow)
	return workflow, res.Error
}

func (r *repository) GetWorkflowByDepartmentID(ctx context.Context, departmentID int64) (*Workflow, []Stage, error) {
	var workflow Workflow
	if err := r.db.WithContext(ctx).Where("department_id = ?", departmentID).First(&workflow).Error; err != nil {
		return nil, []Stage{}, err
	}
	var stages []Stage
	if err := r.db.WithContext(ctx).Where("workflow_id=?", workflow.ID).Order("\"order\"asc").Find(&stages).Error; err != nil {
		return nil, nil, err
	}
	return &workflow, stages, nil
}

func (r *repository) CreateStage(ctx context.Context, name string, workflowID int64, order int64, roleID int64, deadline int64) (*Stage, error) {
	stage := &Stage{
		Name:              name,
		WorkflowID:        workflowID,
		Order:             order,
		ResponsibleRoleID: roleID,
		DeadlineDays:      deadline,
	}
	res := r.db.WithContext(ctx).Create(stage)
	return stage, res.Error
}
func (r *repository) GetStageByID(ctx context.Context, workflowID int64) (*Stage, error) {
	var stage Stage
	err := r.db.WithContext(ctx).First(&stage, stage.ID).Error
	return &stage, err
}
func (r *repository) GetNextStage(ctx context.Context, workflowID int64, currentOrder int64) (*Stage, error) {
	var stage Stage
	err := r.db.WithContext(ctx).Where("workflow_id = ? AND \"order\" >?", workflowID, currentOrder).Order("\"order\" asc").First(&stage).Error
	return &stage, err
}
