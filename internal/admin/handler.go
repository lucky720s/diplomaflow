package admin

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	adminv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/admin/v1"
	projectv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/project/v1"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Handler struct {
	adminv1.UnimplementedAdminServiceServer
	adminv1.UnimplementedNormControlServiceServer
	adminv1.UnimplementedAntiplagiatServiceServer
	service *Service
	logger  *zap.Logger
}

func NewHandler(service *Service, logger *zap.Logger) *Handler {
	return &Handler{
		service: service,
		logger:  logger,
	}
}

func derefInt64(p *int64) int64 {
	if p == nil {
		return 0
	}
	return *p
}

// ==================== Dashboard ====================

func (h *Handler) GetDashboard(ctx context.Context, req *adminv1.GetDashboardRequest) (*adminv1.GetDashboardResponse, error) {
	if req.DepartmentId == 0 {
		return nil, status.Error(codes.InvalidArgument, "department_id is required")
	}

	resp, err := h.service.GetDashboard(ctx, req.DepartmentId)
	if err != nil {
		h.logger.Error("GetDashboard failed", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "failed to get dashboard: %v", err)
	}

	pbResp := &adminv1.GetDashboardResponse{
		Stats: &adminv1.DashboardStats{
			TotalStudents:             resp.Stats.TotalStudents,
			TotalTeams:                resp.Stats.TotalTeams,
			TotalProjects:             resp.Stats.TotalProjects,
			CompletedProjects:         resp.Stats.CompletedProjects,
			PendingReviews:            resp.Stats.PendingReviews,
			ActiveSupervisors:         resp.Stats.ActiveSupervisors,
			PendingTopicRegistrations: resp.Stats.PendingTopicRegistrations,
			PendingSupervisorRequests: resp.Stats.PendingSupervisorRequests,
			PendingPreDefenses:        resp.Stats.PendingPreDefenses,
			ScheduledPreDefenses:      resp.Stats.ScheduledPreDefenses,
			PreDefensesThisWeek:       resp.Stats.PreDefensesThisWeek,
		},
	}

	for _, sp := range resp.StepProgress {
		var completionPct float32
		if sp.TotalTeams > 0 {
			completionPct = float32(sp.CompletedTeams) / float32(sp.TotalTeams) * 100
		}

		pbResp.StepProgress = append(pbResp.StepProgress, &adminv1.StepProgress{
			StepId:               sp.StepID,
			StepName:             sp.StepName,
			StepType:             sp.StepType,
			TotalTeams:           sp.TotalTeams,
			CompletedTeams:       sp.CompletedTeams,
			PendingTeams:         sp.PendingTeams,
			RejectedTeams:        sp.RejectedTeams,
			CompletionPercentage: completionPct,
		})
	}

	for _, act := range resp.Activities {
		pbResp.RecentActivities = append(pbResp.RecentActivities, &adminv1.RecentActivity{
			Id:           act.ID,
			ActivityType: act.ActivityType,
			Description:  act.Description,
			ActorId:      act.ActorID,
			TargetId:     act.TargetID,
			TargetType:   act.TargetType,
			CreatedAt:    timestamppb.New(act.CreatedAt),
		})
	}

	return pbResp, nil
}

func (h *Handler) GetDepartmentStats(ctx context.Context, req *adminv1.GetDepartmentStatsRequest) (*adminv1.GetDepartmentStatsResponse, error) {
	resp, err := h.service.GetDashboard(ctx, req.DepartmentId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get stats: %v", err)
	}

	pbResp := &adminv1.GetDepartmentStatsResponse{
		Stats: &adminv1.DashboardStats{
			TotalStudents:             resp.Stats.TotalStudents,
			TotalTeams:                resp.Stats.TotalTeams,
			TotalProjects:             resp.Stats.TotalProjects,
			CompletedProjects:         resp.Stats.CompletedProjects,
			PendingReviews:            resp.Stats.PendingReviews,
			ActiveSupervisors:         resp.Stats.ActiveSupervisors,
			PendingTopicRegistrations: resp.Stats.PendingTopicRegistrations,
			PendingSupervisorRequests: resp.Stats.PendingSupervisorRequests,
			PendingPreDefenses:        resp.Stats.PendingPreDefenses,
			ScheduledPreDefenses:      resp.Stats.ScheduledPreDefenses,
			PreDefensesThisWeek:       resp.Stats.PreDefensesThisWeek,
		},
	}

	return pbResp, nil
}

// ==================== Students ====================

