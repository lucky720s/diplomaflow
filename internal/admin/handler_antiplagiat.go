package admin

import (
	"context"

	adminv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/admin/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func scoreOrUnset(p *int32) int32 {
	if p == nil {
		return -1
	}
	return *p
}

func (h *Handler) toAntiplagPB(ctx context.Context, c *AntiplagCheck) *adminv1.AntiplagPendingDocument {
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

	return &adminv1.AntiplagPendingDocument{
		Id:              c.SubmissionID,
		ProjectId:       c.ProjectID,
		TeamId:          teamID,
		Status:          c.Status,
		FileId:          fileID,
		Version:         c.DocumentVersion,
		CheckerId:       checkerID,
		PlagiarismScore: scoreOrUnset(c.PlagiarismScore),
		AiScore:         scoreOrUnset(c.AIScore),
		SubmittedAt:     timestamppb.New(c.CreatedAt),
		File:            h.service.GetFileRef(ctx, fileID),
	}
}

func (h *Handler) ListAntiplagPending(ctx context.Context, req *adminv1.ListAntiplagPendingRequest) (*adminv1.ListAntiplagPendingResponse, error) {
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

	filter := AntiplagCheckFilter{
		DepartmentID: deptID,
		Status:       req.Status,
		TeamID:       req.TeamId,
		CheckerID:    req.CheckerId,
		Limit:        pageSize,
		Offset:       (page - 1) * pageSize,
	}

	list, total, err := h.service.AntiplagListPending(ctx, filter)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list failed: %v", err)
	}

	out := &adminv1.ListAntiplagPendingResponse{TotalCount: total}
	for _, c := range list {
		if c == nil {
			continue
		}
		out.Documents = append(out.Documents, h.toAntiplagPB(ctx, c))
	}
	return out, nil
}

func (h *Handler) GetAntiplagDocument(ctx context.Context, req *adminv1.GetAntiplagDocumentRequest) (*adminv1.AntiplagDocumentResponse, error) {
	check, comments, err := h.service.AntiplagGetDocument(ctx, req.Id)
	if err != nil {
		if st, ok := status.FromError(err); ok {
			return nil, st.Err()
		}
		return nil, status.Errorf(codes.Internal, "get failed: %v", err)
	}

	resp := &adminv1.AntiplagDocumentResponse{
		Document:       h.toAntiplagPB(ctx, check),
		OverallComment: check.OverallComment,
	}
	for _, it := range comments {
		if it == nil {
			continue
		}
		pn := int32(0)
		if it.PageNumber != nil {
			pn = *it.PageNumber
		}
		resp.Comments = append(resp.Comments, &adminv1.AntiplagComment{
			Id:         it.ID,
			CheckId:    it.SubmissionID,
			Text:       it.Text,
			PageNumber: pn,
			CreatedAt:  timestamppb.New(it.CreatedAt),
		})
	}
	return resp, nil
}

func (h *Handler) StartAntiplagReview(ctx context.Context, req *adminv1.StartAntiplagReviewRequest) (*adminv1.StartAntiplagReviewResponse, error) {
	actorID := actorIDFromMD(ctx)
	check, err := h.service.AntiplagStartReview(ctx, req.Id, actorID)
	if err != nil {
		if st, ok := status.FromError(err); ok {
			return nil, st.Err()
		}
		return nil, status.Errorf(codes.Internal, "start failed: %v", err)
	}
	return &adminv1.StartAntiplagReviewResponse{Success: true, Status: check.Status}, nil
}

func (h *Handler) SetAntiplagScores(ctx context.Context, req *adminv1.SetAntiplagScoresRequest) (*adminv1.SetAntiplagScoresResponse, error) {
	actorID := actorIDFromMD(ctx)
	check, err := h.service.AntiplagSetScores(ctx, req.Id, actorID,
		req.HasPlagiarism, req.PlagiarismScore, req.HasAi, req.AiScore)
	if err != nil {
		if st, ok := status.FromError(err); ok {
			return nil, st.Err()
		}
		return nil, status.Errorf(codes.Internal, "set scores failed: %v", err)
	}
	return &adminv1.SetAntiplagScoresResponse{
		PlagiarismScore: scoreOrUnset(check.PlagiarismScore),
		AiScore:         scoreOrUnset(check.AIScore),
	}, nil
}

