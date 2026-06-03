package task

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	authv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/auth/v1"
	taskv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/task/v1"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Handler struct {
	taskv1.UnimplementedTaskServiceServer
	service       *Service
	accessChecker *AccessChecker
	authClient    authv1.AuthServiceClient
	logger        *zap.Logger
}

func NewHandler(svc *Service, accessChecker *AccessChecker, authClient authv1.AuthServiceClient, logger *zap.Logger) *Handler {
	return &Handler{
		service:       svc,
		accessChecker: accessChecker,
		authClient:    authClient,
		logger:        logger,
	}
}

func (h *Handler) GetBoard(ctx context.Context, req *taskv1.GetBoardRequest) (*taskv1.GetBoardResponse, error) {
	auth, ok := GetAuthContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "unauthorized")
	}

	if err := h.accessChecker.CheckBoardAccess(ctx, req.BoardId, auth); err != nil {
		h.logger.Warn("access denied to board",
			zap.Int64("board_id", req.BoardId),
			zap.Int64("user_id", auth.UserID),
			zap.String("role", auth.Role),
			zap.Error(err),
		)
		return nil, toGRPCError(err)
	}

	board, err := h.service.GetBoard(ctx, req.BoardId, req.IncludeColumns, req.IncludeStats)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "board not found: %v", err)
	}

	return &taskv1.GetBoardResponse{Board: h.boardToProto(board)}, nil
}

func (h *Handler) GetBoardByProject(ctx context.Context, req *taskv1.GetBoardByProjectRequest) (*taskv1.GetBoardResponse, error) {
	auth, ok := GetAuthContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "unauthorized")
	}

	if err := h.accessChecker.CheckBoardAccessByProject(ctx, req.ProjectId, auth); err != nil {
		h.logger.Warn("access denied to board by project",
			zap.Int64("project_id", req.ProjectId),
			zap.Int64("user_id", auth.UserID),
			zap.Error(err),
		)
		return nil, toGRPCError(err)
	}

	board, err := h.service.GetBoardByProject(ctx, req.ProjectId, req.IncludeColumns, req.IncludeStats)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "board not found: %v", err)
	}

	return &taskv1.GetBoardResponse{Board: h.boardToProto(board)}, nil
}

func (h *Handler) ListMyBoards(ctx context.Context, req *taskv1.ListMyBoardsRequest) (*taskv1.ListMyBoardsResponse, error) {
	auth, ok := GetAuthContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "unauthorized")
	}

	// Используем данные из auth context, а не из request
	boards, err := h.service.ListMyBoards(ctx, auth.UserID, auth.Role, auth.UniversityID, auth.DepartmentID, req.IncludeColumns, req.IncludeStats)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list boards: %v", err)
	}

	out := &taskv1.ListMyBoardsResponse{}
	for _, b := range boards {
		out.Boards = append(out.Boards, h.boardToProto(b))
	}
	return out, nil
}

func (h *Handler) UpdateBoard(ctx context.Context, req *taskv1.UpdateBoardRequest) (*taskv1.Board, error) {
	auth, ok := GetAuthContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "unauthorized")
	}

	if err := h.accessChecker.CheckBoardAccess(ctx, req.BoardId, auth); err != nil {
		return nil, toGRPCError(err)
	}

	var settings *BoardSettings
	if req.Settings != nil {
		settings = &BoardSettings{
			DefaultColumn:      req.Settings.DefaultColumn,
			AllowCustomColumns: req.Settings.AllowCustomColumns,
			ShowCompleted:      req.Settings.ShowCompleted,
			Labels:             req.Settings.Labels,
		}
	}

	board, err := h.service.UpdateBoard(ctx, &UpdateBoardInput{
		BoardID:     req.BoardId,
		Name:        req.Name,
		Description: req.Description,
		Settings:    settings,
		UpdateMask:  req.UpdateMask,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "update board: %v", err)
	}

	return h.boardToProto(board), nil
}

func (h *Handler) ListColumns(ctx context.Context, req *taskv1.ListColumnsRequest) (*taskv1.ListColumnsResponse, error) {
	auth, ok := GetAuthContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "unauthorized")
	}

	if err := h.accessChecker.CheckBoardAccess(ctx, req.BoardId, auth); err != nil {
		return nil, toGRPCError(err)
	}

	columns, err := h.service.ListColumns(ctx, req.BoardId, req.IncludeTaskCount)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list columns: %v", err)
	}

	out := &taskv1.ListColumnsResponse{}
	for _, c := range columns {
		out.Columns = append(out.Columns, h.columnToProto(c))
	}
	return out, nil
}

func (h *Handler) CreateColumn(ctx context.Context, req *taskv1.CreateColumnRequest) (*taskv1.Column, error) {
	auth, ok := GetAuthContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "unauthorized")
	}

	if err := h.accessChecker.CheckBoardAccess(ctx, req.BoardId, auth); err != nil {
		return nil, toGRPCError(err)
	}

	col, err := h.service.CreateColumn(ctx, &CreateColumnInput{
		BoardID:      req.BoardId,
		Name:         req.Name,
		Slug:         req.Slug,
		Description:  req.Description,
		Color:        req.Color,
		Icon:         req.Icon,
		WIPLimit:     req.WipLimit,
		IsDoneColumn: req.IsDoneColumn,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "create column: %v", err)
	}

	return h.columnToProto(col), nil
}

