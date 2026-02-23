package task

import (
	"context"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
)

// TaskCounts - счётчики для задачи (для batch-запроса)
type TaskCounts struct {
	Comments    int32
	Attachments int32
	Watchers    int32
}

// Repository - интерфейс репозитория для task_service
type Repository interface {
	// Board
	CreateBoard(ctx context.Context, board *Board) error
	GetBoard(ctx context.Context, id int64) (*Board, error)
	GetBoardByProject(ctx context.Context, projectID int64) (*Board, error)
	ListMyBoards(ctx context.Context, userID int64, role string) ([]*Board, error)
	UpdateBoard(ctx context.Context, board *Board) error
	DeleteBoard(ctx context.Context, id int64) error

	// Column
	CreateColumn(ctx context.Context, column *Column) error
	GetColumn(ctx context.Context, id int64) (*Column, error)
	ListColumns(ctx context.Context, boardID int64) ([]*Column, error)
	UpdateColumn(ctx context.Context, column *Column) error
	DeleteColumn(ctx context.Context, id int64) error
	GetDefaultColumn(ctx context.Context, boardID int64) (*Column, error)
	GetColumnBySlug(ctx context.Context, boardID int64, slug string) (*Column, error)
	ReorderColumns(ctx context.Context, boardID int64, columnIDs []int64) error
	GetMaxColumnOrder(ctx context.Context, boardID int64) (int32, error)

	// Task
	CreateTask(ctx context.Context, task *Task) error
	GetTask(ctx context.Context, id int64) (*Task, error)
	UpdateTask(ctx context.Context, task *Task) error
	DeleteTask(ctx context.Context, id int64) error
	ListTasks(ctx context.Context, filter TaskFilter) ([]*Task, int64, error)
	MoveTask(ctx context.Context, taskID, toColumnID int64, position int32) error
	ReorderTasks(ctx context.Context, columnID int64, taskIDs []int64) error
	GetMaxPosition(ctx context.Context, columnID int64) (int32, error)
	BulkUpdateTasks(ctx context.Context, taskIDs []int64, updates map[string]interface{}) error
	BulkDeleteTasks(ctx context.Context, taskIDs []int64) error

	// Task Counts
	GetTaskCounts(ctx context.Context, taskID int64) (comments, attachments, watchers int32, err error)
	GetColumnTaskCounts(ctx context.Context, boardID int64) (map[int64]int32, error)
	GetTasksCountsBatch(ctx context.Context, taskIDs []int64) (map[int64]TaskCounts, error)

	// Comment
	CreateComment(ctx context.Context, comment *Comment) error
	GetComment(ctx context.Context, id int64) (*Comment, error)
	UpdateComment(ctx context.Context, comment *Comment) error
	DeleteComment(ctx context.Context, id int64) error
	ListComments(ctx context.Context, taskID int64, limit, offset int) ([]*Comment, int64, error)

	// Attachment
	CreateAttachment(ctx context.Context, attachment *Attachment) error
	GetAttachment(ctx context.Context, id int64) (*Attachment, error)
	DeleteAttachment(ctx context.Context, id int64) error
	ListAttachments(ctx context.Context, taskID int64) ([]*Attachment, error)

	// Activity Log
	LogActivity(ctx context.Context, log *ActivityLog) error
	ListActivity(ctx context.Context, taskID int64, limit, offset int) ([]*ActivityLog, int64, error)

	// Watcher
	AddWatcher(ctx context.Context, watcher *Watcher) error
	RemoveWatcher(ctx context.Context, taskID, userID int64) error
	ListWatchers(ctx context.Context, taskID int64) ([]*Watcher, error)
	IsWatching(ctx context.Context, taskID, userID int64) (bool, error)

	// Stats
	GetBoardStats(ctx context.Context, boardID int64) (*BoardStats, error)
	GetMemberStats(ctx context.Context, boardID int64) ([]*MemberStats, error)
	GetDailyStats(ctx context.Context, boardID int64, days int) ([]*DailyStats, error)

	// My Tasks
	GetMyTasks(ctx context.Context, userID int64, filter MyTasksFilter) ([]*Task, int64, error)
	GetOverdueTasks(ctx context.Context, boardID int64, assigneeID *int64, limit, offset int) ([]*Task, int64, error)
	GetUpcomingDeadlines(ctx context.Context, boardID int64, userID *int64, daysAhead, limit, offset int) ([]*Task, int64, error)

	// Deadline notifier
	ListTasksDueOn(ctx context.Context, dueDate time.Time) ([]*Task, error)
	ListOverdueOpenTasks(ctx context.Context, beforeDate time.Time) ([]*Task, error)
	TryInsertDeadlineRun(ctx context.Context, run *DeadlineNotificationRun) (bool, error)
}

