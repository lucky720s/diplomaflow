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
}

type repository struct{ db *gorm.DB }

func NewRepository(db *gorm.DB) Repository { return &repository{db: db} }

func (r *repository) Transaction(ctx context.Context, fn func(tx *gorm.DB) error) error {
	return r.db.WithContext(ctx).Transaction(fn)
}

func (r *repository) CreateWithOutbox(ctx context.Context, project *Project, eventType string, topic string, payloadBase map[string]interface{}) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(project).Error; err != nil {
			return err
		}

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
		Where("status = ? AND deadline_at < ? AND deadline_processed = ?", "active", time.Now().UTC(), false).
		Find(&projects).Error
	return projects, err
}
