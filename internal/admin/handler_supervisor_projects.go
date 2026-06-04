package admin

import (
	"context"
	"encoding/json"
	"fmt"

	adminv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/admin/v1"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// gRPC handlers for the supervisor-facing project module. Each method takes
// supervisor_id from the request (set by the gateway from the JWT) and enforces
// ownership via requireSupervisorProjectAccess before returning any data.

const supervisorMaxListPageSize = 100

// ==================== Access check ====================

func (h *Handler) CheckSupervisorProjectAccess(ctx context.Context, req *adminv1.CheckSupervisorProjectAccessRequest) (*adminv1.CheckSupervisorProjectAccessResponse, error) {
	if req.SupervisorId <= 0 || req.ProjectId <= 0 {
		return nil, status.Error(codes.InvalidArgument, "supervisor_id and project_id are required")
	}
	exists, owned, err := h.service.repo.IsSupervisorOfProject(ctx, req.SupervisorId, req.ProjectId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "access check failed: %v", err)
	}
	return &adminv1.CheckSupervisorProjectAccessResponse{Exists: exists, HasAccess: owned}, nil
}

// ==================== List projects ====================

func (h *Handler) ListSupervisorProjects(ctx context.Context, req *adminv1.ListSupervisorProjectsRequest) (*adminv1.ListSupervisorProjectsResponse, error) {
	if req.SupervisorId <= 0 {
		return nil, status.Error(codes.InvalidArgument, "supervisor_id is required")
	}

	page := req.Page
	if page <= 0 {
		page = 1
	}
	pageSize := req.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > supervisorMaxListPageSize {
		pageSize = supervisorMaxListPageSize
	}

	f := SupervisorProjectsFilter{
		SupervisorID: req.SupervisorId,
		Status:       req.Status,
		CurrentState: req.CurrentState,
		Search:       req.Search,
		Limit:        int(pageSize),
		Offset:       int((page - 1) * pageSize),
		Sort:         req.Sort,
		Order:        req.Order,
	}

	entries, total, err := h.service.ListSupervisorProjects(ctx, f)
	if err != nil {
		h.logger.Error("ListSupervisorProjects failed", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "failed to list supervisor projects: %v", err)
	}

	resp := &adminv1.ListSupervisorProjectsResponse{
		TotalCount: total,
		Page:       page,
		PageSize:   pageSize,
	}
	for _, e := range entries {
		resp.Projects = append(resp.Projects, toPbSupervisorProjectListItem(e))
	}
	return resp, nil
}

func toPbSupervisorProjectListItem(e *SupervisorProjectListEntry) *adminv1.SupervisorProjectListItem {
	r := e.Row
	item := &adminv1.SupervisorProjectListItem{
		Id:                      r.ProjectID,
		Title:                   r.Title,
		Description:             r.Description,
		Status:                  r.Status,
		CurrentStateId:          r.CurrentStateID,
		CurrentStateName:        r.CurrentStateName,
		CurrentStateDisplayName: r.CurrentDisplay,
		ProgressPercent:         computeProgressPercent(r.CurrentOrder, r.TotalSteps),
		CreatedAt:               timestamppb.New(r.CreatedAt),
		UpdatedAt:               timestamppb.New(r.UpdatedAt),
		Stats: &adminv1.SupervisorProjectStats{
			SubmissionsCount:    r.SubmissionsCount,
			PendingReviewsCount: r.PendingReviewsCount,
			FilesCount:          r.FilesCount,
			GradesCount:         r.GradesCount,
		},
	}
	if r.LastActivityAt != nil {
		item.Stats.LastActivityAt = timestamppb.New(*r.LastActivityAt)
	}
	if r.TeamID > 0 {
		item.Team = &adminv1.TeamAdminInfo{
			Id:          r.TeamID,
			Name:        r.TeamName,
			MemberCount: int32(len(e.Members)),
			Members:     toPbDPMembers(e.Members),
		}
	}
	item.Supervisor = toPbDPSupervisor(e.Supervisor)
	return item
}

// ==================== Project details ====================

