package admin

import (
	"context"
	"fmt"

	adminv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/admin/v1"
	projectv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/project/v1"
	teamv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/team/v1"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// ==================== Pre-Defense gRPC Handlers ====================

func (h *Handler) SubmitPreDefense(ctx context.Context, req *adminv1.SubmitPreDefenseRequest) (*adminv1.SubmitPreDefenseResponse, error) {
	if req.TeamId == 0 || req.ProjectId == 0 || req.SubmittedBy == 0 {
		return nil, status.Error(codes.InvalidArgument, "team_id, project_id, and submitted_by are required")
	}

	id, sub, err := h.service.SubmitPreDefense(ctx, req.TeamId, req.ProjectId, req.SubmittedBy, req.Message, req.DocumentIds)
	if err != nil {
		h.logger.Error("SubmitPreDefense failed", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "failed to submit pre-defense: %v", err)
	}

	return &adminv1.SubmitPreDefenseResponse{
		Success:      true,
		SubmissionId: id,
		Message:      "Pre-defense submission created successfully",
		Submission:   h.convertPreDefenseToProto(sub),
	}, nil
}

func (h *Handler) ListPreDefenseSubmissions(ctx context.Context, req *adminv1.ListPreDefenseSubmissionsRequest) (*adminv1.ListPreDefenseSubmissionsResponse, error) {
	pageSize := int(req.PageSize)
	if pageSize <= 0 {
		pageSize = 20
	}
	page := int(req.Page)
	if page <= 0 {
		page = 1
	}

	filter := PreDefenseFilter{
		DepartmentID: req.DepartmentId,
		SupervisorID: req.SupervisorId,
		TeamID:       req.TeamId,
		Status:       req.Status,
		Limit:        pageSize,
		Offset:       (page - 1) * pageSize,
	}
	if req.DateFrom != nil {
		t := req.DateFrom.AsTime()
		filter.DateFrom = &t
	}
	if req.DateTo != nil {
		t := req.DateTo.AsTime()
		filter.DateTo = &t
	}

	subs, total, err := h.service.ListPreDefenseSubmissions(ctx, filter)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list pre-defense submissions: %v", err)
	}

	var pbSubs []*adminv1.PreDefenseSubmission
	for _, s := range subs {
		pbSubs = append(pbSubs, h.convertPreDefenseToProto(s))
	}

	return &adminv1.ListPreDefenseSubmissionsResponse{
		Submissions: pbSubs,
		TotalCount:  total,
	}, nil
}

