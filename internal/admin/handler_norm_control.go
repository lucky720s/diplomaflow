package admin

import (
	"context"
	"encoding/json"
	"strconv"

	adminv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/admin/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func actorIDFromMD(ctx context.Context) int64 {
	if md, ok := metadata.FromIncomingContext(ctx); ok && md != nil {
		if v := md.Get("x-user-id"); len(v) > 0 {
			id, _ := strconv.ParseInt(v[0], 10, 64)
			return id
		}
	}
	return 0
}

func deptIDFromMD(ctx context.Context) int64 {
	if md, ok := metadata.FromIncomingContext(ctx); ok && md != nil {
		if v := md.Get("x-department-id"); len(v) > 0 {
			id, _ := strconv.ParseInt(v[0], 10, 64)
			return id
		}
	}
	return 0
}
func (h *Handler) ListPendingDocuments(ctx context.Context, req *adminv1.ListPendingDocumentsRequest) (*adminv1.ListPendingDocumentsResponse, error) {
	deptID := deptIDFromMD(ctx)
	if deptID == 0 {
		return nil, status.Error(codes.PermissionDenied, "department_id is required in metadata")
	}

	page := int(req.Page)
	if page <= 0 {
		page = 1
	}

	pageSize := int(req.PageSize)
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	statusFilter := req.Status

	// ВАЖНО:
	// /norm-control/pending по умолчанию НЕ должен возвращать approved/returned историю.
	// Если фронт явно передал status=approved, тогда можно вернуть approved.
	// Но без status показываем только активные проверки.
	filter := NormCheckFilter{
		DepartmentID: deptID,
		Status:       statusFilter,
		TeamID:       req.TeamId,
		CheckerID:    req.CheckerId,
		Limit:        pageSize,
		Offset:       (page - 1) * pageSize,
	}

	list, total, err := h.service.NormListPending(ctx, filter)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list failed: %v", err)
	}

	out := &adminv1.ListPendingDocumentsResponse{TotalCount: total}

	for _, c := range list {
		if c == nil {
			continue
		}

		// Дополнительная защита на уровне handler:
		// даже если repo случайно вернул approved/returned, в pending endpoint не отдаём.
		if statusFilter == "" && c.Status != "submitted" && c.Status != "in_review" {
			continue
		}

		fileID := ""
		if c.PrimaryFileID != nil {
			fileID = *c.PrimaryFileID
		}

		teamID := int64(0)
		if c.TeamID != nil {
			teamID = *c.TeamID
		}

		checkerID := int64(0)
		if c.CheckerID != nil {
			checkerID = *c.CheckerID
		}

		out.Documents = append(out.Documents, &adminv1.PendingDocument{
			Id:          c.SubmissionID,
			ProjectId:   c.ProjectID,
			TeamId:      teamID,
			Status:      c.Status,
			FileId:      fileID,
			Version:     c.DocumentVersion,
			CheckerId:   checkerID,
			SubmittedAt: timestamppb.New(c.CreatedAt),
			File:        h.service.GetFileRef(ctx, fileID),
		})
	}

	return out, nil
}

func (h *Handler) GetDocumentForReview(ctx context.Context, req *adminv1.GetDocumentRequest) (*adminv1.DocumentReviewResponse, error) {
	check, issues, err := h.service.NormGetDocument(ctx, req.Id)
	if err != nil {
		if st, ok := status.FromError(err); ok {
			return nil, st.Err()
		}
		return nil, status.Errorf(codes.Internal, "get failed: %v", err)
	}

	fileID := ""
	if check.PrimaryFileID != nil {
		fileID = *check.PrimaryFileID
	}
	teamID := int64(0)
	if check.TeamID != nil {
		teamID = *check.TeamID
	}
	checkerID := int64(0)
	if check.CheckerID != nil {
		checkerID = *check.CheckerID
	}

	doc := &adminv1.PendingDocument{
		Id:          check.SubmissionID,
		ProjectId:   check.ProjectID,
		TeamId:      teamID,
		Status:      check.Status,
		FileId:      fileID,
		Version:     check.DocumentVersion,
		CheckerId:   checkerID,
		SubmittedAt: timestamppb.New(check.CreatedAt),
		File:        h.service.GetFileRef(ctx, fileID),
	}

	resp := &adminv1.DocumentReviewResponse{
		Document:       doc,
		OverallComment: check.OverallComment,
	}

	for _, it := range issues {
		if it == nil {
			continue
		}
		pn := int32(0)
		if it.PageNumber != nil {
			pn = *it.PageNumber
		}
		ln := int32(0)
		if it.LineNumber != nil {
			ln = *it.LineNumber
		}
		pb := &adminv1.Issue{
			Id:          it.ID,
			CheckId:     it.SubmissionID,
			Category:    it.Category,
			Severity:    it.Severity,
			Description: it.Description,
			PageNumber:  pn,
			LineNumber:  ln,
			IsResolved:  it.IsResolved,
			CreatedAt:   timestamppb.New(it.CreatedAt),
		}
		if it.ResolvedAt != nil {
			pb.ResolvedAt = timestamppb.New(*it.ResolvedAt)
		}
		resp.Issues = append(resp.Issues, pb)
	}

	return resp, nil
}

