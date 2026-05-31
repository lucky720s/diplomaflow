package project

import (
	"context"
	"encoding/json"
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type Repository interface {
	Transaction(ctx context.Context, fn func(tx *gorm.DB) error) error
	CreateWithOutbox(ctx context.Context, project *Project, eventType string, topic string, payloadBase map[string]interface{}) error
	GetByID(ctx context.Context, id int64) (*Project, error)
	GetRuntimeByID(ctx context.Context, id int64) (*Project, error)
	GetPendingEvents(ctx context.Context, limit int) ([]OutboxEvent, error)
	MarkEventProcessed(ctx context.Context, id int64) error

	// Proto compatibility: GetStudentProjects(student_id) is used as "projects visible for user"
	ListByStudent(ctx context.Context, studentID int64) ([]*Project, error)

	GetProjectsWithExpiredDeadlines(ctx context.Context) ([]*Project, error)

	// Admin/internal
	ListProjectsRuntime(ctx context.Context, f ProjectFilter) ([]*Project, int64, error)
	ListStateHistory(ctx context.Context, projectID int64, limit int, order string) ([]StateHistory, error)
}

type ProjectFilter struct {
	DepartmentID int64
	UniversityID int64
	WorkflowID   int64
	StateID      int64
	TeamID       int64
	StudentID    int64
	Status       string
	Search       string
	Page         int32
	PageSize     int32
}

type repository struct{ db *gorm.DB }

func NewRepository(db *gorm.DB) Repository { return &repository{db: db} }

func (r *repository) Transaction(ctx context.Context, fn func(tx *gorm.DB) error) error {
	return r.db.WithContext(ctx).Transaction(fn)
}

// --- minimal rows for task tables (создаём из project_service транзакционно) ---

type taskBoardRow struct {
	ID          int64  `gorm:"primaryKey;column:id"`
	TeamID      int64  `gorm:"column:team_id"`
	ProjectID   int64  `gorm:"column:project_id"`
	Name        string `gorm:"column:name"`
	Description string `gorm:"column:description"`
	CreatedBy   int64  `gorm:"column:created_by"`
}

func (taskBoardRow) TableName() string { return "task_boards" }

type taskColumnRow struct {
	ID           int64  `gorm:"primaryKey;column:id"`
	BoardID      int64  `gorm:"column:board_id"`
	Name         string `gorm:"column:name"`
	Slug         string `gorm:"column:slug"`
	Color        string `gorm:"column:color"`
	OrderIndex   int32  `gorm:"column:order_index"`
	IsDefault    bool   `gorm:"column:is_default"`
	IsDoneColumn bool   `gorm:"column:is_done_column"`
}

func (taskColumnRow) TableName() string { return "task_columns" }

type defaultColumn struct {
	Name         string
	Slug         string
	Color        string
	OrderIndex   int32
	IsDefault    bool
	IsDoneColumn bool
}

var defaultColumns = []defaultColumn{
	{Name: "К выполнению", Slug: "todo", Color: "#6B7280", OrderIndex: 0, IsDefault: true, IsDoneColumn: false},
	{Name: "В работе", Slug: "in_progress", Color: "#3B82F6", OrderIndex: 1, IsDefault: false, IsDoneColumn: false},
	{Name: "На проверке", Slug: "review", Color: "#F59E0B", OrderIndex: 2, IsDefault: false, IsDoneColumn: false},
	{Name: "Готово", Slug: "done", Color: "#10B981", OrderIndex: 3, IsDefault: false, IsDoneColumn: true},
}

func createBoardAndDefaultColumns(tx *gorm.DB, project *Project) error {
	// создаём board с именем проекта
	b := &taskBoardRow{
		TeamID:      project.TeamID,
		ProjectID:   project.ID,
		Name:        project.Title,
		Description: project.Description,
		CreatedBy:   project.StudentID,
	}

	// settings/created_at/updated_at берём дефолтами БД (NOT NULL defaults) [[16]]
	if err := tx.Omit("Settings").Create(b).Error; err != nil {
		return err
	}

	// дефолтные колонки
	for _, c := range defaultColumns {
		row := &taskColumnRow{
			BoardID:      b.ID,
			Name:         c.Name,
			Slug:         c.Slug,
			Color:        c.Color,
			OrderIndex:   c.OrderIndex,
			IsDefault:    c.IsDefault,
			IsDoneColumn: c.IsDoneColumn,
		}
		if err := tx.Create(row).Error; err != nil {
			return err
		}
	}

	return nil
}

func (r *repository) CreateWithOutbox(ctx context.Context, project *Project, eventType string, topic string, payloadBase map[string]interface{}) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 1) create project
		if err := tx.Create(project).Error; err != nil {
			return err
		}

		// 2) create task board + default columns (в той же транзакции)
		if err := createBoardAndDefaultColumns(tx, project); err != nil {
			return err
		}

		// 3) outbox
		payloadBase["project_id"] = project.ID
		payloadBytes, err := json.Marshal(payloadBase)
		if err != nil {
			return err
		}

		event := &OutboxEvent{
			Topic:     topic,
			EventType: eventType,
			Payload:   datatypes.JSON(payloadBytes),
			Status:    "pending",
			CreatedAt: time.Now().UTC(),
		}
		return tx.Create(event).Error
	})
}