func (h *Handler) AddAntiplagComment(ctx context.Context, req *adminv1.AddAntiplagCommentRequest) (*adminv1.AddAntiplagCommentResponse, error) {
	actorID := actorIDFromMD(ctx)
	var page *int32
	if req.PageNumber != 0 {
		v := req.PageNumber
		page = &v
	}
	created, err := h.service.AntiplagAddComment(ctx, req.DocumentId, actorID, req.Text, page)
	if err != nil {
		if st, ok := status.FromError(err); ok {
			return nil, st.Err()
		}
		return nil, status.Errorf(codes.Internal, "add comment failed: %v", err)
	}
	pb := &adminv1.AntiplagComment{
		Id:        created.ID,
		CheckId:   created.SubmissionID,
		Text:      created.Text,
		CreatedAt: timestamppb.New(created.CreatedAt),
	}
	if created.PageNumber != nil {
		pb.PageNumber = *created.PageNumber
	}
	return &adminv1.AddAntiplagCommentResponse{Comment: pb}, nil
}

func (h *Handler) UpdateAntiplagComment(ctx context.Context, req *adminv1.UpdateAntiplagCommentRequest) (*adminv1.UpdateAntiplagCommentResponse, error) {
	actorID := actorIDFromMD(ctx)
	var page *int32
	if req.PageNumber != 0 {
		v := req.PageNumber
		page = &v
	}
	c, err := h.service.AntiplagUpdateComment(ctx, req.Id, actorID, req.Text, page)
	if err != nil {
		if st, ok := status.FromError(err); ok {
			return nil, st.Err()
		}
		return nil, status.Errorf(codes.Internal, "update comment failed: %v", err)
	}
	pb := &adminv1.AntiplagComment{
		Id:        c.ID,
		CheckId:   c.SubmissionID,
		Text:      c.Text,
		CreatedAt: timestamppb.New(c.CreatedAt),
	}
	if c.PageNumber != nil {
		pb.PageNumber = *c.PageNumber
	}
	return &adminv1.UpdateAntiplagCommentResponse{Comment: pb}, nil
}

func (h *Handler) DeleteAntiplagComment(ctx context.Context, req *adminv1.DeleteAntiplagCommentRequest) (*adminv1.DeleteAntiplagCommentResponse, error) {
	actorID := actorIDFromMD(ctx)
	if err := h.service.AntiplagDeleteComment(ctx, req.Id, actorID); err != nil {
		if st, ok := status.FromError(err); ok {
			return nil, st.Err()
		}
		return nil, status.Errorf(codes.Internal, "delete comment failed: %v", err)
	}
	return &adminv1.DeleteAntiplagCommentResponse{Success: true}, nil
}

func (h *Handler) ApproveAntiplagDocument(ctx context.Context, req *adminv1.ApproveAntiplagRequest) (*adminv1.ApproveAntiplagResponse, error) {
	actorID := actorIDFromMD(ctx)
	check, err := h.service.AntiplagApprove(ctx, req.Id, actorID, req.Comment)
	if err != nil {
		if st, ok := status.FromError(err); ok {
			return nil, st.Err()
		}
		return nil, status.Errorf(codes.Internal, "approve failed: %v", err)
	}
	return &adminv1.ApproveAntiplagResponse{Success: true, Status: check.Status}, nil
}

func (h *Handler) ReturnAntiplagForRevision(ctx context.Context, req *adminv1.ReturnAntiplagRequest) (*adminv1.ReturnAntiplagResponse, error) {
	actorID := actorIDFromMD(ctx)
	check, err := h.service.AntiplagReturn(ctx, req.Id, actorID, req.Comment)
	if err != nil {
		if st, ok := status.FromError(err); ok {
			return nil, st.Err()
		}
		return nil, status.Errorf(codes.Internal, "return failed: %v", err)
	}
	return &adminv1.ReturnAntiplagResponse{Success: true, Status: check.Status}, nil
}

func (h *Handler) GetAntiplagHistory(ctx context.Context, req *adminv1.GetAntiplagHistoryRequest) (*adminv1.GetAntiplagHistoryResponse, error) {
	hist, err := h.service.AntiplagHistory(ctx, req.ProjectId)
	if err != nil {
		if st, ok := status.FromError(err); ok {
			return nil, st.Err()
		}
		return nil, status.Errorf(codes.Internal, "history failed: %v", err)
	}
	out := &adminv1.GetAntiplagHistoryResponse{}
	for _, it := range hist {
		if it == nil {
			continue
		}
		actor := int64(0)
		if it.ActorID != nil {
			actor = *it.ActorID
		}
		out.History = append(out.History, &adminv1.AntiplagHistoryItem{
			Id:        "",
			ProjectId: it.ProjectID,
			Action:    it.Action,
			ActorId:   actor,
			Comment:   it.Comment,
			CreatedAt: timestamppb.New(it.CreatedAt),
		})
	}
	return out, nil
}
