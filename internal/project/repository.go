package project

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/lucky720s/diplomaflow/internal/project/service"
	projectv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/project/v1"
	workflowv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/workflow/v1"
	rkpostgres "github.com/rookie-ninja/rk-db/postgres"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

type Repository interface {
	CreateProject(ctx context.Context, req *projectv1.CreateProjectRequest) (*Project, error)
	GetProject(ctx context.Context, projectID int64) (*Project, error)
	ListProjects(ctx context.Context, req *projectv1.ListProjectsRequest) ([]*Project, error)
	PerformStateAction(ctx context.Context, req *projectv1.PerformStateActionRequest) error
}

type repository struct {
	db              *gorm.DB
	workflowClient  workflowv1.WorkflowServiceClient
	stateProcessors map[string]service.StateProcessor
}

func (Project) TableName() string          { return "project_schema.projects" }
func (ProjectStateData) TableName() string { return "project_schema.project_state_data" }

func NewRepository(wfClient workflowv1.WorkflowServiceClient) (Repository, error) {
	pgEntry := rkpostgres.GetPostgresEntry("project-conn")
	dbName := os.Getenv("MAIN_POSTGRES_DB_NAME")
	db := pgEntry.GetDB(dbName)
	if db == nil {
		return nil, fmt.Errorf("Database '%s' not found", dbName)
	}

	if err := db.AutoMigrate(&Project{}, &ProjectStateData{}); err != nil {
		return nil, fmt.Errorf("AutoMigrate error: %v", err)
	}
	processors := map[string]service.StateProcessor{
		workflowv1.StateType_SUPERVISOR_SELECTION.String(): &service.SupervisorSelectionProcessor{},
		workflowv1.StateType_DOCUMENT_UPLOAD.String():      &service.DocumentUploadProcessor{},
	}

	return &repository{db: db, workflowClient: wfClient, stateProcessors: processors}, nil
}

func (r *repository) CreateProject(ctx context.Context, req *projectv1.CreateProjectRequest) (*Project, error) {
	initialStateID := int64(1)

	newProject := &Project{
		Title:          req.GetTitle(),
		WorkflowID:     req.GetWorkflowId(),
		CurrentStateID: initialStateID,
		Status:         "IN_PROGRESS",
	}

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(newProject).Error; err != nil {
			return err
		}

		initialStateInfo, err := r.workflowClient.GetState(ctx, &workflowv1.GetStateRequest{StateId: initialStateID})
		if err != nil {
			return err
		}

		var deadline *time.Time
		if duration := initialStateInfo.GetDurationDays(); duration > 0 {
			deadlineTime := time.Now().UTC().AddDate(0, 0, int(duration))
			deadline = &deadlineTime
		}

		initialStateData := &ProjectStateData{
			ProjectID:  newProject.ID,
			StateID:    initialStateID,
			Status:     "IN_PROGRESS",
			Data:       []byte("{}"),
			DeadlineAt: deadline,
		}
		return tx.Create(initialStateData).Error
	})

	return newProject, err
}

func (r *repository) GetProject(ctx context.Context, projectID int64) (*Project, error) {
	var p Project
	err := r.db.WithContext(ctx).First(&p, projectID).Error
	return &p, err
}
func (r *repository) ListProjects(ctx context.Context, req *projectv1.ListProjectsRequest) ([]*Project, error) {
	var projects []*Project
	err := r.db.WithContext(ctx).Where("department_id = ?", req.GetDepartmentId()).Find(&projects).Error
	if err != nil {
		return nil, err
	}
	return projects, nil
}
func (r *repository) PerformStateAction(ctx context.Context, req *projectv1.PerformStateActionRequest) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var activeStateData ProjectStateData
		if err := tx.Where("project_id = ? AND status = ?", req.GetProjectId(), "IN_PROGRESS").First(&activeStateData).Error; err != nil {
			return status.Errorf(codes.NotFound, "no active state found for project %d: %v", req.GetProjectId(), err)
		}

		stateInfo, err := r.workflowClient.GetState(ctx, &workflowv1.GetStateRequest{StateId: activeStateData.StateID})
		if err != nil {
			return status.Errorf(codes.FailedPrecondition, "cannot get state info: %v", err)
		}

		processor, ok := r.stateProcessors[stateInfo.GetType().String()]
		if !ok {
			return status.Errorf(codes.Internal, "no processor for state type %s", stateInfo.GetType().String())
		}

		payloadMap := req.GetPayload().AsMap()
		newData, isCompleted, err := processor.ProcessAction(ctx, activeStateData.Data, req.GetAction(), payloadMap)
		if err != nil {
			return status.Errorf(codes.InvalidArgument, "action processing failed: %v", err)
		}

		activeStateData.Data = newData
		if isCompleted {
			activeStateData.Status = "COMPLETED"
		}
		if err := tx.Save(&activeStateData).Error; err != nil {
			return err
		}

		if isCompleted {
			r.processSideEffects(ctx, activeStateData.StateID, workflowv1.StateAction_ON_EXIT)

			var project Project
			if err := tx.First(&project, req.GetProjectId()).Error; err != nil {
				return err
			}

			eventName := "STATE_COMPLETED"

			nextState, err := r.workflowClient.GetNextState(ctx, &workflowv1.GetNextStateRequest{CurrentStateId: activeStateData.StateID, EventName: eventName})
			if err != nil {
				if status.Code(err) == codes.NotFound {
					project.Status = "COMPLETED"
					return tx.Save(&project).Error
				}
				return err
			}

			project.CurrentStateID = nextState.GetId()
			if err := tx.Save(&project).Error; err != nil {
				return err
			}

			var deadline *time.Time
			if duration := nextState.GetDurationDays(); duration > 0 {
				deadlineTime := time.Now().UTC().AddDate(0, 0, int(duration))
				deadline = &deadlineTime
			}

			newStateData := &ProjectStateData{
				ProjectID:  project.ID,
				StateID:    project.CurrentStateID,
				Status:     "IN_PROGRESS",
				Data:       []byte("{}"),
				DeadlineAt: deadline,
			}
			if err := tx.Create(&newStateData).Error; err != nil {
				return err
			}

			r.processSideEffects(ctx, newStateData.StateID, workflowv1.StateAction_ON_ENTER)
		}
		return nil
	})
}

func (r *repository) processSideEffects(ctx context.Context, stateID int64, trigger workflowv1.StateAction_Trigger) {
	actions, err := r.workflowClient.ListStateActions(ctx, &workflowv1.ListStateActionsRequest{StateId: stateID})
	if err != nil {
		log.Printf("ERROR: failed to list side effects for state %d: %v", stateID, err)
		return
	}

	for _, action := range actions.GetActions() {
		if action.GetTrigger() == trigger {
			config := action.GetConfig().AsMap()
			log.Printf("INFO: Executing side effect: Type=%s, Trigger=%s, Config=%v", action.GetType(), trigger, config)
		}
	}
}
