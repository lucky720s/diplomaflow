package task

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	teamv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/team/v1"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Service - бизнес-логика для task_service
type Service struct {
	repo       Repository
	teamClient teamv1.TeamServiceClient
	logger     *zap.Logger
}

// NewService создает новый сервис
func NewService(repo Repository, teamClient teamv1.TeamServiceClient, logger *zap.Logger) *Service {
	return &Service{
		repo:       repo,
		teamClient: teamClient,
		logger:     logger,
	}
}

// ==================== Board ====================

// CreateBoardInput - входные данные для создания доски
type CreateBoardInput struct {
	TeamID               int64
	ProjectID            int64
	Name                 string
	Description          string
	CreatedBy            int64
	CreateDefaultColumns bool
}

// CreateBoard - создает доску для команды
func (s *Service) CreateBoard(ctx context.Context, input *CreateBoardInput) (*Board, error) {
	// Проверяем, нет ли уже доски для этой команды
	existing, err := s.repo.GetBoardByTeam(ctx, input.TeamID)
	if err == nil && existing != nil {
		return nil, errors.New("board already exists for this team")
	}

	// Настройки по умолчанию
	settings := BoardSettings{
		DefaultColumn:      "todo",
		AllowCustomColumns: false,
		ShowCompleted:      true,
		Labels:             []string{"backend", "frontend", "docs", "research", "urgent"},
	}
	settingsJSON, _ := json.Marshal(settings)

	board := &Board{
		TeamID:      input.TeamID,
		ProjectID:   input.ProjectID,
		Name:        input.Name,
		Description: input.Description,
		Settings:    settingsJSON,
		CreatedBy:   input.CreatedBy,
	}

	if err := s.repo.CreateBoard(ctx, board); err != nil {
		s.logger.Error("Failed to create board", zap.Error(err))
		return nil, fmt.Errorf("failed to create board: %w", err)
	}

	// Создаём стандартные колонки
	if input.CreateDefaultColumns {
		for _, col := range DefaultColumns {
			column := &Column{
				BoardID:      board.ID,
				Name:         col.Name,
				Slug:         col.Slug,
				Color:        col.Color,
				OrderIndex:   col.OrderIndex,
				IsDefault:    col.IsDefault,
				IsDoneColumn: col.IsDoneColumn,
			}
			if err := s.repo.CreateColumn(ctx, column); err != nil {
				s.logger.Error("Failed to create default column", zap.Error(err), zap.String("slug", col.Slug))
			}
		}
	}

	s.logger.Info("Board created", zap.Int64("board_id", board.ID), zap.Int64("team_id", input.TeamID))
	return board, nil
}

// GetBoard - получает доску по ID
func (s *Service) GetBoard(ctx context.Context, boardID int64, includeColumns, includeStats bool) (*Board, error) {
	board, err := s.repo.GetBoard(ctx, boardID)
	if err != nil {
		return nil, fmt.Errorf("board not found: %w", err)
	}

	if includeColumns {
		columns, err := s.repo.ListColumns(ctx, boardID)
		if err == nil {
			counts, _ := s.repo.GetColumnTaskCounts(ctx, boardID)
			for _, col := range columns {
				if count, ok := counts[col.ID]; ok {
					col.TaskCount = count
				}
			}
			for _, col := range columns {
				board.Columns = append(board.Columns, *col)
			}
		}
	}

	return board, nil
}

// GetBoardByTeam - получает доску по ID команды
func (s *Service) GetBoardByTeam(ctx context.Context, teamID int64, includeColumns, includeStats bool) (*Board, error) {
	board, err := s.repo.GetBoardByTeam(ctx, teamID)
	if err != nil {
		return nil, fmt.Errorf("board not found for team: %w", err)
	}

	if includeColumns {
		columns, err := s.repo.ListColumns(ctx, board.ID)
		if err == nil {
			counts, _ := s.repo.GetColumnTaskCounts(ctx, board.ID)
			for _, col := range columns {
				if count, ok := counts[col.ID]; ok {
					col.TaskCount = count
				}
			}
			for _, col := range columns {
				board.Columns = append(board.Columns, *col)
			}
		}
	}

	return board, nil
}

// UpdateBoard - обновляет доску
func (s *Service) UpdateBoard(ctx context.Context, boardID int64, name, description string, settings *BoardSettings) (*Board, error) {
	board, err := s.repo.GetBoard(ctx, boardID)
	if err != nil {
		return nil, fmt.Errorf("board not found: %w", err)
	}

	if name != "" {
		board.Name = name
	}
	if description != "" {
		board.Description = description
	}
	if settings != nil {
		settingsJSON, _ := json.Marshal(settings)
		board.Settings = settingsJSON
	}

	if err := s.repo.UpdateBoard(ctx, board); err != nil {
		return nil, fmt.Errorf("failed to update board: %w", err)
	}

	return board, nil
}