func (h *Handler) GetPreDefenseSubmission(ctx context.Context, req *adminv1.GetPreDefenseSubmissionRequest) (*adminv1.GetPreDefenseSubmissionResponse, error) {
	if req.SubmissionId == "" {
		return nil, status.Error(codes.InvalidArgument, "submission_id is required")
	}

	sub, history, err := h.service.GetPreDefenseSubmission(ctx, req.SubmissionId)
	if err != nil {
		h.logger.Error("GetPreDefenseSubmission failed", zap.Error(err))
		return nil, status.Errorf(codes.NotFound, "submission not found: %v", err)
	}

	var pbHistory []*adminv1.PreDefenseHistory
	for _, hist := range history {
		pbHistory = append(pbHistory, &adminv1.PreDefenseHistory{
			Id:        hist.ID,
			Action:    hist.Action,
			ActorId:   hist.ActorID,
			OldValue:  hist.OldValue,
			NewValue:  hist.NewValue,
			Comment:   hist.Comment,
			CreatedAt: timestamppb.New(hist.CreatedAt),
		})
	}

	return &adminv1.GetPreDefenseSubmissionResponse{
		Submission: h.convertPreDefenseToProto(sub),
		History:    pbHistory,
	}, nil
}
func (h *Handler) SchedulePreDefense(ctx context.Context, req *adminv1.SchedulePreDefenseRequest) (*adminv1.SchedulePreDefenseResponse, error) {
	if req.SubmissionId == "" || req.ScheduledBy == 0 {
		return nil, status.Error(codes.InvalidArgument, "submission_id and scheduled_by are required")
	}
	if req.ScheduledDate == nil {
		return nil, status.Error(codes.InvalidArgument, "scheduled_date is required")
	}

	scheduledDate := req.ScheduledDate.AsTime()

	err := h.service.SchedulePreDefense(ctx,
		req.SubmissionId,
		scheduledDate,
		req.ScheduledTime,
		req.Location,
		req.MeetingLink,
		req.DurationMinutes,
		req.CommissionMemberIds,
		req.ScheduledBy,
		req.Comment,
	)
	if err != nil {
		h.logger.Error("SchedulePreDefense failed", zap.Error(err))

		if st, ok := status.FromError(err); ok {
			return nil, st.Err()
		}

		return nil, status.Errorf(codes.Internal, "failed to schedule pre-defense: %v", err)
	}

	return &adminv1.SchedulePreDefenseResponse{
		Success: true,
		Message: "Pre-defense scheduled successfully",
	}, nil
}
func (h *Handler) GradePreDefense(ctx context.Context, req *adminv1.GradePreDefenseRequest) (*adminv1.GradePreDefenseResponse, error) {
	if req.SubmissionId == "" || req.GradedBy == 0 {
		return nil, status.Error(codes.InvalidArgument, "submission_id and graded_by are required")
	}
	if req.Grade < 0 || req.Grade > 100 {
		return nil, status.Error(codes.InvalidArgument, "grade must be between 0 and 100")
	}

	var memberGrades []MemberGradeInput
	for _, mg := range req.MemberGrades {
		memberGrades = append(memberGrades, MemberGradeInput{
			MemberID: mg.MemberId,
			Grade:    mg.Grade,
			Comment:  mg.Comment,
		})
	}

	err := h.service.GradePreDefense(ctx,
		req.SubmissionId,
		req.GradedBy,
		req.Grade,
		req.Comment,
		memberGrades,
	)
	if err != nil {
		h.logger.Error("GradePreDefense failed", zap.Error(err))

		if st, ok := status.FromError(err); ok {
			return nil, st.Err()
		}

		return nil, status.Errorf(codes.Internal, "failed to grade pre-defense: %v", err)
	}

	return &adminv1.GradePreDefenseResponse{
		Success: true,
		Message: "Pre-defense graded successfully",
	}, nil
}

func (h *Handler) CompletePreDefense(ctx context.Context, req *adminv1.CompletePreDefenseRequest) (*adminv1.CompletePreDefenseResponse, error) {
	if req.SubmissionId == "" || req.CompletedBy == 0 {
		return nil, status.Error(codes.InvalidArgument, "submission_id and completed_by are required")
	}
	if req.Result == "" {
		return nil, status.Error(codes.InvalidArgument, "result is required (passed, failed, conditional)")
	}

	err := h.service.CompletePreDefense(ctx,
		req.SubmissionId,
		req.CompletedBy,
		req.Result,
		req.ResultComment,
		req.Recommendations,
		req.AllowResubmission,
	)
	if err != nil {
		h.logger.Error("CompletePreDefense failed", zap.Error(err))
		if st, ok := status.FromError(err); ok {
			return nil, st.Err()
		}
		return nil, status.Errorf(codes.Internal, "failed to complete pre-defense: %v", err)
	}

	return &adminv1.CompletePreDefenseResponse{
		Success: true,
		Message: fmt.Sprintf("Pre-defense completed with result: %s", req.Result),
	}, nil
}
func (h *Handler) ReschedulePreDefense(ctx context.Context, req *adminv1.ReschedulePreDefenseRequest) (*adminv1.ReschedulePreDefenseResponse, error) {
	if req.SubmissionId == "" || req.RescheduledBy == 0 {
		return nil, status.Error(codes.InvalidArgument, "submission_id and rescheduled_by are required")
	}
	if req.NewDate == nil {
		return nil, status.Error(codes.InvalidArgument, "new_date is required")
	}

	newDate := req.NewDate.AsTime()

	err := h.service.ReschedulePreDefense(ctx,
		req.SubmissionId,
		req.RescheduledBy,
		newDate,
		req.NewTime,
		req.NewLocation,
		req.Reason,
	)
	if err != nil {
		h.logger.Error("ReschedulePreDefense failed", zap.Error(err))

		if st, ok := status.FromError(err); ok {
			return nil, st.Err()
		}

		return nil, status.Errorf(codes.Internal, "failed to reschedule pre-defense: %v", err)
	}

	return &adminv1.ReschedulePreDefenseResponse{
		Success: true,
		Message: "Pre-defense rescheduled successfully",
	}, nil
}