// TaskFilter - фильтр для списка задач
type TaskFilter struct {
	BoardID        int64
	ColumnID       int64
	AssigneeID     int64
	Status         string
	Priority       string
	Search         string
	Labels         []string
	OnlyOverdue    bool
	OnlyUnassigned bool
	SortBy         string
	SortOrder      string
	Limit          int
	Offset         int
}

// MyTasksFilter - фильтр для "моих задач"
type MyTasksFilter struct {
	OnlyAssigned     bool
	OnlyCreated      bool
	OnlyWatching     bool
	IncludeCompleted bool
	Limit            int
	Offset           int
}

type repository struct {
	db *gorm.DB
}

// NewRepository создает новый репозиторий
func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

// ==================== Board ====================

func (r *repository) CreateBoard(ctx context.Context, board *Board) error {
	board.CreatedAt = time.Now()
	board.UpdatedAt = time.Now()
	return r.db.WithContext(ctx).Create(board).Error
}

func (r *repository) GetBoard(ctx context.Context, id int64) (*Board, error) {
	var board Board
	if err := r.db.WithContext(ctx).First(&board, id).Error; err != nil {
		return nil, err
	}
	return &board, nil
}

func (r *repository) GetBoardByProject(ctx context.Context, projectID int64) (*Board, error) {
	var board Board
	if err := r.db.WithContext(ctx).
		Where("project_id = ? AND deleted_at IS NULL", projectID).
		First(&board).Error; err != nil {
		return nil, err
	}
	return &board, nil
}

// ListMyBoards:
// - student: доски проектов, где user либо владелец (projects.student_id), либо участник команды (team_members)
// - teacher: доски проектов, где user научрук команды (admin_supervisor_assignments.team_id)
// - admin: все доски
func (r *repository) ListMyBoards(ctx context.Context, userID int64, role string) ([]*Board, error) {
	var boards []*Board

	q := r.db.WithContext(ctx).Model(&Board{}).
		Where("task_boards.deleted_at IS NULL")

	switch role {
	case "teacher":
		q = q.
			Joins("JOIN admin_supervisor_assignments a ON a.team_id = task_boards.team_id").
			Where("a.supervisor_id = ?", userID)

	case "admin":
		// all boards

	default: // student (и любые прочие роли — как student)
		q = q.
			Joins("JOIN projects p ON p.id = task_boards.project_id").
			Joins("LEFT JOIN team_members tm ON tm.team_id = task_boards.team_id AND tm.user_id = ?", userID).
			Where("p.student_id = ? OR tm.user_id = ?", userID, userID).
			Select("DISTINCT task_boards.*")
	}

	if err := q.Order("task_boards.created_at DESC").Find(&boards).Error; err != nil {
		return nil, err
	}
	return boards, nil
}

func (r *repository) UpdateBoard(ctx context.Context, board *Board) error {
	board.UpdatedAt = time.Now()
	return r.db.WithContext(ctx).Save(board).Error
}

func (r *repository) DeleteBoard(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&Board{}, id).Error
}

// ==================== Columns ====================