func (h *Handler) GetSupervisorProjectDetails(ctx context.Context, req *adminv1.GetSupervisorProjectDetailsRequest) (*adminv1.GetSupervisorProjectDetailsResponse, error) {
	if err := h.service.requireSupervisorProjectAccess(ctx, req.SupervisorId, req.ProjectId, req.CallerRole); err != nil {
		return nil, err
	}

	d, err := h.service.GetSupervisorProjectDetails(ctx, req.ProjectId)
	if err != nil {
		if _, ok := status.FromError(err); ok {
			return nil, err
		}
		h.logger.Error("GetSupervisorProjectDetails failed", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "failed to get project details: %v", err)
	}

	head := d.Head
	resp := &adminv1.GetSupervisorProjectDetailsResponse{
		Project: &adminv1.SupervisorProjectDetail{
			Id:                      head.ProjectID,
			Title:                   head.Title,
			Description:             head.Description,
			Status:                  head.Status,
			CurrentStateId:          head.CurrentStateID,
			CurrentStateName:        head.CurrentStateName,
			CurrentStateDisplayName: head.CurrentDisplay,
			CreatedAt:               timestamppb.New(head.CreatedAt),
			UpdatedAt:               timestamppb.New(head.UpdatedAt),
		},
		Team: &adminv1.TeamAdminInfo{
			Id:          head.TeamID,
			Name:        head.TeamName,
			Status:      head.TeamStatus,
			MemberCount: int32(len(d.Members)),
			Members:     toPbDPMembers(d.Members),
			Supervisor:  toPbDPSupervisor(d.Supervisor),
			CreatedAt:   timestamppb.New(head.TeamCreatedAt),
			UpdatedAt:   timestamppb.New(head.TeamUpdatedAt),
		},
		Workflow: &adminv1.SupervisorProjectWorkflow{
			WorkflowId:       head.WorkflowID,
			WorkflowName:     head.WorkflowName,
			CurrentStateId:   head.CurrentStateID,
			CurrentStateName: head.CurrentStateName,
			ProgressPercent:  d.Progress,
		},
		Permissions: &adminv1.SupervisorProjectPermissions{
			CanReviewSubmissions: true,
			CanGrade:             true,
			CanDownloadFiles:     true,
			CanChangeWorkflow:    false,
			CanFinalizeStage:     false,
		},
	}
	if head.TopicRegisteredAt != nil {
		resp.Project.TopicRegisteredAt = timestamppb.New(*head.TopicRegisteredAt)
	}
	if d.Topic != nil {
		resp.Project.TopicKz = d.Topic.ProposedTopicKZ
		resp.Project.TopicRu = d.Topic.ProposedTopicRU
		resp.Project.TopicEn = d.Topic.ProposedTopicEN
	}

	for _, st := range d.Steps {
		resp.Workflow.Steps = append(resp.Workflow.Steps, toPbStepStatus(st))
	}

	// Submitter + grader names (batched).
	nameIDs := make([]int64, 0, len(d.Submissions)+len(d.Grades))
	for _, s := range d.Submissions {
		if s.SubmittedBy > 0 {
			nameIDs = append(nameIDs, s.SubmittedBy)
		}
	}
	for _, g := range d.Grades {
		if g.GradedBy > 0 {
			nameIDs = append(nameIDs, g.GradedBy)
		}
	}
	names := h.service.ResolveUserNames(ctx, nameIDs)

	for _, s := range d.Submissions {
		resp.Submissions = append(resp.Submissions, toPbSupervisorSubmission(s, names[s.SubmittedBy]))
	}
	for _, g := range d.Grades {
		resp.Grades = append(resp.Grades, &adminv1.GradeInfo{
			Id:         g.ID,
			ProjectId:  g.ProjectID,
			StepId:     g.StepID,
			Grade:      g.Grade,
			GradedBy:   g.GradedBy,
			GraderName: names[g.GradedBy],
			Comment:    g.Comment,
			GradedAt:   timestamppb.New(g.CreatedAt),
		})
	}
	for _, hi := range d.History {
		resp.History = append(resp.History, toPbUnifiedHistory(hi))
	}

	return resp, nil
}