func (h *Handler) ListStudents(ctx context.Context, req *adminv1.ListStudentsRequest) (*adminv1.ListStudentsResponse, error) {
	students, total, err := h.service.ListStudents(ctx, &ListStudentsRequest{
		UniversityID:    req.UniversityId,
		DepartmentID:    req.DepartmentId,
		Search:          req.Search,
		OnlyWithoutTeam: req.OnlyWithoutTeam,
		Page:            req.Page,
		PageSize:        req.PageSize,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list students: %v", err)
	}

	var pbStudents []*adminv1.StudentInfo
	for _, s := range students {
		pbStudents = append(pbStudents, &adminv1.StudentInfo{
			Id:           s.ID,
			Email:        s.Email,
			FirstName:    s.FirstName,
			LastName:     s.LastName,
			TeamId:       s.TeamID,
			TeamName:     s.TeamName,
			ProjectId:    s.ProjectID,
			ProjectTitle: s.ProjectTitle,
			CurrentStep:  s.CurrentStep,
		})
	}

	return &adminv1.ListStudentsResponse{
		Students:   pbStudents,
		TotalCount: total,
	}, nil
}

func (h *Handler) GetStudent(ctx context.Context, req *adminv1.GetStudentRequest) (*adminv1.GetStudentResponse, error) {
	if req.StudentId == 0 {
		return nil, status.Error(codes.InvalidArgument, "student_id is required")
	}

	resp, err := h.service.GetStudentFullInfo(ctx, req.StudentId)
	if err != nil {
		h.logger.Error("GetStudent failed", zap.Error(err))
		return nil, status.Errorf(codes.NotFound, "student not found: %v", err)
	}

	pbStudent := &adminv1.StudentInfo{
		Id:           resp.Student.ID,
		Email:        resp.Student.Email,
		FirstName:    resp.Student.FirstName,
		LastName:     resp.Student.LastName,
		TeamId:       resp.Student.TeamID,
		TeamName:     resp.Student.TeamName,
		ProjectId:    resp.Student.ProjectID,
		ProjectTitle: resp.Student.ProjectTitle,
		CurrentStep:  resp.Student.CurrentStep,
		CreatedAt:    timestamppb.New(resp.Student.CreatedAt),
	}

	var pbGrades []*adminv1.GradeInfo
	for _, g := range resp.Grades {
		pbGrades = append(pbGrades, &adminv1.GradeInfo{
			Id:        g.ID,
			ProjectId: g.ProjectID,
			StepId:    g.StepID,
			Grade:     g.Grade,
			GradedBy:  g.GradedBy,
			Comment:   g.Comment,
			GradedAt:  timestamppb.New(g.CreatedAt),
		})
	}

	var pbSubmissions []*adminv1.SubmissionPreview
	for _, s := range resp.Submissions {
		pbSubmissions = append(pbSubmissions, &adminv1.SubmissionPreview{
			Id:          s.ID,
			Status:      s.Status,
			SubmittedAt: timestamppb.New(s.CreatedAt),
		})
	}

	return &adminv1.GetStudentResponse{
		Student:     pbStudent,
		Grades:      pbGrades,
		Submissions: pbSubmissions,
	}, nil
}

// ==================== Teams ====================

func (h *Handler) ListAllTeams(ctx context.Context, req *adminv1.ListAllTeamsRequest) (*adminv1.ListAllTeamsResponse, error) {
	teams, total, err := h.service.ListAllTeams(ctx, &ListAllTeamsRequest{
		DepartmentID: req.DepartmentId,
		Status:       req.Status,
		Search:       req.Search,
		Page:         req.Page,
		PageSize:     req.PageSize,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list teams: %v", err)
	}

	var pbTeams []*adminv1.TeamAdminInfo
	for _, t := range teams {
		pbTeams = append(pbTeams, &adminv1.TeamAdminInfo{
			Id:           t.ID,
			Name:         t.Name,
			ProjectTitle: t.ProjectTitle,
			CurrentStep:  t.CurrentStep,
			MemberCount:  t.MemberCount,
			Status:       t.Status,
		})
	}

	return &adminv1.ListAllTeamsResponse{
		Teams:      pbTeams,
		TotalCount: total,
	}, nil
}

func (h *Handler) GetTeamDetails(ctx context.Context, req *adminv1.GetTeamDetailsRequest) (*adminv1.GetTeamDetailsResponse, error) {
	if req.TeamId == 0 {
		return nil, status.Error(codes.InvalidArgument, "team_id is required")
	}

	resp, err := h.service.GetTeamFullDetails(ctx, req.TeamId)
	if err != nil {
		h.logger.Error("GetTeamDetails failed", zap.Error(err))
		return nil, status.Errorf(codes.NotFound, "team not found: %v", err)
	}

	var pbMembers []*adminv1.TeamMemberInfo
	for _, m := range resp.Team.Members {
		pbMembers = append(pbMembers, &adminv1.TeamMemberInfo{
			UserId:   m.UserID,
			FullName: m.FullName,
			Email:    m.Email,
			Role:     m.Role,
		})
	}

	pbTeam := &adminv1.TeamAdminInfo{
		Id:           resp.Team.ID,
		Name:         resp.Team.Name,
		ProjectTitle: resp.Team.ProjectTitle,
		CurrentStep:  resp.Team.CurrentStep,
		MemberCount:  int32(len(resp.Team.Members)),
		Members:      pbMembers,
		Status:       resp.Team.Status,
		CreatedAt:    timestamppb.New(resp.Team.CreatedAt),
		UpdatedAt:    timestamppb.New(resp.Team.UpdatedAt),
	}

	if resp.Supervisor != nil {
		pbTeam.Supervisor = &adminv1.SupervisorInfo{
			Id:       resp.Supervisor.ID,
			FullName: resp.Supervisor.FullName,
			Email:    resp.Supervisor.Email,
		}
	}

	pbResp := &adminv1.GetTeamDetailsResponse{
		Team: pbTeam,
	}

	for _, g := range resp.Grades {
		pbResp.Grades = append(pbResp.Grades, &adminv1.GradeInfo{
			Id:        g.ID,
			ProjectId: g.ProjectID,
			StepId:    g.StepID,
			Grade:     g.Grade,
			GradedBy:  g.GradedBy,
			Comment:   g.Comment,
			GradedAt:  timestamppb.New(g.CreatedAt),
		})
	}

	for _, s := range resp.Submissions {
		pbResp.Submissions = append(pbResp.Submissions, &adminv1.SubmissionPreview{
			Id:          s.ID,
			Status:      s.Status,
			SubmittedAt: timestamppb.New(s.CreatedAt),
		})
	}

	if resp.ProjectProgress != nil {
		pbResp.ProjectProgress = toPbProjectProgress(resp.ProjectProgress)
	}

	return pbResp, nil
}

func toPbProjectProgress(p *TeamProgressResponse) *adminv1.ProjectProgress {
	if p == nil {
		return nil
	}
	pbSteps := make([]*adminv1.StepStatus, 0, len(p.Steps))
	for _, st := range p.Steps {
		pbStep := &adminv1.StepStatus{
			StepId:   st.StepID,
			StepName: st.StepName,
			Status:   st.Status,
		}
		if st.CompletedAt != nil {
			pbStep.CompletedAt = timestamppb.New(*st.CompletedAt)
		}
		if st.Grade != nil {
			pbStep.Grade = &adminv1.GradeInfo{
				Id:        st.Grade.ID,
				ProjectId: st.Grade.ProjectID,
				StepId:    st.Grade.StepID,
				StepName:  st.StepName,
				Grade:     st.Grade.Grade,
				GradedBy:  st.Grade.GradedBy,
				Comment:   st.Grade.Comment,
				GradedAt:  timestamppb.New(st.Grade.CreatedAt),
			}
		}
		pbSteps = append(pbSteps, pbStep)
	}
	return &adminv1.ProjectProgress{
		ProjectId:     p.ProjectID,
		Title:         p.ProjectTitle,
		CurrentState:  p.CurrentStateName,
		CurrentStepId: p.CurrentStateID,
		Steps:         pbSteps,
	}
}

func getActorIDFromContext(ctx context.Context) int64 {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return 0
	}
	if vals := md.Get("x-user-id"); len(vals) > 0 {
		id, _ := strconv.ParseInt(vals[0], 10, 64)
		return id
	}
	return 0
}
func (h *Handler) UpdateTeamAdmin(ctx context.Context, req *adminv1.UpdateTeamAdminRequest) (*adminv1.UpdateTeamAdminResponse, error) {
	if req.TeamId == 0 {
		return nil, status.Error(codes.InvalidArgument, "team_id is required")
	}

	actorID := getActorIDFromContext(ctx)
	if actorID <= 0 {
		return nil, status.Error(codes.Unauthenticated, "actor_id is required in metadata (x-user-id)")
	}

	team, err := h.service.UpdateTeamByAdmin(ctx, &UpdateTeamByAdminRequest{
		TeamID:       req.TeamId,
		Name:         req.Name,
		SupervisorID: req.SupervisorId,
		MemberIDs:    req.MemberIds,
		UpdatedBy:    actorID,
	})
	if err != nil {
		h.logger.Error("UpdateTeamAdmin failed", zap.Error(err))
		// FIX: если уже status-ошибка — возвращаем как есть
		if st, ok := status.FromError(err); ok {
			return nil, st.Err()
		}
		return nil, status.Errorf(codes.Internal, "failed to update team: %v", err)
	}

	var pbMembers []*adminv1.TeamMemberInfo
	for _, m := range team.Members {
		pbMembers = append(pbMembers, &adminv1.TeamMemberInfo{
			UserId:   m.UserID,
			FullName: m.FullName,
			Email:    m.Email,
			Role:     m.Role,
		})
	}

	return &adminv1.UpdateTeamAdminResponse{
		Team: &adminv1.TeamAdminInfo{
			Id:           team.ID,
			Name:         team.Name,
			ProjectTitle: team.ProjectTitle,
			CurrentStep:  team.CurrentStep,
			MemberCount:  int32(len(team.Members)),
			Members:      pbMembers,
			Status:       team.Status,
			CreatedAt:    timestamppb.New(team.CreatedAt),
			UpdatedAt:    timestamppb.New(team.UpdatedAt),
		},
	}, nil
}
func (h *Handler) DeleteTeamAdmin(ctx context.Context, req *adminv1.DeleteTeamAdminRequest) (*emptypb.Empty, error) {
	if req.TeamId == 0 {
		return nil, status.Error(codes.InvalidArgument, "team_id is required")
	}

	actorID := getActorIDFromContext(ctx)
	if actorID <= 0 {
		return nil, status.Error(codes.Unauthenticated, "actor_id is required in metadata (x-user-id)")
	}

	reason := req.Reason
	if reason == "" {
		reason = "Deleted by admin"
	}

	err := h.service.DeleteTeamByAdmin(ctx, req.TeamId, reason, actorID)
	if err != nil {
		h.logger.Error("DeleteTeamAdmin failed", zap.Error(err))
		if st, ok := status.FromError(err); ok {
			return nil, st.Err()
		}
		return nil, status.Errorf(codes.Internal, "failed to delete team: %v", err)
	}

	return &emptypb.Empty{}, nil
}

// ==================== Supervisors ====================

func (h *Handler) ListSupervisors(ctx context.Context, req *adminv1.ListSupervisorsRequest) (*adminv1.ListSupervisorsResponse, error) {
	supervisors, total, err := h.service.ListSupervisors(ctx, req.DepartmentId, req.UniversityId, req.Page, req.PageSize)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list supervisors: %v", err)
	}

	var pbSupervisors []*adminv1.SupervisorDetails
	for _, s := range supervisors {
		pbSupervisors = append(pbSupervisors, &adminv1.SupervisorDetails{
			Id:         s.ID,
			FirstName:  s.FirstName,
			LastName:   s.LastName,
			Email:      s.Email,
			Position:   s.Position,
			TeamsCount: s.TeamsCount,
			MaxTeams:   s.MaxTeams,
		})
	}

	return &adminv1.ListSupervisorsResponse{
		Supervisors: pbSupervisors,
		TotalCount:  total,
	}, nil
}