// GetOrCreateBoardForTeam - получает или создаёт доску для команды
func (s *Service) GetOrCreateBoardForTeam(ctx context.Context, teamID, projectID, createdBy int64) (*Board, error) {
	board, err := s.repo.GetBoardByTeam(ctx, teamID)
	if err == nil {
		return board, nil
	}

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return s.CreateBoard(ctx, &CreateBoardInput{
			TeamID:               teamID,
			ProjectID:            projectID,
			Name:                 "Задачи команды",
			CreatedBy:            createdBy,
			CreateDefaultColumns: true,
		})
	}

	return nil, err
}

// ==================== Columns ====================

// CreateColumnInput - входные данные для создания колонки
type CreateColumnInput struct {
	BoardID      int64
	Name         string
	Slug         string
	Description  string
	Color        string
	WIPLimit     int32
	IsDoneColumn bool
	IsDefault    bool
}

// CreateColumn - создает колонку
func (s *Service) CreateColumn(ctx context.Context, input *CreateColumnInput) (*Column, error) {
	_, err := s.repo.GetBoard(ctx, input.BoardID)
	if err != nil {
		return nil, fmt.Errorf("board not found: %w", err)
	}

	existing, _ := s.repo.GetColumnBySlug(ctx, input.BoardID, input.Slug)
	if existing != nil {
		return nil, errors.New("column with this slug already exists")
	}

	maxOrder, _ := s.repo.GetMaxColumnOrder(ctx, input.BoardID)

	color := input.Color
	if color == "" {
		color = "#6B7280"
	}

	// Если новая колонка is_default - снимаем флаг со старой
	if input.IsDefault {
		columns, _ := s.repo.ListColumns(ctx, input.BoardID)
		for _, col := range columns {
			if col.IsDefault {
				col.IsDefault = false
				_ = s.repo.UpdateColumn(ctx, col)
			}
		}
	}

	// Если новая колонка is_done_column - снимаем флаг со старых
	if input.IsDoneColumn {
		columns, _ := s.repo.ListColumns(ctx, input.BoardID)
		for _, col := range columns {
			if col.IsDoneColumn {
				col.IsDoneColumn = false
				_ = s.repo.UpdateColumn(ctx, col)
			}
		}
	}

	column := &Column{
		BoardID:      input.BoardID,
		Name:         input.Name,
		Slug:         input.Slug,
		Description:  input.Description,
		Color:        color,
		OrderIndex:   maxOrder + 1,
		WIPLimit:     input.WIPLimit,
		IsDoneColumn: input.IsDoneColumn,
		IsDefault:    input.IsDefault,
	}

	if err := s.repo.CreateColumn(ctx, column); err != nil {
		return nil, fmt.Errorf("failed to create column: %w", err)
	}

	s.logger.Info("Column created", zap.Int64("column_id", column.ID), zap.String("slug", input.Slug))
	return column, nil
}

// ListColumns - список колонок доски
func (s *Service) ListColumns(ctx context.Context, boardID int64) ([]*Column, error) {
	columns, err := s.repo.ListColumns(ctx, boardID)
	if err != nil {
		return nil, fmt.Errorf("failed to list columns: %w", err)
	}

	counts, _ := s.repo.GetColumnTaskCounts(ctx, boardID)
	for _, col := range columns {
		if count, ok := counts[col.ID]; ok {
			col.TaskCount = count
		}
	}

	return columns, nil
}

// UpdateColumn - обновляет колонку (без isDoneColumn для совместимости с handler)
func (s *Service) UpdateColumn(ctx context.Context, columnID int64, name, description, color string, wipLimit int32) (*Column, error) {
	column, err := s.repo.GetColumn(ctx, columnID)
	if err != nil {
		return nil, fmt.Errorf("column not found: %w", err)
	}

	if name != "" {
		column.Name = name
	}
	if description != "" {
		column.Description = description
	}
	if color != "" {
		column.Color = color
	}
	if wipLimit >= 0 {
		column.WIPLimit = wipLimit
	}

	if err := s.repo.UpdateColumn(ctx, column); err != nil {
		return nil, fmt.Errorf("failed to update column: %w", err)
	}

	s.logger.Info("Column updated", zap.Int64("column_id", columnID))
	return column, nil
}