func (h *Handler) StartReview(ctx context.Context, req *adminv1.StartReviewRequest) (*adminv1.StartReviewResponse, error) {
	actorID := actorIDFromMD(ctx)
	check, err := h.service.NormStartReview(ctx, req.Id, actorID)
	if err != nil {
		if st, ok := status.FromError(err); ok {
			return nil, st.Err()
		}
		return nil, status.Errorf(codes.Internal, "start failed: %v", err)
	}
	return &adminv1.StartReviewResponse{Success: true, Status: check.Status}, nil
}

func (h *Handler) AddIssue(ctx context.Context, req *adminv1.AddIssueRequest) (*adminv1.AddIssueResponse, error) {
	actorID := actorIDFromMD(ctx)

	var page *int32
	if req.PageNumber != 0 {
		v := req.PageNumber
		page = &v
	}
	var line *int32
	if req.LineNumber != 0 {
		v := req.LineNumber
		line = &v
	}

	issue := &NormControlIssue{
		Category:    req.Category,
		Severity:    req.Severity,
		Description: req.Description,
		PageNumber:  page,
		LineNumber:  line,
		IsResolved:  false,
	}

	created, err := h.service.NormAddIssue(ctx, req.DocumentId, actorID, issue)
	if err != nil {
		if st, ok := status.FromError(err); ok {
			return nil, st.Err()
		}
		return nil, status.Errorf(codes.Internal, "add issue failed: %v", err)
	}

	pb := &adminv1.Issue{
		Id:          created.ID,
		CheckId:     created.SubmissionID,
		Category:    created.Category,
		Severity:    created.Severity,
		Description: created.Description,
		IsResolved:  created.IsResolved,
		CreatedAt:   timestamppb.New(created.CreatedAt),
	}
	if created.PageNumber != nil {
		pb.PageNumber = *created.PageNumber
	}
	if created.LineNumber != nil {
		pb.LineNumber = *created.LineNumber
	}

	return &adminv1.AddIssueResponse{Issue: pb}, nil
}

func (h *Handler) UpdateIssue(ctx context.Context, req *adminv1.UpdateIssueRequest) (*adminv1.UpdateIssueResponse, error) {
	actorID := actorIDFromMD(ctx)

	var page *int32
	if req.PageNumber != 0 {
		v := req.PageNumber
		page = &v
	}
	var line *int32
	if req.LineNumber != 0 {
		v := req.LineNumber
		line = &v
	}

	upd := &NormControlIssue{
		Category:    req.Category,
		Severity:    req.Severity,
		Description: req.Description,
		PageNumber:  page,
		LineNumber:  line,
		IsResolved:  req.IsResolved,
	}

	issue, err := h.service.NormUpdateIssue(ctx, req.Id, actorID, upd)
	if err != nil {
		if st, ok := status.FromError(err); ok {
			return nil, st.Err()
		}
		return nil, status.Errorf(codes.Internal, "update issue failed: %v", err)
	}

	pb := &adminv1.Issue{
		Id:          issue.ID,
		CheckId:     issue.SubmissionID,
		Category:    issue.Category,
		Severity:    issue.Severity,
		Description: issue.Description,
		IsResolved:  issue.IsResolved,
		CreatedAt:   timestamppb.New(issue.CreatedAt),
	}
	if issue.PageNumber != nil {
		pb.PageNumber = *issue.PageNumber
	}
	if issue.LineNumber != nil {
		pb.LineNumber = *issue.LineNumber
	}
	if issue.ResolvedAt != nil {
		pb.ResolvedAt = timestamppb.New(*issue.ResolvedAt)
	}

	return &adminv1.UpdateIssueResponse{Issue: pb}, nil
}

func (h *Handler) DeleteIssue(ctx context.Context, req *adminv1.DeleteIssueRequest) (*adminv1.DeleteIssueResponse, error) {
	actorID := actorIDFromMD(ctx)
	if err := h.service.NormDeleteIssue(ctx, req.Id, actorID); err != nil {
		if st, ok := status.FromError(err); ok {
			return nil, st.Err()
		}
		return nil, status.Errorf(codes.Internal, "delete issue failed: %v", err)
	}
	return &adminv1.DeleteIssueResponse{Success: true}, nil
}

