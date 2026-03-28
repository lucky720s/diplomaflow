package task

import (
	"context"
	"fmt"
	"time"

	notificationv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/notification/v1"
	teamv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/team/v1"
	"go.uber.org/zap"
	"google.golang.org/grpc/metadata"
)

type Service struct {
	repo               Repository
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
	col, err := s.repo.GetColumn(ctx, columnID)
	if err != nil {
		return err
	}

	if moveTasksToColumnID > 0 {
		tasks, _, err := s.repo.ListTasks(ctx, TaskFilter{ColumnID: columnID})
		if err == nil && len(tasks) > 0 {
			taskIDs := make([]int64, len(tasks))
			for i, t := range tasks {
				taskIDs[i] = t.ID
			}
			_ = s.repo.BulkUpdateTasks(ctx, taskIDs, map[string]interface{}{
				"column_id": moveTasksToColumnID,
			})
		}
	}

	_ = col
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
	var columnID int64
	if input.ColumnID > 0 {
		columnID = input.ColumnID
	} else {
		defaultCol, err := s.repo.GetDefaultColumn(ctx, input.BoardID)
		if err != nil {
			return nil, fmt.Errorf("no default column found: %w", err)
		}
		columnID = defaultCol.ID
	}

	maxPos, _ := s.repo.GetMaxPosition(ctx, columnID)

	task := &Task{
		BoardID:          input.BoardID,
		ColumnID:         columnID,
		Title:            input.Title,
		Description:      input.Description,
		Priority:         input.Priority,
		CreatedBy:        input.CreatedBy,
		DueDate:          input.DueDate,
		EstimatedMinutes: input.EstimatedMinutes,
		Position:         maxPos + 1,
		Status:           TaskStatusTodo,
	}

	if input.AssigneeID > 0 {
		task.AssigneeID = &input.AssigneeID
	}

	if input.WorkflowStepID > 0 {
		task.WorkflowStepID = &input.WorkflowStepID
	}

	if len(input.Labels) > 0 {
		task.Labels = JSONArray(input.Labels)
	}

	if err := s.repo.CreateTask(ctx, task); err != nil {
		return nil, err
	}

	// ✅ ИСПРАВЛЕНО errcheck
	if err := s.repo.LogActivity(ctx, &ActivityLog{
		TaskID:  task.ID,
		ActorID: input.CreatedBy,
		Action:  ActionCreated,
	}); err != nil {
		s.logger.Warn("failed to log activity", zap.String("action", ActionCreated), zap.Error(err))
	}

	return task, nil
}

func (s *Service) GetTask(ctx context.Context, taskID int64) (*Task, error) {
	task, err := s.repo.GetTask(ctx, taskID)
	if err != nil {
		return nil, err
	}

	comments, attachments, watchers, err := s.repo.GetTaskCounts(ctx, taskID)
	if err == nil {
		task.CommentsCount = comments
		task.AttachmentsCount = attachments
		task.WatchersCount = watchers
	}

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
		if len(input.Labels) > 0 {
			task.Labels = JSONArray(input.Labels)
		}
		if input.WorkflowStepID > 0 {
			task.WorkflowStepID = &input.WorkflowStepID
		}
	}

	if err := s.repo.UpdateTask(ctx, task); err != nil {
		return nil, err
	}

	// ✅ ИСПРАВЛЕНО errcheck
	if err := s.repo.LogActivity(ctx, &ActivityLog{
		TaskID:  task.ID,
		ActorID: input.UpdatedBy,
		Action:  ActionUpdated,
	}); err != nil {
		s.logger.Warn("failed to log activity", zap.String("action", ActionUpdated), zap.Error(err))
	}

	return task, nil
}

func (s *Service) DeleteTask(ctx context.Context, taskID int64) error {
	return s.repo.DeleteTask(ctx, taskID)
}

