package task

import (
	"context"
	"encoding/json"
	"time"

	taskv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/task/v1"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Handler - gRPC handler для TaskService
type Handler struct {
	taskv1.UnimplementedTaskServiceServer
	service *Service
	logger  *zap.Logger
}

// NewHandler создает новый handler
func NewHandler(service *Service, logger *zap.Logger) *Handler {
	return &Handler{
		service: service,
		logger:  logger,
	}
}

// ==================== Board ====================

func (h *Handler) GetBoard(ctx context.Context, req *taskv1.GetBoardRequest) (*taskv1.GetBoardResponse, error) {
	board, err := h.service.GetBoard(ctx, req.BoardId, req.IncludeColumns, req.IncludeStats)
	if err != nil {
		h.logger.Error("GetBoard failed", zap.Error(err), zap.Int64("board_id", req.BoardId))
		return nil, status.Errorf(codes.NotFound, "board not found: %v", err)
	}

	return &taskv1.GetBoardResponse{
		Board: h.boardToProto(board),
	}, nil
}

func (h *Handler) GetBoardByTeam(ctx context.Context, req *taskv1.GetBoardByTeamRequest) (*taskv1.GetBoardResponse, error) {
	board, err := h.service.GetBoardByTeam(ctx, req.TeamId, req.IncludeColumns, req.IncludeStats)
	if err != nil {
		h.logger.Error("GetBoardByTeam failed", zap.Error(err), zap.Int64("team_id", req.TeamId))
		return nil, status.Errorf(codes.NotFound, "board not found for team: %v", err)
	}

	return &taskv1.GetBoardResponse{
		Board: h.boardToProto(board),
	}, nil
}

func (h *Handler) CreateBoard(ctx context.Context, req *taskv1.CreateBoardRequest) (*taskv1.Board, error) {
	board, err := h.service.CreateBoard(ctx, &CreateBoardInput{
		TeamID:               req.TeamId,
		ProjectID:            req.ProjectId,
		Name:                 req.Name,
		Description:          req.Description,
		CreatedBy:            req.CreatedBy,
		CreateDefaultColumns: req.CreateDefaultColumns,
	})
	if err != nil {
		h.logger.Error("CreateBoard failed", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "failed to create board: %v", err)
	}

	return h.boardToProto(board), nil
}

func (h *Handler) UpdateBoard(ctx context.Context, req *taskv1.UpdateBoardRequest) (*taskv1.Board, error) {
	var settings *BoardSettings
	if req.Settings != nil {
		settings = &BoardSettings{
			DefaultColumn:      req.Settings.DefaultColumn,
			AllowCustomColumns: req.Settings.AllowCustomColumns,
			ShowCompleted:      req.Settings.ShowCompleted,
			Labels:             req.Settings.Labels,
		}
	}

	board, err := h.service.UpdateBoard(ctx, req.BoardId, req.Name, req.Description, settings)
	if err != nil {
		h.logger.Error("UpdateBoard failed", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "failed to update board: %v", err)
	}

	return h.boardToProto(board), nil
}

// ==================== Columns ====================

func (h *Handler) ListColumns(ctx context.Context, req *taskv1.ListColumnsRequest) (*taskv1.ListColumnsResponse, error) {
	columns, err := h.service.ListColumns(ctx, req.BoardId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list columns: %v", err)
	}

	var pbColumns []*taskv1.Column
	for _, col := range columns {
		pbColumns = append(pbColumns, h.columnToProto(col))
	}

	return &taskv1.ListColumnsResponse{
		Columns: pbColumns,
	}, nil
}

func (h *Handler) CreateColumn(ctx context.Context, req *taskv1.CreateColumnRequest) (*taskv1.Column, error) {
	column, err := h.service.CreateColumn(ctx, &CreateColumnInput{
		BoardID:      req.BoardId,
		Name:         req.Name,
		Slug:         req.Slug,
		Description:  req.Description,
		Color:        req.Color,
		WIPLimit:     req.WipLimit,
		IsDoneColumn: req.IsDoneColumn,
	})
	if err != nil {
		h.logger.Error("CreateColumn failed", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "failed to create column: %v", err)
	}

	return h.columnToProto(column), nil
}

func (h *Handler) UpdateColumn(ctx context.Context, req *taskv1.UpdateColumnRequest) (*taskv1.Column, error) {
	column, err := h.service.UpdateColumn(ctx, req.ColumnId, req.Name, req.Description, req.Color, req.WipLimit)
	if err != nil {
		h.logger.Error("UpdateColumn failed", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "failed to update column: %v", err)
	}

	return h.columnToProto(column), nil
}

func (h *Handler) DeleteColumn(ctx context.Context, req *taskv1.DeleteColumnRequest) (*emptypb.Empty, error) {
	err := h.service.DeleteColumn(ctx, req.ColumnId, req.MoveTasksToColumnId)
	if err != nil {
		h.logger.Error("DeleteColumn failed", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "failed to delete column: %v", err)
	}

	return &emptypb.Empty{}, nil
}

func (h *Handler) ReorderColumns(ctx context.Context, req *taskv1.ReorderColumnsRequest) (*taskv1.ListColumnsResponse, error) {
	err := h.service.ReorderColumns(ctx, req.BoardId, req.ColumnIds)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to reorder columns: %v", err)
	}

	// Возвращаем обновлённый список
	columns, _ := h.service.ListColumns(ctx, req.BoardId)
	var pbColumns []*taskv1.Column
	for _, col := range columns {
		pbColumns = append(pbColumns, h.columnToProto(col))
	}

	return &taskv1.ListColumnsResponse{
		Columns: pbColumns,
	}, nil
}