func (r *repository) CreateColumn(ctx context.Context, column *Column) error {
	column.CreatedAt = time.Now()
	column.UpdatedAt = time.Now()
	return r.db.WithContext(ctx).Create(column).Error
}

func (r *repository) GetColumn(ctx context.Context, id int64) (*Column, error) {
	var column Column
	if err := r.db.WithContext(ctx).First(&column, id).Error; err != nil {
		return nil, err
	}
	return &column, nil
}

func (r *repository) ListColumns(ctx context.Context, boardID int64) ([]*Column, error) {
	var columns []*Column
	err := r.db.WithContext(ctx).
		Where("board_id = ?", boardID).
		Order("order_index ASC").
		Find(&columns).Error
	return columns, err
}

func (r *repository) UpdateColumn(ctx context.Context, column *Column) error {
	column.UpdatedAt = time.Now()
	return r.db.WithContext(ctx).Save(column).Error
}

func (r *repository) DeleteColumn(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&Column{}, id).Error
}

func (r *repository) GetDefaultColumn(ctx context.Context, boardID int64) (*Column, error) {
	var column Column
	err := r.db.WithContext(ctx).
		Where("board_id = ? AND is_default = ?", boardID, true).
		First(&column).Error
	if err != nil {
		// Fallback: первая колонка
		err = r.db.WithContext(ctx).
			Where("board_id = ?", boardID).
			Order("order_index ASC").
			First(&column).Error
	}
	return &column, err
}

func (r *repository) GetColumnBySlug(ctx context.Context, boardID int64, slug string) (*Column, error) {
	var column Column
	err := r.db.WithContext(ctx).
		Where("board_id = ? AND slug = ?", boardID, slug).
		First(&column).Error
	return &column, err
}

func (r *repository) ReorderColumns(ctx context.Context, boardID int64, columnIDs []int64) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for i, id := range columnIDs {
			if err := tx.Model(&Column{}).
				Where("id = ? AND board_id = ?", id, boardID).
				Update("order_index", i).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *repository) GetMaxColumnOrder(ctx context.Context, boardID int64) (int32, error) {
	var maxOrder *int32
	err := r.db.WithContext(ctx).
		Model(&Column{}).
		Where("board_id = ?", boardID).
		Select("MAX(order_index)").
		Scan(&maxOrder).Error
	if err != nil || maxOrder == nil {
		return 0, err
	}
	return *maxOrder, nil
}

// ==================== Task ====================

func (r *repository) CreateTask(ctx context.Context, task *Task) error {
	task.CreatedAt = time.Now()
	task.UpdatedAt = time.Now()
	return r.db.WithContext(ctx).Create(task).Error
}

func (r *repository) GetTask(ctx context.Context, id int64) (*Task, error) {
	var task Task
	if err := r.db.WithContext(ctx).First(&task, id).Error; err != nil {
		return nil, err
	}
	return &task, nil
}

func (r *repository) UpdateTask(ctx context.Context, task *Task) error {
	task.UpdatedAt = time.Now()
	return r.db.WithContext(ctx).Save(task).Error
}

func (r *repository) DeleteTask(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&Task{}, id).Error
}

func (r *repository) ListTasks(ctx context.Context, filter TaskFilter) ([]*Task, int64, error) {
	var tasks []*Task
	var total int64

	query := r.db.WithContext(ctx).Model(&Task{}).Where("deleted_at IS NULL")

	if filter.BoardID > 0 {
		query = query.Where("board_id = ?", filter.BoardID)
	}
	if filter.ColumnID > 0 {
		query = query.Where("column_id = ?", filter.ColumnID)
	}
	if filter.AssigneeID > 0 {
		query = query.Where("assignee_id = ?", filter.AssigneeID)
	}
	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}
	if filter.Priority != "" {
		query = query.Where("priority = ?", filter.Priority)
	}
	if filter.Search != "" {
		search := "%" + filter.Search + "%"
		query = query.Where("title ILIKE ? OR description ILIKE ?", search, search)
	}
	if filter.OnlyOverdue {
		query = query.Where("due_date < ? AND status != ?", time.Now().Format("2006-01-02"), TaskStatusDone)
	}
	if filter.OnlyUnassigned {
		query = query.Where("assignee_id IS NULL")
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	sortBy := "position"
	if filter.SortBy != "" {
		sortBy = filter.SortBy
	}
	sortOrder := "ASC"
	if filter.SortOrder == "desc" {
		sortOrder = "DESC"
	}
	query = query.Order(fmt.Sprintf("%s %s", sortBy, sortOrder))

	if filter.Limit > 0 {
		query = query.Limit(filter.Limit).Offset(filter.Offset)
	}

	err := query.Find(&tasks).Error
	return tasks, total, err
}