// DeleteColumn - удаляет колонку с корректным перемещением задач
func (s *Service) DeleteColumn(ctx context.Context, columnID, moveTasksToColumnID int64) error {
	column, err := s.repo.GetColumn(ctx, columnID)
	if err != nil {
		return fmt.Errorf("column not found: %w", err)
	}

	if column.IsDefault {
		return errors.New("cannot delete default column")
	}

	tasks, totalTasks, err := s.repo.ListTasks(ctx, TaskFilter{
		ColumnID: columnID,
		Limit:    0,
	})
	if err != nil {
		return fmt.Errorf("failed to check tasks in column: %w", err)
	}

	if totalTasks > 0 {
		if moveTasksToColumnID == 0 {
			return fmt.Errorf("column has %d tasks, specify move_tasks_to_column_id", totalTasks)
		}

		targetColumn, err := s.repo.GetColumn(ctx, moveTasksToColumnID)
		if err != nil {
			return fmt.Errorf("target column not found: %w", err)
		}

		if targetColumn.BoardID != column.BoardID {
			return errors.New("target column must be on the same board")
		}

		if moveTasksToColumnID == columnID {
			return errors.New("cannot move tasks to the same column being deleted")
		}

		if targetColumn.WIPLimit > 0 {
			currentCount, _ := s.repo.GetColumnTaskCounts(ctx, column.BoardID)
			targetCurrentTasks := currentCount[moveTasksToColumnID]
			if int32(totalTasks)+targetCurrentTasks > targetColumn.WIPLimit {
				return fmt.Errorf("moving %d tasks would exceed WIP limit (%d) of target column '%s'",
					totalTasks, targetColumn.WIPLimit, targetColumn.Name)
			}
		}

		// Собираем ID задач
		taskIDs := make([]int64, 0, len(tasks))
		for _, task := range tasks {
			taskIDs = append(taskIDs, task.ID)
		}

		maxPos, _ := s.repo.GetMaxPosition(ctx, moveTasksToColumnID)

		// Перемещаем задачи
		if err := s.repo.BulkUpdateTasks(ctx, taskIDs, map[string]interface{}{
			"column_id": moveTasksToColumnID,
		}); err != nil {
			return fmt.Errorf("failed to move tasks: %w", err)
		}

		for i, taskID := range taskIDs {
			_ = s.repo.BulkUpdateTasks(ctx, []int64{taskID}, map[string]interface{}{
				"position": maxPos + int32(i) + 1,
			})
		}

		s.logger.Info("Tasks moved before column deletion",
			zap.Int64("from_column", columnID),
			zap.Int64("to_column", moveTasksToColumnID),
			zap.Int("tasks_count", len(taskIDs)))
	}

	if err := s.repo.DeleteColumn(ctx, columnID); err != nil {
		return fmt.Errorf("failed to delete column: %w", err)
	}

	s.logger.Info("Column deleted", zap.Int64("column_id", columnID), zap.Int64("board_id", column.BoardID))
	return nil
}

// ReorderColumns - изменяет порядок колонок
func (s *Service) ReorderColumns(ctx context.Context, boardID int64, columnIDs []int64) error {
	return s.repo.ReorderColumns(ctx, boardID, columnIDs)
}

// ==================== Tasks ====================

// CreateTaskInput - входные данные для создания задачи
type CreateTaskInput struct {
	BoardID          int64
	Title            string
	Description      string
	Priority         string
	AssigneeID       int64
	DueDate          *time.Time
	EstimatedMinutes int32
	Labels           []string
	ColumnID         int64
	CreatedBy        int64
	WorkflowStepID   int64
}

// CreateTask - создает задачу с проверкой WIP-лимита
func (s *Service) CreateTask(ctx context.Context, input *CreateTaskInput) (*Task, error) {
	board, err := s.repo.GetBoard(ctx, input.BoardID)
	if err != nil {
		return nil, fmt.Errorf("board not found: %w", err)
	}

	columnID := input.ColumnID
	var targetColumn *Column

	if columnID == 0 {
		defaultColumn, err := s.repo.GetDefaultColumn(ctx, board.ID)
		if err != nil {
			return nil, fmt.Errorf("no default column found: %w", err)
		}
		columnID = defaultColumn.ID
		targetColumn = defaultColumn
	} else {
		col, err := s.repo.GetColumn(ctx, columnID)
		if err != nil {
			return nil, fmt.Errorf("column not found: %w", err)
		}
		if col.BoardID != board.ID {
			return nil, errors.New("column does not belong to this board")
		}
		targetColumn = col
	}

	// Проверка WIP-лимита
	if targetColumn.WIPLimit > 0 {
		counts, err := s.repo.GetColumnTaskCounts(ctx, board.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to check column task count: %w", err)
		}

		currentTasks := counts[columnID]
		if currentTasks >= targetColumn.WIPLimit {
			return nil, fmt.Errorf("WIP limit reached: column '%s' already has %d/%d tasks",
				targetColumn.Name, currentTasks, targetColumn.WIPLimit)
		}
	}

	maxPos, _ := s.repo.GetMaxPosition(ctx, columnID)

	priority := input.Priority
	if priority == "" {
		priority = TaskPriorityMedium
	}

	labelsJSON, _ := json.Marshal(input.Labels)

	task := &Task{
		BoardID:          input.BoardID,
		ColumnID:         columnID,
		Title:            input.Title,
		Description:      input.Description,
		Status:           TaskStatusTodo,
		Priority:         priority,
		CreatedBy:        input.CreatedBy,
		DueDate:          input.DueDate,
		EstimatedMinutes: input.EstimatedMinutes,
		Position:         maxPos + 1,
		Labels:           labelsJSON,
	}

	if input.AssigneeID > 0 {
		task.AssigneeID = &input.AssigneeID
	}
	if input.WorkflowStepID > 0 {
		task.WorkflowStepID = &input.WorkflowStepID
	}

	if err := s.repo.CreateTask(ctx, task); err != nil {
		return nil, fmt.Errorf("failed to create task: %w", err)
	}

	s.logTaskActivity(ctx, task.ID, input.CreatedBy, ActionCreated, "", "", "")
	_ = s.repo.AddWatcher(ctx, &Watcher{TaskID: task.ID, UserID: input.CreatedBy})

	s.logger.Info("Task created", zap.Int64("task_id", task.ID), zap.String("title", input.Title))
	return task, nil
}