func toPbStepStatus(st *DPStep) *adminv1.StepStatus {
	pb := &adminv1.StepStatus{
		StepId:           st.StepID,
		StepName:         st.StepName,
		DisplayName:      st.DisplayName,
		StepType:         st.StepType,
		OrderIndex:       st.OrderIndex,
		Status:           st.Status,
		SubmissionStatus: st.SubmissionState,
	}
	if st.ReviewedAt != nil {
		pb.CompletedAt = timestamppb.New(*st.ReviewedAt)
	}
	if st.Grade != nil {
		grade := &adminv1.GradeInfo{
			StepId:     st.StepID,
			StepName:   st.StepName,
			Grade:      *st.Grade,
			GraderName: st.ReviewerName,
			Comment:    st.Comment,
		}
		if st.ReviewedAt != nil {
			grade.GradedAt = timestamppb.New(*st.ReviewedAt)
		}
		pb.Grade = grade
	}
	return pb
}

// ==================== Submissions ====================

func (h *Handler) ListSupervisorProjectSubmissions(ctx context.Context, req *adminv1.ListSupervisorProjectSubmissionsRequest) (*adminv1.ListSubmissionsResponse, error) {
	if err := h.service.requireSupervisorProjectAccess(ctx, req.SupervisorId, req.ProjectId, req.CallerRole); err != nil {
		return nil, err
	}

	page := req.Page
	if page <= 0 {
		page = 1
	}
	pageSize := req.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > supervisorMaxListPageSize {
		pageSize = supervisorMaxListPageSize
	}

	subs, total, err := h.service.ListSubmissions(ctx, SubmissionFilter{
		ProjectID: req.ProjectId,
		StepID:    req.StepId,
		Status:    req.Status,
		Limit:     int(pageSize),
		Offset:    int((page - 1) * pageSize),
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list submissions: %v", err)
	}

	nameIDs := make([]int64, 0, len(subs))
	for _, s := range subs {
		if s.SubmittedBy > 0 {
			nameIDs = append(nameIDs, s.SubmittedBy)
		}
	}
	names := h.service.ResolveUserNames(ctx, nameIDs)

	resp := &adminv1.ListSubmissionsResponse{TotalCount: total}
	for _, s := range subs {
		resp.Submissions = append(resp.Submissions, toPbSupervisorSubmission(s, names[s.SubmittedBy]))
	}
	return resp, nil
}

func (h *Handler) GetSupervisorProjectSubmission(ctx context.Context, req *adminv1.GetSupervisorProjectSubmissionRequest) (*adminv1.GetSubmissionResponse, error) {
	if err := h.service.requireSupervisorProjectAccess(ctx, req.SupervisorId, req.ProjectId, req.CallerRole); err != nil {
		return nil, err
	}

	sub, reviews, err := h.service.GetSubmission(ctx, req.SubmissionId)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "submission not found: %v", err)
	}
	if sub.ProjectID != req.ProjectId {
		return nil, status.Error(codes.NotFound, "submission does not belong to this project")
	}

	names := h.service.ResolveUserNames(ctx, []int64{sub.SubmittedBy})
	pbSub := toPbSupervisorSubmission(sub, names[sub.SubmittedBy])

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

	return &adminv1.GetSubmissionResponse{Submission: pbSub, History: pbHistory}, nil
}

// ==================== Grades ====================

func (h *Handler) GetSupervisorProjectGrades(ctx context.Context, req *adminv1.GetSupervisorProjectGradesRequest) (*adminv1.GetProjectGradesResponse, error) {
	if err := h.service.requireSupervisorProjectAccess(ctx, req.SupervisorId, req.ProjectId, req.CallerRole); err != nil {
		return nil, err
	}

	grades, avg, err := h.service.GetProjectGrades(ctx, req.ProjectId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get grades: %v", err)
	}

	graderIDs := make([]int64, 0, len(grades))
	for _, g := range grades {
		if g.GradedBy > 0 {
			graderIDs = append(graderIDs, g.GradedBy)
		}
	}
	names := h.service.ResolveUserNames(ctx, graderIDs)

	resp := &adminv1.GetProjectGradesResponse{
		ProjectId:    req.ProjectId,
		AverageGrade: avg,
		TotalScore:   avg,
	}
	for _, g := range grades {
		resp.StepGrades = append(resp.StepGrades, &adminv1.GradeInfo{
			Id:         g.ID,
			ProjectId:  g.ProjectID,
			StepId:     g.StepID,
			Grade:      g.Grade,
			GradedBy:   g.GradedBy,
			GraderName: names[g.GradedBy],
			Comment:    g.Comment,
			GradedAt:   timestamppb.New(g.CreatedAt),
		})
	}
	return resp, nil
}