func (h *Handler) ApproveDocument(ctx context.Context, req *adminv1.ApproveDocumentRequest) (*adminv1.ApproveDocumentResponse, error) {
	actorID := actorIDFromMD(ctx)
	check, err := h.service.NormApprove(ctx, req.Id, actorID, req.Comment)
	if err != nil {
		if st, ok := status.FromError(err); ok {
			return nil, st.Err()
		}
		return nil, status.Errorf(codes.Internal, "approve failed: %v", err)
	}
	return &adminv1.ApproveDocumentResponse{Success: true, Status: check.Status}, nil
}

func (h *Handler) ReturnForRevision(ctx context.Context, req *adminv1.ReturnForRevisionRequest) (*adminv1.ReturnForRevisionResponse, error) {
	actorID := actorIDFromMD(ctx)
	check, err := h.service.NormReturn(ctx, req.Id, actorID, req.Comment)
	if err != nil {
		if st, ok := status.FromError(err); ok {
			return nil, st.Err()
		}
		return nil, status.Errorf(codes.Internal, "return failed: %v", err)
	}
	return &adminv1.ReturnForRevisionResponse{Success: true, Status: check.Status}, nil
}

func (h *Handler) GetCheckHistory(ctx context.Context, req *adminv1.GetCheckHistoryRequest) (*adminv1.GetCheckHistoryResponse, error) {
	hist, err := h.service.NormHistory(ctx, req.ProjectId)
	if err != nil {
		if st, ok := status.FromError(err); ok {
			return nil, st.Err()
		}
		return nil, status.Errorf(codes.Internal, "history failed: %v", err)
	}

	resp := &adminv1.GetCheckHistoryResponse{}
	for _, it := range hist {
		if it == nil {
			continue
		}
		actor := int64(0)
		if it.ActorID != nil {
			actor = *it.ActorID
		}
		resp.History = append(resp.History, &adminv1.HistoryItem{
			Id:        strconv.FormatInt(it.ID, 10),
			ProjectId: it.ProjectID,
			Action:    it.Action,
			ActorId:   actor,
			Comment:   it.Comment,
			CreatedAt: timestamppb.New(it.CreatedAt),
		})
	}
	return resp, nil
}

func (h *Handler) GetErrorStatistics(ctx context.Context, req *adminv1.GetStatisticsRequest) (*adminv1.GetStatisticsResponse, error) {
	mdDept := deptIDFromMD(ctx)
	if mdDept == 0 {
		return nil, status.Error(codes.PermissionDenied, "department_id is required in metadata")
	}

	deptID := mdDept
	if roleFromMD(ctx) == "admin" && req.DepartmentId != 0 {
		deptID = req.DepartmentId
	}

	stats, err := h.service.NormStats(ctx, deptID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "stats failed: %v", err)
	}

	resp := &adminv1.GetStatisticsResponse{}
	for cat, cnt := range stats {
		resp.Stats = append(resp.Stats, &adminv1.CategoryStat{
			Category:   cat,
			IssueCount: cnt,
		})
	}
	return resp, nil
}

func (h *Handler) ListChecklists(ctx context.Context, _ *adminv1.ListChecklistsRequest) (*adminv1.ListChecklistsResponse, error) {
	list, err := h.service.NormListChecklists(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list checklists failed: %v", err)
	}

	resp := &adminv1.ListChecklistsResponse{}
	for _, c := range list {
		if c == nil {
			continue
		}
		resp.Checklists = append(resp.Checklists, &adminv1.Checklist{
			Id:        c.ID,
			Name:      c.Name,
			Category:  c.Category,
			ItemsJson: string(c.Items),
			IsActive:  c.IsActive,
		})
	}
	return resp, nil
}

func (h *Handler) CreateChecklist(ctx context.Context, req *adminv1.CreateChecklistRequest) (*adminv1.CreateChecklistResponse, error) {
	actorID := actorIDFromMD(ctx)
	c, err := h.service.NormCreateChecklist(ctx, actorID, req.Name, req.Category, req.ItemsJson)
	if err != nil {
		if st, ok := status.FromError(err); ok {
			return nil, st.Err()
		}
		return nil, status.Errorf(codes.Internal, "create checklist failed: %v", err)
	}

	// Ensure stored JSON is canonical
	var tmp any
	_ = json.Unmarshal(c.Items, &tmp)
	b, _ := json.Marshal(tmp)

	return &adminv1.CreateChecklistResponse{
		Checklist: &adminv1.Checklist{
			Id:        c.ID,
			Name:      c.Name,
			Category:  c.Category,
			ItemsJson: string(b),
			IsActive:  c.IsActive,
		},
	}, nil
}
func roleFromMD(ctx context.Context) string {
	if md, ok := metadata.FromIncomingContext(ctx); ok && md != nil {
		if v := md.Get("x-user-role"); len(v) > 0 {
			return v[0]
		}
	}
	return ""
}