func (h *Handler) UpdateColumn(ctx context.Context, req *taskv1.UpdateColumnRequest) (*taskv1.Column, error) {
	auth, ok := GetAuthContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "unauthorized")
	}

	if err := h.accessChecker.CanModifyColumn(ctx, req.ColumnId, auth); err != nil {
		return nil, toGRPCError(err)
	}

	col, err := h.service.UpdateColumn(ctx, &UpdateColumnInput{
		ColumnID:    req.ColumnId,
		Name:        req.Name,
		Description: req.Description,
		Color:       req.Color,
		Icon:        req.Icon,
		WIPLimit:    req.WipLimit,
		UpdateMask:  req.UpdateMask,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "update column: %v", err)
	}

	return h.columnToProto(col), nil
}

func (h *Handler) DeleteColumn(ctx context.Context, req *taskv1.DeleteColumnRequest) (*emptypb.Empty, error) {
	auth, ok := GetAuthContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "unauthorized")
	}

	if err := h.accessChecker.CanModifyColumn(ctx, req.ColumnId, auth); err != nil {
		return nil, toGRPCError(err)
	}

	if err := h.service.DeleteColumn(ctx, req.ColumnId, req.MoveTasksToColumnId); err != nil {
		return nil, status.Errorf(codes.Internal, "delete column: %v", err)
	}

	return &emptypb.Empty{}, nil
}

func (h *Handler) ReorderColumns(ctx context.Context, req *taskv1.ReorderColumnsRequest) (*taskv1.ListColumnsResponse, error) {
	auth, ok := GetAuthContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "unauthorized")
	}

	if err := h.accessChecker.CheckBoardAccess(ctx, req.BoardId, auth); err != nil {
		return nil, toGRPCError(err)
	}

	columns, err := h.service.ReorderColumns(ctx, req.BoardId, req.ColumnIds)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "reorder columns: %v", err)
	}

	out := &taskv1.ListColumnsResponse{}
	for _, c := range columns {
		out.Columns = append(out.Columns, h.columnToProto(c))
	}
	return out, nil
}

func (h *Handler) CreateTask(ctx context.Context, req *taskv1.CreateTaskRequest) (*taskv1.Task, error) {
	auth, ok := GetAuthContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "unauthorized")
	}

	if err := h.accessChecker.CheckBoardAccess(ctx, req.BoardId, auth); err != nil {
		return nil, toGRPCError(err)
	}

	var dueDate *time.Time
	if req.DueDate != nil {
		t := req.DueDate.AsTime()
		dueDate = &t
	}

	task, err := h.service.CreateTask(ctx, &CreateTaskInput{
		BoardID:          req.BoardId,
		ColumnID:         req.ColumnId,
		Title:            req.Title,
		Description:      req.Description,
		Priority:         h.priorityFromProto(req.Priority),
		AssigneeID:       req.AssigneeId,
		DueDate:          dueDate,
		EstimatedMinutes: req.EstimatedMinutes,
		Labels:           req.Labels,
		CreatedBy:        auth.UserID, // Используем auth.UserID вместо req.CreatedBy
		WorkflowStepID:   req.WorkflowStepId,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "create task: %v", err)
	}

	return h.taskToProto(task), nil
}

func (h *Handler) GetTask(ctx context.Context, req *taskv1.GetTaskRequest) (*taskv1.GetTaskResponse, error) {
	auth, ok := GetAuthContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "unauthorized")
	}

	if err := h.accessChecker.CheckTaskAccess(ctx, req.TaskId, auth); err != nil {
		return nil, toGRPCError(err)
	}

	task, err := h.service.GetTask(ctx, req.TaskId)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "task not found: %v", err)
	}

	// Загружаем дополнительные данные
	comments, _ := h.service.GetRecentComments(ctx, req.TaskId, 5)
	attachments, _ := h.service.ListAttachments(ctx, req.TaskId)
	activity, _ := h.service.GetRecentActivity(ctx, req.TaskId, 10)
	watchers, _ := h.service.ListWatchers(ctx, req.TaskId)

	resp := &taskv1.GetTaskResponse{
		Task: h.taskToProto(task),
	}

	for _, c := range comments {
		resp.RecentComments = append(resp.RecentComments, h.commentToProto(c))
	}
	for _, a := range attachments {
		resp.Attachments = append(resp.Attachments, h.attachmentToProto(a))
	}
	for _, act := range activity {
		resp.RecentActivity = append(resp.RecentActivity, h.activityToProto(act))
	}
	for _, w := range watchers {
		resp.Watchers = append(resp.Watchers, h.userPreviewToProto(w))
	}

	h.enrichGetTask(ctx, resp)
	return resp, nil
}

func (h *Handler) UpdateTask(ctx context.Context, req *taskv1.UpdateTaskRequest) (*taskv1.Task, error) {
	auth, ok := GetAuthContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "unauthorized")
	}

	existingTask, err := h.service.GetTask(ctx, req.TaskId)
	if err != nil {
		return nil, status.Error(codes.NotFound, "task not found")
	}

	err = h.accessChecker.CanModifyTask(ctx, existingTask, auth)
	if err != nil {
		return nil, toGRPCError(err)
	}

	var dueDate *time.Time
	if req.DueDate != nil {
		t := req.DueDate.AsTime()
		dueDate = &t
	}

	task, err := h.service.UpdateTask(ctx, &UpdateTaskInput{
		TaskID:           req.TaskId,
		Title:            req.Title,
		Description:      req.Description,
		Priority:         h.priorityPtrFromProto(req.Priority),
		DueDate:          dueDate,
		EstimatedMinutes: req.EstimatedMinutes,
		ActualMinutes:    req.ActualMinutes,
		Labels:           req.Labels,
		WorkflowStepID:   req.WorkflowStepId,
		UpdateMask:       req.UpdateMask,
		UpdatedBy:        auth.UserID, // Используем auth.UserID
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "update task: %v", err)
	}

	return h.taskToProto(task), nil
}