func (h *Handler) GetSupervisorProjectGradingHistory(ctx context.Context, req *adminv1.GetSupervisorProjectGradingHistoryRequest) (*adminv1.GetGradingHistoryResponse, error) {
	if err := h.service.requireSupervisorProjectAccess(ctx, req.SupervisorId, req.ProjectId, req.CallerRole); err != nil {
		return nil, err
	}

	history, err := h.service.GetGradingHistoryFull(ctx, req.ProjectId, req.StepId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get grading history: %v", err)
	}

	resp := &adminv1.GetGradingHistoryResponse{}
	for _, item := range history {
		resp.History = append(resp.History, &adminv1.GradeHistoryItem{
			Id:          item.ID,
			OldGrade:    item.OldGrade,
			NewGrade:    item.NewGrade,
			ChangedBy:   item.ChangedBy,
			ChangerName: item.ChangerName,
			Reason:      item.Reason,
			ChangedAt:   timestamppb.New(item.ChangedAt),
		})
	}
	return resp, nil
}

// ==================== Workflow history ====================

func (h *Handler) GetSupervisorProjectWorkflowHistory(ctx context.Context, req *adminv1.GetSupervisorProjectWorkflowHistoryRequest) (*adminv1.SupervisorProjectWorkflowHistoryResponse, error) {
	if err := h.service.requireSupervisorProjectAccess(ctx, req.SupervisorId, req.ProjectId, req.CallerRole); err != nil {
		return nil, err
	}

	head, err := h.service.repo.GetSupervisorProjectHead(ctx, req.ProjectId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to load project: %v", err)
	}
	items, err := h.service.repo.GetDPUnifiedHistory(ctx, req.ProjectId, head.TeamID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get workflow history: %v", err)
	}

	resp := &adminv1.SupervisorProjectWorkflowHistoryResponse{}
	for _, hi := range items {
		resp.History = append(resp.History, toPbUnifiedHistory(hi))
	}
	return resp, nil
}

// ==================== Files ====================

func (h *Handler) ListSupervisorProjectFiles(ctx context.Context, req *adminv1.ListSupervisorProjectFilesRequest) (*adminv1.SupervisorProjectFilesResponse, error) {
	if err := h.service.requireSupervisorProjectAccess(ctx, req.SupervisorId, req.ProjectId, req.CallerRole); err != nil {
		return nil, err
	}

	rows, err := h.service.repo.ListSupervisorProjectFiles(ctx, req.ProjectId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list files: %v", err)
	}

	resp := &adminv1.SupervisorProjectFilesResponse{}
	for _, row := range rows {
		for _, fa := range parseSubmissionFiles(row.Files) {
			f := &adminv1.SupervisorProjectFile{
				Id:             fa.ID,
				FileName:       fa.FileName,
				DisplayName:    fa.DisplayName,
				FileType:       fa.FileType,
				Size:           fa.Size,
				DownloadUrl:    fa.DownloadURL,
				UploadedBy:     row.UploadedBy,
				UploadedByName: row.UploadedByName,
				UploadedAt:     timestamppb.New(row.UploadedAt),
				SubmissionId:   row.SubmissionID,
				StepId:         row.StepID,
				StepName:       row.StepName,
			}
			resp.Files = append(resp.Files, f)
		}
	}
	resp.TotalCount = int64(len(resp.Files))
	return resp, nil
}

// ==================== Timeline ====================

