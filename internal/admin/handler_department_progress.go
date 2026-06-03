package admin

import (
	"context"

	adminv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/admin/v1"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"
)

const dpMaxPageSize = 100

// ==================== Summary ====================

func (h *Handler) GetDepartmentProgressSummary(ctx context.Context, req *adminv1.DepartmentProgressSummaryRequest) (*adminv1.DepartmentProgressSummaryResponse, error) {
	if req.DepartmentId <= 0 {
		return nil, status.Error(codes.InvalidArgument, "department_id is required")
	}

	stats, workflow, activities, actors, err := h.service.GetDepartmentProgressSummary(ctx, req.DepartmentId)
	if err != nil {
		h.logger.Error("GetDepartmentProgressSummary failed", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "failed to get department progress summary: %v", err)
	}

	resp := &adminv1.DepartmentProgressSummaryResponse{
		Stats: toPbDPStats(stats),
	}
	for _, sp := range workflow {
		var pct float32
		if sp.TotalTeams > 0 {
			pct = float32(sp.CompletedTeams) / float32(sp.TotalTeams) * 100
		}
		resp.Workflow = append(resp.Workflow, &adminv1.StepProgress{
			StepId:               sp.StepID,
			StepName:             sp.StepName,
			StepType:             sp.StepType,
			TotalTeams:           sp.TotalTeams,
			CompletedTeams:       sp.CompletedTeams,
			PendingTeams:         sp.PendingTeams,
			RejectedTeams:        sp.RejectedTeams,
			CompletionPercentage: pct,
		})
	}
	for _, a := range activities {
		name := ""
		if u, ok := actors[a.ActorID]; ok && u != nil {
			name = u.FullName
		}
		resp.RecentActivities = append(resp.RecentActivities, &adminv1.RecentActivity{
			Id:           a.ID,
			ActivityType: a.ActivityType,
			Description:  a.Description,
			ActorId:      a.ActorID,
			ActorName:    name,
			TargetId:     a.TargetID,
			TargetType:   a.TargetType,
			CreatedAt:    timestamppb.New(a.CreatedAt),
		})
	}
	return resp, nil
}

func toPbDPStats(s *DepartmentProgressStatsData) *adminv1.DepartmentProgressStats {
	if s == nil {
		return &adminv1.DepartmentProgressStats{}
	}
	return &adminv1.DepartmentProgressStats{
		TotalStudents:             s.TotalStudents,
		TotalTeams:                s.TotalTeams,
		TotalProjects:             s.TotalProjects,
		CompletedProjects:         s.CompletedProjects,
		PendingTopicRegistrations: s.PendingTopicRegistrations,
		PendingSupervisorRequests: s.PendingSupervisorRequests,
		PendingNormControl:        s.PendingNormControl,
		PendingAntiplagiat:        s.PendingAntiplagiat,
		PendingPreDefenses:        s.PendingPreDefenses,
		ScheduledPreDefenses:      s.ScheduledPreDefenses,
		AdmittedCount:             s.AdmittedCount,
		NotAdmittedCount:          s.NotAdmittedCount,
		RevisionRequiredCount:     s.RevisionRequiredCount,
		AverageGrade:              s.AverageGrade,
		HasAverageGrade:           s.HasAverageGrade,
	}
}

// ==================== Team list ====================

func (h *Handler) ListDepartmentProgressTeams(ctx context.Context, req *adminv1.ListDepartmentProgressTeamsRequest) (*adminv1.ListDepartmentProgressTeamsResponse, error) {
	if req.DepartmentId <= 0 {
		return nil, status.Error(codes.InvalidArgument, "department_id is required")
	}

	page := req.Page
	if page <= 0 {
		page = 1
	}
	pageSize := req.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > dpMaxPageSize {
		pageSize = dpMaxPageSize
	}

	f := DepartmentProgressTeamFilter{
		DepartmentID:      req.DepartmentId,
		Search:            req.Search,
		SupervisorID:      req.SupervisorId,
		CurrentStage:      req.CurrentStage,
		TopicStatus:       req.TopicStatus,
		NormControlStatus: req.NormControlStatus,
		AntiplagiatStatus: req.AntiplagiatStatus,
		PreDefenseStatus:  req.PreDefenseStatus,
		AdmissionStatus:   req.AdmissionStatus,
		Page:              page,
		PageSize:          pageSize,
		SortBy:            req.SortBy,
		SortOrder:         req.SortOrder,
	}
	if req.HasMinGrade {
		v := req.MinGrade
		f.MinGrade = &v
	}
	if req.HasMaxGrade {
		v := req.MaxGrade
		f.MaxGrade = &v
	}

	teams, total, err := h.service.ListDepartmentProgress(ctx, f)
	if err != nil {
		h.logger.Error("ListDepartmentProgressTeams failed", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "failed to list department progress teams: %v", err)
	}

	resp := &adminv1.ListDepartmentProgressTeamsResponse{
		TotalCount: total,
		Page:       page,
		PageSize:   pageSize,
	}
	for _, t := range teams {
		resp.Teams = append(resp.Teams, toPbDPTeam(t))
	}
	return resp, nil
}