// MoveTask - перемещает задачу
func (r *repository) MoveTask(ctx context.Context, taskID, toColumnID int64, position int32) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var task Task
		if err := tx.First(&task, taskID).Error; err != nil {
			return err
		}

		oldColumnID := task.ColumnID
		oldPosition := task.Position

		var toColumn Column
		if err := tx.First(&toColumn, toColumnID).Error; err != nil {
			return err
		}

		newStatus := task.Status
		switch toColumn.Slug {
		case "todo":
			newStatus = TaskStatusTodo
		case "in_progress":
			newStatus = TaskStatusInProgress
		case "review":
			newStatus = TaskStatusReview
		case "done":
			newStatus = TaskStatusDone
		}

		if oldColumnID == toColumnID {
			if oldPosition == position {
				return nil
			}

			if oldPosition < position {
				tx.Model(&Task{}).
					Where("column_id = ? AND position > ? AND position <= ? AND id != ?",
						toColumnID, oldPosition, position, taskID).
					UpdateColumn("position", gorm.Expr("position - 1"))
			} else {
				tx.Model(&Task{}).
					Where("column_id = ? AND position >= ? AND position < ? AND id != ?",
						toColumnID, position, oldPosition, taskID).
					UpdateColumn("position", gorm.Expr("position + 1"))
			}
		} else {
			tx.Model(&Task{}).
				Where("column_id = ? AND position > ?", oldColumnID, oldPosition).
				UpdateColumn("position", gorm.Expr("position - 1"))

			tx.Model(&Task{}).
				Where("column_id = ? AND position >= ?", toColumnID, position).
				UpdateColumn("position", gorm.Expr("position + 1"))
		}

		updates := map[string]interface{}{
			"column_id":  toColumnID,
			"position":   position,
			"status":     newStatus,
			"updated_at": time.Now(),
		}

		if toColumn.IsDoneColumn && task.CompletedAt == nil {
			now := time.Now()
			updates["completed_at"] = now
		}
		if !toColumn.IsDoneColumn && task.CompletedAt != nil {
			updates["completed_at"] = nil
		}
		if toColumn.Slug == "in_progress" && task.StartedAt == nil {
			now := time.Now()
			updates["started_at"] = now
		}

		return tx.Model(&Task{}).Where("id = ?", taskID).Updates(updates).Error
	})
}

func (r *repository) ReorderTasks(ctx context.Context, columnID int64, taskIDs []int64) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for i, id := range taskIDs {
			if err := tx.Model(&Task{}).
				Where("id = ? AND column_id = ?", id, columnID).
				Update("position", i).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *repository) GetMaxPosition(ctx context.Context, columnID int64) (int32, error) {
	var maxPos *int32
	err := r.db.WithContext(ctx).
		Model(&Task{}).
		Where("column_id = ? AND deleted_at IS NULL", columnID).
		Select("MAX(position)").
		Scan(&maxPos).Error
	if err != nil || maxPos == nil {
		return 0, err
	}
	return *maxPos, nil
}

func (r *repository) BulkUpdateTasks(ctx context.Context, taskIDs []int64, updates map[string]interface{}) error {
	if len(taskIDs) == 0 {
		return nil
	}
	updates["updated_at"] = time.Now()
	return r.db.WithContext(ctx).
		Model(&Task{}).
		Where("id IN ?", taskIDs).
		Updates(updates).Error
}