func (h *Handler) GetSupervisorProjectTimeline(ctx context.Context, req *adminv1.GetSupervisorProjectTimelineRequest) (*adminv1.SupervisorProjectTimelineResponse, error) {
	if err := h.service.requireSupervisorProjectAccess(ctx, req.SupervisorId, req.ProjectId, req.CallerRole); err != nil {
		return nil, err
	}

	head, err := h.service.repo.GetSupervisorProjectHead(ctx, req.ProjectId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to load project: %v", err)
	}
	items, err := h.service.repo.GetDPUnifiedHistory(ctx, req.ProjectId, head.TeamID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get timeline: %v", err)
	}

	resp := &adminv1.SupervisorProjectTimelineResponse{}
	for _, hi := range items {
		resp.Events = append(resp.Events, toPbTimelineEvent(hi))
	}
	return resp, nil
}

// ==================== Review submission ====================

func (h *Handler) ReviewSupervisorProjectSubmission(ctx context.Context, req *adminv1.ReviewSupervisorProjectSubmissionRequest) (*adminv1.ReviewSubmissionResponse, error) {
	if err := h.service.requireSupervisorProjectAccess(ctx, req.SupervisorId, req.ProjectId, req.CallerRole); err != nil {
		return nil, err
	}
	if req.Action == "" {
		return nil, status.Error(codes.InvalidArgument, "action is required")
	}

	// Submission must belong to the guarded project.
	sub, _, err := h.service.GetSubmission(ctx, req.SubmissionId)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "submission not found: %v", err)
	}
	if sub.ProjectID != req.ProjectId {
		return nil, status.Error(codes.NotFound, "submission does not belong to this project")
	}

	// SECURITY:
	// Supervisor can review only states where reviewer_roles contains teacher/supervisor.
	// This prevents supervisor from reviewing norm_control / antiplagiat / commission-only stages.
	if req.CallerRole != "admin" {
		allowed, err := h.service.CanSupervisorReviewSubmissionState(ctx, sub.StepID) //nolint:govet
		if err != nil {
			if _, ok := status.FromError(err); ok {
				return nil, err
			}
			return nil, status.Errorf(codes.Internal, "failed to check submission reviewer roles: %v", err)
		}
		if !allowed {
			return nil, status.Error(codes.PermissionDenied, "supervisor cannot review this workflow state")
		}
	}

	updated, err := h.service.ReviewSubmission(ctx, &ReviewSubmissionRequest{
		SubmissionID: req.SubmissionId,
		ReviewerID:   req.SupervisorId,
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

	names := h.service.ResolveUserNames(ctx, []int64{updated.SubmittedBy})
	return &adminv1.ReviewSubmissionResponse{
		Success:           true,
		Message:           "Submission reviewed successfully",
		UpdatedSubmission: toPbSupervisorSubmission(updated, names[updated.SubmittedBy]),
	}, nil
}

// ==================== Conversion helpers ====================

func toPbSupervisorSubmission(sub *Submission, submitterName string) *adminv1.SubmissionInfo {
	var dataMap map[string]interface{}
	_ = json.Unmarshal(sub.Data, &dataMap)
	pbData, _ := structpb.NewStruct(dataMap)

	pbSub := &adminv1.SubmissionInfo{
		Id:            sub.ID,
		ProjectId:     sub.ProjectID,
		TeamId:        sub.TeamID,
		TeamName:      sub.TeamName,
		StepId:        sub.StepID,
		StepName:      sub.StepName,
		SubmittedBy:   sub.SubmittedBy,
		SubmitterName: submitterName,
		Status:        sub.Status,
		Data:          pbData,
		Files:         toPbFileAttachments(sub.Files),
		SubmittedAt:   timestamppb.New(sub.CreatedAt),
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
	return pbSub
}

func toPbFileAttachments(raw []byte) []*adminv1.FileAttachment {
	var out []*adminv1.FileAttachment
	for _, fa := range parseSubmissionFiles(raw) {
		out = append(out, &adminv1.FileAttachment{
			Id:          fa.ID,
			FileName:    fa.FileName,
			FileType:    fa.FileType,
			Size:        fa.Size,
			DownloadUrl: fa.DownloadURL,
		})
	}
	return out
}

// fileAttachment is the normalized form of a single entry in admin_submissions.files.
type fileAttachment struct {
	ID          string
	FileName    string
	DisplayName string
	FileType    string
	Size        int64
	DownloadURL string
}

// parseSubmissionFiles handles both JSONB shapes found in admin_submissions.files:
// a plain array of file-id strings, or an array of file-attachment objects.
func parseSubmissionFiles(raw []byte) []fileAttachment {
	if len(raw) == 0 {
		return nil
	}
	var elems []json.RawMessage
	if err := json.Unmarshal(raw, &elems); err != nil {
		return nil
	}

	out := make([]fileAttachment, 0, len(elems))
	for _, el := range elems {
		// Shape 1: a bare file-id string.
		var idStr string
		if err := json.Unmarshal(el, &idStr); err == nil && idStr != "" {
			out = append(out, fileAttachment{
				ID:          idStr,
				DownloadURL: downloadURLForFile(idStr),
			})
			continue
		}
		// Shape 2: an object with metadata.
		var obj struct {
			ID          string `json:"id"`
			FileID      string `json:"file_id"`
			FileName    string `json:"file_name"`
			Name        string `json:"name"`
			DisplayName string `json:"display_name"`
			FileType    string `json:"file_type"`
			MimeType    string `json:"mime_type"`
			Size        int64  `json:"size"`
			DownloadURL string `json:"download_url"`
		}
		if err := json.Unmarshal(el, &obj); err != nil {
			continue
		}
		fa := fileAttachment{
			ID:          firstNonEmpty(obj.ID, obj.FileID),
			FileName:    firstNonEmpty(obj.FileName, obj.Name),
			DisplayName: obj.DisplayName,
			FileType:    firstNonEmpty(obj.FileType, obj.MimeType),
			Size:        obj.Size,
			DownloadURL: obj.DownloadURL,
		}
		if fa.DownloadURL == "" && fa.ID != "" {
			fa.DownloadURL = downloadURLForFile(fa.ID)
		}
		out = append(out, fa)
	}
	return out
}

func downloadURLForFile(id string) string {
	return fmt.Sprintf("/api/v1/files/%s", id)
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

func toPbUnifiedHistory(hi *UnifiedHistoryItem) *adminv1.UnifiedHistoryItem {
	return &adminv1.UnifiedHistoryItem{
		Id:        hi.ID,
		Source:    hi.Source,
		Action:    hi.Action,
		ActorId:   hi.ActorID,
		ActorName: hi.ActorName,
		OldValue:  hi.OldValue,
		NewValue:  hi.NewValue,
		Comment:   hi.Comment,
		CreatedAt: timestamppb.New(hi.CreatedAt),
	}
}

func toPbTimelineEvent(hi *UnifiedHistoryItem) *adminv1.SupervisorTimelineEvent {
	eventType, title, description := timelineDescriptor(hi)
	meta, _ := structpb.NewStruct(map[string]interface{}{
		"source":    hi.Source,
		"action":    hi.Action,
		"old_value": hi.OldValue,
		"new_value": hi.NewValue,
	})
	return &adminv1.SupervisorTimelineEvent{
		Id:          fmt.Sprintf("%s-%s", hi.Source, hi.ID),
		Type:        eventType,
		Title:       title,
		Description: description,
		ActorId:     hi.ActorID,
		ActorName:   hi.ActorName,
		CreatedAt:   timestamppb.New(hi.CreatedAt),
		Metadata:    meta,
	}
}

// timelineDescriptor maps a unified-history item to a UI-friendly type/title/description.
func timelineDescriptor(hi *UnifiedHistoryItem) (eventType, title, description string) {
	switch hi.Source {
	case "project":
		eventType = "workflow"
		title = "Переход этапа"
		if hi.OldValue != "" || hi.NewValue != "" {
			description = fmt.Sprintf("%s → %s", hi.OldValue, hi.NewValue)
		} else {
			description = hi.Comment
		}
	case "grade":
		eventType = "grade"
		title = "Выставлена оценка"
		description = fmt.Sprintf("%s баллов", hi.NewValue)
	case "submission":
		eventType = "submission"
		title = "Проверка документа"
		description = hi.Comment
	default:
		eventType = hi.Source
		title = hi.Action
		description = hi.Comment
	}
	return eventType, title, description
}