func (h *Handler) AssignSupervisor(ctx context.Context, req *adminv1.AssignSupervisorRequest) (*adminv1.AssignSupervisorResponse, error) {
	if req.TeamId == 0 || req.SupervisorId == 0 {
		return nil, status.Error(codes.InvalidArgument, "team_id and supervisor_id are required")
	}

	actorID := getActorIDFromContext(ctx)
	if actorID == 0 {
		return nil, status.Error(codes.Unauthenticated, "actor_id is required for audit trail")
	}

	err := h.service.AssignSupervisor(ctx, req.TeamId, req.SupervisorId, actorID, false)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to assign supervisor: %v", err)
	}

	return &adminv1.AssignSupervisorResponse{
		Success: true,
		Message: "Supervisor assigned successfully",
	}, nil
}

// ==================== Topic Registration ====================

func (h *Handler) SubmitTopicRegistration(ctx context.Context, req *adminv1.SubmitTopicRegistrationRequest) (*adminv1.SubmitTopicRegistrationResponse, error) {
	if req.ProjectId == 0 {
		return nil, status.Error(codes.InvalidArgument, "project_id is required")
	}
	if strings.TrimSpace(req.ProposedTopicKz) == "" {
		return nil, status.Error(codes.InvalidArgument, "proposed_topic_kz is required")
	}
	if strings.TrimSpace(req.ProposedTopicRu) == "" {
		return nil, status.Error(codes.InvalidArgument, "proposed_topic_ru is required")
	}
	if strings.TrimSpace(req.ProposedTopicEn) == "" {
		return nil, status.Error(codes.InvalidArgument, "proposed_topic_en is required")
	}
	if req.SupervisorId == 0 {
		return nil, status.Error(codes.InvalidArgument, "supervisor_id is required")
	}
	if req.SubmittedBy == 0 {
		return nil, status.Error(codes.InvalidArgument, "submitted_by is required")
	}

	reg, err := h.service.SubmitTopicRegistration(ctx, &SubmitTopicRegistrationRequest{
		ProjectID:        req.ProjectId,
		TeamID:           req.TeamId,
		ProposedTopicKz:  req.ProposedTopicKz,
		ProposedTopicRu:  req.ProposedTopicRu,
		ProposedTopicEn:  req.ProposedTopicEn,
		TopicDescription: req.TopicDescription,
		SupervisorID:     req.SupervisorId,
		SubmittedBy:      req.SubmittedBy,
	})
	if err != nil {
		if _, ok := status.FromError(err); ok {
			return nil, err
		}
		return nil, status.Errorf(codes.Internal,
			"failed to submit topic registration: %v", err)
	}

	return &adminv1.SubmitTopicRegistrationResponse{
		Success:        true,
		RegistrationId: reg.ID,
		Message:        "Заявление на регистрацию темы успешно подано",
	}, nil
}

func (h *Handler) ListTopicRegistrations(ctx context.Context, req *adminv1.ListTopicRegistrationsRequest) (*adminv1.ListTopicRegistrationsResponse, error) {
	pageSize := int(req.PageSize)
	if pageSize <= 0 {
		pageSize = 20
	}
	page := int(req.Page)
	if page <= 0 {
		page = 1
	}

	filter := TopicRegistrationFilter{
		DepartmentID: req.DepartmentId,
		Status:       req.Status,
		SupervisorID: req.SupervisorId,
		Limit:        pageSize,
		Offset:       (page - 1) * pageSize,
	}

	regs, total, err := h.service.ListTopicRegistrations(ctx, filter)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list topic registrations: %v", err)
	}

	var pbRegs []*adminv1.TopicRegistrationInfo
	for _, r := range regs {
		pbReg := &adminv1.TopicRegistrationInfo{
			Id:               r.ID,
			TeamId:           r.TeamID,
			ProjectId:        r.ProjectID,
			ProposedTopicKz:  r.ProposedTopicKz,
			ProposedTopicRu:  r.ProposedTopicRu,
			ProposedTopicEn:  r.ProposedTopicEn,
			TopicDescription: r.TopicDescription,
			SupervisorId:     r.SupervisorID,
			SubmittedBy:      r.SubmittedBy,
			Status:           r.Status,
			RejectionReason:  r.RejectionReason,
			Comment:          r.Comment,
			SubmittedAt:      timestamppb.New(r.CreatedAt),
		}
		if r.ReviewerID != nil {
			pbReg.ReviewedBy = *r.ReviewerID
		}
		if r.ReviewedAt != nil {
			pbReg.ReviewedAt = timestamppb.New(*r.ReviewedAt)
		}
		pbRegs = append(pbRegs, pbReg)
	}

	return &adminv1.ListTopicRegistrationsResponse{
		Registrations: pbRegs,
		TotalCount:    total,
	}, nil
}

func (h *Handler) GetTopicRegistration(ctx context.Context, req *adminv1.GetTopicRegistrationRequest) (*adminv1.GetTopicRegistrationResponse, error) {
	if req.RegistrationId == "" {
		return nil, status.Error(codes.InvalidArgument, "registration_id is required")
	}

	reg, reviews, err := h.service.GetTopicRegistration(ctx, req.RegistrationId)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "topic registration not found: %v", err)
	}

	pbReg := &adminv1.TopicRegistrationInfo{
		Id:               reg.ID,
		TeamId:           reg.TeamID,
		ProjectId:        reg.ProjectID,
		ProposedTopicKz:  reg.ProposedTopicKz,
		ProposedTopicRu:  reg.ProposedTopicRu,
		ProposedTopicEn:  reg.ProposedTopicEn,
		TopicDescription: reg.TopicDescription,
		SupervisorId:     reg.SupervisorID,
		SubmittedBy:      reg.SubmittedBy,
		Status:           reg.Status,
		RejectionReason:  reg.RejectionReason,
		Comment:          reg.Comment,
		SubmittedAt:      timestamppb.New(reg.CreatedAt),
	}
	if reg.ReviewerID != nil {
		pbReg.ReviewedBy = *reg.ReviewerID
	}
	if reg.ReviewedAt != nil {
		pbReg.ReviewedAt = timestamppb.New(*reg.ReviewedAt)
	}

	var pbHistory []*adminv1.TopicRegistrationHistory
	for _, r := range reviews {
		pbHistory = append(pbHistory, &adminv1.TopicRegistrationHistory{
			Id:         r.ID,
			ReviewerId: r.ReviewerID,
			Action:     r.Action,
			Comment:    r.Comment,
			CreatedAt:  timestamppb.New(r.CreatedAt),
		})
	}

	return &adminv1.GetTopicRegistrationResponse{
		Registration: pbReg,
		History:      pbHistory,
	}, nil
}