// ==================== Tasks ====================

func (h *Handler) CreateTask(ctx context.Context, req *taskv1.CreateTaskRequest) (*taskv1.Task, error) {
	var dueDate *time.Time
	if req.DueDate != nil {
		t := req.DueDate.AsTime()
		dueDate = &t
	}

	task, err := h.service.CreateTask(ctx, &CreateTaskInput{
		BoardID:          req.BoardId,
		Title:            req.Title,
		Description:      req.Description,
		Priority:         h.priorityFromProto(req.Priority),
		AssigneeID:       req.AssigneeId,
		DueDate:          dueDate,
		EstimatedMinutes: req.EstimatedMinutes,
		Labels:           req.Labels,
		ColumnID:         req.ColumnId,
		CreatedBy:        req.CreatedBy,
		WorkflowStepID:   req.WorkflowStepId,
	})
	if err != nil {
		h.logger.Error("CreateTask failed", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "failed to create task: %v", err)
	}

	return h.taskToProto(task), nil
}

func (h *Handler) GetTask(ctx context.Context, req *taskv1.GetTaskRequest) (*taskv1.GetTaskResponse, error) {
	task, comments, attachments, activity, watchers, err := h.service.GetTask(ctx, req.TaskId)
	if err != nil {
		h.logger.Error("GetTask failed", zap.Error(err), zap.Int64("task_id", req.TaskId))
		return nil, status.Errorf(codes.NotFound, "task not found: %v", err)
	}

	// Конвертируем комментарии
	var pbComments []*taskv1.Comment
	for _, c := range comments {
		pbComments = append(pbComments, h.commentToProto(c))
	}

	// Конвертируем вложения
	var pbAttachments []*taskv1.Attachment
	for _, a := range attachments {
		pbAttachments = append(pbAttachments, h.attachmentToProto(a))
	}

	// Конвертируем активность
	var pbActivity []*taskv1.ActivityLogEntry
	for _, act := range activity {
		pbActivity = append(pbActivity, h.activityToProto(act))
	}

	// Конвертируем watchers
	var pbWatchers []*taskv1.UserPreview
	for _, w := range watchers {
		pbWatchers = append(pbWatchers, &taskv1.UserPreview{
			Id: w.UserID,
			// TODO: загрузить данные пользователя
		})
	}

	return &taskv1.GetTaskResponse{
		Task:           h.taskToProto(task),
		RecentComments: pbComments,
		Attachments:    pbAttachments,
		RecentActivity: pbActivity,
		Watchers:       pbWatchers,
	}, nil
}

func (h *Handler) UpdateTask(ctx context.Context, req *taskv1.UpdateTaskRequest) (*taskv1.Task, error) {
	var dueDate *time.Time
	if req.DueDate != nil {
		t := req.DueDate.AsTime()
		dueDate = &t
	}

	task, err := h.service.UpdateTask(ctx, &UpdateTaskInput{
		TaskID:           req.TaskId,
		Title:            req.Title,
		Description:      req.Description,
		Priority:         h.priorityFromProto(req.Priority),
		DueDate:          dueDate,
		EstimatedMinutes: req.EstimatedMinutes,
		ActualMinutes:    req.ActualMinutes,
		Labels:           req.Labels,
		WorkflowStepID:   req.WorkflowStepId,
		UpdatedBy:        req.UpdatedBy,
	})
	if err != nil {
		h.logger.Error("UpdateTask failed", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "failed to update task: %v", err)
	}

	return h.taskToProto(task), nil
}

