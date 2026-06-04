package admin

import (
	"context"

	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ==========================================================================
// Supervisor Projects service (supervisor-facing).
//
// A supervisor sees ONLY their own projects. Ownership = projects.team_id ->
// admin_supervisor_assignments.supervisor_id (projects has no supervisor_id
// column). Every method is guarded by requireSupervisorProjectAccess. Business
// logic for grades/reviews/history is reused from the existing admin service.
// ==========================================================================

// requireSupervisorProjectAccess enforces supervisor ownership of a project.
// Admin callers bypass ownership (but still get a 404 for a missing project).
// Returns gRPC NotFound (missing) or PermissionDenied (exists but not owned).
func (s *Service) requireSupervisorProjectAccess(ctx context.Context, supervisorID, projectID int64, callerRole string) error {
	exists, owned, err := s.repo.IsSupervisorOfProject(ctx, supervisorID, projectID)
	if err != nil {
		return status.Errorf(codes.Internal, "access check failed: %v", err)
	}
	if !exists {
		return status.Error(codes.NotFound, "project not found")
	}
	if callerRole == "admin" {
		return nil
	}
	if !owned {
		return status.Error(codes.PermissionDenied, "you are not the supervisor of this project")
	}
	return nil
}

// SupervisorProjectListEntry is an assembled list item (row + batched relations).
type SupervisorProjectListEntry struct {
	Row        *SupervisorProjectRow
	Members    []*DPMember
	Supervisor *DPUser
}

// ListSupervisorProjects runs the paginated owner-scoped query and batch-loads
// members + supervisors (no N+1).
func (s *Service) ListSupervisorProjects(ctx context.Context, f SupervisorProjectsFilter) ([]*SupervisorProjectListEntry, int64, error) {
	rows, total, err := s.repo.ListSupervisorProjects(ctx, f)
	if err != nil {
		return nil, 0, err
	}
	if len(rows) == 0 {
		return []*SupervisorProjectListEntry{}, total, nil
	}

	teamIDs := make([]int64, 0, len(rows))
	supSet := make(map[int64]struct{}, len(rows))
	supIDs := make([]int64, 0, len(rows))
	for _, r := range rows {
		if r.TeamID > 0 {
			teamIDs = append(teamIDs, r.TeamID)
		}
		if r.SupervisorID > 0 {
			if _, ok := supSet[r.SupervisorID]; !ok {
				supSet[r.SupervisorID] = struct{}{}
				supIDs = append(supIDs, r.SupervisorID)
			}
		}
	}

	membersByTeam, err := s.repo.GetDPTeamMembers(ctx, teamIDs)
	if err != nil {
		return nil, 0, err
	}
	supByID, err := s.repo.GetDPSupervisors(ctx, supIDs)
	if err != nil {
		return nil, 0, err
	}

	out := make([]*SupervisorProjectListEntry, 0, len(rows))
	for _, r := range rows {
		entry := &SupervisorProjectListEntry{
			Row:     r,
			Members: membersByTeam[r.TeamID],
		}
		if r.SupervisorID > 0 {
			entry.Supervisor = supByID[r.SupervisorID]
		}
		out = append(out, entry)
	}
	return out, total, nil
}

// SupervisorProjectDetailsData is the assembled detail view for one project.
type SupervisorProjectDetailsData struct {
	Head        *SupervisorProjectHead
	Members     []*DPMember
	Supervisor  *DPUser
	Topic       *DPTopicRegistration
	Steps       []*DPStep
	Progress    int32
	Submissions []*Submission
	Grades      []*Grade
	History     []*UnifiedHistoryItem
}

// GetSupervisorProjectDetails assembles the full supervisor project card.
// Caller must have already passed requireSupervisorProjectAccess.
func (s *Service) GetSupervisorProjectDetails(ctx context.Context, projectID int64) (*SupervisorProjectDetailsData, error) {
	head, err := s.repo.GetSupervisorProjectHead(ctx, projectID)
	if err != nil {
		return nil, err
	}
	if head == nil || head.ProjectID == 0 {
		return nil, status.Error(codes.NotFound, "project not found")
	}

	d := &SupervisorProjectDetailsData{Head: head}

	if head.TeamID > 0 {
		if members, mErr := s.repo.GetDPTeamMembers(ctx, []int64{head.TeamID}); mErr == nil {
			d.Members = members[head.TeamID]
		}
	}
	if head.SupervisorID > 0 {
		if sup, sErr := s.repo.GetDPSupervisors(ctx, []int64{head.SupervisorID}); sErr == nil {
			d.Supervisor = sup[head.SupervisorID]
		}
	}

	d.Topic, _ = s.repo.GetDPTopicRegistration(ctx, projectID)

	steps, err := s.repo.GetDPWorkflowSteps(ctx, projectID, head.WorkflowID)
	if err != nil {
		return nil, err
	}
	var currentOrder int32
	for _, st := range steps {
		if st.StepID == head.CurrentStateID {
			currentOrder = st.OrderIndex
		}
	}
	for _, st := range steps {
		st.Status = normalizeStepStatus(st, head.CurrentStateID, currentOrder)
	}
	d.Steps = steps
	d.Progress = computeProgressPercent(currentOrder, int32(len(steps)))

	if subs, _, sErr := s.repo.ListSubmissions(ctx, SubmissionFilter{ProjectID: projectID, Limit: 200}); sErr == nil {
		d.Submissions = subs
	}
	if grades, gErr := s.repo.GetGradesByProject(ctx, projectID); gErr == nil {
		d.Grades = grades
	}
	if history, hErr := s.repo.GetDPUnifiedHistory(ctx, projectID, head.TeamID); hErr == nil {
		d.History = history
	} else {
		s.logger.Warn("supervisor projects: unified history unavailable", zap.Error(hErr))
		d.History = []*UnifiedHistoryItem{}
	}

	return d, nil
}

// computeProgressPercent mirrors the department-progress percentage logic:
// (currentOrder+1)/totalSteps, clamped to [0,100].
func computeProgressPercent(currentOrder, totalSteps int32) int32 {
	if totalSteps <= 0 {
		return 0
	}
	pct := (float32(currentOrder) + 1) / float32(totalSteps) * 100
	if pct > 100 {
		pct = 100
	}
	if pct < 0 {
		pct = 0
	}
	return int32(pct)
}