func (r *repository) BulkDeleteTasks(ctx context.Context, taskIDs []int64) error {
	if len(taskIDs) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).
		Where("id IN ?", taskIDs).
		Delete(&Task{}).Error
}

func (r *repository) GetTaskCounts(ctx context.Context, taskID int64) (comments, attachments, watchers int32, err error) {
	var c, a, w int64

	r.db.WithContext(ctx).Model(&Comment{}).Where("task_id = ? AND deleted_at IS NULL", taskID).Count(&c)
	r.db.WithContext(ctx).Model(&Attachment{}).Where("task_id = ?", taskID).Count(&a)
	r.db.WithContext(ctx).Model(&Watcher{}).Where("task_id = ?", taskID).Count(&w)

	return int32(c), int32(a), int32(w), nil
}

func (r *repository) GetColumnTaskCounts(ctx context.Context, boardID int64) (map[int64]int32, error) {
	type result struct {
		ColumnID int64
		Count    int64
	}
	var results []result

	err := r.db.WithContext(ctx).
		Model(&Task{}).
		Select("column_id, COUNT(*) as count").
		Where("board_id = ? AND deleted_at IS NULL", boardID).
		Group("column_id").
		Find(&results).Error

	counts := make(map[int64]int32)
	for _, r := range results {
		counts[r.ColumnID] = int32(r.Count)
	}
	return counts, err
}

func (r *repository) GetTasksCountsBatch(ctx context.Context, taskIDs []int64) (map[int64]TaskCounts, error) {
	if len(taskIDs) == 0 {
		return make(map[int64]TaskCounts), nil
	}

	result := make(map[int64]TaskCounts)
	for _, id := range taskIDs {
		result[id] = TaskCounts{}
	}

	type countResult struct {
		TaskID int64
		Count  int64
	}

	var commentCounts []countResult
	r.db.WithContext(ctx).
		Model(&Comment{}).
		Select("task_id, COUNT(*) as count").
		Where("task_id IN ? AND deleted_at IS NULL", taskIDs).
		Group("task_id").
		Find(&commentCounts)
	for _, c := range commentCounts {
		if counts, ok := result[c.TaskID]; ok {
			counts.Comments = int32(c.Count)
			result[c.TaskID] = counts
		}
	}

	var attachmentCounts []countResult
	r.db.WithContext(ctx).
		Model(&Attachment{}).
		Select("task_id, COUNT(*) as count").
		Where("task_id IN ?", taskIDs).
		Group("task_id").
		Find(&attachmentCounts)
	for _, c := range attachmentCounts {
		if counts, ok := result[c.TaskID]; ok {
			counts.Attachments = int32(c.Count)
			result[c.TaskID] = counts
		}
	}

	var watcherCounts []countResult
	r.db.WithContext(ctx).
		Model(&Watcher{}).
		Select("task_id, COUNT(*) as count").
		Where("task_id IN ?", taskIDs).
		Group("task_id").
		Find(&watcherCounts)
	for _, c := range watcherCounts {
		if counts, ok := result[c.TaskID]; ok {
			counts.Watchers = int32(c.Count)
			result[c.TaskID] = counts
		}
	}

	return result, nil
}

// ==================== Comment ====================

func (r *repository) CreateComment(ctx context.Context, comment *Comment) error {
	comment.CreatedAt = time.Now()
	comment.UpdatedAt = time.Now()
	return r.db.WithContext(ctx).Create(comment).Error
}

func (r *repository) GetComment(ctx context.Context, id int64) (*Comment, error) {
	var comment Comment
	if err := r.db.WithContext(ctx).First(&comment, id).Error; err != nil {
		return nil, err
	}
	return &comment, nil
}

