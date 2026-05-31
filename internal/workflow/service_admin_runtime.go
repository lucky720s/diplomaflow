package workflow

import (
	"context"
	"errors"
	"fmt"
	"time"

	projectv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/project/v1"
	workflowv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/workflow/v1"
	"go.uber.org/zap"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var (
	ErrProjectNotFound       = errors.New("project not found")
	ErrInvalidTransition     = errors.New("invalid transition")
	ErrRollbackNotAllowed    = errors.New("rollback not allowed for current state")
	ErrForbidden             = errors.New("forbidden")
	ErrActiveWFNotConfigured = errors.New("active workflow is not configured for department")
)

// AdminRuntimeService handles admin workflow management over project runtime.
type AdminRuntimeService struct {
	wfService     *Service
	projectClient ProjectClient
	logger        *zap.Logger
}

func NewAdminRuntimeService(wfSvc *Service, projectClient ProjectClient, logger *zap.Logger) *AdminRuntimeService {
	return &AdminRuntimeService{
		wfService:     wfSvc,
		projectClient: projectClient,
		logger:        logger,
	}
}

// projectInternalCtx tags outgoing gRPC calls as coming from workflow_service.
func projectInternalCtx(ctx context.Context, userID int64, userRole string) context.Context {
	out := metadata.AppendToOutgoingContext(ctx, "x-internal-service", "workflow_service")
	if userID > 0 {
		out = metadata.AppendToOutgoingContext(out, "x-user-id", fmt.Sprintf("%d", userID))
	}
	if userRole != "" {
		out = metadata.AppendToOutgoingContext(out, "x-user-role", userRole)
	}
	return out
}

// GetProjectWorkflowState returns full project workflow state for admin panel.
func (s *AdminRuntimeService) GetProjectWorkflowState(ctx context.Context, projectID int64) (*workflowv1.ProjectWorkflowStateResponse, error) {
	snap, err := s.projectClient.GetProjectRuntime(projectInternalCtx(ctx, 0, ""), &projectv1.GetProjectRuntimeRequest{
		ProjectId: projectID,
	})
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrProjectNotFound, err)
	}

	wf, err := s.wfService.GetWorkflowFull(ctx, snap.WorkflowId)
	if err != nil {
		return nil, fmt.Errorf("get workflow: %w", err)
	}

	// History
	histResp, err := s.projectClient.ListProjectStateHistory(projectInternalCtx(ctx, 0, ""), &projectv1.ListProjectStateHistoryRequest{
		ProjectId: projectID,
		Limit:     50,
		Order:     "desc",
	})
	if err != nil {
		s.logger.Warn("failed to load project history", zap.Int64("project_id", projectID), zap.Error(err))
	}

	return s.buildProjectWorkflowStateResponse(snap, wf, histResp), nil
}