func (h *Handler) ReviewTopicRegistration(ctx context.Context, req *adminv1.ReviewTopicRegistrationRequest) (*adminv1.ReviewTopicRegistrationResponse, error) {
	if req.RegistrationId == "" {
		return nil, status.Error(codes.InvalidArgument, "registration_id is required")
	}
	if req.ReviewerId == 0 {
		return nil, status.Error(codes.InvalidArgument, "reviewer_id is required")
	}
	if req.Action == "" {
		return nil, status.Error(codes.InvalidArgument, "action is required")
	}

	reg, err := h.service.ReviewTopicRegistration(ctx, &ReviewTopicRegistrationRequest{
		RegistrationID:  req.RegistrationId,
		ReviewerID:      req.ReviewerId,
		Action:          req.Action,
		Comment:         req.Comment,
		RejectionReason: req.RejectionReason,
	})
	if err != nil {
		if _, ok := status.FromError(err); ok {
			return nil, err
		}
		return nil, status.Errorf(codes.Internal,
			"failed to review topic registration: %v", err)
	}
	pbReg := &adminv1.TopicRegistrationInfo{
		Id:               reg.ID,
		TeamId:           reg.TeamID,
		ProjectId:        reg.ProjectID,
		ProposedTopicKz:  reg.ProposedTopicKz,
		ProposedTopicRu:  reg.ProposedTopicRu,
		ProposedTopicEn:  reg.ProposedTopicEn,
		TopicDescription: reg.TopicDescription,
		SupervisorId:     reg.SupervisorID,
		Status:           reg.Status,
		Comment:          reg.Comment,
		RejectionReason:  reg.RejectionReason,
		SubmittedAt:      timestamppb.New(reg.CreatedAt),
	}
	if reg.ReviewerID != nil {
		pbReg.ReviewedBy = *reg.ReviewerID
	}
	if reg.ReviewedAt != nil {
		pbReg.ReviewedAt = timestamppb.New(*reg.ReviewedAt)
	}

	return &adminv1.ReviewTopicRegistrationResponse{
		Success:             true,
		Message:             "Заявление успешно рассмотрено",
		UpdatedRegistration: pbReg,
	}, nil
}

// ==================== Submissions ====================

func (h *Handler) ListSubmissions(ctx context.Context, req *adminv1.ListSubmissionsRequest) (*adminv1.ListSubmissionsResponse, error) {
	pageSize := int(req.PageSize)
	if pageSize <= 0 {
		pageSize = 20
	}
	page := int(req.Page)
	if page <= 0 {
		page = 1
	}

	filter := SubmissionFilter{
		DepartmentID: req.DepartmentId,
		StepID:       req.StepId,
		TeamID:       req.TeamId,
		ReviewerID:   req.ReviewerId,
		Status:       req.Status,
		Limit:        pageSize,
		Offset:       (page - 1) * pageSize,
		ProjectID:    req.ProjectId,
	}

	submissions, total, err := h.service.ListSubmissions(ctx, filter)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list submissions: %v", err)
	}

	var pbSubmissions []*adminv1.SubmissionInfo

	for _, sub := range submissions {
		var dataMap map[string]interface{}
		_ = json.Unmarshal(sub.Data, &dataMap)
		pbData, _ := structpb.NewStruct(dataMap)

		// ✅ files
		var files []struct {
			FileName    string `json:"file_name"`
			DownloadURL string `json:"download_url"`
		}

		if len(sub.Files) > 0 {
			_ = json.Unmarshal(sub.Files, &files)
		}

		var pbFiles []*adminv1.FileAttachment
		for _, f := range files {
			pbFiles = append(pbFiles, &adminv1.FileAttachment{
				FileName:    f.FileName,
				DownloadUrl: f.DownloadURL,
			})
		}

		pbSub := &adminv1.SubmissionInfo{
			Id:          sub.ID,
			ProjectId:   sub.ProjectID,
			TeamId:      sub.TeamID,
			StepId:      sub.StepID,
			SubmittedBy: sub.SubmittedBy,
			Status:      sub.Status,
			Data:        pbData,
			SubmittedAt: timestamppb.New(sub.CreatedAt),

			// ✅ КЛЮЧЕВОЕ
			TeamName: sub.TeamName,
			StepName: sub.StepName,

			Files: pbFiles,
		}

		if sub.ReviewerID != nil {
			pbSub.Review = &adminv1.ReviewInfo{
				ReviewerId: *sub.ReviewerID,
				Status:     sub.Status,
				Comment:    sub.ReviewComment,
			}
			if sub.ReviewedAt != nil {
				pbSub.Review.ReviewedAt = timestamppb.New(*sub.ReviewedAt)
			}
		}

		pbSubmissions = append(pbSubmissions, pbSub)
	}

	return &adminv1.ListSubmissionsResponse{
		Submissions: pbSubmissions,
		TotalCount:  total,
	}, nil
}

func (h *Handler) GetSubmission(ctx context.Context, req *adminv1.GetSubmissionRequest) (*adminv1.GetSubmissionResponse, error) {
	sub, reviews, err := h.service.GetSubmission(ctx, req.SubmissionId)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "submission not found: %v", err)
	}

	var dataMap map[string]interface{}
	_ = json.Unmarshal(sub.Data, &dataMap)
	pbData, _ := structpb.NewStruct(dataMap)

	pbSub := &adminv1.SubmissionInfo{
		Id:          sub.ID,
		ProjectId:   sub.ProjectID,
		TeamId:      sub.TeamID,
		StepId:      sub.StepID,
		SubmittedBy: sub.SubmittedBy,
		Status:      sub.Status,
		Data:        pbData,
		SubmittedAt: timestamppb.New(sub.CreatedAt),
	}

	var pbHistory []*adminv1.ReviewHistory
	for _, r := range reviews {
		pbHistory = append(pbHistory, &adminv1.ReviewHistory{
			Id:         r.ID,
			ReviewerId: r.ReviewerID,
			Action:     r.Action,
			Comment:    r.Comment,
			CreatedAt:  timestamppb.New(r.CreatedAt),
		})
	}

	return &adminv1.GetSubmissionResponse{
		Submission: pbSub,
		History:    pbHistory,
	}, nil
}

func (h *Handler) ReviewSubmission(ctx context.Context, req *adminv1.ReviewSubmissionRequest) (*adminv1.ReviewSubmissionResponse, error) {
	if req.SubmissionId == "" {
		return nil, status.Error(codes.InvalidArgument, "submission_id is required")
	}
	if req.ReviewerId == 0 {
		return nil, status.Error(codes.InvalidArgument, "reviewer_id is required")
	}
	if req.Action == "" {
		return nil, status.Error(codes.InvalidArgument, "action is required")
	}

	sub, err := h.service.ReviewSubmission(ctx, &ReviewSubmissionRequest{
		SubmissionID: req.SubmissionId,
		ReviewerID:   req.ReviewerId,
		Action:       req.Action,
		Comment:      req.Comment,
		Grade:        req.Grade,
	})
	if err != nil {
		if _, ok := status.FromError(err); ok {
			return nil, err
		}
		return nil, status.Errorf(codes.Internal, "failed to review submission: %v", err)
	}

	var dataMap map[string]interface{}
	_ = json.Unmarshal(sub.Data, &dataMap)
	pbData, _ := structpb.NewStruct(dataMap)

	return &adminv1.ReviewSubmissionResponse{
		Success: true,
		Message: "Submission reviewed successfully",
		UpdatedSubmission: &adminv1.SubmissionInfo{
			Id:          sub.ID,
			ProjectId:   sub.ProjectID,
			Status:      sub.Status,
			Data:        pbData,
			SubmittedAt: timestamppb.New(sub.CreatedAt),
		},
	}, nil
}