func (h *Handler) DeleteTask(ctx context.Context, req *taskv1.DeleteTaskRequest) (*emptypb.Empty, error) {
	auth, ok := GetAuthContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "unauthorized")
	}

	existingTask, err := h.service.GetTask(ctx, req.TaskId)
	if err != nil {
		return nil, status.Error(codes.NotFound, "task not found")
	}

	if err := h.accessChecker.CanDeleteTask(ctx, existingTask, auth); err != nil {
		return nil, toGRPCError(err)
	}

	if err := h.service.DeleteTask(ctx, req.TaskId); err != nil {
		return nil, status.Errorf(codes.Internal, "delete task: %v", err)
	}

	return &emptypb.Empty{}, nil
}

func (h *Handler) ListTasks(ctx context.Context, req *taskv1.ListTasksRequest) (*taskv1.ListTasksResponse, error) {
	auth, ok := GetAuthContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "unauthorized")
	}

	// Проверяем доступ к доске если указана
	if req.BoardId > 0 {
		if err := h.accessChecker.CheckBoardAccess(ctx, req.BoardId, auth); err != nil {
			return nil, toGRPCError(err)
		}
	}

	// Проверяем доступ к колонке если указана
	if req.ColumnId > 0 {
		if err := h.accessChecker.CheckColumnAccess(ctx, req.ColumnId, auth); err != nil {
			return nil, toGRPCError(err)
		}
	}

	tasks, total, err := h.service.ListTasks(ctx, TaskFilter{
		BoardID:        req.BoardId,
		ColumnID:       req.ColumnId,
		AssigneeID:     req.AssigneeId,
		Status:         h.statusFromProto(req.Status),
		Priority:       h.priorityStringFromProto(req.Priority),
		Search:         req.Search,
		Labels:         req.Labels,
		OnlyOverdue:    req.OnlyOverdue,
		OnlyUnassigned: req.OnlyUnassigned,
		SortBy:         req.SortBy,
		SortOrder:      req.SortOrder,
		Limit:          int(req.PageSize),
		Offset:         int((req.Page - 1) * req.PageSize),
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list tasks: %v", err)
	}

	out := &taskv1.ListTasksResponse{
		TotalCount: total,
	}
	for _, t := range tasks {
		out.Tasks = append(out.Tasks, h.taskToProto(t))
	}
	h.enrichTasks(ctx, out.Tasks)
	return out, nil
}
func (h *Handler) MoveTask(ctx context.Context, req *taskv1.MoveTaskRequest) (*taskv1.Task, error) {
	auth, ok := GetAuthContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "unauthorized")
	}

	existingTask, err := h.service.GetTask(ctx, req.TaskId)
	if err != nil {
		return nil, status.Error(codes.NotFound, "task not found")
	}

	err = h.accessChecker.CanMoveTask(ctx, existingTask, req.ToColumnId, auth)
	if err != nil {
		return nil, toGRPCError(err)
	}

	task, err := h.service.MoveTask(ctx, &MoveTaskInput{
		TaskID:   req.TaskId,
		ColumnID: req.ToColumnId,
		Position: int(req.Position),
		MovedBy:  auth.UserID,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "move task: %v", err)
	}

	return h.taskToProto(task), nil
}

func (h *Handler) ReorderTasks(ctx context.Context, req *taskv1.ReorderTasksRequest) (*emptypb.Empty, error) {
	auth, ok := GetAuthContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "unauthorized")
	}

	if err := h.accessChecker.CheckColumnAccess(ctx, req.ColumnId, auth); err != nil {
		return nil, toGRPCError(err)
	}

	if err := h.service.ReorderTasks(ctx, req.ColumnId, req.TaskIds); err != nil {
		return nil, status.Errorf(codes.Internal, "reorder tasks: %v", err)
	}

	return &emptypb.Empty{}, nil
}

func (h *Handler) BulkUpdateTasks(ctx context.Context, req *taskv1.BulkUpdateTasksRequest) (*taskv1.BulkUpdateTasksResponse, error) {
	auth, ok := GetAuthContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "unauthorized")
	}

	// Проверяем доступ к каждой задаче
	for _, taskID := range req.TaskIds {
		if err := h.accessChecker.CheckTaskAccess(ctx, taskID, auth); err != nil {
			return nil, toGRPCError(err)
		}
	}

	tasks, err := h.service.BulkUpdateTasks(ctx, &BulkUpdateInput{
		TaskIDs:        req.TaskIds,
		AssigneeID:     req.AssigneeId,
		Priority:       h.priorityPtrFromProto(req.Priority),
		AddLabels:      req.AddLabels,
		RemoveLabels:   req.RemoveLabels,
		MoveToColumnID: req.MoveToColumnId,
		UpdatedBy:      auth.UserID,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "bulk update: %v", err)
	}

	out := &taskv1.BulkUpdateTasksResponse{
		UpdatedCount: int32(len(tasks)),
	}
	for _, t := range tasks {
		out.UpdatedTasks = append(out.UpdatedTasks, h.taskToProto(t))
	}
	return out, nil
}