func (h *Handler) DeleteTask(ctx context.Context, req *taskv1.DeleteTaskRequest) (*emptypb.Empty, error) {
	err := h.service.DeleteTask(ctx, req.TaskId, req.DeletedBy)
	if err != nil {
		h.logger.Error("DeleteTask failed", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "failed to delete task: %v", err)
	}

	return &emptypb.Empty{}, nil
}

func (h *Handler) ListTasks(ctx context.Context, req *taskv1.ListTasksRequest) (*taskv1.ListTasksResponse, error) {
	filter := TaskFilter{
		BoardID:        req.BoardId,
		ColumnID:       req.ColumnId,
		AssigneeID:     req.AssigneeId,
		Status:         h.statusFromProto(req.Status),
		Priority:       h.priorityFromProto(req.Priority),
		Search:         req.Search,
		Labels:         req.Labels,
		OnlyOverdue:    req.OnlyOverdue,
		OnlyUnassigned: req.OnlyUnassigned,
		SortBy:         req.SortBy,
		SortOrder:      req.SortOrder,
		Limit:          int(req.PageSize),
		Offset:         int((req.Page - 1) * req.PageSize),
	}

	if filter.Limit <= 0 {
		filter.Limit = 20
	}
	if filter.Offset < 0 {
		filter.Offset = 0
	}

	tasks, total, err := h.service.ListTasks(ctx, filter)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list tasks: %v", err)
	}

	var pbTasks []*taskv1.Task
	for _, t := range tasks {
		pbTasks = append(pbTasks, h.taskToProto(t))
	}

	return &taskv1.ListTasksResponse{
		Tasks:      pbTasks,
		TotalCount: total,
	}, nil
}

// ==================== Kanban Operations ====================

func (h *Handler) MoveTask(ctx context.Context, req *taskv1.MoveTaskRequest) (*taskv1.Task, error) {
	task, err := h.service.MoveTask(ctx, req.TaskId, req.ToColumnId, req.Position, req.MovedBy)
	if err != nil {
		h.logger.Error("MoveTask failed", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "failed to move task: %v", err)
	}

	return h.taskToProto(task), nil
}

func (h *Handler) ReorderTasks(ctx context.Context, req *taskv1.ReorderTasksRequest) (*emptypb.Empty, error) {
	err := h.service.ReorderTasks(ctx, req.ColumnId, req.TaskIds)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to reorder tasks: %v", err)
	}

	return &emptypb.Empty{}, nil
}

// ==================== Bulk Operations ====================

func (h *Handler) BulkUpdateTasks(ctx context.Context, req *taskv1.BulkUpdateTasksRequest) (*taskv1.BulkUpdateTasksResponse, error) {
	tasks, err := h.service.BulkUpdateTasks(ctx, &BulkUpdateTasksInput{
		TaskIDs:      req.TaskIds,
		AssigneeID:   req.AssigneeId,
		Priority:     h.priorityFromProto(req.Priority),
		ColumnID:     req.MoveToColumnId,
		AddLabels:    req.AddLabels,
		RemoveLabels: req.RemoveLabels,
		UpdatedBy:    req.UpdatedBy,
	})
	if err != nil {
		h.logger.Error("BulkUpdateTasks failed", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "failed to bulk update tasks: %v", err)
	}

	var pbTasks []*taskv1.Task
	for _, t := range tasks {
		pbTasks = append(pbTasks, h.taskToProto(t))
	}

	return &taskv1.BulkUpdateTasksResponse{
		UpdatedCount: int32(len(tasks)),
		UpdatedTasks: pbTasks,
	}, nil
}

func (h *Handler) BulkDeleteTasks(ctx context.Context, req *taskv1.BulkDeleteTasksRequest) (*emptypb.Empty, error) {
	err := h.service.BulkDeleteTasks(ctx, req.TaskIds, req.DeletedBy)
	if err != nil {
		h.logger.Error("BulkDeleteTasks failed", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "failed to bulk delete tasks: %v", err)
	}

	return &emptypb.Empty{}, nil
}