// ==================== Grading ====================

func (h *Handler) GetProjectGrades(ctx context.Context, req *adminv1.GetProjectGradesRequest) (*adminv1.GetProjectGradesResponse, error) {
	grades, avg, err := h.service.GetProjectGrades(ctx, req.ProjectId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get grades: %v", err)
	}

	var pbGrades []*adminv1.GradeInfo
	for _, g := range grades {
		pbGrades = append(pbGrades, &adminv1.GradeInfo{
			Id:        g.ID,
			ProjectId: g.ProjectID,
			StepId:    g.StepID,
			Grade:     g.Grade,
			GradedBy:  g.GradedBy,
			Comment:   g.Comment,
			GradedAt:  timestamppb.New(g.CreatedAt),
		})
	}

	return &adminv1.GetProjectGradesResponse{
		ProjectId:    req.ProjectId,
		StepGrades:   pbGrades,
		AverageGrade: avg,
		TotalScore:   avg,
	}, nil
}

func (h *Handler) SetStepGrade(ctx context.Context, req *adminv1.SetStepGradeRequest) (*adminv1.SetStepGradeResponse, error) {
	if req.ProjectId == 0 || req.StepId == 0 {
		return nil, status.Error(codes.InvalidArgument, "project_id and step_id are required")
	}
	if req.Grade < 0 || req.Grade > 100 {
		return nil, status.Error(codes.InvalidArgument, "grade must be between 0 and 100")
	}

	grade, err := h.service.SetStepGrade(ctx, &SetGradeRequest{
		ProjectID: req.ProjectId,
		StepID:    req.StepId,
		Grade:     req.Grade,
		Comment:   req.Comment,
		GraderID:  req.GraderId,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to set grade: %v", err)
	}

	return &adminv1.SetStepGradeResponse{
		Success: true,
		Grade: &adminv1.GradeInfo{
			Id:        grade.ID,
			ProjectId: grade.ProjectID,
			StepId:    grade.StepID,
			Grade:     grade.Grade,
			GradedBy:  grade.GradedBy,
			Comment:   grade.Comment,
			GradedAt:  timestamppb.New(grade.CreatedAt),
		},
	}, nil
}

func (h *Handler) GetGradingHistory(ctx context.Context, req *adminv1.GetGradingHistoryRequest) (*adminv1.GetGradingHistoryResponse, error) {
	if req.ProjectId == 0 {
		return nil, status.Error(codes.InvalidArgument, "project_id is required")
	}

	history, err := h.service.GetGradingHistoryFull(ctx, req.ProjectId, req.StepId)
	if err != nil {
		h.logger.Error("GetGradingHistory failed", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "failed to get grading history: %v", err)
	}

	var pbHistory []*adminv1.GradeHistoryItem
	for _, item := range history {
		pbHistory = append(pbHistory, &adminv1.GradeHistoryItem{
			Id:          item.ID,
			OldGrade:    item.OldGrade,
			NewGrade:    item.NewGrade,
			ChangedBy:   item.ChangedBy,
			ChangerName: item.ChangerName,
			Reason:      item.Reason,
			ChangedAt:   timestamppb.New(item.ChangedAt),
		})
	}

	return &adminv1.GetGradingHistoryResponse{
		History: pbHistory,
	}, nil
}

// ==================== Workflow Progress ====================

func (h *Handler) GetWorkflowProgress(ctx context.Context, req *adminv1.GetWorkflowProgressRequest) (*adminv1.GetWorkflowProgressResponse, error) {
	progress, err := h.service.GetWorkflowProgress(ctx, req.DepartmentId, req.WorkflowId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get progress: %v", err)
	}

	var pbSteps []*adminv1.StepProgress
	for _, sp := range progress {
		var completionPct float32
		if sp.TotalTeams > 0 {
			completionPct = float32(sp.CompletedTeams) / float32(sp.TotalTeams) * 100
		}

		pbSteps = append(pbSteps, &adminv1.StepProgress{
			StepId:               sp.StepID,
			StepName:             sp.StepName,
			StepType:             sp.StepType,
			TotalTeams:           sp.TotalTeams,
			CompletedTeams:       sp.CompletedTeams,
			PendingTeams:         sp.PendingTeams,
			RejectedTeams:        sp.RejectedTeams,
			CompletionPercentage: completionPct,
		})
	}

	return &adminv1.GetWorkflowProgressResponse{
		WorkflowId: req.WorkflowId,
		Steps:      pbSteps,
	}, nil
}

func (h *Handler) GetTeamProgress(ctx context.Context, req *adminv1.GetTeamProgressRequest) (*adminv1.GetTeamProgressResponse, error) {
	if req.GetTeamId() <= 0 && req.GetProjectId() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "team_id or project_id is required")
	}

	resp, err := h.service.GetTeamProgress(ctx, req.GetTeamId(), req.GetProjectId())
	if err != nil {
		if st, ok := status.FromError(err); ok {
			return nil, st.Err()
		}
		h.logger.Error("GetTeamProgress failed", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "failed to get team progress: %v", err)
	}

	pbHistory := make([]*adminv1.StateHistoryItem, 0, len(resp.History))
	for _, h := range resp.History {
		item := &adminv1.StateHistoryItem{
			FromStepId:    h.FromStateID.Int64,
			FromStepName:  h.FromStateName.String,
			ToStepId:      h.ToStateID.Int64,
			ToStepName:    h.ToStateName.String,
			EventName:     h.EventName.String,
			ChangedBy:     h.ChangedBy.Int64,
			ChangedByName: h.ChangedByName.String,
			Comment:       h.Comment.String,
			CreatedAt:     timestamppb.New(h.CreatedAt),
		}
		pbHistory = append(pbHistory, item)
	}

	return &adminv1.GetTeamProgressResponse{
		Progress: toPbProjectProgress(resp),
		History:  pbHistory,
	}, nil
}

func (h *Handler) ListPendingReviews(ctx context.Context, req *adminv1.ListPendingReviewsRequest) (*adminv1.ListPendingReviewsResponse, error) {
	pageSize := req.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}
	page := req.Page
	if page <= 0 {
		page = 1
	}

	submissions, total, err := h.service.ListPendingReviews(ctx, req.DepartmentId, page, pageSize)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list pending reviews: %v", err)
	}

	var pbReviews []*adminv1.PendingReview
	for _, sub := range submissions {
		daysPending := int32(0)
		if !sub.CreatedAt.IsZero() {
			daysPending = int32(time.Since(sub.CreatedAt).Hours() / 24)
		}

		priority := "low"
		if daysPending > 7 {
			priority = "urgent"
		} else if daysPending > 3 {
			priority = "high"
		} else if daysPending > 1 {
			priority = "medium"
		}

		pbReviews = append(pbReviews, &adminv1.PendingReview{
			SubmissionId: sub.ID,
			ProjectId:    sub.ProjectID,
			TeamId:       sub.TeamID,
			StepId:       sub.StepID,
			SubmittedAt:  timestamppb.New(sub.CreatedAt),
			DaysPending:  daysPending,
			Priority:     priority,
		})
	}

	return &adminv1.ListPendingReviewsResponse{
		Reviews:    pbReviews,
		TotalCount: total,
	}, nil
}

// ==================== Supervisor Requests ====================

