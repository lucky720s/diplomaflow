package project

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	projectv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/project/v1"
	workflowv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/workflow/v1"
	"go.uber.org/zap"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Service struct {
	repo           Repository
	workflowClient WorkflowClient
	logger         *zap.Logger
}

func NewService(repo Repository, wfClient WorkflowClient, logger *zap.Logger) *Service {
	return &Service{repo: repo, workflowClient: wfClient, logger: logger}
}

func (s *Service) CreateProject(ctx context.Context, req *projectv1.CreateProjectRequest) (*projectv1.CreateProjectResponse, error) {
	wf, err := s.workflowClient.GetActiveWorkflowByDepartment(ctx, &workflowv1.GetActiveWorkflowByDepartmentRequest{
		DepartmentId: req.DepartmentId,
	})
	if err != nil {
		return nil, errors.New("no active workflow configured for this department")
	}
	if len(wf.States) == 0 {
		return nil, errors.New("workflow has no states configured")
	}

	// initial state
	initial := wf.States[0]
	for _, st := range wf.States {
		if st.IsInitial {
			initial = st
			break
		}
	}

	var deadlineAt *time.Time
	if initial.DurationDays > 0 {
		d := time.Now().UTC().AddDate(0, 0, int(initial.DurationDays))
		deadlineAt = &d
	} else if initial.FixedDeadline != nil {
		d := initial.FixedDeadline.AsTime().UTC()
		deadlineAt = &d
	}

	p := &Project{
		Title:            req.Title,
		Description:      req.Description,
		StudentID:        req.StudentId,
		UniversityID:     req.UniversityId,
		DepartmentID:     req.DepartmentId,
		TeamID:           req.TeamId,
		WorkflowID:       wf.Id,
		WorkflowVersion:  wf.Version,
		WorkflowName:     wf.Name,
		CurrentStateID:   initial.Id,
		CurrentStateName: initial.Name,
		Status:           "active",
		Data:             datatypes.JSON([]byte(`{}`)),
		DeadlineAt:       deadlineAt,
		CreatedAt:        time.Now().UTC(),
		UpdatedAt:        time.Now().UTC(),
	}

	// IMPORTANT: backward/forward compatible contract
	eventPayload := map[string]interface{}{
		"student_id":       req.StudentId,
		"university_id":    req.UniversityId,
		"department_id":    req.DepartmentId,
		"workflow_id":      wf.Id,
		"initial_state_id": initial.Id,
		"first_state_id":   initial.Id,
		"initial_state":    initial.Name,
		"title":            req.Title,
	}

	if err := s.repo.CreateWithOutbox(ctx, p, "ProjectCreated", "project-events", eventPayload); err != nil {
		return nil, err
	}

	return &projectv1.CreateProjectResponse{
		ProjectId: p.ID,
		Status:    p.Status,
	}, nil
}

func (s *Service) GetProject(ctx context.Context, id int64) (*Project, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *Service) GetStudentProjects(ctx context.Context, studentID int64) ([]*Project, error) {
	return s.repo.ListByStudent(ctx, studentID)
}

