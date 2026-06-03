package admin

import (
	"context"

	"go.uber.org/zap"
)

// ==========================================================================
// Department Progress service (teacher-facing, read-only).
//
// Owns the business normalization: admission decisions, workflow step status,
// and the raw->canonical status mapping. The frontend only displays these.
// ==========================================================================

// normalizeAdmission is the single source of truth for the admission decision.
// IMPORTANT: must stay in sync with admissionCaseSQL in
// repository_department_progress.go (used for list filtering/sorting/tally).
func normalizeAdmission(preDefenseResult string, admGrade *int32, admPassing int32) string {
	switch preDefenseResult {
	case "passed":
		return AdmissionAdmitted
	case "failed":
		return AdmissionNotAdmitted
	case "conditional":
		return AdmissionRevisionRequired
	}
	if admGrade != nil {
		if *admGrade >= admPassing {
			return AdmissionAdmitted
		}
		return AdmissionNotAdmitted
	}
	return AdmissionNotDecided
}

// normalizeStepStatus resolves the display status of a single workflow step.
func normalizeStepStatus(step *DPStep, currentStateID int64, currentOrder int32) string {
	switch step.SubmissionState {
	case "approved":
		return "approved"
	case "rejected":
		return "rejected"
	case "revision_requested", "request_changes":
		return "revision_requested"
	case "submitted":
		return "submitted"
	case "pending":
		return "pending"
	}
	if step.Grade != nil {
		return "completed"
	}
	if step.StepID == currentStateID {
		return "pending"
	}
	if step.OrderIndex < currentOrder {
		return "completed"
	}
	return "not_started"
}

// ResolveUserNames returns id -> "First Last" for the given user ids (batch).
func (s *Service) ResolveUserNames(ctx context.Context, ids []int64) map[int64]string {
	out := map[int64]string{}
	if len(ids) == 0 {
		return out
	}
	users, err := s.repo.GetDPSupervisors(ctx, ids)
	if err != nil {
		return out
	}
	for id, u := range users {
		out[id] = u.FullName
	}
	return out
}

// GetDepartmentProgressSummary returns aggregated department stats + active
// workflow step progress + recent activities (with resolved actor names).
func (s *Service) GetDepartmentProgressSummary(ctx context.Context, departmentID int64) (*DepartmentProgressStatsData, []*StepProgressData, []*AdminActivity, map[int64]*DPUser, error) {
	stats, err := s.repo.GetDepartmentProgressSummaryStats(ctx, departmentID)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	// Active-workflow step progress (best effort — empty if no active workflow).
	workflow, wfErr := s.GetWorkflowProgress(ctx, departmentID, 0)
	if wfErr != nil {
		s.logger.Warn("department progress: workflow progress unavailable", zap.Error(wfErr))
		workflow = []*StepProgressData{}
	}

	activities, actErr := s.repo.GetRecentActivities(ctx, departmentID, 20)
	if actErr != nil {
		s.logger.Warn("department progress: recent activities unavailable", zap.Error(actErr))
		activities = []*AdminActivity{}
	}

	actorIDs := make([]int64, 0, len(activities))
	for _, a := range activities {
		if a.ActorID > 0 {
			actorIDs = append(actorIDs, a.ActorID)
		}
	}
	actors, _ := s.repo.GetDPSupervisors(ctx, actorIDs)

	return stats, workflow, activities, actors, nil
}