// AdvanceProjectWorkflow moves a project forward along the workflow.
func (s *AdminRuntimeService) AdvanceProjectWorkflow(ctx context.Context, req *workflowv1.AdvanceProjectWorkflowRequest, actorID int64, actorRole string) (*workflowv1.ProjectWorkflowStateResponse, error) {
	snap, err := s.projectClient.GetProjectRuntime(projectInternalCtx(ctx, actorID, actorRole), &projectv1.GetProjectRuntimeRequest{
		ProjectId: req.ProjectId,
	})
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrProjectNotFound, err)
	}

	if snap.Status == "archived" || snap.Status == "cancelled" {
		return nil, fmt.Errorf("%w: cannot advance %s project", ErrInvalidTransition, snap.Status)
	}

	var tr *Transition
	if req.TransitionId > 0 {
		var t *Transition
		t, err = s.wfService.GetTransition(ctx, req.TransitionId)
		if err != nil {
			return nil, fmt.Errorf("%w: transition %d not found", ErrInvalidTransition, req.TransitionId)
		}
		if t.WorkflowID != snap.WorkflowId {
			return nil, fmt.Errorf("%w: transition belongs to different workflow", ErrInvalidTransition)
		}
		if t.FromStateID != snap.CurrentStateId {
			return nil, fmt.Errorf("%w: transition from_state_id mismatch (expected %d got %d)", ErrInvalidTransition, snap.CurrentStateId, t.FromStateID)
		}
		tr = t
	} else if req.ToStateId > 0 {
		// find transition current -> to_state
		var transitions []Transition
		transitions, err = s.wfService.repo.GetTransitionsFromState(ctx, snap.CurrentStateId)
		if err != nil {
			return nil, fmt.Errorf("get transitions: %w", err)
		}
		for i := range transitions {
			if transitions[i].ToStateID == req.ToStateId {
				tr = &transitions[i]
				break
			}
		}
	} else {
		// Use first available forward transition
		var transitions []Transition
		transitions, err = s.wfService.repo.GetTransitionsFromState(ctx, snap.CurrentStateId)
		if err != nil {
			return nil, fmt.Errorf("get transitions: %w", err)
		}
		if len(transitions) > 0 {
			tr = &transitions[0]
		}
	}

	if tr == nil {
		// Admin manual: need to know target state
		if req.ToStateId <= 0 {
			return nil, fmt.Errorf("%w: no transition found and no to_state_id provided", ErrInvalidTransition)
		}
		// Admin force-advance without transition
		return s.adminSetState(ctx, snap, req.ToStateId, actorID, actorRole, "forward", req.Comment)
	}

	toState, err := s.wfService.GetState(ctx, tr.ToStateID)
	if err != nil {
		return nil, fmt.Errorf("get target state: %w", err)
	}

	newDeadline := computeDeadline(toState)
	setStatus := ""
	if toState.IsFinal {
		setStatus = "completed"
	}

	histMeta, _ := structpb.NewStruct(map[string]interface{}{
		"direction":     "forward",
		"actor_role":    actorRole,
		"transition_id": tr.ID,
		"manual":        false,
	})

	eventName := tr.EventName
	if eventName == "" {
		eventName = "__admin_advance__"
	}

	commitCtx := projectInternalCtx(ctx, actorID, actorRole)
	commitReq := &projectv1.CommitTransitionRequest{
		ProjectId:           req.ProjectId,
		ExpectedFromStateId: snap.CurrentStateId,
		TransitionId:        tr.ID,
		EventName:           eventName,
		ToStateId:           toState.ID,
		ToStateName:         toState.Name,
		ChangedBy:           actorID,
		Comment:             req.Comment,
		SetStatus:           setStatus,
		HistoryMetadata:     histMeta,
	}
	if newDeadline != nil {
		commitReq.NewDeadlineAt = timestamppb.New(*newDeadline)
	}

	if _, err := s.projectClient.CommitTransition(commitCtx, commitReq); err != nil {
		return nil, fmt.Errorf("commit transition: %w", err)
	}

	return s.GetProjectWorkflowState(ctx, req.ProjectId)
}