func (r *repository) UpdateComment(ctx context.Context, comment *Comment) error {
	now := time.Now()
	comment.EditedAt = &now
	comment.UpdatedAt = now
	return r.db.WithContext(ctx).Save(comment).Error
}

func (r *repository) DeleteComment(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&Comment{}, id).Error
}

func (r *repository) ListComments(ctx context.Context, taskID int64, limit, offset int) ([]*Comment, int64, error) {
	var comments []*Comment
	var total int64

	query := r.db.WithContext(ctx).Model(&Comment{}).Where("task_id = ? AND deleted_at IS NULL", taskID)

	query.Count(&total)

	if limit <= 0 {
		limit = 20
	}

	err := query.
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&comments).Error

	return comments, total, err
}

// ==================== Attachment ====================

func (r *repository) CreateAttachment(ctx context.Context, attachment *Attachment) error {
	attachment.CreatedAt = time.Now()
	return r.db.WithContext(ctx).Create(attachment).Error
}

func (r *repository) GetAttachment(ctx context.Context, id int64) (*Attachment, error) {
	var attachment Attachment
	if err := r.db.WithContext(ctx).First(&attachment, id).Error; err != nil {
		return nil, err
	}
	return &attachment, nil
}

func (r *repository) DeleteAttachment(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&Attachment{}, id).Error
}

func (r *repository) ListAttachments(ctx context.Context, taskID int64) ([]*Attachment, error) {
	var attachments []*Attachment
	err := r.db.WithContext(ctx).
		Where("task_id = ?", taskID).
		Order("created_at DESC").
		Find(&attachments).Error
	return attachments, err
}

// ==================== Activity Log ====================

func (r *repository) LogActivity(ctx context.Context, log *ActivityLog) error {
	log.CreatedAt = time.Now()
	return r.db.WithContext(ctx).Create(log).Error
}

func (r *repository) ListActivity(ctx context.Context, taskID int64, limit, offset int) ([]*ActivityLog, int64, error) {
	var logs []*ActivityLog
	var total int64

	query := r.db.WithContext(ctx).Model(&ActivityLog{}).Where("task_id = ?", taskID)
	query.Count(&total)

	if limit <= 0 {
		limit = 20
	}

	err := query.
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&logs).Error

	return logs, total, err
}

// ==================== Watcher ====================

func (r *repository) AddWatcher(ctx context.Context, watcher *Watcher) error {
	watcher.CreatedAt = time.Now()
	return r.db.WithContext(ctx).Create(watcher).Error
}

func (r *repository) RemoveWatcher(ctx context.Context, taskID, userID int64) error {
	return r.db.WithContext(ctx).
		Where("task_id = ? AND user_id = ?", taskID, userID).
		Delete(&Watcher{}).Error
}

func (r *repository) ListWatchers(ctx context.Context, taskID int64) ([]*Watcher, error) {
	var watchers []*Watcher
	err := r.db.WithContext(ctx).
		Where("task_id = ?", taskID).
		Find(&watchers).Error
	return watchers, err
}

func (r *repository) IsWatching(ctx context.Context, taskID, userID int64) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&Watcher{}).
		Where("task_id = ? AND user_id = ?", taskID, userID).
		Count(&count).Error
	return count > 0, err
}

// ==================== Stats ====================