func (h *Handler) BulkDeleteTasks(ctx context.Context, req *taskv1.BulkDeleteTasksRequest) (*emptypb.Empty, error) {
	auth, ok := GetAuthContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "unauthorized")
	}

	// Проверяем права на удаление каждой задачи
	for _, taskID := range req.TaskIds {
		task, err := h.service.GetTask(ctx, taskID)
		if err != nil {
			continue // Пропускаем несуществующие
		}
		if err := h.accessChecker.CanDeleteTask(ctx, task, auth); err != nil {
			return nil, toGRPCError(err)
		}
	}

	if err := h.service.BulkDeleteTasks(ctx, req.TaskIds); err != nil {
		return nil, status.Errorf(codes.Internal, "bulk delete: %v", err)
	}

	return &emptypb.Empty{}, nil
}

func (h *Handler) AssignTask(ctx context.Context, req *taskv1.AssignTaskRequest) (*taskv1.Task, error) {
	auth, ok := GetAuthContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "unauthorized")
	}

	existingTask, err := h.service.GetTask(ctx, req.TaskId)
	if err != nil {
		return nil, status.Error(codes.NotFound, "task not found")
	}

	err = h.accessChecker.CanModifyTask(ctx, existingTask, auth)
	if err != nil {
		return nil, toGRPCError(err)
	}

	task, err := h.service.AssignTask(ctx, req.TaskId, req.AssigneeId, auth.UserID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "assign task: %v", err)
	}

	return h.taskToProto(task), nil
}

func (h *Handler) UnassignTask(ctx context.Context, req *taskv1.UnassignTaskRequest) (*taskv1.Task, error) {
	auth, ok := GetAuthContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "unauthorized")
	}

	existingTask, err := h.service.GetTask(ctx, req.TaskId)
	if err != nil {
		return nil, status.Error(codes.NotFound, "task not found")
	}

	err = h.accessChecker.CanModifyTask(ctx, existingTask, auth)
	if err != nil {
		return nil, toGRPCError(err)
	}

	task, err := h.service.UnassignTask(ctx, req.TaskId, auth.UserID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "unassign task: %v", err)
	}

	return h.taskToProto(task), nil
}

func (h *Handler) CreateComment(ctx context.Context, req *taskv1.CreateCommentRequest) (*taskv1.Comment, error) {
	auth, ok := GetAuthContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "unauthorized")
	}

	if err := h.accessChecker.CheckTaskAccess(ctx, req.TaskId, auth); err != nil {
		return nil, toGRPCError(err)
	}

	comment, err := h.service.CreateComment(ctx, &CreateCommentInput{
		TaskID:         req.TaskId,
		AuthorID:       auth.UserID, // Используем auth.UserID
		Content:        req.Content,
		MentionUserIDs: req.MentionUserIds,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "create comment: %v", err)
	}

	pb := h.commentToProto(comment)
	h.enrichComments(ctx, []*taskv1.Comment{pb})
	return pb, nil
}

func (h *Handler) UpdateComment(ctx context.Context, req *taskv1.UpdateCommentRequest) (*taskv1.Comment, error) {
	auth, ok := GetAuthContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "unauthorized")
	}

	comment, err := h.service.GetComment(ctx, req.CommentId)
	if err != nil {
		return nil, status.Error(codes.NotFound, "comment not found")
	}

	err = h.accessChecker.CheckTaskAccess(ctx, comment.TaskID, auth)
	if err != nil {
		return nil, toGRPCError(err)
	}

	// Только автор или admin/teacher может редактировать
	if comment.AuthorID != auth.UserID && auth.Role != "teacher" && auth.Role != "admin" {
		return nil, status.Error(codes.PermissionDenied, "can only edit own comments")
	}

	updated, err := h.service.UpdateComment(ctx, req.CommentId, req.Content)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "update comment: %v", err)
	}

	return h.commentToProto(updated), nil
}

func (h *Handler) DeleteComment(ctx context.Context, req *taskv1.DeleteCommentRequest) (*emptypb.Empty, error) {
	auth, ok := GetAuthContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "unauthorized")
	}

	comment, err := h.service.GetComment(ctx, req.CommentId)
	if err != nil {
		return nil, status.Error(codes.NotFound, "comment not found")
	}

	if err := h.accessChecker.CheckTaskAccess(ctx, comment.TaskID, auth); err != nil {
		return nil, toGRPCError(err)
	}

	// Только автор или admin/teacher может удалять
	if comment.AuthorID != auth.UserID && auth.Role != "teacher" && auth.Role != "admin" {
		return nil, status.Error(codes.PermissionDenied, "can only delete own comments")
	}

	if err := h.service.DeleteComment(ctx, req.CommentId); err != nil {
		return nil, status.Errorf(codes.Internal, "delete comment: %v", err)
	}

	return &emptypb.Empty{}, nil
}

func (h *Handler) ListComments(ctx context.Context, req *taskv1.ListCommentsRequest) (*taskv1.ListCommentsResponse, error) {
	auth, ok := GetAuthContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "unauthorized")
	}

	if err := h.accessChecker.CheckTaskAccess(ctx, req.TaskId, auth); err != nil {
		return nil, toGRPCError(err)
	}

	limit := int(req.PageSize)
	if limit <= 0 {
		limit = 20
	}
	offset := int((req.Page - 1) * req.PageSize)
	if offset < 0 {
		offset = 0
	}

	comments, total, err := h.service.ListComments(ctx, req.TaskId, limit, offset)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list comments: %v", err)
	}

	out := &taskv1.ListCommentsResponse{
		TotalCount: total,
	}
	for _, c := range comments {
		out.Comments = append(out.Comments, h.commentToProto(c))
	}
	h.enrichComments(ctx, out.Comments)
	return out, nil
}

