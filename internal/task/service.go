package task

import (
	"context"
	"fmt"
	"time"

	notificationv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/notification/v1"
	teamv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/team/v1"
	"go.uber.org/zap"
)

type Service struct {
	repo               Repository // ✅ интерфейс, не *repository
	teamClient         teamv1.TeamServiceClient
	notificationClient notificationv1.NotificationServiceClient
	logger             *zap.Logger
}

func NewService(
	repo Repository,
	teamClient teamv1.TeamServiceClient,
	notificationClient notificationv1.NotificationServiceClient,
	logger *zap.Logger,
) *Service {
	return &Service{
		repo:               repo,
		teamClient:         teamClient,
		notificationClient: notificationClient,
		logger:             logger,
	}
}

// ==================== BOARD ====================

func (s *Service) CreateBoard(ctx context.Context, input *CreateBoardInput) (*Board, error) {
	board := &Board{
		TeamID:      input.TeamID,
		ProjectID:   input.ProjectID,
		Name:        input.Name,
		Description: input.Description,
		CreatedBy:   input.CreatedBy,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := s.repo.CreateBoard(ctx, board); err != nil {
		return nil, err
	}

	// Создаём дефолтные колонки
	defaultColumns := []struct {
		Name         string
		Slug         string
		Color        string
		Icon         string
		IsDefault    bool
		IsDoneColumn bool
	}{
		{Name: "To Do", Slug: "todo", Color: "#6B7280", Icon: "📋", IsDefault: true},
		{Name: "In Progress", Slug: "in_progress", Color: "#3B82F6", Icon: "🔄"},
		{Name: "Review", Slug: "review", Color: "#F59E0B", Icon: "👀"},
		{Name: "Done", Slug: "done", Color: "#10B981", Icon: "✅", IsDoneColumn: true},
	}

	for i, dc := range defaultColumns {
		col := &Column{
			BoardID:      board.ID,
			Name:         dc.Name,
			Slug:         dc.Slug,
			Color:        dc.Color,
			Icon:         dc.Icon,
			OrderIndex:   int32(i),
			IsDefault:    dc.IsDefault,
			IsDoneColumn: dc.IsDoneColumn,
		}
		if err := s.repo.CreateColumn(ctx, col); err != nil {
			s.logger.Warn("failed to create default column", zap.Error(err))
		}
	}

	return board, nil
}

func (s *Service) GetBoard(ctx context.Context, boardID int64, includeColumns, includeStats bool) (*Board, error) {
	board, err := s.repo.GetBoard(ctx, boardID)
	if err != nil {
		return nil, err
	}

	if includeColumns {
		columns, err := s.repo.ListColumns(ctx, boardID)
		if err == nil {
			cols := make([]Column, len(columns))
			for i, c := range columns {
				cols[i] = *c
			}
			board.Columns = cols

			// Добавляем TaskCount для каждой колонки
			taskCounts, err := s.repo.GetColumnTaskCounts(ctx, boardID)
			if err == nil {
				for i := range board.Columns {
					board.Columns[i].TaskCount = taskCounts[board.Columns[i].ID]
				}
			}
		}
	}

	if includeStats {
		stats, err := s.repo.GetBoardStats(ctx, boardID)
		if err == nil {
			board.Stats = stats
		}
	}

	return board, nil
}

func (s *Service) GetBoardByProject(ctx context.Context, projectID int64, includeColumns, includeStats bool) (*Board, error) {
	board, err := s.repo.GetBoardByProject(ctx, projectID)
	if err != nil {
		return nil, err
	}

	if includeColumns {
		columns, err := s.repo.ListColumns(ctx, board.ID)
		if err == nil {
			cols := make([]Column, len(columns))
			for i, c := range columns {
				cols[i] = *c
			}
			board.Columns = cols
		}
	}

	if includeStats {
		stats, err := s.repo.GetBoardStats(ctx, board.ID)
		if err == nil {
			board.Stats = stats
		}
	}

	return board, nil
}

func (s *Service) ListMyBoards(ctx context.Context, userID int64, role string, universityID, departmentID int64, includeColumns, includeStats bool) ([]*Board, error) {
	return s.repo.ListMyBoards(ctx, userID, role, universityID, departmentID, includeColumns, includeStats)
}

func (s *Service) UpdateBoard(ctx context.Context, input *UpdateBoardInput) (*Board, error) {
	board, err := s.repo.GetBoard(ctx, input.BoardID)
	if err != nil {
		return nil, err
	}

	if input.UpdateMask != nil {
		for _, path := range input.UpdateMask.GetPaths() {
			switch path {
			case "name":
				board.Name = input.Name
			case "description":
				board.Description = input.Description
			}
		}
	} else {
		if input.Name != "" {
			board.Name = input.Name
		}
		if input.Description != "" {
			board.Description = input.Description
		}
	}

	board.UpdatedAt = time.Now()

	if err := s.repo.UpdateBoard(ctx, board); err != nil {
		return nil, err
	}
	return board, nil
}

func (s *Service) GetBoardStats(ctx context.Context, boardID int64) (*BoardStats, error) {
	return s.repo.GetBoardStats(ctx, boardID)
}

// ==================== COLUMN ====================

func (s *Service) ListColumns(ctx context.Context, boardID int64, includeTaskCount bool) ([]*Column, error) {
	columns, err := s.repo.ListColumns(ctx, boardID)
	if err != nil {
		return nil, err
	}

	if includeTaskCount {
		taskCounts, err := s.repo.GetColumnTaskCounts(ctx, boardID)
		if err == nil {
			for _, col := range columns {
				col.TaskCount = taskCounts[col.ID]
			}
		}
	}

	return columns, nil
}

func (s *Service) CreateColumn(ctx context.Context, input *CreateColumnInput) (*Column, error) {
	maxOrder, _ := s.repo.GetMaxColumnOrder(ctx, input.BoardID)

	col := &Column{
		BoardID:      input.BoardID,
		Name:         input.Name,
		Slug:         input.Slug,
		Description:  input.Description,
		Color:        input.Color,
		Icon:         input.Icon,
		OrderIndex:   maxOrder + 1,
		WIPLimit:     input.WIPLimit,
		IsDoneColumn: input.IsDoneColumn,
	}

	if err := s.repo.CreateColumn(ctx, col); err != nil {
		return nil, err
	}
	return col, nil
}

func (s *Service) UpdateColumn(ctx context.Context, input *UpdateColumnInput) (*Column, error) {
	col, err := s.repo.GetColumn(ctx, input.ColumnID)
	if err != nil {
		return nil, err
	}

	if input.UpdateMask != nil {
		for _, path := range input.UpdateMask.GetPaths() {
			switch path {
			case "name":
				col.Name = input.Name
			case "description":
				col.Description = input.Description
			case "color":
				col.Color = input.Color
			case "icon":
				col.Icon = input.Icon
			case "wip_limit":
				col.WIPLimit = input.WIPLimit
			}
		}
	} else {
		if input.Name != "" {
			col.Name = input.Name
		}
		if input.Description != "" {
			col.Description = input.Description
		}
		if input.Color != "" {
			col.Color = input.Color
		}
		if input.Icon != "" {
			col.Icon = input.Icon
		}
		if input.WIPLimit > 0 {
			col.WIPLimit = input.WIPLimit
		}
	}

	if err := s.repo.UpdateColumn(ctx, col); err != nil {
		return nil, err
	}
	return col, nil
}

func (s *Service) DeleteColumn(ctx context.Context, columnID int64, moveTasksToColumnID int64) error {
	if moveTasksToColumnID > 0 {
		// Перемещаем задачи в другую колонку
		col, err := s.repo.GetColumn(ctx, columnID)
		if err != nil {
			return err
		}

		tasks, _, err := s.repo.ListTasks(ctx, TaskFilter{
			BoardID:  col.BoardID,
			ColumnID: columnID,
		})
		if err != nil {
			return err
		}

		if len(tasks) > 0 {
			taskIDs := make([]int64, len(tasks))
			for i, t := range tasks {
				taskIDs[i] = t.ID
			}
			if err := s.repo.BulkUpdateTasks(ctx, taskIDs, map[string]interface{}{
				"column_id": moveTasksToColumnID,
			}); err != nil {
				return err
			}
		}
	}

	return s.repo.DeleteColumn(ctx, columnID)
}

func (s *Service) ReorderColumns(ctx context.Context, boardID int64, columnIDs []int64) ([]*Column, error) {
	if err := s.repo.ReorderColumns(ctx, boardID, columnIDs); err != nil {
		return nil, err
	}
	return s.repo.ListColumns(ctx, boardID)
}

// ==================== TASK ====================

func (s *Service) CreateTask(ctx context.Context, input *CreateTaskInput) (*Task, error) {
	columnID := input.ColumnID
	if columnID == 0 {
		defCol, err := s.repo.GetDefaultColumn(ctx, input.BoardID)
		if err != nil {
			return nil, fmt.Errorf("no default column: %w", err)
		}
		columnID = defCol.ID
	}

	maxPos, _ := s.repo.GetMaxPosition(ctx, columnID)

	var assigneeID *int64
	if input.AssigneeID > 0 {
		assigneeID = &input.AssigneeID
	}

	var workflowStepID *int64
	if input.WorkflowStepID > 0 {
		workflowStepID = &input.WorkflowStepID
	}

	task := &Task{
		BoardID:          input.BoardID,
		ColumnID:         columnID,
		Title:            input.Title,
		Description:      input.Description,
		Status:           TaskStatusTodo,
		Priority:         input.Priority,
		AssigneeID:       assigneeID,
		DueDate:          input.DueDate,
		EstimatedMinutes: input.EstimatedMinutes,
		Labels:           JSONArray(input.Labels),
		Position:         maxPos + 1,
		CreatedBy:        input.CreatedBy,
		WorkflowStepID:   workflowStepID,
	}

	if err := s.repo.CreateTask(ctx, task); err != nil {
		return nil, err
	}

	// Логируем создание
	s.repo.LogActivity(ctx, &ActivityLog{
		TaskID:  task.ID,
		ActorID: input.CreatedBy,
		Action:  "created",
	})

	return task, nil
}

func (s *Service) GetTask(ctx context.Context, taskID int64) (*Task, error) {
	task, err := s.repo.GetTask(ctx, taskID)
	if err != nil {
		return nil, err
	}

	// Заполняем computed fields
	comments, attachments, watchers, _ := s.repo.GetTaskCounts(ctx, taskID)
	task.CommentsCount = comments
	task.AttachmentsCount = attachments
	task.WatchersCount = watchers

	if task.DueDate != nil && task.Status != TaskStatusDone {
		task.IsOverdue = task.DueDate.Before(time.Now())
	}

	return task, nil
}

func (s *Service) UpdateTask(ctx context.Context, input *UpdateTaskInput) (*Task, error) {
	task, err := s.repo.GetTask(ctx, input.TaskID)
	if err != nil {
		return nil, err
	}

	if input.UpdateMask != nil {
		for _, path := range input.UpdateMask.GetPaths() {
			switch path {
			case "title":
				task.Title = input.Title
			case "description":
				task.Description = input.Description
			case "priority":
				if input.Priority != nil {
					task.Priority = *input.Priority
				}
			case "due_date":
				task.DueDate = input.DueDate
			case "estimated_minutes":
				task.EstimatedMinutes = input.EstimatedMinutes
			case "actual_minutes":
				task.ActualMinutes = input.ActualMinutes
			case "labels":
				task.Labels = JSONArray(input.Labels)
			case "workflow_step_id":
				if input.WorkflowStepID > 0 {
					task.WorkflowStepID = &input.WorkflowStepID
				}
			}
		}
	} else {
		if input.Title != "" {
			task.Title = input.Title
		}
		if input.Description != "" {
			task.Description = input.Description
		}
		if input.Priority != nil {
			task.Priority = *input.Priority
		}
		if input.DueDate != nil {
			task.DueDate = input.DueDate
		}
		if input.EstimatedMinutes > 0 {
			task.EstimatedMinutes = input.EstimatedMinutes
		}
		if input.ActualMinutes > 0 {
			task.ActualMinutes = input.ActualMinutes
		}
		if input.Labels != nil {
			task.Labels = JSONArray(input.Labels)
		}
	}

	task.UpdatedAt = time.Now()

	if err := s.repo.UpdateTask(ctx, task); err != nil {
		return nil, err
	}

	// Логируем обновление
	s.repo.LogActivity(ctx, &ActivityLog{
		TaskID:  task.ID,
		ActorID: input.UpdatedBy,
		Action:  "updated",
	})

	return task, nil
}

func (s *Service) DeleteTask(ctx context.Context, taskID int64) error {
	return s.repo.DeleteTask(ctx, taskID)
}

func (s *Service) MoveTask(ctx context.Context, input *MoveTaskInput) (*Task, error) {
	if err := s.repo.MoveTask(ctx, input.TaskID, input.ColumnID, int32(input.Position)); err != nil {
		return nil, err
	}

	// Логируем перемещение
	s.repo.LogActivity(ctx, &ActivityLog{
		TaskID:    input.TaskID,
		ActorID:   input.MovedBy,
		Action:    "moved",
		FieldName: "column_id",
		NewValue:  fmt.Sprintf("%d", input.ColumnID),
	})

	return s.repo.GetTask(ctx, input.TaskID)
}

func (s *Service) ReorderTasks(ctx context.Context, columnID int64, order []int64) error {
	return s.repo.ReorderTasks(ctx, columnID, order)
}

func (s *Service) ListTasks(ctx context.Context, filter TaskFilter) ([]*Task, int64, error) {
	tasks, total, err := s.repo.ListTasks(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	// Заполняем computed fields batch-запросом
	if len(tasks) > 0 {
		taskIDs := make([]int64, len(tasks))
		for i, t := range tasks {
			taskIDs[i] = t.ID
		}

		counts, _ := s.repo.GetTasksCountsBatch(ctx, taskIDs)
		for _, t := range tasks {
			if c, ok := counts[t.ID]; ok {
				t.CommentsCount = c.Comments
				t.AttachmentsCount = c.Attachments
				t.WatchersCount = c.Watchers
			}
			if t.DueDate != nil && t.Status != TaskStatusDone {
				t.IsOverdue = t.DueDate.Before(time.Now())
			}
		}
	}

	return tasks, total, nil
}

func (s *Service) BulkUpdateTasks(ctx context.Context, input *BulkUpdateInput) ([]*Task, error) {
	updates := make(map[string]interface{})

	if input.AssigneeID > 0 {
		updates["assignee_id"] = input.AssigneeID
	}
	if input.Priority != nil {
		updates["priority"] = *input.Priority
	}
	if input.MoveToColumnID > 0 {
		updates["column_id"] = input.MoveToColumnID
	}

	if len(updates) > 0 {
		if err := s.repo.BulkUpdateTasks(ctx, input.TaskIDs, updates); err != nil {
			return nil, err
		}
	}

	// Возвращаем обновлённые задачи
	var tasks []*Task
	for _, id := range input.TaskIDs {
		task, err := s.repo.GetTask(ctx, id)
		if err == nil {
			tasks = append(tasks, task)
		}
	}
	return tasks, nil
}

func (s *Service) BulkDeleteTasks(ctx context.Context, taskIDs []int64) error {
	return s.repo.BulkDeleteTasks(ctx, taskIDs)
}

// ==================== ASSIGNMENT ====================

func (s *Service) AssignTask(ctx context.Context, taskID, assigneeID, assignedBy int64) (*Task, error) {
	task, err := s.repo.GetTask(ctx, taskID)
	if err != nil {
		return nil, err
	}

	task.AssigneeID = &assigneeID
	task.UpdatedAt = time.Now()

	if err := s.repo.UpdateTask(ctx, task); err != nil {
		return nil, err
	}

	s.repo.LogActivity(ctx, &ActivityLog{
		TaskID:    taskID,
		ActorID:   assignedBy,
		Action:    "assigned",
		FieldName: "assignee_id",
		NewValue:  fmt.Sprintf("%d", assigneeID),
	})

	return task, nil
}

func (s *Service) UnassignTask(ctx context.Context, taskID, unassignedBy int64) (*Task, error) {
	task, err := s.repo.GetTask(ctx, taskID)
	if err != nil {
		return nil, err
	}

	oldAssignee := ""
	if task.AssigneeID != nil {
		oldAssignee = fmt.Sprintf("%d", *task.AssigneeID)
	}

	task.AssigneeID = nil
	task.UpdatedAt = time.Now()

	if err := s.repo.UpdateTask(ctx, task); err != nil {
		return nil, err
	}

	s.repo.LogActivity(ctx, &ActivityLog{
		TaskID:    taskID,
		ActorID:   unassignedBy,
		Action:    "unassigned",
		FieldName: "assignee_id",
		OldValue:  oldAssignee,
	})

	return task, nil
}

// ==================== COMMENT ====================

func (s *Service) CreateComment(ctx context.Context, input *CreateCommentInput) (*Comment, error) {
	comment := &Comment{
		TaskID:   input.TaskID,
		AuthorID: input.AuthorID,
		Content:  input.Content,
	}

	if err := s.repo.CreateComment(ctx, comment); err != nil {
		return nil, err
	}

	s.repo.LogActivity(ctx, &ActivityLog{
		TaskID:  input.TaskID,
		ActorID: input.AuthorID,
		Action:  "commented",
	})

	return comment, nil
}

func (s *Service) GetComment(ctx context.Context, commentID int64) (*Comment, error) {
	return s.repo.GetComment(ctx, commentID)
}

func (s *Service) UpdateComment(ctx context.Context, commentID int64, content string) (*Comment, error) {
	comment, err := s.repo.GetComment(ctx, commentID)
	if err != nil {
		return nil, err
	}

	comment.Content = content
	now := time.Now()
	comment.EditedAt = &now

	if err := s.repo.UpdateComment(ctx, comment); err != nil {
		return nil, err
	}
	return comment, nil
}

func (s *Service) DeleteComment(ctx context.Context, commentID int64) error {
	return s.repo.DeleteComment(ctx, commentID)
}

func (s *Service) ListComments(ctx context.Context, taskID int64, limit, offset int) ([]*Comment, int64, error) {
	return s.repo.ListComments(ctx, taskID, limit, offset)
}

func (s *Service) GetRecentComments(ctx context.Context, taskID int64, limit int) ([]*Comment, error) {
	comments, _, err := s.repo.ListComments(ctx, taskID, limit, 0)
	return comments, err
}

// ==================== ATTACHMENT ====================

func (s *Service) AddAttachment(ctx context.Context, input *AddAttachmentInput) (*Attachment, error) {
	attachment := &Attachment{
		TaskID:     input.TaskID,
		FileID:     input.FileID,
		FileName:   input.FileName,
		FileType:   input.FileType,
		FileSize:   input.FileSize,
		UploadedBy: input.UploadedBy,
	}

	if err := s.repo.CreateAttachment(ctx, attachment); err != nil {
		return nil, err
	}

	s.repo.LogActivity(ctx, &ActivityLog{
		TaskID:   input.TaskID,
		ActorID:  input.UploadedBy,
		Action:   "attachment_added",
		NewValue: input.FileName,
	})

	return attachment, nil
}

func (s *Service) GetAttachment(ctx context.Context, attachmentID int64) (*Attachment, error) {
	return s.repo.GetAttachment(ctx, attachmentID)
}

func (s *Service) RemoveAttachment(ctx context.Context, attachmentID int64) error {
	return s.repo.DeleteAttachment(ctx, attachmentID)
}

func (s *Service) ListAttachments(ctx context.Context, taskID int64) ([]*Attachment, error) {
	return s.repo.ListAttachments(ctx, taskID)
}

// ==================== ACTIVITY ====================

func (s *Service) ListActivity(ctx context.Context, taskID int64, limit, offset int) ([]*ActivityLog, int64, error) {
	return s.repo.ListActivity(ctx, taskID, limit, offset)
}

func (s *Service) GetRecentActivity(ctx context.Context, taskID int64, limit int) ([]*ActivityLog, error) {
	activities, _, err := s.repo.ListActivity(ctx, taskID, limit, 0)
	return activities, err
}

// ==================== WATCHER ====================

func (s *Service) AddWatcher(ctx context.Context, taskID, userID int64) error {
	watcher := &Watcher{
		TaskID: taskID,
		UserID: userID,
	}
	return s.repo.AddWatcher(ctx, watcher)
}

func (s *Service) RemoveWatcher(ctx context.Context, taskID, userID int64) error {
	return s.repo.RemoveWatcher(ctx, taskID, userID)
}

func (s *Service) ListWatchers(ctx context.Context, taskID int64) ([]*UserPreview, error) {
	watchers, err := s.repo.ListWatchers(ctx, taskID)
	if err != nil {
		return nil, err
	}

	var previews []*UserPreview
	for _, w := range watchers {
		previews = append(previews, &UserPreview{
			ID: w.UserID,
		})
	}
	return previews, nil
}

// ==================== STATS ====================

func (s *Service) GetMemberStats(ctx context.Context, boardID int64) ([]*MemberStats, error) {
	return s.repo.GetMemberStats(ctx, boardID)
}

func (s *Service) GetDailyStats(ctx context.Context, boardID int64, days int) ([]*DailyStats, error) {
	return s.repo.GetDailyStats(ctx, boardID, days)
}

// ==================== MY TASKS ====================

func (s *Service) GetMyTasks(ctx context.Context, userID int64, filter MyTasksFilter) ([]*Task, int64, error) {
	return s.repo.GetMyTasks(ctx, userID, filter)
}

func (s *Service) GetOverdueTasks(ctx context.Context, boardID int64, assigneeID *int64, limit, offset int) ([]*Task, int64, error) {
	return s.repo.GetOverdueTasks(ctx, boardID, assigneeID, limit, offset)
}

func (s *Service) GetUpcomingDeadlines(ctx context.Context, boardID int64, userID *int64, daysAhead, limit, offset int) ([]*Task, int64, error) {
	return s.repo.GetUpcomingDeadlines(ctx, boardID, userID, daysAhead, limit, offset)
}

func (s *Service) StartBackgroundJobs(ctx context.Context) {
	s.logger.Info("background jobs started (deadline notifier)")
	// TODO: реализовать deadline notifier
}
