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
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
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

func (s *Service) wfInternalCtx(ctx context.Context) context.Context {
	return metadata.AppendToOutgoingContext(ctx, "x-internal-service", "project_service")
}

func (s *Service) CreateProject(ctx context.Context, req *projectv1.CreateProjectRequest) (*projectv1.CreateProjectResponse, error) {
	// workflow call as internal (otherwise workflow_service may deny/require metadata)
	wf, err := s.workflowClient.GetActiveWorkflowByDepartment(s.wfInternalCtx(ctx), &workflowv1.GetActiveWorkflowByDepartmentRequest{
		DepartmentId: req.DepartmentId,
	})
	if err != nil {
		return nil, status.Errorf(codes.FailedPrecondition, "active workflow is not configured for department %d", req.DepartmentId)
	}
	if len(wf.States) == 0 {
		return nil, status.Error(codes.FailedPrecondition, "active workflow has no states configured")
	}

	// pick initial state
	initial := wf.States[0]
	foundInitial := false
	for _, st := range wf.States {
		if st.IsInitial {
			initial = st
			foundInitial = true
			break
		}
	}
	if !foundInitial {
		s.logger.Warn("workflow has no is_initial state, using first state by order_index",
			zap.Int64("workflow_id", wf.Id))
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
		Status:           initial.Name,
		Data:             datatypes.JSON([]byte(`{}`)),
		DeadlineAt:       deadlineAt,
		CreatedAt:        time.Now().UTC(),
		UpdatedAt:        time.Now().UTC(),
	}

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

	return &projectv1.CreateProjectResponse{ProjectId: p.ID, Status: p.Status}, nil
}

func (s *Service) GetProject(ctx context.Context, id int64) (*Project, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *Service) GetStudentProjects(ctx context.Context, studentID int64) ([]*Project, error) {
	return s.repo.ListByStudent(ctx, studentID)
}

func (s *Service) PerformAction(ctx context.Context, projectID int64, actionName string, payload map[string]interface{}, userID int64, userRole string) (*Project, error) {
	p, err := s.repo.GetRuntimeByID(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("project not found: %w", err)
	}

	if p.Status == "completed" || p.Status == "cancelled" || p.Status == "archived" {
		return nil, errors.New("cannot perform action on inactive project")
	}
	outCtx := metadata.AppendToOutgoingContext(
		ctx,
		"x-internal-service", "project_service",
		"x-user-id", fmt.Sprintf("%d", userID),
		"x-user-role", userRole,
	)

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
	if p.TopicRegisteredAt != nil {
		resp.TopicRegisteredAt = timestamppb.New(*p.TopicRegisteredAt)
	}
	return resp, nil
}