// Project-first (оставляем как есть)
func (h *Handler) CreateSupervisorRequest(ctx context.Context, req *adminv1.CreateSupervisorRequestReq) (*adminv1.CreateSupervisorRequestResp, error) {
	if req.ProjectId == 0 {
		return nil, status.Error(codes.InvalidArgument, "project_id is required")
	}
	if req.SupervisorId == 0 {
		return nil, status.Error(codes.InvalidArgument, "supervisor_id is required")
	}
	if req.RequestedBy == 0 {
		return nil, status.Error(codes.InvalidArgument, "requested_by is required")
	}

	result, err := h.service.CreateSupervisorRequest(ctx, &CreateSupervisorRequestInput{
		ProjectID:     req.ProjectId,
		TeamID:        req.TeamId, // optional (0 allowed)
		SupervisorID:  req.SupervisorId,
		RequestedBy:   req.RequestedBy,
		Message:       req.Message,
		ProposedTopic: req.ProposedTopic,
	})
	if err != nil {
		if st, ok := status.FromError(err); ok {
			return nil, st.Err()
		}
		h.logger.Error("CreateSupervisorRequest failed", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "failed to create supervisor request: %v", err)
	}

	return &adminv1.CreateSupervisorRequestResp{
		Success:   true,
		RequestId: result.ID,
		Message:   "Запрос успешно отправлен супервайзеру",
		Request:   h.convertSupervisorRequestToProto(result),
	}, nil
}

// Team-first (NEW)
func (h *Handler) CreateSupervisorRequestByTeam(ctx context.Context, req *adminv1.CreateSupervisorRequestByTeamReq) (*adminv1.CreateSupervisorRequestResp, error) {
	if req.TeamId == 0 {
		return nil, status.Error(codes.InvalidArgument, "team_id is required")
	}
	if req.SupervisorId == 0 {
		return nil, status.Error(codes.InvalidArgument, "supervisor_id is required")
	}
	if req.RequestedBy == 0 {
		return nil, status.Error(codes.InvalidArgument, "requested_by is required")
	}

	result, err := h.service.CreateSupervisorRequest(ctx, &CreateSupervisorRequestInput{
		ProjectID:     0,
		TeamID:        req.TeamId,
		SupervisorID:  req.SupervisorId,
		RequestedBy:   req.RequestedBy,
		Message:       req.Message,
		ProposedTopic: req.ProposedTopic,
	})
	if err != nil {
		if st, ok := status.FromError(err); ok {
			return nil, st.Err()
		}
		h.logger.Error("CreateSupervisorRequest failed", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "failed to create supervisor request: %v", err)
	}

	return &adminv1.CreateSupervisorRequestResp{
		Success:   true,
		RequestId: result.ID,
		Message:   "Запрос успешно отправлен супервайзеру",
		Request:   h.convertSupervisorRequestToProto(result),
	}, nil
}

func (h *Handler) ListSupervisorRequests(ctx context.Context, req *adminv1.ListSupervisorRequestsReq) (*adminv1.ListSupervisorRequestsResp, error) {
	pageSize := int(req.PageSize)
	if pageSize <= 0 {
		pageSize = 20
	}
	page := int(req.Page)
	if page <= 0 {
		page = 1
	}

	filter := SupervisorRequestFilter{
		DepartmentID: req.DepartmentId,
		SupervisorID: req.SupervisorId,
		TeamID:       req.TeamId,
		Status:       req.Status,
		Limit:        pageSize,
		Offset:       (page - 1) * pageSize,
	}

	requests, total, err := h.service.ListSupervisorRequests(ctx, filter)
	if err != nil {
		h.logger.Error("ListSupervisorRequests failed", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "failed to list supervisor requests: %v", err)
	}

	var pbRequests []*adminv1.SupervisorRequest
	for _, r := range requests {
		pbRequests = append(pbRequests, h.convertSupervisorRequestToProto(r))
	}

	return &adminv1.ListSupervisorRequestsResp{
		Requests:   pbRequests,
		TotalCount: total,
	}, nil
}

func (h *Handler) GetSupervisorRequest(ctx context.Context, req *adminv1.GetSupervisorRequestReq) (*adminv1.GetSupervisorRequestResp, error) {
	if req.RequestId == "" {
		return nil, status.Error(codes.InvalidArgument, "request_id is required")
	}

	supervisorReq, history, err := h.service.GetSupervisorRequest(ctx, req.RequestId)
	if err != nil {
		h.logger.Error("GetSupervisorRequest failed", zap.Error(err))
		return nil, status.Errorf(codes.NotFound, "supervisor request not found: %v", err)
	}

	var pbHistory []*adminv1.SupervisorRequestHistory
	for _, hist := range history {
		pbHistory = append(pbHistory, &adminv1.SupervisorRequestHistory{
			Id:        hist.ID,
			Action:    hist.Action,
			ActorId:   hist.ActorID,
			Comment:   hist.Comment,
			CreatedAt: timestamppb.New(hist.CreatedAt),
		})
	}

	return &adminv1.GetSupervisorRequestResp{
		Request: h.convertSupervisorRequestToProto(supervisorReq),
		History: pbHistory,
	}, nil
}

func (h *Handler) RespondToSupervisorRequest(ctx context.Context, req *adminv1.RespondToSupervisorRequestReq) (*adminv1.RespondToSupervisorRequestResp, error) {
	if req.RequestId == "" {
		return nil, status.Error(codes.InvalidArgument, "request_id is required")
	}
	if req.SupervisorId == 0 {
		return nil, status.Error(codes.InvalidArgument, "supervisor_id is required")
	}
	if req.Action == "" {
		return nil, status.Error(codes.InvalidArgument, "action is required")
	}
	if req.Action != "approve" && req.Action != "reject" {
		return nil, status.Error(codes.InvalidArgument, "action must be 'approve' or 'reject'")
	}
	if req.Action == "reject" && req.RejectReason == "" {
		return nil, status.Error(codes.InvalidArgument, "reject_reason is required when rejecting")
	}

	result, err := h.service.RespondToSupervisorRequest(ctx, &RespondToSupervisorRequestInput{
		RequestID:    req.RequestId,
		SupervisorID: req.SupervisorId,
		Action:       req.Action,
		RejectReason: req.RejectReason,
		Comment:      req.Comment,
	})
	if err != nil {
		h.logger.Error("RespondToSupervisorRequest failed", zap.Error(err))
		if _, ok := status.FromError(err); ok {
			return nil, err
		}
		return nil, status.Errorf(codes.Internal, "failed to respond to supervisor request: %v", err)
	}

	message := "Запрос одобрен"
	if req.Action == "reject" {
		message = "Запрос отклонён"
	}

	return &adminv1.RespondToSupervisorRequestResp{
		Success:        true,
		Message:        message,
		UpdatedRequest: h.convertSupervisorRequestToProto(result),
	}, nil
}

