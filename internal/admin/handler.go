package admin

import (
	"context"
	"encoding/json"
	"time"

	adminv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/admin/v1"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Handler struct {
	adminv1.UnimplementedAdminServiceServer
	service *Service
	logger  *zap.Logger
}

func NewHandler(service *Service, logger *zap.Logger) *Handler {
	return &Handler{
		service: service,
		logger:  logger,
	}
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
			TotalStudents:     resp.Stats.TotalStudents,
			TotalTeams:        resp.Stats.TotalTeams,
			TotalProjects:     resp.Stats.TotalProjects,
			CompletedProjects: resp.Stats.CompletedProjects,
			PendingReviews:    resp.Stats.PendingReviews,
			ActiveSupervisors: resp.Stats.ActiveSupervisors,
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
			TotalStudents:     resp.Stats.TotalStudents,
			TotalTeams:        resp.Stats.TotalTeams,
			TotalProjects:     resp.Stats.TotalProjects,
			CompletedProjects: resp.Stats.CompletedProjects,
			PendingReviews:    resp.Stats.PendingReviews,
			ActiveSupervisors: resp.Stats.ActiveSupervisors,
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
	// Implementation
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

// ==================== Teams ====================

func (h *Handler) ListAllTeams(ctx context.Context, req *adminv1.ListAllTeamsRequest) (*adminv1.ListAllTeamsResponse, error) {
	teams, total, err := h.service.ListAllTeams(ctx, &ListAllTeamsRequest{
		DepartmentID: req.DepartmentId,
		ProjectID:    req.ProjectId,
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
			ProjectId:    t.ProjectID,
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
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (h *Handler) UpdateTeamAdmin(ctx context.Context, req *adminv1.UpdateTeamAdminRequest) (*adminv1.UpdateTeamAdminResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (h *Handler) DeleteTeamAdmin(ctx context.Context, req *adminv1.DeleteTeamAdminRequest) (*emptypb.Empty, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
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
			FullName:   s.FullName,
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

	// TODO: Get actual actor ID from context
	actorID := int64(1)

	err := h.service.AssignSupervisor(ctx, req.TeamId, req.SupervisorId, actorID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to assign supervisor: %v", err)
	}

	return &adminv1.AssignSupervisorResponse{
		Success: true,
		Message: "Supervisor assigned successfully",
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
		Status:       req.Status,
		Limit:        pageSize,
		Offset:       (page - 1) * pageSize,
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
			Id:          g.ID,
			ProjectId:   g.ProjectID,
			StepId:      g.StepID,
			Grade:       g.Grade,
			LetterGrade: g.LetterGrade,
			GradedBy:    g.GradedBy,
			Comment:     g.Comment,
			GradedAt:    timestamppb.New(g.CreatedAt),
		})
	}

	return &adminv1.GetProjectGradesResponse{
		ProjectId:        req.ProjectId,
		StepGrades:       pbGrades,
		AverageGrade:     avg,
		FinalLetterGrade: CalculateLetterGrade(int32(avg)),
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
			Id:          grade.ID,
			ProjectId:   grade.ProjectID,
			StepId:      grade.StepID,
			Grade:       grade.Grade,
			LetterGrade: grade.LetterGrade,
			GradedBy:    grade.GradedBy,
			Comment:     grade.Comment,
			GradedAt:    timestamppb.New(grade.CreatedAt),
		},
	}, nil
}

func (h *Handler) GetGradingHistory(ctx context.Context, req *adminv1.GetGradingHistoryRequest) (*adminv1.GetGradingHistoryResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
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