func (s *Service) CommitTransition(ctx context.Context, req *projectv1.CommitTransitionRequest) (*projectv1.CommitTransitionResponse, error) {
	var newStatus string

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
		} else {
			p.Status = req.ToStateName
		}

		p.UpdatedAt = time.Now().UTC()
		if err := tx.WithContext(ctx).Save(&p).Error; err != nil {
			return err
		}

		changedBy := req.ChangedBy

		// Build history metadata: merge history_metadata from request with derived fields
		histMeta := map[string]interface{}{}
		if req.HistoryMetadata != nil {
			histMeta = req.HistoryMetadata.AsMap()
		}
		// Ensure direction/actor_role are always set from explicit fields too
		direction, _ := histMeta["direction"].(string)
		actorRole, _ := histMeta["actor_role"].(string)

		histMetaBytes, _ := json.Marshal(histMeta)

		toStateIDForHistory := p.CurrentStateID
		var transitionIDPtr *int64
		if req.TransitionId > 0 {
			tid := req.TransitionId
			transitionIDPtr = &tid
		}
		h := &StateHistory{
			ProjectID:     p.ID,
			EventName:     req.EventName,
			Status:        "completed",
			ChangedBy:     &changedBy,
			Comment:       req.Comment,
			FromStateID:   &fromID,
			ToStateID:     &toStateIDForHistory,
			FromStateName: fromName,
			ToStateName:   p.CurrentStateName,
			Metadata:      datatypes.JSON(histMetaBytes),
			TransitionID:  transitionIDPtr,
			Direction:     direction,
			ActorRole:     actorRole,
			CreatedAt:     time.Now().UTC(),
		}
		if err := tx.WithContext(ctx).Create(h).Error; err != nil {
			return err
		}

		// outbox post-actions
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

		newStatus = p.Status
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
		Status:       newStatus,
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
func (s *Service) UpdateProject(ctx context.Context, req *projectv1.UpdateProjectRequest) (*projectv1.UpdateProjectResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is nil")
	}
	if req.ProjectId <= 0 {
		return nil, status.Error(codes.InvalidArgument, "project_id is required")
	}

	now := time.Now().UTC()

	err := s.repo.Transaction(ctx, func(tx *gorm.DB) error {
		var p Project
		if err := tx.WithContext(ctx).
			Clauses(clause.Locking{Strength: "UPDATE"}).
			First(&p, "id = ?", req.ProjectId).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return status.Error(codes.NotFound, "project not found")
			}
			return status.Errorf(codes.Internal, "load project failed: %v", err)
		}

		updates := map[string]any{
			"updated_at": now,
		}

		// Семантика без field_mask: пустые строки/нули считаем "не менять"
		if req.Title != "" {
			updates["title"] = req.Title
		}
		if req.Description != "" {
			updates["description"] = req.Description
		}
		if req.StudentId > 0 {
			updates["student_id"] = req.StudentId
		}
		if req.TeamId > 0 {
			updates["team_id"] = req.TeamId
		}

		if req.DataPatch != nil {
			merged := mergeJSON(p.Data, req.DataPatch.AsMap())
			updates["data"] = merged
		}

		// Если кроме updated_at нечего менять — всё равно ок
		if len(updates) > 1 {
			if err := tx.WithContext(ctx).Model(&Project{}).
				Where("id = ?", p.ID).
				Updates(updates).Error; err != nil {
				return status.Errorf(codes.Internal, "update project failed: %v", err)
			}
		} else {
			// просто touch updated_at
			if err := tx.WithContext(ctx).Model(&Project{}).
				Where("id = ?", p.ID).
				Update("updated_at", now).Error; err != nil {
				return status.Errorf(codes.Internal, "touch project failed: %v", err)
			}
		}

		// best-effort: синхронизируем task_boards.name/description с проектом
		boardUpdates := map[string]any{"updated_at": now}
		if req.Title != "" {
			boardUpdates["name"] = req.Title
		}
		if req.Description != "" {
			boardUpdates["description"] = req.Description
		}
		if len(boardUpdates) > 1 {
			_ = tx.WithContext(ctx).Table("task_boards").
				Where("project_id = ?", p.ID).
				Updates(boardUpdates).Error
		}

		return nil
	})

	if err != nil {
		if _, ok := status.FromError(err); ok {
			return nil, err
		}
		return nil, status.Errorf(codes.Internal, "update failed: %v", err)
	}

	return &projectv1.UpdateProjectResponse{Success: true}, nil
}
func (s *Service) ArchiveProject(ctx context.Context, req *projectv1.ArchiveProjectRequest) (*projectv1.ArchiveProjectResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is nil")
	}
	if req.ProjectId <= 0 {
		return nil, status.Error(codes.InvalidArgument, "project_id is required")
	}

	now := time.Now().UTC()

	err := s.repo.Transaction(ctx, func(tx *gorm.DB) error {
		var p Project
		if err := tx.WithContext(ctx).
			Clauses(clause.Locking{Strength: "UPDATE"}).
			First(&p, "id = ?", req.ProjectId).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return status.Error(codes.NotFound, "project not found")
			}
			return status.Errorf(codes.Internal, "load project failed: %v", err)
		}

		// idempotent
		if p.Status == "archived" {
			return nil
		}

		if err := tx.WithContext(ctx).Model(&Project{}).
			Where("id = ?", p.ID).
			Updates(map[string]any{
				"status":     "archived",
				"updated_at": now,
			}).Error; err != nil {
			return status.Errorf(codes.Internal, "archive failed: %v", err)
		}

		return nil
	})

	if err != nil {
		if _, ok := status.FromError(err); ok {
			return nil, err
		}
		return nil, status.Errorf(codes.Internal, "archive failed: %v", err)
	}

	return &projectv1.ArchiveProjectResponse{Success: true, Status: "archived"}, nil
}
func (s *Service) ListProjectsRuntime(ctx context.Context, req *projectv1.ListProjectsRuntimeRequest) (*projectv1.ListProjectsRuntimeResponse, error) {
	f := ProjectFilter{
		DepartmentID: req.DepartmentId,
		UniversityID: req.UniversityId,
		WorkflowID:   req.WorkflowId,
		StateID:      req.StateId,
		TeamID:       req.TeamId,
		StudentID:    req.StudentId,
		Status:       req.Status,
		Search:       req.Search,
		Page:         req.Page,
		PageSize:     req.PageSize,
	}

	projects, total, err := s.repo.ListProjectsRuntime(ctx, f)
	if err != nil {
		return nil, fmt.Errorf("list projects runtime: %w", err)
	}

	resp := &projectv1.ListProjectsRuntimeResponse{
		Total:    int32(total),
		Page:     req.Page,
		PageSize: req.PageSize,
	}
	for _, p := range projects {
		item := &projectv1.ProjectRuntimeSummary{
			ProjectId:        p.ID,
			Title:            p.Title,
			StudentId:        p.StudentID,
			TeamId:           p.TeamID,
			UniversityId:     p.UniversityID,
			DepartmentId:     p.DepartmentID,
			WorkflowId:       p.WorkflowID,
			WorkflowVersion:  p.WorkflowVersion,
			WorkflowName:     p.WorkflowName,
			CurrentStateId:   p.CurrentStateID,
			CurrentStateName: p.CurrentStateName,
			Status:           p.Status,
			CreatedAt:        timestamppb.New(p.CreatedAt),
			UpdatedAt:        timestamppb.New(p.UpdatedAt),
		}
		if p.DeadlineAt != nil {
			item.DeadlineAt = timestamppb.New(*p.DeadlineAt)
		}
		if p.TopicRegisteredAt != nil {
			item.TopicRegisteredAt = timestamppb.New(*p.TopicRegisteredAt)
		}
		resp.Projects = append(resp.Projects, item)
	}
	return resp, nil
}