// GetTask - получает задачу с деталями
func (s *Service) GetTask(ctx context.Context, taskID int64) (*Task, []*Comment, []*Attachment, []*ActivityLog, []*Watcher, error) {
	task, err := s.repo.GetTask(ctx, taskID)
	if err != nil {
		return nil, nil, nil, nil, nil, fmt.Errorf("task not found: %w", err)
	}

	comments, attachments, watchers, _ := s.repo.GetTaskCounts(ctx, taskID)
	task.CommentsCount = comments
	task.AttachmentsCount = attachments
	task.WatchersCount = watchers

	if task.DueDate != nil && task.Status != TaskStatusDone {
		task.IsOverdue = task.DueDate.Before(time.Now())
	}

	recentComments, _, _ := s.repo.ListComments(ctx, taskID, 5, 0)
	attachmentsList, _ := s.repo.ListAttachments(ctx, taskID)
	activityList, _, _ := s.repo.ListActivity(ctx, taskID, 10, 0)
	watchersList, _ := s.repo.ListWatchers(ctx, taskID)

	return task, recentComments, attachmentsList, activityList, watchersList, nil
}

// UpdateTaskInput - входные данные для обновления задачи
type UpdateTaskInput struct {
	TaskID           int64
	Title            string
	Description      string
	Priority         string
	DueDate          *time.Time
	EstimatedMinutes int32
	ActualMinutes    int32
	Labels           []string
	WorkflowStepID   int64
	UpdatedBy        int64
}

// UpdateTask - обновляет задачу
func (s *Service) UpdateTask(ctx context.Context, input *UpdateTaskInput) (*Task, error) {
	task, err := s.repo.GetTask(ctx, input.TaskID)
	if err != nil {
		return nil, fmt.Errorf("task not found: %w", err)
	}

	changes := make(map[string][2]string)

	if input.Title != "" && input.Title != task.Title {
		changes["title"] = [2]string{task.Title, input.Title}
		task.Title = input.Title
	}
	if input.Description != "" && input.Description != task.Description {
		changes["description"] = [2]string{task.Description, input.Description}
		task.Description = input.Description
	}
	if input.Priority != "" && input.Priority != task.Priority {
		changes["priority"] = [2]string{task.Priority, input.Priority}
		task.Priority = input.Priority
	}
	if input.DueDate != nil {
		oldDue := ""
		if task.DueDate != nil {
			oldDue = task.DueDate.Format("2006-01-02")
		}
		newDue := input.DueDate.Format("2006-01-02")
		if oldDue != newDue {
			changes["due_date"] = [2]string{oldDue, newDue}
			task.DueDate = input.DueDate
		}
	}
	if input.EstimatedMinutes > 0 {
		task.EstimatedMinutes = input.EstimatedMinutes
	}
	if input.ActualMinutes > 0 {
		task.ActualMinutes = input.ActualMinutes
	}
	if len(input.Labels) > 0 {
		labelsJSON, _ := json.Marshal(input.Labels)
		task.Labels = labelsJSON
	}
	if input.WorkflowStepID > 0 {
		task.WorkflowStepID = &input.WorkflowStepID
	}

	if err := s.repo.UpdateTask(ctx, task); err != nil {
		return nil, fmt.Errorf("failed to update task: %w", err)
	}

	for field, vals := range changes {
		s.logTaskActivity(ctx, task.ID, input.UpdatedBy, ActionUpdated, field, vals[0], vals[1])
	}

	s.logger.Info("Task updated", zap.Int64("task_id", task.ID))
	return task, nil
}

// DeleteTask - удаляет задачу с проверкой прав через team_service
func (s *Service) DeleteTask(ctx context.Context, taskID, deletedBy int64) error {
	task, err := s.repo.GetTask(ctx, taskID)
	if err != nil {
		return fmt.Errorf("task not found: %w", err)
	}

	canDelete := false
	reason := ""

	// 1. Создатель задачи
	if task.CreatedBy == deletedBy {
		canDelete = true
		reason = "creator"
	}

	// 2. Assignee
	if !canDelete && task.AssigneeID != nil && *task.AssigneeID == deletedBy {
		canDelete = true
		reason = "assignee"
	}

	// 3. Лидер команды (через team_service)
	if !canDelete && s.teamClient != nil {
		board, _ := s.repo.GetBoard(ctx, task.BoardID)
		if board != nil {
			teamResp, err := s.teamClient.GetTeam(ctx, &teamv1.GetTeamRequest{TeamId: board.TeamID})
			if err == nil {
				for _, member := range teamResp.Members {
					if member.UserId == deletedBy && member.Role == "leader" {
						canDelete = true
						reason = "team_leader"
						break
					}
				}
			} else {
				s.logger.Warn("Failed to get team for delete permission check",
					zap.Error(err),
					zap.Int64("team_id", board.TeamID))
			}
		}
	}

	if !canDelete {
		s.logger.Warn("Unauthorized task deletion attempt",
			zap.Int64("task_id", taskID),
			zap.Int64("user_id", deletedBy),
			zap.Int64("created_by", task.CreatedBy))
		return errors.New("you don't have permission to delete this task")
	}

	if err := s.repo.DeleteTask(ctx, taskID); err != nil {
		return fmt.Errorf("failed to delete task: %w", err)
	}

	s.logTaskActivity(ctx, taskID, deletedBy, ActionDeleted, "", "", "")
	s.logger.Info("Task deleted",
		zap.Int64("task_id", taskID),
		zap.Int64("deleted_by", deletedBy),
		zap.String("reason", reason))
	return nil
}

