package task

import (
	"context"
	"time"

	"go.uber.org/zap"
)

type Service struct {
	repo   *repository
	logger *zap.Logger
}

func NewService(repo *repository, logger *zap.Logger) *Service {
	return &Service{
		repo:   repo,
		logger: logger,
	}
}

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
	defaultColumns := []string{"To Do", "In Progress", "Done"}
	for i, name := range defaultColumns {
		col := &Column{
			BoardID:   board.ID,
			Name:      name,
			Position:  i,
			CreatedAt: time.Now(),
		}
		if err := s.repo.CreateColumn(ctx, col); err != nil {
			s.logger.Warn("failed to create default column", zap.Error(err))
		}
	}

	return board, nil
}

func (s *Service) GetBoard(ctx context.Context, boardID int64, includeColumns, includeStats bool) (*Board, error) {
	return s.repo.GetBoard(ctx, boardID, includeColumns, includeStats)
}

func (s *Service) GetBoardByProject(ctx context.Context, projectID int64, includeColumns, includeStats bool) (*Board, error) {
	return s.repo.GetBoardByProject(ctx, projectID, includeColumns, includeStats)
}

// ИЗМЕНЕНО: добавлены параметры universityID и departmentID
func (s *Service) ListMyBoards(ctx context.Context, userID int64, role string, universityID, departmentID int64, includeColumns, includeStats bool) ([]*Board, error) {
	return s.repo.ListMyBoards(ctx, userID, role, universityID, departmentID, includeColumns, includeStats)
}

func (s *Service) GetBoardStats(ctx context.Context, boardID int64) (*BoardStats, error) {
	return s.repo.GetBoardStats(ctx, boardID)
}

func (s *Service) CreateColumn(ctx context.Context, boardID int64, name string, position int) (*Column, error) {
	col := &Column{
		BoardID:   boardID,
		Name:      name,
		Position:  position,
		CreatedAt: time.Now(),
	}
	if err := s.repo.CreateColumn(ctx, col); err != nil {
		return nil, err
	}
	return col, nil
}

func (s *Service) UpdateColumn(ctx context.Context, columnID int64, name string) (*Column, error) {
	return s.repo.UpdateColumn(ctx, columnID, name)
}

func (s *Service) DeleteColumn(ctx context.Context, columnID int64) error {
	return s.repo.DeleteColumn(ctx, columnID)
}

func (s *Service) ReorderColumns(ctx context.Context, boardID int64, order []int64) error {
	return s.repo.ReorderColumns(ctx, boardID, order)
}

func (s *Service) CreateTask(ctx context.Context, input *CreateTaskInput) (*Task, error) {
	// Если колонка не указана, используем первую колонку доски
	columnID := input.ColumnID
	if columnID == 0 {
		columns, err := s.repo.GetColumnsByBoard(ctx, input.BoardID)
		if err != nil || len(columns) == 0 {
			return nil, err
		}
		columnID = columns[0].ID
	}

	maxPos, _ := s.repo.GetMaxTaskPosition(ctx, columnID)

	task := &Task{
		BoardID:          input.BoardID,
		ColumnID:         columnID,
		Title:            input.Title,
		Description:      input.Description,
		Priority:         input.Priority,
		AssigneeID:       input.AssigneeID,
		DueDate:          input.DueDate,
		EstimatedMinutes: input.EstimatedMinutes,
		Labels:           input.Labels,
		Position:         maxPos + 1,
		CreatedBy:        input.CreatedBy,
		WorkflowStepID:   input.WorkflowStepID,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	if err := s.repo.CreateTask(ctx, task); err != nil {
		return nil, err
	}
	return task, nil
}

func (s *Service) GetTask(ctx context.Context, taskID int64) (*Task, error) {
	return s.repo.GetTask(ctx, taskID)
}

func (s *Service) UpdateTask(ctx context.Context, input *UpdateTaskInput) (*Task, error) {
	task, err := s.repo.GetTask(ctx, input.TaskID)
	if err != nil {
		return nil, err
	}

	if input.Title != "" {
		task.Title = input.Title
	}
	if input.Description != "" {
		task.Description = input.Description
	}
	if input.Priority != nil {
		task.Priority = *input.Priority
	}
	if input.AssigneeID != nil {
		task.AssigneeID = input.AssigneeID
	}
	if input.DueDate != nil {
		task.DueDate = input.DueDate
	}
	if input.EstimatedMinutes != nil {
		task.EstimatedMinutes = input.EstimatedMinutes
	}
	if input.Labels != nil {
		task.Labels = input.Labels
	}

	task.UpdatedAt = time.Now()

	if err := s.repo.UpdateTask(ctx, task); err != nil {
		return nil, err
	}
	return task, nil
}

func (s *Service) DeleteTask(ctx context.Context, taskID int64) error {
	return s.repo.DeleteTask(ctx, taskID)
}

func (s *Service) MoveTask(ctx context.Context, input *MoveTaskInput) (*Task, error) {
	return s.repo.MoveTask(ctx, input.TaskID, input.ColumnID, input.Position)
}

func (s *Service) ReorderTasks(ctx context.Context, columnID int64, order []int64) error {
	return s.repo.ReorderTasks(ctx, columnID, order)
}

func (s *Service) GetTasksByAssignee(ctx context.Context, userID int64) ([]*Task, error) {
	return s.repo.GetTasksByAssignee(ctx, userID)
}

func (s *Service) CreateComment(ctx context.Context, taskID, authorID int64, content string) (*Comment, error) {
	comment := &Comment{
		TaskID:    taskID,
		AuthorID:  authorID,
		Content:   content,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := s.repo.CreateComment(ctx, comment); err != nil {
		return nil, err
	}
	return comment, nil
}

func (s *Service) GetComments(ctx context.Context, taskID int64) ([]*Comment, error) {
	return s.repo.GetComments(ctx, taskID)
}

func (s *Service) GetComment(ctx context.Context, commentID int64) (*Comment, error) {
	return s.repo.GetComment(ctx, commentID)
}

func (s *Service) DeleteComment(ctx context.Context, commentID int64) error {
	return s.repo.DeleteComment(ctx, commentID)
}