func (h *Handler) AddAttachment(ctx context.Context, req *taskv1.AddAttachmentRequest) (*taskv1.Attachment, error) {
	auth, ok := GetAuthContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "unauthorized")
	}

	if err := h.accessChecker.CheckTaskAccess(ctx, req.TaskId, auth); err != nil {
		return nil, toGRPCError(err)
	}

	attachment, err := h.service.AddAttachment(ctx, &AddAttachmentInput{
		TaskID:     req.TaskId,
		FileID:     req.FileId,
		FileName:   req.FileName,
		FileType:   req.FileType,
		FileSize:   req.FileSize,
		UploadedBy: auth.UserID,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "add attachment: %v", err)
	}

	pb := h.attachmentToProto(attachment)
	h.enrichAttachments(ctx, []*taskv1.Attachment{pb})
	return pb, nil
}

func (h *Handler) RemoveAttachment(ctx context.Context, req *taskv1.RemoveAttachmentRequest) (*emptypb.Empty, error) {
	auth, ok := GetAuthContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "unauthorized")
	}

	attachment, err := h.service.GetAttachment(ctx, req.AttachmentId)
	if err != nil {
		return nil, status.Error(codes.NotFound, "attachment not found")
	}

	if err := h.accessChecker.CheckTaskAccess(ctx, attachment.TaskID, auth); err != nil {
		return nil, toGRPCError(err)
	}

	// Только загрузивший или admin/teacher может удалять
	if attachment.UploadedBy != auth.UserID && auth.Role != "teacher" && auth.Role != "admin" {
		return nil, status.Error(codes.PermissionDenied, "can only remove own attachments")
	}

	if err := h.service.RemoveAttachment(ctx, req.AttachmentId); err != nil {
		return nil, status.Errorf(codes.Internal, "remove attachment: %v", err)
	}

	return &emptypb.Empty{}, nil
}

func (h *Handler) ListAttachments(ctx context.Context, req *taskv1.ListAttachmentsRequest) (*taskv1.ListAttachmentsResponse, error) {
	auth, ok := GetAuthContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "unauthorized")
	}

	if err := h.accessChecker.CheckTaskAccess(ctx, req.TaskId, auth); err != nil {
		return nil, toGRPCError(err)
	}

	attachments, err := h.service.ListAttachments(ctx, req.TaskId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list attachments: %v", err)
	}

	out := &taskv1.ListAttachmentsResponse{}
	for _, a := range attachments {
		out.Attachments = append(out.Attachments, h.attachmentToProto(a))
	}
	h.enrichAttachments(ctx, out.Attachments)
	return out, nil
}

func (h *Handler) GetTaskActivity(ctx context.Context, req *taskv1.GetTaskActivityRequest) (*taskv1.GetTaskActivityResponse, error) {
	auth, ok := GetAuthContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "unauthorized")
	}

	if err := h.accessChecker.CheckTaskAccess(ctx, req.TaskId, auth); err != nil {
		return nil, toGRPCError(err)
	}

	limit := int(req.PageSize)
	if limit <= 0 {
		limit = 20
	}
	offset := int((req.Page - 1) * req.PageSize)
	if offset < 0 {
		offset = 0
	}

	activities, total, err := h.service.ListActivity(ctx, req.TaskId, limit, offset)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "get activity: %v", err)
	}

	out := &taskv1.GetTaskActivityResponse{
		TotalCount: total,
	}
	for _, a := range activities {
		out.Activities = append(out.Activities, h.activityToProto(a))
	}
	h.enrichActivities(ctx, out.Activities)
	return out, nil
}

func (h *Handler) AddWatcher(ctx context.Context, req *taskv1.AddWatcherRequest) (*emptypb.Empty, error) {
	auth, ok := GetAuthContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "unauthorized")
	}

	if err := h.accessChecker.CheckTaskAccess(ctx, req.TaskId, auth); err != nil {
		return nil, toGRPCError(err)
	}

	// Пользователь может добавить только себя (или admin/teacher может добавить любого)
	if req.UserId != auth.UserID && auth.Role != "teacher" && auth.Role != "admin" {
		return nil, status.Error(codes.PermissionDenied, "can only add yourself as watcher")
	}

	if err := h.service.AddWatcher(ctx, req.TaskId, req.UserId); err != nil {
		return nil, status.Errorf(codes.Internal, "add watcher: %v", err)
	}

	return &emptypb.Empty{}, nil
}

func (h *Handler) RemoveWatcher(ctx context.Context, req *taskv1.RemoveWatcherRequest) (*emptypb.Empty, error) {
	auth, ok := GetAuthContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "unauthorized")
	}

	if err := h.accessChecker.CheckTaskAccess(ctx, req.TaskId, auth); err != nil {
		return nil, toGRPCError(err)
	}

	// Пользователь может удалить только себя
	if req.UserId != auth.UserID && auth.Role != "teacher" && auth.Role != "admin" {
		return nil, status.Error(codes.PermissionDenied, "can only remove yourself as watcher")
	}

	if err := h.service.RemoveWatcher(ctx, req.TaskId, req.UserId); err != nil {
		return nil, status.Errorf(codes.Internal, "remove watcher: %v", err)
	}

	return &emptypb.Empty{}, nil
}