// ListTasks - список задач с фильтрацией (batch-оптимизация)
func (s *Service) ListTasks(ctx context.Context, filter TaskFilter) ([]*Task, int64, error) {
	tasks, total, err := s.repo.ListTasks(ctx, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list tasks: %w", err)
	}

	if len(tasks) == 0 {
		return tasks, total, nil
	}

	taskIDs := make([]int64, len(tasks))
	for i, task := range tasks {
		taskIDs[i] = task.ID
	}

	counts, err := s.repo.GetTasksCountsBatch(ctx, taskIDs)
	if err != nil {
		s.logger.Warn("Failed to get task counts batch", zap.Error(err))
		counts = make(map[int64]TaskCounts)
	}

	for _, task := range tasks {
		if c, ok := counts[task.ID]; ok {
			task.CommentsCount = c.Comments
			task.AttachmentsCount = c.Attachments
			task.WatchersCount = c.Watchers
		}

		if task.DueDate != nil && task.Status != TaskStatusDone {
			task.IsOverdue = task.DueDate.Before(time.Now())
		}
	}

	return tasks, total, nil
}

// MoveTask - перемещает задачу с проверкой WIP-лимита
func (s *Service) MoveTask(ctx context.Context, taskID, toColumnID int64, position int32, movedBy int64) (*Task, error) {
	task, err := s.repo.GetTask(ctx, taskID)
	if err != nil {
		return nil, fmt.Errorf("task not found: %w", err)
	}

	oldColumnID := task.ColumnID

	oldColumn, _ := s.repo.GetColumn(ctx, oldColumnID)
	newColumn, err := s.repo.GetColumn(ctx, toColumnID)
	if err != nil {
		return nil, fmt.Errorf("target column not found: %w", err)
	}

	if newColumn.BoardID != task.BoardID {
		return nil, errors.New("cannot move task to column on different board")
	}

	// Проверка WIP-лимита
	if oldColumnID != toColumnID && newColumn.WIPLimit > 0 {
		counts, err := s.repo.GetColumnTaskCounts(ctx, newColumn.BoardID)
		if err != nil {
			return nil, fmt.Errorf("failed to check column task count: %w", err)
		}

		currentTasksInTarget := counts[toColumnID]
		if currentTasksInTarget >= newColumn.WIPLimit {
			return nil, fmt.Errorf("WIP limit reached: column '%s' already has %d/%d tasks",
				newColumn.Name, currentTasksInTarget, newColumn.WIPLimit)
		}
	}

	if err := s.repo.MoveTask(ctx, taskID, toColumnID, position); err != nil {
		return nil, fmt.Errorf("failed to move task: %w", err)
	}

	oldVal := ""
	if oldColumn != nil {
		oldVal = oldColumn.Name
	}
	s.logTaskActivity(ctx, taskID, movedBy, ActionMoved, "column", oldVal, newColumn.Name)

	s.logger.Info("Task moved",
		zap.Int64("task_id", taskID),
		zap.Int64("from_column", oldColumnID),
		zap.Int64("to_column", toColumnID))

	return s.repo.GetTask(ctx, taskID)
}

// ReorderTasks - изменяет порядок задач с валидацией
func (s *Service) ReorderTasks(ctx context.Context, columnID int64, taskIDs []int64) error {
	if len(taskIDs) == 0 {
		return nil
	}

	column, err := s.repo.GetColumn(ctx, columnID)
	if err != nil {
		return fmt.Errorf("column not found: %w", err)
	}

	existingTasks, _, err := s.repo.ListTasks(ctx, TaskFilter{
		ColumnID: columnID,
		Limit:    0,
	})
	if err != nil {
		return fmt.Errorf("failed to get column tasks: %w", err)
	}

	existingTaskIDs := make(map[int64]bool)
	for _, task := range existingTasks {
		existingTaskIDs[task.ID] = true
	}

	for _, taskID := range taskIDs {
		if !existingTaskIDs[taskID] {
			return fmt.Errorf("task %d does not belong to column %d", taskID, columnID)
		}
	}

	if len(taskIDs) != len(existingTasks) {
		return fmt.Errorf("task_ids count (%d) does not match column tasks count (%d)",
			len(taskIDs), len(existingTasks))
	}

	seen := make(map[int64]bool)
	for _, taskID := range taskIDs {
		if seen[taskID] {
			return fmt.Errorf("duplicate task_id: %d", taskID)
		}
		seen[taskID] = true
	}

	s.logger.Debug("Reordering tasks",
		zap.Int64("column_id", columnID),
		zap.Int64("board_id", column.BoardID),
		zap.Int("tasks_count", len(taskIDs)))

	return s.repo.ReorderTasks(ctx, columnID, taskIDs)
}