func (h *Handler) ListMySupervisorRequests(ctx context.Context, req *adminv1.ListMySupervisorRequestsReq) (*adminv1.ListMySupervisorRequestsResp, error) {
	if req.SupervisorId == 0 {
		return nil, status.Error(codes.InvalidArgument, "supervisor_id is required")
	}

	result, err := h.service.ListMySupervisorRequests(ctx, req.SupervisorId, req.Status, req.Page, req.PageSize)
	if err != nil {
		h.logger.Error("ListMySupervisorRequests failed", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "failed to list supervisor requests: %v", err)
	}

	// requests
	var pbRequests []*adminv1.SupervisorRequest
	for _, r := range result.Requests {
		pbRequests = append(pbRequests, h.convertSupervisorRequestToProto(r))
	}

	// teams_report
	var pbTeams []*adminv1.SupervisorTeamReport
	for _, t := range result.Teams {
		if t == nil {
			continue
		}
		rep := &adminv1.SupervisorTeamReport{
			TeamId:       t.ID,
			TeamName:     t.Name,
			MemberCount:  int32(len(t.Members)),
			ProjectId:    t.ProjectID,
			ProjectTitle: t.ProjectTitle,
			CurrentStep:  t.CurrentStep,
			Status:       t.Status,
		}
		for _, m := range t.Members {
			if m == nil {
				continue
			}
			rep.Members = append(rep.Members, &adminv1.TeamMemberPreview{
				UserId:   m.UserID,
				FullName: m.FullName,
				Email:    m.Email,
				Role:     m.Role,
			})
		}
		pbTeams = append(pbTeams, rep)
	}

	return &adminv1.ListMySupervisorRequestsResp{
		Requests:      pbRequests,
		TotalCount:    result.TotalCount,
		PendingCount:  result.PendingCount,
		ActiveTeams:   result.ActiveTeams,
		TotalStudents: result.TotalStudents,
		TeamsReport:   pbTeams,
	}, nil
}

func (h *Handler) CancelSupervisorRequest(ctx context.Context, req *adminv1.CancelSupervisorRequestReq) (*adminv1.CancelSupervisorRequestResp, error) {
	if req.RequestId == "" {
		return nil, status.Error(codes.InvalidArgument, "request_id is required")
	}
	if req.CancelledBy == 0 {
		return nil, status.Error(codes.InvalidArgument, "cancelled_by is required")
	}

	err := h.service.CancelSupervisorRequest(ctx, req.RequestId, req.CancelledBy, req.Reason)
	if err != nil {
		h.logger.Error("CancelSupervisorRequest failed", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "failed to cancel supervisor request: %v", err)
	}

	return &adminv1.CancelSupervisorRequestResp{
		Success: true,
		Message: "Запрос успешно отменён",
	}, nil
}

func (h *Handler) convertSupervisorRequestToProto(req *SupervisorRequestWithDetails) *adminv1.SupervisorRequest {
	if req == nil {
		return nil
	}

	pbReq := &adminv1.SupervisorRequest{
		Id:              req.ID,
		TeamId:          req.TeamID,
		TeamName:        req.TeamName,
		ProjectId:       derefInt64(req.ProjectID), // 0 if not created yet
		SupervisorId:    req.SupervisorID,
		SupervisorName:  req.SupervisorName,
		SupervisorEmail: req.SupervisorEmail,
		RequestedBy:     req.RequestedBy,
		RequesterName:   req.RequesterName,
		Status:          req.Status,
		Message:         req.Message,
		ProposedTopic:   req.ProposedTopic,
		RejectReason:    req.RejectReason,
		CreatedAt:       timestamppb.New(req.CreatedAt),
	}

	if req.RespondedAt != nil {
		pbReq.RespondedAt = timestamppb.New(*req.RespondedAt)
	}
	if req.ExpiresAt != nil {
		pbReq.ExpiresAt = timestamppb.New(*req.ExpiresAt)
	}

	return pbReq
}
func (h *Handler) ListAvailableTeams(ctx context.Context, req *adminv1.ListAvailableTeamsRequest) (*adminv1.ListAvailableTeamsResponse, error) {
	if req.DepartmentId == 0 {
		return nil, status.Error(codes.InvalidArgument, "department_id is required")
	}

	teams, total, err := h.service.ListAvailableTeams(ctx, req.DepartmentId, req.Page, req.PageSize)
	if err != nil {
		h.logger.Error("ListAvailableTeams failed", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "failed to list available teams: %v", err)
	}

	pb := make([]*adminv1.AvailableTeam, 0, len(teams))
	for _, t := range teams {
		if t == nil {
			continue
		}
		members := make([]*adminv1.TeamMemberInfo, 0, len(t.Members))
		for _, m := range t.Members {
			if m == nil {
				continue
			}
			members = append(members, &adminv1.TeamMemberInfo{
				UserId:   m.UserID,
				FullName: m.FullName,
				Email:    m.Email,
				Role:     m.Role,
			})
		}
		pb = append(pb, &adminv1.AvailableTeam{
			Id:          t.ID,
			Name:        t.Name,
			MemberCount: t.MemberCount,
			Members:     members,
			CreatedAt:   timestamppb.New(t.CreatedAt),
		})
	}

	return &adminv1.ListAvailableTeamsResponse{
		Teams:      pb,
		TotalCount: total,
	}, nil
}

func (h *Handler) SubmitDocument(ctx context.Context, req *adminv1.SubmitDocumentRequest) (*adminv1.SubmitDocumentResponse, error) {
	if req.ProjectId == 0 {
		return nil, status.Error(codes.InvalidArgument, "project_id is required")
	}
	if req.SubmittedBy == 0 {
		return nil, status.Error(codes.InvalidArgument, "submitted_by is required")
	}
	if len(req.FileIds) == 0 {
		return nil, status.Error(codes.InvalidArgument, "file_ids is required")
	}

	sub, err := h.service.SubmitDocumentForStep(ctx, &SubmitDocumentRequest{
		ProjectID:   req.ProjectId,
		SubmittedBy: req.SubmittedBy,
		FileIDs:     req.FileIds,
		Comment:     req.Comment,
	})
	if err != nil {
		if _, ok := status.FromError(err); ok {
			return nil, err
		}
		return nil, status.Errorf(codes.Internal, "failed to submit document: %v", err)
	}

	if _, err := h.service.EnsurePreDefenseSubmissionForSubmission(ctx, sub); err != nil {
		h.logger.Error("failed to ensure pre-defense submission after document submit", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "failed to create pre-defense submission: %v", err)
	}

	return &adminv1.SubmitDocumentResponse{
		Success:      true,
		SubmissionId: sub.ID,
		NewState:     "",
		Message:      "Document submitted successfully",
	}, nil
}

func (h *Handler) GetSupervisorSettings(ctx context.Context, req *adminv1.GetSupervisorSettingsRequest) (*adminv1.GetSupervisorSettingsResponse, error) {
	if req.SupervisorId <= 0 {
		return nil, status.Error(codes.InvalidArgument, "supervisor_id is required")
	}

	departmentID := req.DepartmentId
	if departmentID == 0 {
		departmentID = deptIDFromMD(ctx)
	}

	currentCount, err := h.service.repo.CountSupervisorTeams(ctx, req.SupervisorId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "count teams: %v", err)
	}

	maxTeams, isCustom, err := h.service.getEffectiveMaxTeams(ctx, req.SupervisorId, departmentID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "get max teams: %v", err)
	}

	rawMax := int32(0)
	if isCustom {
		rawMax = maxTeams
	}

	return &adminv1.GetSupervisorSettingsResponse{
		SupervisorId:      req.SupervisorId,
		MaxTeams:          rawMax,
		CurrentTeams:      currentCount,
		EffectiveMaxTeams: maxTeams,
		IsCustom:          isCustom,
	}, nil
}

func (h *Handler) UpdateSupervisorMaxTeams(ctx context.Context, req *adminv1.UpdateSupervisorMaxTeamsRequest) (*adminv1.UpdateSupervisorMaxTeamsResponse, error) {
	if req.SupervisorId <= 0 || req.DepartmentId <= 0 {
		return nil, status.Error(codes.InvalidArgument, "supervisor_id and department_id are required")
	}

	actorID := getActorIDFromContext(ctx)

	s := &SupervisorSettings{
		UserID:       req.SupervisorId,
		DepartmentID: req.DepartmentId,
		MaxTeams:     req.MaxTeams,
		UpdatedBy:    &actorID,
	}

	if err := h.service.repo.UpsertSupervisorSettings(ctx, s); err != nil {
		return nil, status.Errorf(codes.Internal, "upsert: %v", err)
	}

	return &adminv1.UpdateSupervisorMaxTeamsResponse{
		Success:     true,
		Message:     fmt.Sprintf("Max teams set to %d", req.MaxTeams),
		NewMaxTeams: req.MaxTeams,
	}, nil
}