func (r *repository) GetBoardStats(ctx context.Context, boardID int64) (*BoardStats, error) {
	stats := &BoardStats{
		TasksByStatus:   make(map[string]int32),
		TasksByPriority: make(map[string]int32),
	}

	var total int64
	r.db.WithContext(ctx).Model(&Task{}).Where("board_id = ? AND deleted_at IS NULL", boardID).Count(&total)
	stats.TotalTasks = int32(total)

	var completed int64
	r.db.WithContext(ctx).Model(&Task{}).Where("board_id = ? AND status = ? AND deleted_at IS NULL", boardID, TaskStatusDone).Count(&completed)
	stats.CompletedTasks = int32(completed)

	var overdue int64
	r.db.WithContext(ctx).Model(&Task{}).Where("board_id = ? AND due_date < ? AND status != ? AND deleted_at IS NULL", boardID, time.Now().Format("2006-01-02"), TaskStatusDone).Count(&overdue)
	stats.OverdueTasks = int32(overdue)

	var noAssignee int64
	r.db.WithContext(ctx).Model(&Task{}).Where("board_id = ? AND assignee_id IS NULL AND deleted_at IS NULL", boardID).Count(&noAssignee)
	stats.TasksWithoutAssignee = int32(noAssignee)

	type statusCount struct {
		Status string
		Count  int64
	}
	var statusCounts []statusCount
	r.db.WithContext(ctx).Model(&Task{}).
		Select("status, COUNT(*) as count").
		Where("board_id = ? AND deleted_at IS NULL", boardID).
		Group("status").
		Find(&statusCounts)
	for _, sc := range statusCounts {
		stats.TasksByStatus[sc.Status] = int32(sc.Count)
	}

	type priorityCount struct {
		Priority string
		Count    int64
	}
	var priorityCounts []priorityCount
	r.db.WithContext(ctx).Model(&Task{}).
		Select("priority, COUNT(*) as count").
		Where("board_id = ? AND deleted_at IS NULL", boardID).
		Group("priority").
		Find(&priorityCounts)
	for _, pc := range priorityCounts {
		stats.TasksByPriority[pc.Priority] = int32(pc.Count)
	}

	return stats, nil
}

func (r *repository) GetMemberStats(ctx context.Context, boardID int64) ([]*MemberStats, error) {
	type memberRow struct {
		AssigneeID int64
		Assigned   int64
		Completed  int64
		Overdue    int64
		InProgress int64
	}

	var rows []memberRow
	r.db.WithContext(ctx).Raw(`
		SELECT 
			assignee_id,
			COUNT(*) as assigned,
			SUM(CASE WHEN status = 'done' THEN 1 ELSE 0 END) as completed,
			SUM(CASE WHEN due_date < CURRENT_DATE AND status != 'done' THEN 1 ELSE 0 END) as overdue,
			SUM(CASE WHEN status = 'in_progress' THEN 1 ELSE 0 END) as in_progress
		FROM tasks
		WHERE board_id = ? AND deleted_at IS NULL AND assignee_id IS NOT NULL
		GROUP BY assignee_id
	`, boardID).Scan(&rows)

	var stats []*MemberStats
	for _, row := range rows {
		stats = append(stats, &MemberStats{
			User:            &UserPreview{ID: row.AssigneeID},
			AssignedTasks:   int32(row.Assigned),
			CompletedTasks:  int32(row.Completed),
			OverdueTasks:    int32(row.Overdue),
			InProgressTasks: int32(row.InProgress),
		})
	}
	return stats, nil
}

func (r *repository) GetDailyStats(ctx context.Context, boardID int64, days int) ([]*DailyStats, error) {
	var stats []*DailyStats

	r.db.WithContext(ctx).Raw(`
		SELECT 
			TO_CHAR(created_at, 'YYYY-MM-DD') as date,
			COUNT(*) as created,
			0 as completed,
			0 as moved
		FROM tasks
		WHERE board_id = ? AND created_at >= NOW() - INTERVAL '1 day' * ? AND deleted_at IS NULL
		GROUP BY TO_CHAR(created_at, 'YYYY-MM-DD')
		ORDER BY date DESC
	`, boardID, days).Scan(&stats)

	return stats, nil
}

// ==================== My Tasks ====================