func toPbDPTeam(t *DepartmentProgressTeam) *adminv1.DepartmentProgressTeam {
	r := t.Row
	pb := &adminv1.DepartmentProgressTeam{
		TeamId:              r.TeamID,
		TeamName:            r.TeamName,
		ProjectId:           r.ProjectID,
		ProjectTitle:        r.ProjectTitle,
		Supervisor:          toPbDPSupervisor(t.Supervisor),
		Members:             toPbDPMembers(t.Members),
		CurrentStage:        r.CurrentStateName,
		CurrentStageDisplay: r.CurrentStateName,
		ProgressPercentage:  r.ProgressPercentage,
		TopicStatus:         r.TopicStatus,
		NormControlStatus:   r.NormControlStatus,
		AntiplagiatStatus:   r.AntiplagiatStatus,
		PreDefenseStatus:    r.PreDefenseStatus,
		AdmissionStatus:     r.AdmissionStatus,
		CreatedAt:           timestamppb.New(r.CreatedAt),
		UpdatedAt:           timestamppb.New(r.UpdatedAt),
	}
	if r.AntiplagiatPercent != nil {
		pb.AntiplagiatPercent = *r.AntiplagiatPercent
		pb.HasAntiplagiatPercent = true
	}
	if r.PreDefenseGrade != nil {
		pb.PreDefenseGrade = *r.PreDefenseGrade
		pb.HasPreDefenseGrade = true
	}
	if r.FinalGrade != nil {
		pb.FinalGrade = *r.FinalGrade
		pb.HasFinalGrade = true
	}
	if r.LastActivityAt != nil {
		pb.LastActivityAt = timestamppb.New(*r.LastActivityAt)
	}
	return pb
}

func toPbDPSupervisor(u *DPUser) *adminv1.SupervisorInfo {
	if u == nil {
		return nil
	}
	return &adminv1.SupervisorInfo{
		Id:       u.ID,
		FullName: u.FullName,
		Email:    u.Email,
		Position: u.Position,
	}
}

func toPbDPMembers(members []*DPMember) []*adminv1.TeamMemberInfo {
	out := make([]*adminv1.TeamMemberInfo, 0, len(members))
	for _, m := range members {
		out = append(out, &adminv1.TeamMemberInfo{
			UserId:   m.UserID,
			FullName: m.FullName,
			Email:    m.Email,
			Role:     m.Role,
		})
	}
	return out
}

// ==================== Team details ====================