// ==================== Assignment ====================

func (h *Handler) AssignTask(ctx context.Context, req *taskv1.AssignTaskRequest) (*taskv1.Task, error) {
	task, err := h.service.AssignTask(ctx, req.TaskId, req.AssigneeId, req.AssignedBy)
	if err != nil {
		h.logger.Error("AssignTask failed", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "failed to assign task: %v", err)
	}

	return h.taskToProto(task), nil
}

func (h *Handler) UnassignTask(ctx context.Context, req *taskv1.UnassignTaskRequest) (*taskv1.Task, error) {
	task, err := h.service.UnassignTask(ctx, req.TaskId, req.UnassignedBy)
	if err != nil {
		h.logger.Error("UnassignTask failed", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "failed to unassign task: %v", err)
	}

	return h.taskToProto(task), nil
}

// ==================== Comments ====================

func (h *Handler) CreateComment(ctx context.Context, req *taskv1.CreateCommentRequest) (*taskv1.Comment, error) {
	comment, err := h.service.CreateComment(ctx, &CreateCommentInput{
		TaskID:         req.TaskId,
		AuthorID:       req.AuthorId,
		Content:        req.Content,
		MentionUserIDs: req.MentionUserIds,
	})
	if err != nil {
		h.logger.Error("CreateComment failed", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "failed to create comment: %v", err)
	}

	return h.commentToProto(comment), nil
}

func (h *Handler) UpdateComment(ctx context.Context, req *taskv1.UpdateCommentRequest) (*taskv1.Comment, error) {
	comment, err := h.service.UpdateComment(ctx, req.CommentId, req.Content, req.UpdatedBy)
	if err != nil {
		h.logger.Error("UpdateComment failed", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "failed to update comment: %v", err)
	}

	return h.commentToProto(comment), nil
}

func (h *Handler) DeleteComment(ctx context.Context, req *taskv1.DeleteCommentRequest) (*emptypb.Empty, error) {
	err := h.service.DeleteComment(ctx, req.CommentId, req.DeletedBy)
	if err != nil {
		h.logger.Error("DeleteComment failed", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "failed to delete comment: %v", err)
	}

	return &emptypb.Empty{}, nil
}

func (h *Handler) ListComments(ctx context.Context, req *taskv1.ListCommentsRequest) (*taskv1.ListCommentsResponse, error) {
	comments, total, err := h.service.ListComments(ctx, req.TaskId, req.Page, req.PageSize)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list comments: %v", err)
	}

	var pbComments []*taskv1.Comment
	for _, c := range comments {
		pbComments = append(pbComments, h.commentToProto(c))
	}

	return &taskv1.ListCommentsResponse{
		Comments:   pbComments,
		TotalCount: total,
	}, nil
}

// ==================== Attachments ====================

func (h *Handler) AddAttachment(ctx context.Context, req *taskv1.AddAttachmentRequest) (*taskv1.Attachment, error) {
	attachment, err := h.service.AddAttachment(ctx, &AddAttachmentInput{
		TaskID:     req.TaskId,
		FileID:     req.FileId,
		FileName:   req.FileName,
		FileType:   req.FileType,
		FileSize:   req.FileSize,
		UploadedBy: req.UploadedBy,
	})
	if err != nil {
		h.logger.Error("AddAttachment failed", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "failed to add attachment: %v", err)
	}

	return h.attachmentToProto(attachment), nil
}

func (h *Handler) RemoveAttachment(ctx context.Context, req *taskv1.RemoveAttachmentRequest) (*emptypb.Empty, error) {
	err := h.service.RemoveAttachment(ctx, req.AttachmentId, req.RemovedBy)
	if err != nil {
		h.logger.Error("RemoveAttachment failed", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "failed to remove attachment: %v", err)
	}

	return &emptypb.Empty{}, nil
}

func (h *Handler) ListAttachments(ctx context.Context, req *taskv1.ListAttachmentsRequest) (*taskv1.ListAttachmentsResponse, error) {
	attachments, err := h.service.ListAttachments(ctx, req.TaskId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list attachments: %v", err)
	}

	var pbAttachments []*taskv1.Attachment
	for _, a := range attachments {
		pbAttachments = append(pbAttachments, h.attachmentToProto(a))
	}

	return &taskv1.ListAttachmentsResponse{
		Attachments: pbAttachments,
	}, nil
}