// ListDepartmentProgress runs the single paginated list query then batch-loads
// members and supervisors (no N+1).
func (s *Service) ListDepartmentProgress(ctx context.Context, f DepartmentProgressTeamFilter) ([]*DepartmentProgressTeam, int64, error) {
	rows, total, err := s.repo.ListDepartmentProgressTeams(ctx, f)
	if err != nil {
		return nil, 0, err
	}
	if len(rows) == 0 {
		return []*DepartmentProgressTeam{}, total, nil
	}

	teamIDs := make([]int64, 0, len(rows))
	supIDsSet := make(map[int64]struct{}, len(rows))
	supIDs := make([]int64, 0, len(rows))
	for _, r := range rows {
		teamIDs = append(teamIDs, r.TeamID)
		if r.SupervisorID > 0 {
			if _, ok := supIDsSet[r.SupervisorID]; !ok {
				supIDsSet[r.SupervisorID] = struct{}{}
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

	out := make([]*DepartmentProgressTeam, 0, len(rows))
	for _, r := range rows {
		item := &DepartmentProgressTeam{
			Row:     r,
			Members: membersByTeam[r.TeamID],
		}
		if r.SupervisorID > 0 {
			item.Supervisor = supByID[r.SupervisorID]
		}
		out = append(out, item)
	}
	return out, total, nil
}

// GetDepartmentProgressTeamDetails assembles the full read-only team card.
// The department guard lives in GetDPTeamHead (returns NotFound for other depts).
func (s *Service) GetDepartmentProgressTeamDetails(ctx context.Context, teamID, departmentID int64) (*DepartmentProgressTeamDetails, error) {
	head, err := s.repo.GetDPTeamHead(ctx, teamID, departmentID)
	if err != nil {
		return nil, err
	}

	details := &DepartmentProgressTeamDetails{
		Team:             head,
		CurrentStateName: head.CurrentStateName,
	}

	if members, mErr := s.repo.GetDPTeamMembers(ctx, []int64{teamID}); mErr == nil {
		details.Members = members[teamID]
	}
	if head.SupervisorID > 0 {
		if sup, sErr := s.repo.GetDPSupervisors(ctx, []int64{head.SupervisorID}); sErr == nil {
			details.Supervisor = sup[head.SupervisorID]
		}
	}

	// Project block: title_display from projects.title; i18n from topic registration.
	topic, _ := s.repo.GetDPTopicRegistration(ctx, head.ProjectID)
	details.TopicRegistration = topic

	if head.ProjectID > 0 {
		proj := &DPProjectInfo{
			ProjectID:     head.ProjectID,
			TitleDisplay:  head.ProjectTitle,
			Description:   head.ProjectDesc,
			CurrentState:  head.CurrentStateName,
			CurrentStepID: head.CurrentStateID,
		}
		if topic != nil {
			proj.TitleKZ = topic.ProposedTopicKZ
			proj.TitleRU = topic.ProposedTopicRU
			proj.TitleEN = topic.ProposedTopicEN
		}
		details.Project = proj
	}

	// Workflow steps + computed per-step status/admission + overall progress.
	steps, err := s.repo.GetDPWorkflowSteps(ctx, head.ProjectID, head.WorkflowID)
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
		if st.GradeType == "admission" && st.Grade != nil {
			if *st.Grade >= st.PassingScore {
				st.AdmissionStatus = AdmissionAdmitted
			} else {
				st.AdmissionStatus = AdmissionNotAdmitted
			}
		}
	}
	details.Steps = steps
	if n := len(steps); n > 0 {
		pct := (float32(currentOrder) + 1) / float32(n) * 100
		if pct > 100 {
			pct = 100
		}
		details.Progress = pct
	}

	if nc, ncErr := s.repo.GetDPNormControl(ctx, head.ProjectID); ncErr == nil {
		details.NormControl = nc
	}
	if ap, apErr := s.repo.GetDPAntiplagiat(ctx, head.ProjectID); apErr == nil {
		details.Antiplagiat = ap
	}

	preDefenses, pdErr := s.repo.GetDPPreDefenses(ctx, head.ProjectID)
	if pdErr == nil {
		for _, pd := range preDefenses {
			pd.AdmissionStatus = normalizeAdmission(pd.Result, nil, 0)
		}
		details.PreDefenses = preDefenses
	}

	if grades, gErr := s.repo.GetGradesByProject(ctx, head.ProjectID); gErr == nil {
		details.Grades = grades
	}

	if history, hErr := s.repo.GetDPUnifiedHistory(ctx, head.ProjectID, teamID); hErr == nil {
		details.History = history
	} else {
		s.logger.Warn("department progress: unified history unavailable", zap.Error(hErr))
		details.History = []*UnifiedHistoryItem{}
	}

	return details, nil
}