// RollbackProjectWorkflow rolls back a project to a previous state.
func (s *AdminRuntimeService) RollbackProjectWorkflow(ctx context.Context, req *workflowv1.RollbackProjectWorkflowRequest, actorID int64, actorRole string) (*workflowv1.ProjectWorkflowStateResponse, error) {
	snap, err := s.projectClient.GetProjectRuntime(projectInternalCtx(ctx, actorID, actorRole), &projectv1.GetProjectRuntimeRequest{
		ProjectId: req.ProjectId,
	})
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrProjectNotFound, err)
	}

	var targetStateID int64
	var eventName string
	var transitionID int64

	if req.TransitionId > 0 {
		var tr *Transition
		tr, err = s.wfService.GetTransition(ctx, req.TransitionId)
		if err != nil {
			return nil, fmt.Errorf("%w: rollback transition %d not found", ErrInvalidTransition, req.TransitionId)
		}
		if tr.FromStateID != snap.CurrentStateId {
			return nil, fmt.Errorf("%w: rollback transition from_state mismatch", ErrInvalidTransition)
		}
		targetStateID = tr.ToStateID
		eventName = tr.EventName
		transitionID = tr.ID
	} else if req.ToStateId > 0 {
		targetStateID = req.ToStateId
		eventName = "__admin_rollback__"
	} else {
		// Use previous state from history
		var histResp *projectv1.ListProjectStateHistoryResponse
		histResp, err = s.projectClient.ListProjectStateHistory(projectInternalCtx(ctx, 0, ""), &projectv1.ListProjectStateHistoryRequest{
			ProjectId: req.ProjectId,
			Limit:     5,
			Order:     "desc",
		})
		if err != nil || len(histResp.Items) == 0 {
			return nil, fmt.Errorf("%w: no history found for rollback", ErrRollbackNotAllowed)
		}
		// Find last entry where to_state == current (ie the entry that put us here)
		for _, h := range histResp.Items {
			if h.ToStateId == snap.CurrentStateId && h.FromStateId > 0 {
				targetStateID = h.FromStateId
				break
			}
		}
		if targetStateID == 0 {
			return nil, fmt.Errorf("%w: cannot determine previous state", ErrRollbackNotAllowed)
		}
		eventName = "__admin_rollback__"
	}

	toState, err := s.wfService.GetState(ctx, targetStateID)
	if err != nil {
		return nil, fmt.Errorf("get rollback target state: %w", err)
	}

	// Verify target state belongs to same workflow
	if toState.WorkflowID != snap.WorkflowId {
		return nil, fmt.Errorf("%w: target state belongs to different workflow", ErrInvalidTransition)
	}

	setStatus := toState.Name
	if snap.Status == "completed" {
		setStatus = "active"
	}

	histMeta, _ := structpb.NewStruct(map[string]interface{}{
		"direction":     "rollback",
		"actor_role":    actorRole,
		"transition_id": transitionID,
		"manual":        transitionID == 0,
	})

	commitCtx := projectInternalCtx(ctx, actorID, actorRole)
	commitReq := &projectv1.CommitTransitionRequest{
		ProjectId:           req.ProjectId,
		ExpectedFromStateId: snap.CurrentStateId,
		TransitionId:        transitionID,
		EventName:           eventName,
		ToStateId:           toState.ID,
		ToStateName:         toState.Name,
		ChangedBy:           actorID,
		Comment:             req.Comment,
		SetStatus:           setStatus,
		HistoryMetadata:     histMeta,
	}

	if _, err := s.projectClient.CommitTransition(commitCtx, commitReq); err != nil {
		return nil, fmt.Errorf("commit rollback: %w", err)
	}

	return s.GetProjectWorkflowState(ctx, req.ProjectId)
}

// SetProjectWorkflowState manually forces a project to a given state (admin only).
func (s *AdminRuntimeService) SetProjectWorkflowState(ctx context.Context, req *workflowv1.SetProjectWorkflowStateRequest, actorID int64, actorRole string) (*workflowv1.ProjectWorkflowStateResponse, error) {
	snap, err := s.projectClient.GetProjectRuntime(projectInternalCtx(ctx, actorID, actorRole), &projectv1.GetProjectRuntimeRequest{
		ProjectId: req.ProjectId,
	})
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrProjectNotFound, err)
	}

	return s.adminSetState(ctx, snap, req.ToStateId, actorID, actorRole, "set", req.Comment)
}