func (r *repository) GetMyTasks(ctx context.Context, userID int64, filter MyTasksFilter) ([]*Task, int64, error) {
	var tasks []*Task
	var total int64

	query := r.db.WithContext(ctx).Model(&Task{}).Where("deleted_at IS NULL")

	if filter.OnlyAssigned {
		query = query.Where("assignee_id = ?", userID)
	} else if filter.OnlyCreated {
		query = query.Where("created_by = ?", userID)
	} else {
		query = query.Where("assignee_id = ? OR created_by = ?", userID, userID)
	}

	if !filter.IncludeCompleted {
		query = query.Where("status != ?", TaskStatusDone)
	}

	query.Count(&total)

	if filter.Limit <= 0 {
		filter.Limit = 20
	}

	err := query.
		Order("due_date ASC NULLS LAST, priority DESC, created_at DESC").
		Limit(filter.Limit).
		Offset(filter.Offset).
		Find(&tasks).Error

	return tasks, total, err
}

func (r *repository) GetOverdueTasks(ctx context.Context, boardID int64, assigneeID *int64, limit, offset int) ([]*Task, int64, error) {
	var tasks []*Task
	var total int64

	query := r.db.WithContext(ctx).Model(&Task{}).
		Where("due_date < ? AND status != ? AND deleted_at IS NULL", time.Now().Format("2006-01-02"), TaskStatusDone)

	if boardID > 0 {
		query = query.Where("board_id = ?", boardID)
	}
	if assigneeID != nil {
		query = query.Where("assignee_id = ?", *assigneeID)
	}

	query.Count(&total)

	if limit <= 0 {
		limit = 20
	}

	err := query.
		Order("due_date ASC").
		Limit(limit).
		Offset(offset).
		Find(&tasks).Error

	return tasks, total, err
}

func (r *repository) GetUpcomingDeadlines(ctx context.Context, boardID int64, userID *int64, daysAhead, limit, offset int) ([]*Task, int64, error) {
	var tasks []*Task
	var total int64

	if daysAhead <= 0 {
		daysAhead = 7
	}

	futureDate := time.Now().AddDate(0, 0, daysAhead).Format("2006-01-02")

	query := r.db.WithContext(ctx).Model(&Task{}).
		Where("due_date BETWEEN ? AND ? AND status != ? AND deleted_at IS NULL",
			time.Now().Format("2006-01-02"), futureDate, TaskStatusDone)

	if boardID > 0 {
		query = query.Where("board_id = ?", boardID)
	}
	if userID != nil {
		query = query.Where("assignee_id = ?", *userID)
	}

	query.Count(&total)

	if limit <= 0 {
		limit = 20
	}

	err := query.
		Order("due_date ASC").
		Limit(limit).
		Offset(offset).
		Find(&tasks).Error

	return tasks, total, err
}
func (r *repository) ListTasksDueOn(ctx context.Context, dueDate time.Time) ([]*Task, error) {
	var tasks []*Task
	err := r.db.WithContext(ctx).
		Model(&Task{}).
		Where("deleted_at IS NULL").
		Where("status != ?", TaskStatusDone).
		Where("assignee_id IS NOT NULL").
		Where("due_date = ?", dueDate.Format("2006-01-02")).
		Find(&tasks).Error
	return tasks, err
}

func (r *repository) ListOverdueOpenTasks(ctx context.Context, beforeDate time.Time) ([]*Task, error) {
	var tasks []*Task
	err := r.db.WithContext(ctx).
		Model(&Task{}).
		Where("deleted_at IS NULL").
		Where("status != ?", TaskStatusDone).
		Where("assignee_id IS NOT NULL").
		Where("due_date < ?", beforeDate.Format("2006-01-02")).
		Find(&tasks).Error
	return tasks, err
}

func (r *repository) TryInsertDeadlineRun(ctx context.Context, run *DeadlineNotificationRun) (bool, error) {
	if run == nil || run.DedupKey == "" {
		return false, fmt.Errorf("run/dedup_key required")
	}

	err := r.db.WithContext(ctx).Create(run).Error
	if err == nil {
		return true, nil
	}

	// дедуп: если уже есть — просто "не вставили"
	if strings.Contains(err.Error(), "duplicate key") ||
		strings.Contains(err.Error(), "unique constraint") ||
		strings.Contains(err.Error(), "UNIQUE constraint") {
		return false, nil
	}

	return false, err
}