// ==================== Teams ====================

func (h *Handler) CreateTeamAdmin(ctx context.Context, req *adminv1.CreateTeamAdminRequest) (*adminv1.CreateTeamAdminResponse, error) {
	if req.GetName() == "" {
		return nil, status.Error(codes.InvalidArgument, "name is required")
	}
	if req.GetUniversityId() <= 0 || req.GetDepartmentId() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "university_id and department_id are required")
	}
	if req.GetLeaderId() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "leader_id is required")
	}

	id, err := h.service.CreateTeamAdmin(ctx, req.Name, req.UniversityId, req.DepartmentId, req.LeaderId, req.MemberIds)
	if err != nil {
		return nil, err
	}
	return &adminv1.CreateTeamAdminResponse{TeamId: id}, nil
}

// ==================== Projects ====================

func (h *Handler) ListProjectsAdmin(ctx context.Context, req *adminv1.ListProjectsAdminRequest) (*adminv1.ListProjectsAdminResponse, error) {
	page := int(req.GetPage())
	if page <= 0 {
		page = 1
	}
	pageSize := int(req.GetPageSize())
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 200 {
		pageSize = 200
	}

	deptID := req.GetDepartmentId()
	if deptID == 0 {
		deptID = deptIDFromMD(ctx)
	}
	if deptID == 0 {
		return nil, status.Error(codes.InvalidArgument, "department_id is required (in request or metadata)")
	}

	rows, total, err := h.service.ListProjectsAdmin(ctx, ProjectsAdminFilter{
		DepartmentID: deptID,
		TeamID:       req.GetTeamId(),
		StudentID:    req.GetStudentId(),
		Status:       req.GetStatus(),
		Q:            req.GetQ(),
		Limit:        pageSize,
		Offset:       (page - 1) * pageSize,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list projects failed: %v", err)
	}

	out := &adminv1.ListProjectsAdminResponse{TotalCount: total}
	for _, r := range rows {
		if r == nil {
			continue
		}
		item := &adminv1.ProjectAdminListItem{
			ProjectId:        r.ProjectID,
			Title:            r.Title,
			Description:      r.Description,
			StudentId:        r.StudentID,
			TeamId:           r.TeamID,
			UniversityId:     r.UniversityID,
			DepartmentId:     r.DepartmentID,
			WorkflowName:     r.WorkflowName,
			CurrentStateName: r.CurrentStateName,
			Status:           r.Status,
			CreatedAt:        timestamppb.New(r.CreatedAt),
			UpdatedAt:        timestamppb.New(r.UpdatedAt),
		}
		if r.DeadlineAt != nil {
			item.DeadlineAt = timestamppb.New(*r.DeadlineAt)
		}
		if r.TopicRegisteredAt != nil {
			item.TopicRegisteredAt = timestamppb.New(*r.TopicRegisteredAt)
		}
		out.Projects = append(out.Projects, item)
	}

	return out, nil
}

func (h *Handler) GetProjectAdmin(ctx context.Context, req *adminv1.GetProjectAdminRequest) (*adminv1.GetProjectAdminResponse, error) {
	if req.GetProjectId() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "project_id is required")
	}

	p, rt, err := h.service.GetProjectAdmin(ctx, req.ProjectId)
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, status.Error(codes.NotFound, "project not found")
	}

	d := &adminv1.ProjectAdminDetails{
		ProjectId:         p.ProjectId,
		Title:             p.Title,
		Description:       p.Description,
		StudentId:         p.StudentId,
		TeamId:            p.TeamId,
		WorkflowName:      p.WorkflowName,
		CurrentState:      p.CurrentState,
		CurrentStateName:  p.CurrentState, // fallback
		Status:            p.Status,
		TopicRegisteredAt: p.TopicRegisteredAt,
	}
	// prefer runtime values if available
	if rt != nil {
		d.UniversityId = rt.UniversityId
		d.DepartmentId = rt.DepartmentId
		d.TeamId = rt.TeamId
		d.StudentId = rt.StudentId
		d.WorkflowName = rt.WorkflowName
		d.CurrentStateName = rt.CurrentStateName
		d.Status = rt.Status
		d.Data = rt.Data
		d.DeadlineAt = rt.DeadlineAt
		if rt.TopicRegisteredAt != nil {
			d.TopicRegisteredAt = rt.TopicRegisteredAt
		}
	}

	return &adminv1.GetProjectAdminResponse{Project: d}, nil
}

func (h *Handler) CreateProjectAdmin(ctx context.Context, req *adminv1.CreateProjectAdminRequest) (*adminv1.CreateProjectAdminResponse, error) {
	if req.GetTitle() == "" {
		return nil, status.Error(codes.InvalidArgument, "title is required")
	}
	if req.GetStudentId() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "student_id is required")
	}
	if req.GetWorkflowName() == "" {
		return nil, status.Error(codes.InvalidArgument, "workflow_name is required")
	}

	// FIX: team-first (команда обязательна)
	if req.GetTeamId() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "team_id is required")
	}

	// FIX: university_id / department_id могут быть 0 — service подтянет из TeamContext [[12]]
	resp, err := h.service.CreateProjectAdmin(ctx, &projectv1.CreateProjectRequest{
		Title:        req.Title,
		Description:  req.Description,
		StudentId:    req.StudentId,
		WorkflowName: req.WorkflowName,
		UniversityId: req.UniversityId,
		DepartmentId: req.DepartmentId,
		TeamId:       req.TeamId,
	})
	if err != nil {
		return nil, err
	}
	return &adminv1.CreateProjectAdminResponse{ProjectId: resp.ProjectId, Status: resp.Status}, nil
}

func (h *Handler) UpdateProjectAdmin(ctx context.Context, req *adminv1.UpdateProjectAdminRequest) (*adminv1.UpdateProjectAdminResponse, error) {
	if req.GetProjectId() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "project_id is required")
	}

	_, err := h.service.UpdateProjectAdmin(ctx, &projectv1.UpdateProjectRequest{
		ProjectId:   req.ProjectId,
		Title:       req.Title,
		Description: req.Description,
		StudentId:   req.StudentId,
		TeamId:      req.TeamId,
		DataPatch:   req.DataPatch,
	})
	if err != nil {
		return nil, err
	}
	return &adminv1.UpdateProjectAdminResponse{Success: true}, nil
}

func (h *Handler) ArchiveProjectAdmin(ctx context.Context, req *adminv1.ArchiveProjectAdminRequest) (*adminv1.ArchiveProjectAdminResponse, error) {
	if req.GetProjectId() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "project_id is required")
	}
	resp, err := h.service.ArchiveProjectAdmin(ctx, req.ProjectId, req.Reason)
	if err != nil {
		return nil, err
	}
	return &adminv1.ArchiveProjectAdminResponse{Success: resp.Success, Status: resp.Status}, nil
}

func (h *Handler) DeleteArchivedProjectAdmin(ctx context.Context, req *adminv1.DeleteArchivedProjectAdminRequest) (*adminv1.DeleteArchivedProjectAdminResponse, error) {
	if req.GetProjectId() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "project_id is required")
	}
	_, err := h.service.DeleteArchivedProjectAdmin(ctx, req.ProjectId, req.Reason)
	if err != nil {
		return nil, err
	}
	return &adminv1.DeleteArchivedProjectAdminResponse{Success: true}, nil
}