func (s *AdminRuntimeService) adminSetState(ctx context.Context, snap *projectv1.GetProjectRuntimeResponse, toStateID int64, actorID int64, actorRole string, direction string, comment string) (*workflowv1.ProjectWorkflowStateResponse, error) {
	toState, err := s.wfService.GetState(ctx, toStateID)
	if err != nil {
		return nil, fmt.Errorf("get target state: %w", err)
	}
	if toState.WorkflowID != snap.WorkflowId {
		return nil, fmt.Errorf("%w: target state belongs to different workflow", ErrInvalidTransition)
	}

	setStatus := toState.Name
	if toState.IsFinal {
		setStatus = "completed"
	}
	if snap.Status == "completed" && !toState.IsFinal {
		setStatus = "active"
	}

	histMeta, _ := structpb.NewStruct(map[string]interface{}{
		"direction":  direction,
		"actor_role": actorRole,
		"manual":     true,
	})

	newDeadline := computeDeadline(toState)
	commitReq := &projectv1.CommitTransitionRequest{
		ProjectId:           snap.ProjectId,
		ExpectedFromStateId: snap.CurrentStateId,
		TransitionId:        0,
		EventName:           "__admin_set_state__",
		ToStateId:           toState.ID,
		ToStateName:         toState.Name,
		ChangedBy:           actorID,
		Comment:             comment,
		SetStatus:           setStatus,
		HistoryMetadata:     histMeta,
	}
	if newDeadline != nil {
		commitReq.NewDeadlineAt = timestamppb.New(*newDeadline)
	}

	if _, err := s.projectClient.CommitTransition(projectInternalCtx(ctx, actorID, actorRole), commitReq); err != nil {
		return nil, fmt.Errorf("commit set state: %w", err)
	}

	return s.GetProjectWorkflowState(ctx, snap.ProjectId)
}

// ListWorkflowProjects lists projects for admin monitoring.
func (s *AdminRuntimeService) ListWorkflowProjects(ctx context.Context, req *workflowv1.ListWorkflowProjectsRequest) (*workflowv1.ListWorkflowProjectsResponse, error) {
	listResp, err := s.projectClient.ListProjectsRuntime(projectInternalCtx(ctx, 0, ""), &projectv1.ListProjectsRuntimeRequest{
		DepartmentId: req.DepartmentId,
		UniversityId: req.UniversityId,
		WorkflowId:   req.WorkflowId,
		StateId:      req.StateId,
		TeamId:       req.TeamId,
		StudentId:    req.StudentId,
		Status:       req.Status,
		Search:       req.Search,
		Page:         req.Page,
		PageSize:     req.PageSize,
	})
	if err != nil {
		return nil, fmt.Errorf("list projects: %w", err)
	}

	resp := &workflowv1.ListWorkflowProjectsResponse{
		Total:    listResp.Total,
		Page:     listResp.Page,
		PageSize: listResp.PageSize,
	}

	// Load last transition time from history for each project (best-effort)
	for _, p := range listResp.Projects {
		item := &workflowv1.WorkflowProjectItem{
			ProjectId:         p.ProjectId,
			Title:             p.Title,
			StudentId:         p.StudentId,
			TeamId:            p.TeamId,
			UniversityId:      p.UniversityId,
			DepartmentId:      p.DepartmentId,
			WorkflowId:        p.WorkflowId,
			WorkflowName:      p.WorkflowName,
			WorkflowVersion:   p.WorkflowVersion,
			CurrentStateId:    p.CurrentStateId,
			CurrentStateName:  p.CurrentStateName,
			Status:            p.Status,
			DeadlineAt:        p.DeadlineAt,
			TopicRegisteredAt: p.TopicRegisteredAt,
			CreatedAt:         p.CreatedAt,
			UpdatedAt:         p.UpdatedAt,
		}
		resp.Projects = append(resp.Projects, item)
	}

	return resp, nil
}