// ==================== Activity ====================

func (h *Handler) GetTaskActivity(ctx context.Context, req *taskv1.GetTaskActivityRequest) (*taskv1.GetTaskActivityResponse, error) {
	activities, total, err := h.service.GetTaskActivity(ctx, req.TaskId, req.Page, req.PageSize)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get task activity: %v", err)
	}

	var pbActivities []*taskv1.ActivityLogEntry
	for _, a := range activities {
		pbActivities = append(pbActivities, h.activityToProto(a))
	}

	return &taskv1.GetTaskActivityResponse{
		Activities: pbActivities,
		TotalCount: total,
	}, nil
}

// ==================== Watchers ====================

func (h *Handler) AddWatcher(ctx context.Context, req *taskv1.AddWatcherRequest) (*emptypb.Empty, error) {
	err := h.service.AddWatcher(ctx, req.TaskId, req.UserId)
	if err != nil {
		h.logger.Error("AddWatcher failed", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "failed to add watcher: %v", err)
	}

	return &emptypb.Empty{}, nil
}

func (h *Handler) RemoveWatcher(ctx context.Context, req *taskv1.RemoveWatcherRequest) (*emptypb.Empty, error) {
	err := h.service.RemoveWatcher(ctx, req.TaskId, req.UserId)
	if err != nil {
		h.logger.Error("RemoveWatcher failed", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "failed to remove watcher: %v", err)
	}

	return &emptypb.Empty{}, nil
}

func (h *Handler) ListWatchers(ctx context.Context, req *taskv1.ListWatchersRequest) (*taskv1.ListWatchersResponse, error) {
	watchers, err := h.service.ListWatchers(ctx, req.TaskId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list watchers: %v", err)
	}

	var pbWatchers []*taskv1.UserPreview
	for _, w := range watchers {
		pbWatchers = append(pbWatchers, &taskv1.UserPreview{
			Id: w.UserID,
			// TODO: загрузить данные пользователя через auth_service
		})
	}

	return &taskv1.ListWatchersResponse{
		Watchers: pbWatchers,
	}, nil
}

// ==================== Dashboard & Stats ====================

// Строка с GetBoardStatsRequest - исправить проверку флагов
func (h *Handler) GetBoardStats(ctx context.Context, req *taskv1.GetBoardStatsRequest) (*taskv1.GetBoardStatsResponse, error) {
	stats, err := h.service.GetBoardStats(ctx, req.BoardId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get board stats: %v", err)
	}

	pbResp := &taskv1.GetBoardStatsResponse{
		Stats: &taskv1.BoardStats{
			TotalTasks:           stats.TotalTasks,
			CompletedTasks:       stats.CompletedTasks,
			OverdueTasks:         stats.OverdueTasks,
			TasksWithoutAssignee: stats.TasksWithoutAssignee,
			TasksByStatus:        stats.TasksByStatus,
			TasksByPriority:      stats.TasksByPriority,
		},
	}

	// Исправление: проверяем req.IncludeMemberStats
	if req.IncludeMemberStats {
		memberStats, _ := h.service.GetMemberStats(ctx, req.BoardId)
		for _, ms := range memberStats {
			pbResp.MemberStats = append(pbResp.MemberStats, &taskv1.MemberStats{
				User: &taskv1.UserPreview{
					Id: ms.User.ID,
				},
				AssignedTasks:   ms.AssignedTasks,
				CompletedTasks:  ms.CompletedTasks,
				OverdueTasks:    ms.OverdueTasks,
				InProgressTasks: ms.InProgressTasks,
			})
		}
	}

	// Исправление: проверяем req.IncludeDailyStats
	if req.IncludeDailyStats {
		dailyStats, _ := h.service.GetDailyStats(ctx, req.BoardId, 14)
		for _, ds := range dailyStats {
			pbResp.DailyStats = append(pbResp.DailyStats, &taskv1.DailyStats{
				Date:      ds.Date,
				Created:   ds.Created,
				Completed: ds.Completed,
			})
		}
	}

	return pbResp, nil
}