func (h *Handler) ListWatchers(ctx context.Context, req *taskv1.ListWatchersRequest) (*taskv1.ListWatchersResponse, error) {
	auth, ok := GetAuthContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "unauthorized")
	}

	if err := h.accessChecker.CheckTaskAccess(ctx, req.TaskId, auth); err != nil {
		return nil, toGRPCError(err)
	}

	watchers, err := h.service.ListWatchers(ctx, req.TaskId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list watchers: %v", err)
	}

	out := &taskv1.ListWatchersResponse{}
	for _, w := range watchers {
		out.Watchers = append(out.Watchers, h.userPreviewToProto(w))
	}
	return out, nil
}

func (h *Handler) GetBoardStats(ctx context.Context, req *taskv1.GetBoardStatsRequest) (*taskv1.GetBoardStatsResponse, error) {
	auth, ok := GetAuthContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "unauthorized")
	}

	if err := h.accessChecker.CheckBoardAccess(ctx, req.BoardId, auth); err != nil {
		return nil, toGRPCError(err)
	}

	stats, err := h.service.GetBoardStats(ctx, req.BoardId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "get stats: %v", err)
	}

	resp := &taskv1.GetBoardStatsResponse{
		Stats: h.statsToProto(stats),
	}

	if req.IncludeMemberStats {
		memberStats, _ := h.service.GetMemberStats(ctx, req.BoardId)
		for _, ms := range memberStats {
			resp.MemberStats = append(resp.MemberStats, h.memberStatsToProto(ms))
		}
	}

	if req.IncludeDailyStats {
		dailyStats, _ := h.service.GetDailyStats(ctx, req.BoardId, 14)
		for _, ds := range dailyStats {
			resp.DailyStats = append(resp.DailyStats, h.dailyStatsToProto(ds))
		}
	}

	return resp, nil
}

func (h *Handler) GetMyTasks(ctx context.Context, req *taskv1.GetMyTasksRequest) (*taskv1.ListTasksResponse, error) {
	auth, ok := GetAuthContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "unauthorized")
	}

	// Используем auth.UserID вместо req.UserId
	tasks, total, err := h.service.GetMyTasks(ctx, auth.UserID, MyTasksFilter{
		OnlyAssigned:     req.OnlyAssigned,
		OnlyCreated:      req.OnlyCreated,
		OnlyWatching:     req.OnlyWatching,
		IncludeCompleted: req.IncludeCompleted,
		Limit:            int(req.PageSize),
		Offset:           int((req.Page - 1) * req.PageSize),
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "get my tasks: %v", err)
	}

	out := &taskv1.ListTasksResponse{
		TotalCount: total,
	}
	for _, t := range tasks {
		out.Tasks = append(out.Tasks, h.taskToProto(t))
	}
	h.enrichTasks(ctx, out.Tasks)
	return out, nil
}

func (h *Handler) GetOverdueTasks(ctx context.Context, req *taskv1.GetOverdueTasksRequest) (*taskv1.ListTasksResponse, error) {
	auth, ok := GetAuthContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "unauthorized")
	}

	if req.BoardId > 0 {
		if err := h.accessChecker.CheckBoardAccess(ctx, req.BoardId, auth); err != nil {
			return nil, toGRPCError(err)
		}
	}

	var assigneeID *int64
	if req.AssigneeId > 0 {
		assigneeID = &req.AssigneeId
	}

	limit := int(req.PageSize)
	if limit <= 0 {
		limit = 20
	}
	offset := int((req.Page - 1) * req.PageSize)
	if offset < 0 {
		offset = 0
	}

	tasks, total, err := h.service.GetOverdueTasks(ctx, req.BoardId, assigneeID, limit, offset)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "get overdue tasks: %v", err)
	}

	out := &taskv1.ListTasksResponse{
		TotalCount: total,
	}
	for _, t := range tasks {
		out.Tasks = append(out.Tasks, h.taskToProto(t))
	}
	h.enrichTasks(ctx, out.Tasks)
	return out, nil
}

func (h *Handler) GetUpcomingDeadlines(ctx context.Context, req *taskv1.GetUpcomingDeadlinesRequest) (*taskv1.ListTasksResponse, error) {
	auth, ok := GetAuthContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "unauthorized")
	}

	if req.BoardId > 0 {
		if err := h.accessChecker.CheckBoardAccess(ctx, req.BoardId, auth); err != nil {
			return nil, toGRPCError(err)
		}
	}

	var userID *int64
	if req.UserId > 0 {
		userID = &req.UserId
	}

	daysAhead := int(req.DaysAhead)
	if daysAhead <= 0 {
		daysAhead = 7
	}

	limit := int(req.PageSize)
	if limit <= 0 {
		limit = 20
	}
	offset := int((req.Page - 1) * req.PageSize)
	if offset < 0 {
		offset = 0
	}

	tasks, total, err := h.service.GetUpcomingDeadlines(ctx, req.BoardId, userID, daysAhead, limit, offset)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "get upcoming deadlines: %v", err)
	}

	out := &taskv1.ListTasksResponse{
		TotalCount: total,
	}
	for _, t := range tasks {
		out.Tasks = append(out.Tasks, h.taskToProto(t))
	}
	h.enrichTasks(ctx, out.Tasks)
	return out, nil
}

func (h *Handler) StartBackgroundJobs(ctx context.Context) {
	h.service.StartBackgroundJobs(ctx)
}