func (h *Handler) GetDepartmentProgressTeamDetails(ctx context.Context, req *adminv1.GetDepartmentProgressTeamDetailsRequest) (*adminv1.DepartmentProgressTeamDetailsResponse, error) {
	if req.TeamId <= 0 {
		return nil, status.Error(codes.InvalidArgument, "team_id is required")
	}
	if req.DepartmentId <= 0 {
		return nil, status.Error(codes.InvalidArgument, "department_id is required")
	}

	d, err := h.service.GetDepartmentProgressTeamDetails(ctx, req.TeamId, req.DepartmentId)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, status.Error(codes.NotFound, "team not found in this department")
		}
		h.logger.Error("GetDepartmentProgressTeamDetails failed", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "failed to get team details: %v", err)
	}

	resp := &adminv1.DepartmentProgressTeamDetailsResponse{
		Team: &adminv1.DepartmentProgressTeamInfo{
			TeamId:    d.Team.TeamID,
			TeamName:  d.Team.TeamName,
			Status:    d.Team.Status,
			CreatedAt: timestamppb.New(d.Team.CreatedAt),
			UpdatedAt: timestamppb.New(d.Team.UpdatedAt),
		},
		Supervisor: toPbDPSupervisor(d.Supervisor),
		Members:    toPbDPMembers(d.Members),
		Workflow: &adminv1.DepartmentProgressWorkflow{
			CurrentState:       d.CurrentStateName,
			ProgressPercentage: d.Progress,
		},
	}

	if d.Project != nil {
		resp.Project = &adminv1.DepartmentProgressProjectInfo{
			ProjectId:     d.Project.ProjectID,
			TitleKz:       d.Project.TitleKZ,
			TitleRu:       d.Project.TitleRU,
			TitleEn:       d.Project.TitleEN,
			TitleDisplay:  d.Project.TitleDisplay,
			Description:   d.Project.Description,
			CurrentState:  d.Project.CurrentState,
			CurrentStepId: d.Project.CurrentStepID,
		}
	}

	for _, st := range d.Steps {
		pbStep := &adminv1.DepartmentProgressStep{
			StepId:          st.StepID,
			StepName:        st.StepName,
			DisplayName:     st.DisplayName,
			Status:          st.Status,
			ReviewerName:    st.ReviewerName,
			AdmissionStatus: st.AdmissionStatus,
			Comment:         st.Comment,
		}
		if st.Grade != nil {
			pbStep.Grade = *st.Grade
			pbStep.HasGrade = true
		}
		if st.ReviewedAt != nil {
			pbStep.CompletedAt = timestamppb.New(*st.ReviewedAt)
		}
		resp.Workflow.Steps = append(resp.Workflow.Steps, pbStep)
	}

	if d.TopicRegistration != nil {
		tr := d.TopicRegistration
		pbTr := &adminv1.DepartmentProgressTopicRegistration{
			Status:          tr.Status,
			ProposedTopicKz: tr.ProposedTopicKZ,
			ProposedTopicRu: tr.ProposedTopicRU,
			ProposedTopicEn: tr.ProposedTopicEN,
			ReviewerName:    tr.ReviewerName,
			Comment:         tr.Comment,
			RejectionReason: tr.RejectionReason,
		}
		if tr.ReviewedAt != nil {
			pbTr.ReviewedAt = timestamppb.New(*tr.ReviewedAt)
		}
		resp.TopicRegistration = pbTr
	}

	if d.NormControl != nil {
		nc := d.NormControl
		pbNc := &adminv1.DepartmentProgressNormControl{
			Status:              nc.Status,
			SubmissionId:        nc.SubmissionID,
			ReviewerName:        nc.ReviewerName,
			IssuesCount:         nc.IssuesCount,
			CriticalIssuesCount: nc.CriticalIssuesCount,
		}
		if nc.ReviewedAt != nil {
			pbNc.ReviewedAt = timestamppb.New(*nc.ReviewedAt)
		}
		resp.NormControl = pbNc
	}

	if d.Antiplagiat != nil {
		ap := d.Antiplagiat
		pbAp := &adminv1.DepartmentProgressAntiplagiat{
			Status:       ap.Status,
			ReviewerName: ap.ReviewerName,
		}
		if ap.SimilarityPercent != nil {
			pbAp.SimilarityPercent = *ap.SimilarityPercent
			pbAp.HasSimilarityPercent = true
		}
		if ap.AIPercent != nil {
			pbAp.AiPercent = *ap.AIPercent
			pbAp.HasAiPercent = true
		}
		if ap.CheckedAt != nil {
			pbAp.CheckedAt = timestamppb.New(*ap.CheckedAt)
		}
		resp.Antiplagiat = pbAp
	}

	for _, pd := range d.PreDefenses {
		pbPd := &adminv1.DepartmentProgressPreDefense{
			Id:              pd.ID,
			Status:          pd.Status,
			Location:        pd.Location,
			Result:          pd.Result,
			AdmissionStatus: pd.AdmissionStatus,
			Comment:         pd.Comment,
		}
		if pd.ScheduledDate != nil {
			pbPd.ScheduledDate = timestamppb.New(*pd.ScheduledDate)
		}
		if pd.Grade != nil {
			pbPd.Grade = *pd.Grade
			pbPd.HasGrade = true
		}
		for _, c := range pd.Commission {
			pbPd.Commission = append(pbPd.Commission, &adminv1.PreDefenseCommissionMember{
				Id:        c.ID,
				UserId:    c.UserID,
				FullName:  c.FullName,
				Email:     c.Email,
				Role:      c.Role,
				IsPresent: c.IsPresent,
				Comment:   c.Comment,
			})
		}
		resp.PreDefenses = append(resp.PreDefenses, pbPd)
	}

	// Grades + resolved grader names.
	graderIDs := make([]int64, 0, len(d.Grades))
	for _, g := range d.Grades {
		if g.GradedBy > 0 {
			graderIDs = append(graderIDs, g.GradedBy)
		}
	}
	graderNames := h.service.ResolveUserNames(ctx, graderIDs)
	for _, g := range d.Grades {
		resp.Grades = append(resp.Grades, &adminv1.GradeInfo{
			Id:         g.ID,
			ProjectId:  g.ProjectID,
			StepId:     g.StepID,
			Grade:      g.Grade,
			GradedBy:   g.GradedBy,
			GraderName: graderNames[g.GradedBy],
			Comment:    g.Comment,
			GradedAt:   timestamppb.New(g.CreatedAt),
		})
	}

	for _, hi := range d.History {
		resp.History = append(resp.History, &adminv1.UnifiedHistoryItem{
			Id:        hi.ID,
			Source:    hi.Source,
			Action:    hi.Action,
			ActorId:   hi.ActorID,
			ActorName: hi.ActorName,
			OldValue:  hi.OldValue,
			NewValue:  hi.NewValue,
			Comment:   hi.Comment,
			CreatedAt: timestamppb.New(hi.CreatedAt),
		})
	}

	return resp, nil
}