func (h *Handler) GetMyTasks(ctx context.Context, req *taskv1.GetMyTasksRequest) (*taskv1.ListTasksResponse, error) {
	filter := MyTasksFilter{
		OnlyAssigned:     req.OnlyAssigned,
		OnlyCreated:      req.OnlyCreated,
		OnlyWatching:     req.OnlyWatching,
		IncludeCompleted: req.IncludeCompleted,
		Limit:            int(req.PageSize),
		Offset:           int((req.Page - 1) * req.PageSize),
	}

	if filter.Limit <= 0 {
		filter.Limit = 20
	}
	if filter.Offset < 0 {
		filter.Offset = 0
	}

	tasks, total, err := h.service.GetMyTasks(ctx, req.UserId, filter)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get my tasks: %v", err)
	}

	var pbTasks []*taskv1.Task
	for _, t := range tasks {
		pbTasks = append(pbTasks, h.taskToProto(t))
	}

	return &taskv1.ListTasksResponse{
		Tasks:      pbTasks,
		TotalCount: total,
	}, nil
}

func (h *Handler) GetOverdueTasks(ctx context.Context, req *taskv1.GetOverdueTasksRequest) (*taskv1.ListTasksResponse, error) {
	var assigneeID *int64
	if req.AssigneeId > 0 {
		assigneeID = &req.AssigneeId
	}

	tasks, total, err := h.service.GetOverdueTasks(ctx, req.BoardId, assigneeID, req.Page, req.PageSize)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get overdue tasks: %v", err)
	}

	var pbTasks []*taskv1.Task
	for _, t := range tasks {
		pbTasks = append(pbTasks, h.taskToProto(t))
	}

	return &taskv1.ListTasksResponse{
		Tasks:      pbTasks,
		TotalCount: total,
	}, nil
}

func (h *Handler) GetUpcomingDeadlines(ctx context.Context, req *taskv1.GetUpcomingDeadlinesRequest) (*taskv1.ListTasksResponse, error) {
	var userID *int64
	if req.UserId > 0 {
		userID = &req.UserId
	}

	daysAhead := int(req.DaysAhead)
	if daysAhead <= 0 {
		daysAhead = 7
	}

	tasks, total, err := h.service.GetUpcomingDeadlines(ctx, req.BoardId, userID, daysAhead, req.Page, req.PageSize)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get upcoming deadlines: %v", err)
	}

	var pbTasks []*taskv1.Task
	for _, t := range tasks {
		pbTasks = append(pbTasks, h.taskToProto(t))
	}

	return &taskv1.ListTasksResponse{
		Tasks:      pbTasks,
		TotalCount: total,
	}, nil
}

// ==================== Converters ====================

func (h *Handler) boardToProto(board *Board) *taskv1.Board {
	if board == nil {
		return nil
	}

	pb := &taskv1.Board{
		Id:          board.ID,
		TeamId:      board.TeamID,
		ProjectId:   board.ProjectID,
		Name:        board.Name,
		Description: board.Description,
		CreatedBy:   board.CreatedBy,
		CreatedAt:   timestamppb.New(board.CreatedAt),
		UpdatedAt:   timestamppb.New(board.UpdatedAt),
	}

	// Парсим настройки
	var settings BoardSettings
	if board.Settings != nil {
		_ = json.Unmarshal(board.Settings, &settings)
		pb.Settings = &taskv1.BoardSettings{
			DefaultColumn:      settings.DefaultColumn,
			AllowCustomColumns: settings.AllowCustomColumns,
			ShowCompleted:      settings.ShowCompleted,
			Labels:             settings.Labels,
		}
	}

	// Конвертируем колонки
	for _, col := range board.Columns {
		pb.Columns = append(pb.Columns, h.columnToProto(&col))
	}

	return pb
}

func (h *Handler) columnToProto(col *Column) *taskv1.Column {
	if col == nil {
		return nil
	}

	return &taskv1.Column{
		Id:           col.ID,
		BoardId:      col.BoardID,
		Name:         col.Name,
		Slug:         col.Slug,
		Description:  col.Description,
		Color:        col.Color,
		Icon:         col.Icon,
		OrderIndex:   col.OrderIndex,
		WipLimit:     col.WIPLimit,
		IsDefault:    col.IsDefault,
		IsDoneColumn: col.IsDoneColumn,
		TaskCount:    col.TaskCount,
		CreatedAt:    timestamppb.New(col.CreatedAt),
		UpdatedAt:    timestamppb.New(col.UpdatedAt),
	}
}