func toGRPCError(err error) error {
	switch {
	case errors.Is(err, ErrBoardNotFound):
		return status.Error(codes.NotFound, "board not found")
	case errors.Is(err, ErrTaskNotFound):
		return status.Error(codes.NotFound, "task not found")
	case errors.Is(err, ErrColumnNotFound):
		return status.Error(codes.NotFound, "column not found")
	case errors.Is(err, ErrNotTeamMember):
		return status.Error(codes.PermissionDenied, "you are not a member of this team")
	case errors.Is(err, ErrCrossUniversity):
		return status.Error(codes.PermissionDenied, "access denied: different university")
	case errors.Is(err, ErrCrossDepartment):
		return status.Error(codes.PermissionDenied, "access denied: different department")
	case errors.Is(err, ErrAccessDenied):
		return status.Error(codes.PermissionDenied, "access denied")
	default:
		return status.Error(codes.PermissionDenied, err.Error())
	}
}

func (h *Handler) boardToProto(b *Board) *taskv1.Board {
	if b == nil {
		return nil
	}
	pb := &taskv1.Board{
		Id:          b.ID,
		TeamId:      b.TeamID,
		ProjectId:   b.ProjectID,
		Name:        b.Name,
		Description: b.Description,
		CreatedBy:   b.CreatedBy,
		CreatedAt:   timestamppb.New(b.CreatedAt),
		UpdatedAt:   timestamppb.New(b.UpdatedAt),
	}
	if len(b.Settings) > 0 {
		var s BoardSettings
		if err := json.Unmarshal(b.Settings, &s); err == nil {
			pb.Settings = &taskv1.BoardSettings{
				DefaultColumn:      s.DefaultColumn,
				AllowCustomColumns: s.AllowCustomColumns,
				ShowCompleted:      s.ShowCompleted,
				Labels:             s.Labels,
			}
		}
	}
	for _, col := range b.Columns {
		pb.Columns = append(pb.Columns, h.columnToProto(&col))
	}
	if b.Stats != nil {
		pb.Stats = h.statsToProto(b.Stats)
	}
	return pb
}

func (h *Handler) columnToProto(c *Column) *taskv1.Column {
	if c == nil {
		return nil
	}
	return &taskv1.Column{
		Id:           c.ID,
		BoardId:      c.BoardID,
		Name:         c.Name,
		Slug:         c.Slug,
		Description:  c.Description,
		Color:        c.Color,
		Icon:         c.Icon,
		OrderIndex:   c.OrderIndex,
		WipLimit:     c.WIPLimit,
		IsDefault:    c.IsDefault,
		IsDoneColumn: c.IsDoneColumn,
		TaskCount:    c.TaskCount,
		CreatedAt:    timestamppb.New(c.CreatedAt),
		UpdatedAt:    timestamppb.New(c.UpdatedAt),
	}
}

func (h *Handler) taskToProto(t *Task) *taskv1.Task {
	if t == nil {
		return nil
	}
	pb := &taskv1.Task{
		Id:               t.ID,
		BoardId:          t.BoardID,
		ColumnId:         t.ColumnID,
		Title:            t.Title,
		Description:      t.Description,
		Status:           h.statusToProto(t.Status),
		Priority:         h.priorityToProto(t.Priority),
		EstimatedMinutes: t.EstimatedMinutes,
		ActualMinutes:    t.ActualMinutes,
		Position:         t.Position,
		CommentsCount:    t.CommentsCount,
		AttachmentsCount: t.AttachmentsCount,
		WatchersCount:    t.WatchersCount,
		IsOverdue:        t.IsOverdue,
		CreatedAt:        timestamppb.New(t.CreatedAt),
		UpdatedAt:        timestamppb.New(t.UpdatedAt),
	}

	if t.AssigneeID != nil {
		pb.Assignee = &taskv1.UserPreview{Id: *t.AssigneeID}
	}
	pb.CreatedBy = &taskv1.UserPreview{Id: t.CreatedBy}

	if t.DueDate != nil {
		pb.DueDate = timestamppb.New(*t.DueDate)
	}
	if t.StartedAt != nil {
		pb.StartedAt = timestamppb.New(*t.StartedAt)
	}
	if t.CompletedAt != nil {
		pb.CompletedAt = timestamppb.New(*t.CompletedAt)
	}
	if t.WorkflowStepID != nil {
		pb.WorkflowStepId = *t.WorkflowStepID
	}

	// Labels из JSON
	if t.Labels != nil {
		var labels []string
		_ = t.Labels.Scan(&labels)
		pb.Labels = labels
	}

	return pb
}

func (h *Handler) commentToProto(c *Comment) *taskv1.Comment {
	if c == nil {
		return nil
	}
	pb := &taskv1.Comment{
		Id:        c.ID,
		TaskId:    c.TaskID,
		Author:    &taskv1.UserPreview{Id: c.AuthorID},
		Content:   c.Content,
		CreatedAt: timestamppb.New(c.CreatedAt),
		UpdatedAt: timestamppb.New(c.UpdatedAt),
	}
	if c.EditedAt != nil {
		pb.EditedAt = timestamppb.New(*c.EditedAt)
	}
	return pb
}

func (h *Handler) attachmentToProto(a *Attachment) *taskv1.Attachment {
	if a == nil {
		return nil
	}
	return &taskv1.Attachment{
		Id:         a.ID,
		TaskId:     a.TaskID,
		FileId:     a.FileID,
		FileName:   a.FileName,
		FileType:   a.FileType,
		FileSize:   a.FileSize,
		UploadedBy: &taskv1.UserPreview{Id: a.UploadedBy},
		CreatedAt:  timestamppb.New(a.CreatedAt),
	}
}