// GetWorkflowDashboardStats returns dashboard statistics for admin.
func (s *AdminRuntimeService) GetWorkflowDashboardStats(ctx context.Context, req *workflowv1.GetWorkflowDashboardStatsRequest) (*workflowv1.GetWorkflowDashboardStatsResponse, error) {
	// Determine workflow
	var wfID int64
	var wfName string
	var wfActive bool

	if req.WorkflowId > 0 {
		wf, err := s.wfService.GetWorkflow(ctx, req.WorkflowId)
		if err != nil {
			return nil, fmt.Errorf("get workflow: %w", err)
		}
		wfID = wf.ID
		wfName = wf.Name
		wfActive = wf.IsActive
	} else if req.DepartmentId > 0 {
		wf, err := s.wfService.GetActiveWorkflowByDepartment(ctx, req.DepartmentId)
		if err != nil {
			return nil, fmt.Errorf("%w", ErrActiveWFNotConfigured)
		}
		wfID = wf.ID
		wfName = wf.Name
		wfActive = true
	}

	// Get all projects for the department/workflow
	allResp, err := s.projectClient.ListProjectsRuntime(projectInternalCtx(ctx, 0, ""), &projectv1.ListProjectsRuntimeRequest{
		DepartmentId: req.DepartmentId,
		WorkflowId:   wfID,
		PageSize:     1000,
		Page:         1,
	})
	if err != nil {
		return nil, fmt.Errorf("list projects for stats: %w", err)
	}

	resp := &workflowv1.GetWorkflowDashboardStatsResponse{
		WorkflowId:     wfID,
		WorkflowName:   wfName,
		WorkflowActive: wfActive,
		TotalProjects:  allResp.Total,
	}

	stateCounts := map[string]*workflowv1.StateProjectCount{}
	now := time.Now().UTC()

	for _, p := range allResp.Projects {
		switch p.Status {
		case "completed":
			resp.CompletedProjects++
		case "archived", "cancelled":
			resp.ArchivedProjects++
		default:
			resp.ActiveProjects++
		}
		if p.DeadlineAt != nil && p.DeadlineAt.AsTime().Before(now) && p.Status != "completed" && p.Status != "archived" && p.Status != "cancelled" {
			resp.OverdueProjects++
		}

		key := fmt.Sprintf("%d", p.CurrentStateId)
		if _, ok := stateCounts[key]; !ok {
			stateCounts[key] = &workflowv1.StateProjectCount{
				StateId:   p.CurrentStateId,
				StateName: p.CurrentStateName,
			}
		}
		stateCounts[key].Count++
	}

	for _, sc := range stateCounts {
		resp.ProjectsByState = append(resp.ProjectsByState, sc)
	}

	// Recent transitions: load from last 5 projects histories (best-effort)
	var recentTransitions []*workflowv1.WorkflowHistoryItem
	for i, p := range allResp.Projects {
		if i >= 5 {
			break
		}
		histResp, err := s.projectClient.ListProjectStateHistory(projectInternalCtx(ctx, 0, ""), &projectv1.ListProjectStateHistoryRequest{
			ProjectId: p.ProjectId,
			Limit:     3,
			Order:     "desc",
		})
		if err != nil {
			continue
		}
		for _, h := range histResp.Items {
			recentTransitions = append(recentTransitions, projectHistoryItemToWorkflowItem(h))
		}
	}
	resp.RecentTransitions = recentTransitions

	return resp, nil
}

// ListProjectWorkflowHistory returns transition history for a project.
func (s *AdminRuntimeService) ListProjectWorkflowHistory(ctx context.Context, req *workflowv1.ListProjectWorkflowHistoryRequest) (*workflowv1.ListProjectWorkflowHistoryResponse, error) {
	histResp, err := s.projectClient.ListProjectStateHistory(projectInternalCtx(ctx, 0, ""), &projectv1.ListProjectStateHistoryRequest{
		ProjectId: req.ProjectId,
		Limit:     req.Limit,
		Order:     req.Order,
	})
	if err != nil {
		return nil, fmt.Errorf("list history: %w", err)
	}

	resp := &workflowv1.ListProjectWorkflowHistoryResponse{}
	for _, h := range histResp.Items {
		resp.Items = append(resp.Items, projectHistoryItemToWorkflowItem(h))
	}
	return resp, nil
}

// ==================== helpers ====================

func computeDeadline(state *State) *time.Time {
	if state.FixedDeadline != nil {
		d := *state.FixedDeadline
		return &d
	}
	if state.DurationDays > 0 {
		d := time.Now().UTC().AddDate(0, 0, int(state.DurationDays))
		return &d
	}
	return nil
}