func (h *Handler) ListScheduledPreDefenses(ctx context.Context, req *adminv1.ListScheduledPreDefensesRequest) (*adminv1.ListScheduledPreDefensesResponse, error) {
	filter := ScheduleFilter{
		CommissionMemberID: req.CommissionMemberId,
		DepartmentID:       req.DepartmentId,
		Location:           req.Location,
	}
	if req.DateFrom != nil {
		t := req.DateFrom.AsTime()
		filter.DateFrom = &t
	}
	if req.DateTo != nil {
		t := req.DateTo.AsTime()
		filter.DateTo = &t
	}

	subs, err := h.service.ListScheduledPreDefenses(ctx, filter)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list scheduled pre-defenses: %v", err)
	}

	var pbItems []*adminv1.PreDefenseScheduleItem
	for _, s := range subs {
		item := &adminv1.PreDefenseScheduleItem{
			SubmissionId:    s.ID,
			TeamId:          s.TeamID,
			Status:          s.Status,
			ScheduledTime:   s.ScheduledTime,
			Location:        s.Location,
			DurationMinutes: s.DurationMinutes,
		}
		if s.ScheduledDate != nil {
			item.ScheduledDate = timestamppb.New(*s.ScheduledDate)
		}
		pbItems = append(pbItems, item)
	}

	return &adminv1.ListScheduledPreDefensesResponse{
		Schedule:   pbItems,
		TotalCount: int32(len(pbItems)),
	}, nil
}

func (h *Handler) AddPreDefenseCommissionMember(ctx context.Context, req *adminv1.AddPreDefenseCommissionMemberRequest) (*adminv1.AddPreDefenseCommissionMemberResponse, error) {
	if req.SubmissionId == "" || req.UserId == 0 {
		return nil, status.Error(codes.InvalidArgument, "submission_id and user_id are required")
	}

	role := req.Role
	if role == "" {
		role = "member"
	}

	member, err := h.service.AddPreDefenseCommissionMember(ctx, req.SubmissionId, req.UserId, role, req.AddedBy)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to add commission member: %v", err)
	}

	return &adminv1.AddPreDefenseCommissionMemberResponse{
		Success: true,
		Member: &adminv1.PreDefenseCommissionMember{
			UserId: member.UserID,
			Role:   member.Role,
		},
	}, nil

}

func (h *Handler) RemovePreDefenseCommissionMember(ctx context.Context, req *adminv1.RemovePreDefenseCommissionMemberRequest) (*adminv1.RemovePreDefenseCommissionMemberResponse, error) {
	if req.SubmissionId == "" || req.UserId == 0 {
		return nil, status.Error(codes.InvalidArgument, "submission_id and user_id are required")
	}

	err := h.service.RemovePreDefenseCommissionMember(ctx, req.SubmissionId, req.UserId, req.RemovedBy, req.Reason)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to remove commission member: %v", err)
	}

	return &adminv1.RemovePreDefenseCommissionMemberResponse{
		Success: true,
	}, nil
}

// ==================== Converter ====================