func (s *Service) ListTasks(ctx context.Context, filter TaskFilter) ([]*Task, int64, error) {
	tasks, total, err := s.repo.ListTasks(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	if len(tasks) > 0 {
		taskIDs := make([]int64, len(tasks))
		for i, t := range tasks {
			taskIDs[i] = t.ID
		}

		counts, err := s.repo.GetTasksCountsBatch(ctx, taskIDs)
		if err == nil {
			now := time.Now()
			for _, t := range tasks {
				if c, ok := counts[t.ID]; ok {
					t.CommentsCount = c.Comments
					t.AttachmentsCount = c.Attachments
					t.WatchersCount = c.Watchers
				}
				if t.DueDate != nil && t.Status != TaskStatusDone {
					t.IsOverdue = t.DueDate.Before(now)
				}
			}
		}
	}

	return tasks, total, nil
}

func (s *Service) MoveTask(ctx context.Context, input *MoveTaskInput) (*Task, error) {
	if err := s.repo.MoveTask(ctx, input.TaskID, input.ColumnID, int32(input.Position)); err != nil {
		return nil, err
	}

	// ✅ ИСПРАВЛЕНО errcheck
	if err := s.repo.LogActivity(ctx, &ActivityLog{
		TaskID:    input.TaskID,
		ActorID:   input.MovedBy,
		Action:    ActionMoved,
		FieldName: "column_id",
		NewValue:  fmt.Sprintf("%d", input.ColumnID),
	}); err != nil {
		s.logger.Warn("failed to log activity", zap.String("action", ActionMoved), zap.Error(err))
	}

	return s.repo.GetTask(ctx, input.TaskID)
}

func (s *Service) ReorderTasks(ctx context.Context, columnID int64, taskIDs []int64) error {
	return s.repo.ReorderTasks(ctx, columnID, taskIDs)
}

func (s *Service) BulkUpdateTasks(ctx context.Context, input *BulkUpdateInput) ([]*Task, error) {
	updates := make(map[string]interface{})

	if input.AssigneeID > 0 {
		updates["assignee_id"] = input.AssigneeID
	} else if input.AssigneeID == -1 {
		updates["assignee_id"] = nil
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

	var tasks []*Task
	for _, id := range input.TaskIDs {
		t, err := s.repo.GetTask(ctx, id)
		if err == nil {
			tasks = append(tasks, t)
		}
	}
	return tasks, nil
}

func (s *Service) BulkDeleteTasks(ctx context.Context, taskIDs []int64) error {
	return s.repo.BulkDeleteTasks(ctx, taskIDs)
}

func (s *Service) AssignTask(ctx context.Context, taskID, assigneeID, assignedBy int64) (*Task, error) {
	task, err := s.repo.GetTask(ctx, taskID)
	if err != nil {
		return nil, err
	}

	task.AssigneeID = &assigneeID
	if err := s.repo.UpdateTask(ctx, task); err != nil {
		return nil, err
	}

	if err := s.repo.LogActivity(ctx, &ActivityLog{
		TaskID:   taskID,
		ActorID:  assignedBy,
		Action:   ActionAssigned,
		NewValue: fmt.Sprintf("%d", assigneeID),
	}); err != nil {
		s.logger.Warn("failed to log activity", zap.String("action", ActionAssigned), zap.Error(err))
	}

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
	if err := s.repo.UpdateTask(ctx, task); err != nil {
		return nil, err
	}

	if err := s.repo.LogActivity(ctx, &ActivityLog{
		TaskID:   taskID,
		ActorID:  unassignedBy,
		Action:   ActionUnassigned,
		OldValue: oldAssignee,
	}); err != nil {
		s.logger.Warn("failed to log activity", zap.String("action", ActionUnassigned), zap.Error(err))
	}

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

	if err := s.repo.LogActivity(ctx, &ActivityLog{
		TaskID:  input.TaskID,
		ActorID: input.AuthorID,
		Action:  ActionCommented,
	}); err != nil {
		s.logger.Warn("failed to log activity", zap.String("action", ActionCommented), zap.Error(err))
	}

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
	logs, _, err := s.repo.ListActivity(ctx, taskID, limit, 0)
	return logs, err
}

// ==================== WATCHER ====================

func (s *Service) AddWatcher(ctx context.Context, taskID, userID int64) error {
	watching, err := s.repo.IsWatching(ctx, taskID, userID)
	if err != nil {
		return err
	}
	if watching {
		return nil
	}

	return s.repo.AddWatcher(ctx, &Watcher{
		TaskID: taskID,
		UserID: userID,
	})
}

func (s *Service) RemoveWatcher(ctx context.Context, taskID, userID int64) error {
	return s.repo.RemoveWatcher(ctx, taskID, userID)
}

func (s *Service) ListWatchers(ctx context.Context, taskID int64) ([]*UserPreview, error) {
	watchers, err := s.repo.ListWatchers(ctx, taskID)
	if err != nil {
		return nil, err
	}

	previews := make([]*UserPreview, len(watchers))
	for i, w := range watchers {
		previews[i] = &UserPreview{ID: w.UserID}
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

// ==================== BACKGROUND JOBS ====================

func (s *Service) StartBackgroundJobs(ctx context.Context) {
	go s.runDeadlineNotifier(ctx)
}

func (s *Service) runDeadlineNotifier(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	s.checkDeadlines(ctx)

	for {
		select {
		case <-ctx.Done():
			s.logger.Info("deadline notifier stopped")
			return
		case <-ticker.C:
			s.checkDeadlines(ctx)
		}
	}
}

func (s *Service) checkDeadlines(ctx context.Context) {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	tomorrow := today.AddDate(0, 0, 1)

	tasksDueToday, err := s.repo.ListTasksDueOn(ctx, today)
	if err != nil {
		s.logger.Error("failed to list tasks due today", zap.Error(err))
	} else {
		for _, task := range tasksDueToday {
			s.sendDeadlineNotification(ctx, task, "due_today", today)
		}
	}

	tasksDueTomorrow, err := s.repo.ListTasksDueOn(ctx, tomorrow)
	if err != nil {
		s.logger.Error("failed to list tasks due tomorrow", zap.Error(err))
	} else {
		for _, task := range tasksDueTomorrow {
			s.sendDeadlineNotification(ctx, task, "due_tomorrow", tomorrow)
		}
	}

	overdueTasks, err := s.repo.ListOverdueOpenTasks(ctx, today)
	if err != nil {
		s.logger.Error("failed to list overdue tasks", zap.Error(err))
	} else {
		for _, task := range overdueTasks {
			if task.DueDate != nil {
				s.sendDeadlineNotification(ctx, task, "overdue", *task.DueDate)
			}
		}
	}
}

func (s *Service) sendDeadlineNotification(ctx context.Context, task *Task, kind string, dueDate time.Time) {
	if task.AssigneeID == nil {
		return
	}

	dedupKey := fmt.Sprintf("%s:%d:%d:%s", kind, task.ID, *task.AssigneeID, dueDate.Format("2006-01-02"))

	run := &DeadlineNotificationRun{
		DedupKey: dedupKey,
		Kind:     kind,
		TaskID:   task.ID,
		UserID:   *task.AssigneeID,
		DueDate:  dueDate,
		SentAt:   time.Now(),
	}

	inserted, err := s.repo.TryInsertDeadlineRun(ctx, run)
	if err != nil {
		s.logger.Error("failed to insert deadline run", zap.Error(err))
		return
	}
	if !inserted {
		return // уже отправляли сегодня
	}

	// ✅ Правильные поля согласно notification.proto:
	// title, message, link, type — всё string, никакого enum
	var title, message, notifType string

	switch kind {
	case "due_today":
		notifType = "task_due_today"
		title = "Задача истекает сегодня"
		message = fmt.Sprintf("Задача «%s» должна быть выполнена сегодня", task.Title)
	case "due_tomorrow":
		notifType = "task_due_tomorrow"
		title = "Задача истекает завтра"
		message = fmt.Sprintf("Задача «%s» должна быть выполнена завтра", task.Title)
	case "overdue":
		notifType = "task_overdue"
		title = "Задача просрочена"
		message = fmt.Sprintf("Задача «%s» просрочена (срок: %s)", task.Title, dueDate.Format("02.01.2006"))
	default:
		return
	}

	link := fmt.Sprintf("/boards/%d", task.BoardID)

	// Добавляем internal header чтобы notification_service принял запрос
	// (он проверяет x-internal-service)
	notifCtx := metadata.AppendToOutgoingContext(ctx, "x-internal-service", "task_service")

	_, err = s.notificationClient.SendNotification(notifCtx, &notificationv1.SendNotificationRequest{
		UserId:  *task.AssigneeID,
		Title:   title,
		Message: message,
		Link:    link,
		Type:    notifType,
	})
	if err != nil {
		s.logger.Error("failed to send deadline notification",
			zap.String("kind", kind),
			zap.Int64("task_id", task.ID),
			zap.Int64("user_id", *task.AssigneeID),
			zap.Error(err),
		)
	}
}

// ==================== WORKFLOW ====================

func (s *Service) GetColumn(ctx context.Context, columnID int64) (*Column, error) {
	return s.repo.GetColumn(ctx, columnID)
}

func (s *Service) GetBoardByTeam(ctx context.Context, teamID int64) (*Board, error) {
	return s.repo.GetBoardByTeam(ctx, teamID)
}