func projectHistoryItemToWorkflowItem(h *projectv1.ProjectStateHistoryItem) *workflowv1.WorkflowHistoryItem {
	item := &workflowv1.WorkflowHistoryItem{
		Id:            h.Id,
		ProjectId:     h.ProjectId,
		EventName:     h.EventName,
		Status:        h.Status,
		ChangedBy:     h.ChangedBy,
		Comment:       h.Comment,
		FromStateId:   h.FromStateId,
		ToStateId:     h.ToStateId,
		FromStateName: h.FromStateName,
		ToStateName:   h.ToStateName,
		TransitionId:  h.TransitionId,
		Direction:     h.Direction,
		ActorRole:     h.ActorRole,
		Metadata:      h.Metadata,
		CreatedAt:     h.CreatedAt,
	}
	return item
}

func (s *AdminRuntimeService) buildProjectWorkflowStateResponse(
	snap *projectv1.GetProjectRuntimeResponse,
	wf *Workflow,
	histResp *projectv1.ListProjectStateHistoryResponse,
) *workflowv1.ProjectWorkflowStateResponse {
	resp := &workflowv1.ProjectWorkflowStateResponse{
		ProjectId:         snap.ProjectId,
		StudentId:         snap.StudentId,
		TeamId:            snap.TeamId,
		UniversityId:      snap.UniversityId,
		DepartmentId:      snap.DepartmentId,
		WorkflowId:        snap.WorkflowId,
		WorkflowVersion:   snap.WorkflowVersion,
		WorkflowName:      snap.WorkflowName,
		CurrentStateId:    snap.CurrentStateId,
		CurrentStateName:  snap.CurrentStateName,
		Status:            snap.Status,
		DeadlineAt:        snap.DeadlineAt,
		TopicRegisteredAt: snap.TopicRegisteredAt,
		Data:              snap.Data,
	}

	// Build state/transition protos from wf
	h := &Handler{} // use converter helpers only
	for _, st := range wf.States {
		pbState := h.stateToProto(&st)
		resp.States = append(resp.States, pbState)
		if st.ID == snap.CurrentStateId {
			resp.CurrentState = pbState
		}
	}
	for _, tr := range wf.Transitions {
		resp.Transitions = append(resp.Transitions, h.transitionToProto(&tr))
	}

	// Available forward transitions
	for _, tr := range wf.Transitions {
		if tr.FromStateID == snap.CurrentStateId {
			resp.AvailableForwardTransitions = append(resp.AvailableForwardTransitions, &workflowv1.AvailableTransition{
				Transition: h.transitionToProto(&tr),
				CanExecute: true,
			})
		}
	}

	// Available rollback transitions: look at history for previous state
	var previousStateID int64
	if histResp != nil {
		for _, item := range histResp.Items {
			if item.ToStateId == snap.CurrentStateId && item.FromStateId > 0 {
				previousStateID = item.FromStateId
				break
			}
		}
	}
	if previousStateID > 0 {
		for i := range wf.States {
			if wf.States[i].ID == previousStateID {
				resp.PreviousState = h.stateToProto(&wf.States[i])
				// Synthetic rollback transition
				resp.AvailableRollbackTransitions = append(resp.AvailableRollbackTransitions, &workflowv1.AvailableTransition{
					Transition: &workflowv1.Transition{
						WorkflowId:  wf.ID,
						EventName:   "__admin_rollback__",
						DisplayName: "Откатить",
						FromStateId: snap.CurrentStateId,
						ToStateId:   previousStateID,
					},
					CanExecute: true,
				})
				break
			}
		}
	}

	// History items
	if histResp != nil {
		for _, item := range histResp.Items {
			resp.History = append(resp.History, projectHistoryItemToWorkflowItem(item))
		}
	}

	// Project title from data if available
	if snap.Data != nil {
		if title, ok := snap.Data.Fields["title"]; ok {
			resp.Title = title.GetStringValue()
		}
	}

	return resp
}