func (h *Handler) taskToProto(task *Task) *taskv1.Task {
	if task == nil {
		return nil
	}

	pb := &taskv1.Task{
		Id:               task.ID,
		BoardId:          task.BoardID,
		ColumnId:         task.ColumnID,
		Title:            task.Title,
		Description:      task.Description,
		Status:           h.statusToProto(task.Status),
		Priority:         h.priorityToProto(task.Priority),
		EstimatedMinutes: task.EstimatedMinutes,
		ActualMinutes:    task.ActualMinutes,
		Position:         task.Position,
		CommentsCount:    task.CommentsCount,
		AttachmentsCount: task.AttachmentsCount,
		WatchersCount:    task.WatchersCount,
		IsOverdue:        task.IsOverdue,
		CreatedAt:        timestamppb.New(task.CreatedAt),
		UpdatedAt:        timestamppb.New(task.UpdatedAt),
	}

	// Assignee
	if task.AssigneeID != nil {
		pb.Assignee = &taskv1.UserPreview{
			Id: *task.AssigneeID,
		}
	}

	// Created By
	pb.CreatedBy = &taskv1.UserPreview{
		Id: task.CreatedBy,
	}

	// Due Date
	if task.DueDate != nil {
		pb.DueDate = timestamppb.New(*task.DueDate)
	}

	// Started At
	if task.StartedAt != nil {
		pb.StartedAt = timestamppb.New(*task.StartedAt)
	}

	// Completed At
	if task.CompletedAt != nil {
		pb.CompletedAt = timestamppb.New(*task.CompletedAt)
	}

	// Workflow Step
	if task.WorkflowStepID != nil {
		pb.WorkflowStepId = *task.WorkflowStepID
	}

	// Labels
	var labels []string
	if task.Labels != nil {
		_ = json.Unmarshal(task.Labels, &labels)
	}
	pb.Labels = labels

	return pb
}

func (h *Handler) commentToProto(comment *Comment) *taskv1.Comment {
	if comment == nil {
		return nil
	}

	pb := &taskv1.Comment{
		Id:      comment.ID,
		TaskId:  comment.TaskID,
		Content: comment.Content,
		Author: &taskv1.UserPreview{
			Id: comment.AuthorID,
		},
		CreatedAt: timestamppb.New(comment.CreatedAt),
		UpdatedAt: timestamppb.New(comment.UpdatedAt),
	}

	if comment.EditedAt != nil {
		pb.EditedAt = timestamppb.New(*comment.EditedAt)
	}

	// Mentions
	var mentions []UserMention
	if comment.Mentions != nil {
		_ = json.Unmarshal(comment.Mentions, &mentions)
		for _, m := range mentions {
			pb.Mentions = append(pb.Mentions, &taskv1.UserMention{
				UserId:   m.UserID,
				Username: m.Username,
				Position: m.Position,
			})
		}
	}

	return pb
}

func (h *Handler) attachmentToProto(attachment *Attachment) *taskv1.Attachment {
	if attachment == nil {
		return nil
	}

	return &taskv1.Attachment{
		Id:       attachment.ID,
		TaskId:   attachment.TaskID,
		FileId:   attachment.FileID,
		FileName: attachment.FileName,
		FileType: attachment.FileType,
		FileSize: attachment.FileSize,
		UploadedBy: &taskv1.UserPreview{
			Id: attachment.UploadedBy,
		},
		CreatedAt: timestamppb.New(attachment.CreatedAt),
	}
}

func (h *Handler) activityToProto(activity *ActivityLog) *taskv1.ActivityLogEntry {
	if activity == nil {
		return nil
	}

	return &taskv1.ActivityLogEntry{
		Id:        activity.ID,
		TaskId:    activity.TaskID,
		Action:    activity.Action,
		FieldName: activity.FieldName,
		OldValue:  activity.OldValue,
		NewValue:  activity.NewValue,
		Actor: &taskv1.UserPreview{
			Id: activity.ActorID,
		},
		CreatedAt: timestamppb.New(activity.CreatedAt),
	}
}

// ==================== Enum Converters ====================

func (h *Handler) statusToProto(status string) taskv1.TaskStatus {
	switch status {
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

func (h *Handler) statusFromProto(status taskv1.TaskStatus) string {
	switch status {
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

func (h *Handler) priorityToProto(priority string) taskv1.TaskPriority {
	switch priority {
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

func (h *Handler) priorityFromProto(priority taskv1.TaskPriority) string {
	switch priority {
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