// Legacy frontend endpoint: delegates to workflow runtime
func (s *Service) PerformAction(ctx context.Context, projectID int64, actionName string, payload map[string]interface{}, userID int64, userRole string) (*Project, error) {
	p, err := s.repo.GetRuntimeByID(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("project not found: %w", err)
	}
	if p.Status != "active" {
		return nil, errors.New("cannot perform action on inactive project")
	}

	// forward role via metadata to workflow_service runtime
	outCtx := metadata.AppendToOutgoingContext(ctx, "x-user-role", userRole)

	trResp, err := s.workflowClient.GetAvailableTransitions(outCtx, &workflowv1.GetAvailableTransitionsRequest{
		ProjectId:      projectID,
		CurrentStateId: p.CurrentStateID,
		UserId:         userID,
		UserRole:       userRole,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get transitions: %w", err)
	}

	var transitionID int64
	for _, t := range trResp.Transitions {
		if t != nil && t.Transition != nil && t.Transition.EventName == actionName {
			if !t.CanExecute {
				return nil, fmt.Errorf("transition blocked: %s", t.BlockedReason)
			}
			transitionID = t.Transition.Id
			break
		}
	}

	if transitionID == 0 {
		return nil, fmt.Errorf("unknown action/transition: %s", actionName)
	}

	pbPayload, _ := structpb.NewStruct(payload)
	execResp, err := s.workflowClient.ExecuteTransition(outCtx, &workflowv1.ExecuteTransitionRequest{
		ProjectId:    projectID,
		TransitionId: transitionID,
		UserId:       userID,
		Payload:      pbPayload,
	})
	if err != nil {
		return nil, fmt.Errorf("workflow execute failed: %w", err)
	}
	if !execResp.Success {
		return nil, fmt.Errorf("workflow execute failed: %s", execResp.ErrorMessage)
	}

	return s.repo.GetByID(ctx, projectID)
}

// Internal RPC for workflow ядра
func (s *Service) GetProjectRuntime(ctx context.Context, projectID int64) (*projectv1.GetProjectRuntimeResponse, error) {
	p, err := s.repo.GetRuntimeByID(ctx, projectID)
	if err != nil {
		return nil, err
	}

	dataMap := map[string]interface{}{}
	if len(p.Data) > 0 {
		_ = json.Unmarshal(p.Data, &dataMap)
	}
	dataStruct, _ := structpb.NewStruct(dataMap)

	resp := &projectv1.GetProjectRuntimeResponse{
		ProjectId:        p.ID,
		StudentId:        p.StudentID,
		UniversityId:     p.UniversityID,
		DepartmentId:     p.DepartmentID,
		TeamId:           p.TeamID,
		WorkflowId:       p.WorkflowID,
		WorkflowVersion:  p.WorkflowVersion,
		WorkflowName:     p.WorkflowName,
		CurrentStateId:   p.CurrentStateID,
		CurrentStateName: p.CurrentStateName,
		Status:           p.Status,
		Data:             dataStruct,
	}

	if p.DeadlineAt != nil {
		resp.DeadlineAt = timestamppb.New(*p.DeadlineAt)
	}

	return resp, nil
}

func (s *Service) CommitTransition(ctx context.Context, req *projectv1.CommitTransitionRequest) (*projectv1.CommitTransitionResponse, error) {
	err := s.repo.Transaction(ctx, func(tx *gorm.DB) error {
		var p Project
		if err := tx.WithContext(ctx).
			Clauses(clause.Locking{Strength: "UPDATE"}).
			First(&p, "id = ?", req.ProjectId).Error; err != nil {
			return err
		}

		if p.CurrentStateID != req.ExpectedFromStateId {
			return fmt.Errorf("concurrent update: expected %d got %d", req.ExpectedFromStateId, p.CurrentStateID)
		}

		// merge data_patch
		if req.DataPatch != nil {
			patch := req.DataPatch.AsMap()
			p.Data = mergeJSON(p.Data, patch)
		}

		fromID := p.CurrentStateID
		fromName := p.CurrentStateName

		p.CurrentStateID = req.ToStateId
		p.CurrentStateName = req.ToStateName

		if req.NewDeadlineAt != nil {
			d := req.NewDeadlineAt.AsTime().UTC()
			p.DeadlineAt = &d
			p.DeadlineProcessed = false
		}
		if req.SetStatus != "" {
			p.Status = req.SetStatus
		}

		p.UpdatedAt = time.Now().UTC()

		if err := tx.WithContext(ctx).Save(&p).Error; err != nil {
			return err
		}

		changedBy := req.ChangedBy
		h := &StateHistory{
			ProjectID:     p.ID,
			EventName:     req.EventName,
			Status:        "completed",
			ChangedBy:     &changedBy,
			Comment:       req.Comment,
			FromStateID:   &fromID,
			ToStateID:     &p.CurrentStateID,
			FromStateName: fromName,
			ToStateName:   p.CurrentStateName,
			Metadata:      datatypes.JSON([]byte(`{}`)),
			CreatedAt:     time.Now().UTC(),
		}
		if err := tx.WithContext(ctx).Create(h).Error; err != nil {
			return err
		}

		// post_actions: складываем в outbox
		for _, g := range req.PostActions {
			if g == nil || len(g.ActionIds) == 0 {
				continue
			}
			payload := map[string]interface{}{
				"project_id":    p.ID,
				"state_id":      p.CurrentStateID,
				"transition_id": req.TransitionId,
				"trigger":       g.Trigger,
				"action_ids":    g.ActionIds,
				"user_id":       req.ChangedBy,
				"department_id": p.DepartmentID,
			}
			b, _ := json.Marshal(payload)
			ev := &OutboxEvent{
				Topic:     "workflow-actions",
				EventType: "WorkflowPostCommitActions",
				Payload:   datatypes.JSON(b),
				Status:    "pending",
				CreatedAt: time.Now().UTC(),
			}
			if err := tx.WithContext(ctx).Create(ev).Error; err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return &projectv1.CommitTransitionResponse{
		Success:      true,
		ProjectId:    req.ProjectId,
		NewStateId:   req.ToStateId,
		NewStateName: req.ToStateName,
		Status:       req.SetStatus,
	}, nil
}

func mergeJSON(current datatypes.JSON, patch map[string]interface{}) datatypes.JSON {
	cur := map[string]interface{}{}
	if len(current) > 0 {
		_ = json.Unmarshal(current, &cur)
	}
	deepMerge(cur, patch)
	b, _ := json.Marshal(cur)
	return datatypes.JSON(b)
}

func deepMerge(dst, src map[string]interface{}) {
	for k, v := range src {
		if vm, ok := v.(map[string]interface{}); ok {
			if existing, ok := dst[k].(map[string]interface{}); ok {
				deepMerge(existing, vm)
				dst[k] = existing
			} else {
				dst[k] = vm
			}
		} else {
			dst[k] = v
		}
	}
}