// AssignTask - назначает задачу с проверкой членства через team_service
func (s *Service) AssignTask(ctx context.Context, taskID, assigneeID, assignedBy int64) (*Task, error) {
	task, err := s.repo.GetTask(ctx, taskID)
	if err != nil {
		return nil, fmt.Errorf("task not found: %w", err)
	}

	board, err := s.repo.GetBoard(ctx, task.BoardID)
	if err != nil {
		return nil, fmt.Errorf("board not found: %w", err)
	}

	// Проверка членства в команде через team_service
	if s.teamClient != nil {
		teamResp, err := s.teamClient.GetTeam(ctx, &teamv1.GetTeamRequest{
			TeamId: board.TeamID,
		})
		if err != nil {
			s.logger.Warn("Failed to get team for membership check",
				zap.Error(err),
				zap.Int64("team_id", board.TeamID))
			// Продолжаем без проверки если team_service недоступен
		} else {
			isMember := false
			for _, member := range teamResp.Members {
				if member.UserId == assigneeID {
					isMember = true
					break
				}
			}

			if !isMember {
				return nil, fmt.Errorf("user %d is not a member of team %d", assigneeID, board.TeamID)
			}

			s.logger.Debug("Team membership verified",
				zap.Int64("user_id", assigneeID),
				zap.Int64("team_id", board.TeamID))
		}
	} else {
		s.logger.Warn("Team client is nil, skipping membership check",
			zap.Int64("task_id", taskID),
			zap.Int64("assignee_id", assigneeID))
	}

	oldAssignee := ""
	if task.AssigneeID != nil {
		oldAssignee = fmt.Sprintf("%d", *task.AssigneeID)
	}

	task.AssigneeID = &assigneeID

	if err := s.repo.UpdateTask(ctx, task); err != nil {
		return nil, fmt.Errorf("failed to assign task: %w", err)
	}

	s.logTaskActivity(ctx, taskID, assignedBy, ActionAssigned, "assignee", oldAssignee, fmt.Sprintf("%d", assigneeID))
	_ = s.repo.AddWatcher(ctx, &Watcher{TaskID: taskID, UserID: assigneeID})

	s.logger.Info("Task assigned",
		zap.Int64("task_id", taskID),
		zap.Int64("assignee_id", assigneeID),
		zap.Int64("assigned_by", assignedBy))

	return task, nil
}

// UnassignTask - снимает назначение
func (s *Service) UnassignTask(ctx context.Context, taskID, unassignedBy int64) (*Task, error) {
	task, err := s.repo.GetTask(ctx, taskID)
	if err != nil {
		return nil, fmt.Errorf("task not found: %w", err)
	}

	oldAssignee := ""
	if task.AssigneeID != nil {
		oldAssignee = fmt.Sprintf("%d", *task.AssigneeID)
	}

	task.AssigneeID = nil

	if err := s.repo.UpdateTask(ctx, task); err != nil {
		return nil, fmt.Errorf("failed to unassign task: %w", err)
	}

	s.logTaskActivity(ctx, taskID, unassignedBy, ActionUnassigned, "assignee", oldAssignee, "")

	return task, nil
}

// ==================== Comments ====================

// CreateCommentInput - входные данные для создания комментария
type CreateCommentInput struct {
	TaskID         int64
	AuthorID       int64
	Content        string
	MentionUserIDs []int64
}

// CreateComment - создает комментарий
func (s *Service) CreateComment(ctx context.Context, input *CreateCommentInput) (*Comment, error) {
	_, err := s.repo.GetTask(ctx, input.TaskID)
	if err != nil {
		return nil, fmt.Errorf("task not found: %w", err)
	}

	var mentions []UserMention
	for _, userID := range input.MentionUserIDs {
		mentions = append(mentions, UserMention{UserID: userID})
	}
	mentionsJSON, _ := json.Marshal(mentions)

	comment := &Comment{
		TaskID:   input.TaskID,
		AuthorID: input.AuthorID,
		Content:  input.Content,
		Mentions: mentionsJSON,
	}

	if err := s.repo.CreateComment(ctx, comment); err != nil {
		return nil, fmt.Errorf("failed to create comment: %w", err)
	}

	s.logTaskActivity(ctx, input.TaskID, input.AuthorID, ActionCommented, "", "", "")
	_ = s.repo.AddWatcher(ctx, &Watcher{TaskID: input.TaskID, UserID: input.AuthorID})

	return comment, nil
}