func (s *Service) ListProjectStateHistory(ctx context.Context, req *projectv1.ListProjectStateHistoryRequest) (*projectv1.ListProjectStateHistoryResponse, error) {
	items, err := s.repo.ListStateHistory(ctx, req.ProjectId, int(req.Limit), req.Order)
	if err != nil {
		return nil, fmt.Errorf("list state history: %w", err)
	}

	resp := &projectv1.ListProjectStateHistoryResponse{}
	for _, h := range items {
		var changedBy int64
		if h.ChangedBy != nil {
			changedBy = *h.ChangedBy
		}
		var fromStateID, toStateID, transitionID int64
		if h.FromStateID != nil {
			fromStateID = *h.FromStateID
		}
		if h.ToStateID != nil {
			toStateID = *h.ToStateID
		}
		if h.TransitionID != nil {
			transitionID = *h.TransitionID
		}

		metaMap := map[string]interface{}{}
		if len(h.Metadata) > 0 {
			_ = json.Unmarshal(h.Metadata, &metaMap)
		}
		metaStruct, _ := structpb.NewStruct(metaMap)

		resp.Items = append(resp.Items, &projectv1.ProjectStateHistoryItem{
			Id:            h.ID,
			ProjectId:     h.ProjectID,
			EventName:     h.EventName,
			Status:        h.Status,
			ChangedBy:     changedBy,
			Comment:       h.Comment,
			FromStateId:   fromStateID,
			ToStateId:     toStateID,
			FromStateName: h.FromStateName,
			ToStateName:   h.ToStateName,
			TransitionId:  transitionID,
			Direction:     h.Direction,
			ActorRole:     h.ActorRole,
			Metadata:      metaStruct,
			CreatedAt:     timestamppb.New(h.CreatedAt),
		})
	}
	return resp, nil
}

func (s *Service) DeleteProject(ctx context.Context, req *projectv1.DeleteProjectRequest) (*projectv1.DeleteProjectResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is nil")
	}
	if req.ProjectId <= 0 {
		return nil, status.Error(codes.InvalidArgument, "project_id is required")
	}

	err := s.repo.Transaction(ctx, func(tx *gorm.DB) error {
		var p Project
		if err := tx.WithContext(ctx).
			Clauses(clause.Locking{Strength: "UPDATE"}).
			First(&p, "id = ?", req.ProjectId).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return status.Error(codes.NotFound, "project not found")
			}
			return status.Errorf(codes.Internal, "load project failed: %v", err)
		}

		if p.Status != "archived" {
			return status.Error(codes.FailedPrecondition, "project must be archived before delete")
		}

		if err := tx.WithContext(ctx).Delete(&Project{}, p.ID).Error; err != nil {
			return status.Errorf(codes.Internal, "delete failed: %v", err)
		}

		return nil
	})

	if err != nil {
		if _, ok := status.FromError(err); ok {
			return nil, err
		}
		return nil, status.Errorf(codes.Internal, "delete failed: %v", err)
	}

	return &projectv1.DeleteProjectResponse{Success: true}, nil
}