func (h *Handler) activityToProto(a *ActivityLog) *taskv1.ActivityLogEntry {
	if a == nil {
		return nil
	}
	return &taskv1.ActivityLogEntry{
		Id:        a.ID,
		TaskId:    a.TaskID,
		Actor:     &taskv1.UserPreview{Id: a.ActorID},
		Action:    a.Action,
		FieldName: a.FieldName,
		OldValue:  a.OldValue,
		NewValue:  a.NewValue,
		CreatedAt: timestamppb.New(a.CreatedAt),
	}
}

func (h *Handler) userPreviewToProto(u *UserPreview) *taskv1.UserPreview {
	if u == nil {
		return nil
	}
	return &taskv1.UserPreview{
		Id:        u.ID,
		FullName:  u.FullName,
		Email:     u.Email,
		AvatarUrl: u.AvatarURL,
	}
}

func (h *Handler) statsToProto(s *BoardStats) *taskv1.BoardStats {
	if s == nil {
		return nil
	}
	pb := &taskv1.BoardStats{
		TotalTasks:           s.TotalTasks,
		CompletedTasks:       s.CompletedTasks,
		OverdueTasks:         s.OverdueTasks,
		TasksWithoutAssignee: s.TasksWithoutAssignee,
	}
	if s.TasksByStatus != nil {
		pb.TasksByStatus = make(map[string]int32)
		for k, v := range s.TasksByStatus {
			pb.TasksByStatus[k] = v
		}
	}
	if s.TasksByPriority != nil {
		pb.TasksByPriority = make(map[string]int32)
		for k, v := range s.TasksByPriority {
			pb.TasksByPriority[k] = v
		}
	}
	return pb
}

func (h *Handler) memberStatsToProto(ms *MemberStats) *taskv1.MemberStats {
	if ms == nil {
		return nil
	}
	return &taskv1.MemberStats{
		User:            h.userPreviewToProto(ms.User),
		AssignedTasks:   ms.AssignedTasks,
		CompletedTasks:  ms.CompletedTasks,
		OverdueTasks:    ms.OverdueTasks,
		InProgressTasks: ms.InProgressTasks,
	}
}

func (h *Handler) dailyStatsToProto(ds *DailyStats) *taskv1.DailyStats {
	if ds == nil {
		return nil
	}
	return &taskv1.DailyStats{
		Date:      ds.Date,
		Created:   ds.Created,
		Completed: ds.Completed,
		Moved:     ds.Moved,
	}
}

// Priority converters
func (h *Handler) priorityToProto(p string) taskv1.TaskPriority {
	switch p {
	case TaskPriorityLow:
		return taskv1.TaskPriority_TASK_PRIORITY_LOW
	case TaskPriorityMedium:
		return taskv1.TaskPriority_TASK_PRIORITY_MEDIUM
	case TaskPriorityHigh:
		return taskv1.TaskPriority_TASK_PRIORITY_HIGH
	case TaskPriorityUrgent:
		return taskv1.TaskPriority_TASK_PRIORITY_URGENT
	default:
		return taskv1.TaskPriority_TASK_PRIORITY_UNSPECIFIED
	}
}

func (h *Handler) priorityFromProto(p taskv1.TaskPriority) string {
	switch p {
	case taskv1.TaskPriority_TASK_PRIORITY_LOW:
		return TaskPriorityLow
	case taskv1.TaskPriority_TASK_PRIORITY_MEDIUM:
		return TaskPriorityMedium
	case taskv1.TaskPriority_TASK_PRIORITY_HIGH:
		return TaskPriorityHigh
	case taskv1.TaskPriority_TASK_PRIORITY_URGENT:
		return TaskPriorityUrgent
	default:
		return TaskPriorityMedium
	}
}

func (h *Handler) priorityPtrFromProto(p taskv1.TaskPriority) *string {
	if p == taskv1.TaskPriority_TASK_PRIORITY_UNSPECIFIED {
		return nil
	}
	s := h.priorityFromProto(p)
	return &s
}

func (h *Handler) priorityStringFromProto(p taskv1.TaskPriority) string {
	if p == taskv1.TaskPriority_TASK_PRIORITY_UNSPECIFIED {
		return ""
	}
	return h.priorityFromProto(p)
}

// Status converters
func (h *Handler) statusToProto(s string) taskv1.TaskStatus {
	switch s {
	case TaskStatusTodo:
		return taskv1.TaskStatus_TASK_STATUS_TODO
	case TaskStatusInProgress:
		return taskv1.TaskStatus_TASK_STATUS_IN_PROGRESS
	case TaskStatusReview:
		return taskv1.TaskStatus_TASK_STATUS_REVIEW
	case TaskStatusDone:
		return taskv1.TaskStatus_TASK_STATUS_DONE
	default:
		return taskv1.TaskStatus_TASK_STATUS_UNSPECIFIED
	}
}

func (h *Handler) statusFromProto(s taskv1.TaskStatus) string {
	switch s {
	case taskv1.TaskStatus_TASK_STATUS_TODO:
		return TaskStatusTodo
	case taskv1.TaskStatus_TASK_STATUS_IN_PROGRESS:
		return TaskStatusInProgress
	case taskv1.TaskStatus_TASK_STATUS_REVIEW:
		return TaskStatusReview
	case taskv1.TaskStatus_TASK_STATUS_DONE:
		return TaskStatusDone
	default:
		return ""
	}
}