func (h *Handler) convertPreDefenseToProto(sub *PreDefenseSubmission) *adminv1.PreDefenseSubmission {
	if sub == nil {
		return nil
	}

	pb := &adminv1.PreDefenseSubmission{
		Id:              sub.ID,
		TeamId:          sub.TeamID,
		ProjectId:       sub.ProjectID,
		SubmittedBy:     sub.SubmittedBy,
		Status:          sub.Status,
		Location:        sub.Location,
		MeetingLink:     sub.MeetingLink,
		ScheduledTime:   sub.ScheduledTime,
		DurationMinutes: sub.DurationMinutes,
		GradeComment:    sub.GradeComment,
		Result:          sub.Result,
		ResultComment:   sub.ResultComment,
		TeamName:        sub.TeamName,
		ProjectTitle:    sub.ProjectTitle,
		SupervisorName:  sub.SupervisorName,
		SubmitterName:   sub.SubmitterName,
		GraderName:      sub.GraderName,
		CreatedAt:       timestamppb.New(sub.CreatedAt),
		UpdatedAt:       timestamppb.New(sub.UpdatedAt),
	}

	if !sub.SubmittedAt.IsZero() {
		pb.SubmittedAt = timestamppb.New(sub.SubmittedAt)
	}

	if sub.SupervisorID != nil {
		pb.SupervisorId = *sub.SupervisorID
	}
	if sub.ScheduledDate != nil {
		pb.ScheduledDate = timestamppb.New(*sub.ScheduledDate)
	}
	if sub.Grade != nil {
		pb.Grade = *sub.Grade
	}
	if sub.GradedBy != nil {
		pb.GradedBy = *sub.GradedBy
	}
	if sub.GradedAt != nil {
		pb.GradedAt = timestamppb.New(*sub.GradedAt)
	}
	if sub.CompletedAt != nil {
		pb.CompletedAt = timestamppb.New(*sub.CompletedAt)
	}

	for _, m := range sub.Commission {
		pb.Commission = append(pb.Commission, &adminv1.PreDefenseCommissionMember{
			UserId:          m.UserID,
			Role:            m.Role,
			IsPresent:       m.IsPresent,
			IndividualGrade: m.IndividualGrade,
			Comment:         m.Comment,
		})
	}

	// Documents
	for _, d := range sub.Documents {
		pb.Documents = append(pb.Documents, &adminv1.PreDefenseDocument{
			Id:       d.ID,
			FileName: d.FileName,
		})
	}

	return pb
}

func (s *Service) enrichPreDefenseNames(ctx context.Context, sub *PreDefenseSubmission) {
	if sub == nil {
		return
	}

	if sub.TeamID > 0 {
		teamResp, err := s.teamClient.GetTeam(s.internalCtx(ctx), &teamv1.GetTeamRequest{TeamId: sub.TeamID})
		if err == nil && teamResp != nil {
			sub.TeamName = teamResp.Name
		}
	}

	if sub.ProjectID > 0 {
		rt, err := s.projectClient.GetProjectRuntime(s.internalCtx(ctx), &projectv1.GetProjectRuntimeRequest{
			ProjectId: sub.ProjectID,
		})
		if err == nil && rt != nil && rt.Data != nil {
			if title, ok := rt.Data.AsMap()["title"].(string); ok {
				sub.ProjectTitle = title
			}
		}
	}

	userIDs := []int64{sub.SubmittedBy}
	if sub.SupervisorID != nil {
		userIDs = append(userIDs, *sub.SupervisorID)
	}
	if sub.GradedBy != nil {
		userIDs = append(userIDs, *sub.GradedBy)
	}

	type nameRow struct {
		ID       int64  `gorm:"column:id"`
		FullName string `gorm:"column:full_name"`
	}

	// Avoid panic on repo type assertion; also satisfy errcheck by checking Scan().Error
	r, ok := s.repo.(*repository)
	if !ok || r == nil || r.db == nil {
		s.logger.Warn("enrichPreDefenseNames: repo is not *repository or db is nil (skip DB enrichment)")
	} else {
		var names []nameRow
		err := r.db.WithContext(ctx).Raw(
			`SELECT id, CONCAT(first_name, ' ', last_name) as full_name
			 FROM users WHERE id IN ?`, userIDs,
		).Scan(&names).Error
		if err != nil {
			s.logger.Warn("enrichPreDefenseNames: failed to load user names",
				zap.Error(err),
				zap.Int64s("user_ids", userIDs),
			)
			// continue (best-effort)
		} else {
			nameMap := make(map[int64]string, len(names))
			for _, n := range names {
				nameMap[n.ID] = n.FullName
			}

			sub.SubmitterName = nameMap[sub.SubmittedBy]
			if sub.SupervisorID != nil {
				sub.SupervisorName = nameMap[*sub.SupervisorID]
			}
			if sub.GradedBy != nil {
				sub.GraderName = nameMap[*sub.GradedBy]
			}
		}
	}
}