// UpdateComment - обновляет комментарий
func (s *Service) UpdateComment(ctx context.Context, commentID int64, content string, updatedBy int64) (*Comment, error) {
	comment, err := s.repo.GetComment(ctx, commentID)
	if err != nil {
		return nil, fmt.Errorf("comment not found: %w", err)
	}

	if comment.AuthorID != updatedBy {
		return nil, errors.New("only author can edit comment")
	}

	comment.Content = content

	if err := s.repo.UpdateComment(ctx, comment); err != nil {
		return nil, fmt.Errorf("failed to update comment: %w", err)
	}

	return comment, nil
}

// DeleteComment - удаляет комментарий
func (s *Service) DeleteComment(ctx context.Context, commentID, deletedBy int64) error {
	comment, err := s.repo.GetComment(ctx, commentID)
	if err != nil {
		return fmt.Errorf("comment not found: %w", err)
	}

	if comment.AuthorID != deletedBy {
		return fmt.Errorf("only author can delete comment")
	}

	return s.repo.DeleteComment(ctx, commentID)
}

// ListComments - список комментариев задачи
func (s *Service) ListComments(ctx context.Context, taskID int64, page, pageSize int32) ([]*Comment, int64, error) {
	if pageSize <= 0 {
		pageSize = 20
	}
	offset := int((page - 1) * pageSize)
	if offset < 0 {
		offset = 0
	}

	return s.repo.ListComments(ctx, taskID, int(pageSize), offset)
}

// ==================== Attachments ====================

// AddAttachmentInput - входные данные для добавления вложения
type AddAttachmentInput struct {
	TaskID     int64
	FileID     string
	FileName   string
	FileType   string
	FileSize   int64
	UploadedBy int64
}

// AddAttachment - добавляет вложение
func (s *Service) AddAttachment(ctx context.Context, input *AddAttachmentInput) (*Attachment, error) {
	_, err := s.repo.GetTask(ctx, input.TaskID)
	if err != nil {
		return nil, fmt.Errorf("task not found: %w", err)
	}

	attachment := &Attachment{
		TaskID:     input.TaskID,
		FileID:     input.FileID,
		FileName:   input.FileName,
		FileType:   input.FileType,
		FileSize:   input.FileSize,
		UploadedBy: input.UploadedBy,
	}

	if err := s.repo.CreateAttachment(ctx, attachment); err != nil {
		return nil, fmt.Errorf("failed to add attachment: %w", err)
	}

	return attachment, nil
}

// RemoveAttachment - удаляет вложение
func (s *Service) RemoveAttachment(ctx context.Context, attachmentID, removedBy int64) error {
	attachment, err := s.repo.GetAttachment(ctx, attachmentID)
	if err != nil {
		return fmt.Errorf("attachment not found: %w", err)
	}

	if attachment.UploadedBy != removedBy {
		return fmt.Errorf("only uploader can remove attachment")
	}

	return s.repo.DeleteAttachment(ctx, attachmentID)
}

// ListAttachments - список вложений задачи
func (s *Service) ListAttachments(ctx context.Context, taskID int64) ([]*Attachment, error) {
	return s.repo.ListAttachments(ctx, taskID)
}

// ==================== Activity ====================

// GetTaskActivity - история изменений задачи
func (s *Service) GetTaskActivity(ctx context.Context, taskID int64, page, pageSize int32) ([]*ActivityLog, int64, error) {
	if pageSize <= 0 {
		pageSize = 20
	}
	offset := int((page - 1) * pageSize)
	if offset < 0 {
		offset = 0
	}

	return s.repo.ListActivity(ctx, taskID, int(pageSize), offset)
}

// logTaskActivity - вспомогательный метод для логирования
func (s *Service) logTaskActivity(ctx context.Context, taskID, actorID int64, action, fieldName, oldValue, newValue string) {
	log := &ActivityLog{
		TaskID:    taskID,
		ActorID:   actorID,
		Action:    action,
		FieldName: fieldName,
		OldValue:  oldValue,
		NewValue:  newValue,
	}
	if err := s.repo.LogActivity(ctx, log); err != nil {
		s.logger.Error("Failed to log activity", zap.Error(err))
	}
}

// ==================== Watchers ====================

// AddWatcher - добавляет наблюдателя
func (s *Service) AddWatcher(ctx context.Context, taskID, userID int64) error {
	isWatching, _ := s.repo.IsWatching(ctx, taskID, userID)
	if isWatching {
		return nil
	}

	return s.repo.AddWatcher(ctx, &Watcher{TaskID: taskID, UserID: userID})
}

// RemoveWatcher - удаляет наблюдателя
func (s *Service) RemoveWatcher(ctx context.Context, taskID, userID int64) error {
	return s.repo.RemoveWatcher(ctx, taskID, userID)
}

// ListWatchers - список наблюдателей
func (s *Service) ListWatchers(ctx context.Context, taskID int64) ([]*Watcher, error) {
	return s.repo.ListWatchers(ctx, taskID)
}

// ==================== Stats ====================