func (r *repository) GetByID(ctx context.Context, id int64) (*Project, error) {
	var p Project
	if err := r.db.WithContext(ctx).
		Preload("History", func(db *gorm.DB) *gorm.DB { return db.Order("created_at DESC") }).
		First(&p, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *repository) GetRuntimeByID(ctx context.Context, id int64) (*Project, error) {
	var p Project
	if err := r.db.WithContext(ctx).First(&p, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *repository) GetPendingEvents(ctx context.Context, limit int) ([]OutboxEvent, error) {
	var events []OutboxEvent
	err := r.db.WithContext(ctx).
		Where("status = ?", "pending").
		Order("id asc").
		Limit(limit).
		Find(&events).Error
	return events, err
}

func (r *repository) MarkEventProcessed(ctx context.Context, id int64) error {
	now := time.Now().UTC()
	return r.db.WithContext(ctx).Model(&OutboxEvent{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":       "processed",
			"processed_at": &now,
		}).Error
}

// ListByStudent: returns projects visible for user
// - owned by user (projects.student_id = user_id)
// - OR user is member of the team assigned to the project (team_members.user_id = user_id)
func (r *repository) ListByStudent(ctx context.Context, studentID int64) ([]*Project, error) {
	var projects []*Project

	q := r.db.WithContext(ctx).Model(&Project{})

	// If caller passed 0 - fallback to all (internal/admin usage)
	if studentID == 0 {
		if err := q.Order("created_at DESC").Find(&projects).Error; err != nil {
			return nil, err
		}
		return projects, nil
	}

	q = q.
		Select("DISTINCT projects.*").
		Joins("LEFT JOIN team_members tm ON tm.team_id = projects.team_id").
		Where("projects.student_id = ? OR tm.user_id = ?", studentID, studentID).
		Order("projects.created_at DESC")

	if err := q.Find(&projects).Error; err != nil {
		return nil, err
	}

	return projects, nil
}

func (r *repository) GetProjectsWithExpiredDeadlines(ctx context.Context) ([]*Project, error) {
	var projects []*Project
	err := r.db.WithContext(ctx).
		Where("status NOT IN ? AND deadline_at < ? AND deadline_processed = ?",
			[]string{"completed", "cancelled", "archived"},
			time.Now().UTC(), false).
		Find(&projects).Error
	return projects, err
}

func (r *repository) ListProjectsRuntime(ctx context.Context, f ProjectFilter) ([]*Project, int64, error) {
	q := r.db.WithContext(ctx).Model(&Project{})

	if f.DepartmentID > 0 {
		q = q.Where("department_id = ?", f.DepartmentID)
	}
	if f.UniversityID > 0 {
		q = q.Where("university_id = ?", f.UniversityID)
	}
	if f.WorkflowID > 0 {
		q = q.Where("workflow_id = ?", f.WorkflowID)
	}
	if f.StateID > 0 {
		q = q.Where("current_state_id = ?", f.StateID)
	}
	if f.TeamID > 0 {
		q = q.Where("team_id = ?", f.TeamID)
	}
	if f.StudentID > 0 {
		q = q.Where("student_id = ?", f.StudentID)
	}
	if f.Status != "" {
		q = q.Where("status = ?", f.Status)
	}
	if f.Search != "" {
		q = q.Where("title ILIKE ?", "%"+f.Search+"%")
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	page := f.Page
	if page < 1 {
		page = 1
	}
	pageSize := f.PageSize
	if pageSize < 1 || pageSize > 200 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	var projects []*Project
	err := q.Order("created_at DESC").Offset(int(offset)).Limit(int(pageSize)).Find(&projects).Error
	return projects, total, err
}

func (r *repository) ListStateHistory(ctx context.Context, projectID int64, limit int, order string) ([]StateHistory, error) {
	if order != "asc" {
		order = "desc"
	}
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	var items []StateHistory
	err := r.db.WithContext(ctx).
		Where("project_id = ?", projectID).
		Order("created_at " + order).
		Limit(limit).
		Find(&items).Error
	return items, err
}