// GetBoardStats - статистика доски
func (s *Service) GetBoardStats(ctx context.Context, boardID int64) (*BoardStats, error) {
	return s.repo.GetBoardStats(ctx, boardID)
}

// GetMemberStats - статистика по участникам
func (s *Service) GetMemberStats(ctx context.Context, boardID int64) ([]*MemberStats, error) {
	return s.repo.GetMemberStats(ctx, boardID)
}

// GetDailyStats - статистика за период
func (s *Service) GetDailyStats(ctx context.Context, boardID int64, days int) ([]*DailyStats, error) {
	if days <= 0 {
		days = 14
	}
	return s.repo.GetDailyStats(ctx, boardID, days)
}

// ==================== My Tasks ====================

// GetMyTasks - мои задачи
func (s *Service) GetMyTasks(ctx context.Context, userID int64, filter MyTasksFilter) ([]*Task, int64, error) {
	return s.repo.GetMyTasks(ctx, userID, filter)
}

// GetOverdueTasks - просроченные задачи
func (s *Service) GetOverdueTasks(ctx context.Context, boardID int64, assigneeID *int64, page, pageSize int32) ([]*Task, int64, error) {
	if pageSize <= 0 {
		pageSize = 20
	}
	offset := int((page - 1) * pageSize)
	if offset < 0 {
		offset = 0
	}

	return s.repo.GetOverdueTasks(ctx, boardID, assigneeID, int(pageSize), offset)
}

// GetUpcomingDeadlines - задачи с приближающимися дедлайнами
func (s *Service) GetUpcomingDeadlines(ctx context.Context, boardID int64, userID *int64, daysAhead int, page, pageSize int32) ([]*Task, int64, error) {
	if pageSize <= 0 {
		pageSize = 20
	}
	if daysAhead <= 0 {
		daysAhead = 7
	}
	offset := int((page - 1) * pageSize)
	if offset < 0 {
		offset = 0
	}

	return s.repo.GetUpcomingDeadlines(ctx, boardID, userID, daysAhead, int(pageSize), offset)
}

// ==================== Bulk Operations ====================

// BulkUpdateTasksInput - входные данные для массового обновления
type BulkUpdateTasksInput struct {
	TaskIDs      []int64
	AssigneeID   int64
	Priority     string
	ColumnID     int64
	AddLabels    []string
	RemoveLabels []string
	UpdatedBy    int64
}

// BulkUpdateTasks - массовое обновление с проверкой WIP-лимита
func (s *Service) BulkUpdateTasks(ctx context.Context, input *BulkUpdateTasksInput) ([]*Task, error) {
	if len(input.TaskIDs) == 0 {
		return nil, errors.New("task_ids cannot be empty")
	}

	updates := make(map[string]interface{})

	if input.AssigneeID == -1 {
		updates["assignee_id"] = nil
	} else if input.AssigneeID > 0 {
		updates["assignee_id"] = input.AssigneeID
	}

	if input.Priority != "" {
		updates["priority"] = input.Priority
	}

	if input.ColumnID > 0 {
		targetColumn, err := s.repo.GetColumn(ctx, input.ColumnID)
		if err != nil {
			return nil, fmt.Errorf("target column not found: %w", err)
		}

		if targetColumn.WIPLimit > 0 {
			counts, _ := s.repo.GetColumnTaskCounts(ctx, targetColumn.BoardID)
			currentTasks := counts[input.ColumnID]

			tasksAlreadyInTarget := 0
			for _, taskID := range input.TaskIDs {
				task, err := s.repo.GetTask(ctx, taskID)
				if err == nil && task.ColumnID == input.ColumnID {
					tasksAlreadyInTarget++
				}
			}

			newTasksCount := len(input.TaskIDs) - tasksAlreadyInTarget

			if currentTasks+int32(newTasksCount) > targetColumn.WIPLimit {
				return nil, fmt.Errorf("WIP limit exceeded: moving %d tasks would result in %d/%d tasks in column '%s'",
					newTasksCount, currentTasks+int32(newTasksCount), targetColumn.WIPLimit, targetColumn.Name)
			}
		}

		updates["column_id"] = input.ColumnID
	}

	if len(updates) > 0 {
		if err := s.repo.BulkUpdateTasks(ctx, input.TaskIDs, updates); err != nil {
			return nil, fmt.Errorf("failed to bulk update: %w", err)
		}
	}

	var result []*Task
	for _, id := range input.TaskIDs {
		task, err := s.repo.GetTask(ctx, id)
		if err == nil {
			result = append(result, task)
		}
	}

	s.logger.Info("Bulk update completed",
		zap.Int("tasks_count", len(input.TaskIDs)),
		zap.Int64("updated_by", input.UpdatedBy))

	return result, nil
}

// BulkDeleteTasks - массовое удаление задач
func (s *Service) BulkDeleteTasks(ctx context.Context, taskIDs []int64, deletedBy int64) error {
	if len(taskIDs) == 0 {
		return errors.New("task_ids cannot be empty")
	}
	return s.repo.BulkDeleteTasks(ctx, taskIDs)
}
