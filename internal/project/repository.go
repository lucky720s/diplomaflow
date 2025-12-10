package project

import (
	"context"
	"encoding/json"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type Repository interface {
	CreateWithOutbox(ctx context.Context, project *Project, eventType string, topic string, payloadBase map[string]interface{}) error
	GetByID(ctx context.Context, id uint64) (*Project, error)
	Update(ctx context.Context, project *Project) error
	GetPendingEvents(ctx context.Context, limit int) ([]OutboxEvent, error)
	DeleteEvent(ctx context.Context, id uint) error
	ListByStudent(ctx context.Context, studentID int64) ([]*Project, error)
	AddHistory(ctx context.Context, history *StateHistory) error
	MarkEventProcessed(ctx context.Context, id uint) error
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	_ = db.AutoMigrate(&Project{}, &StateHistory{}, &OutboxEvent{})
	return &repository{db: db}
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
		}

		if err := tx.Create(event).Error; err != nil {
			return err
		}
		return nil
	})
}

func (r *repository) GetByID(ctx context.Context, id uint64) (*Project, error) {
	var project Project
	if err := r.db.WithContext(ctx).Preload("History").First(&project, id).Error; err != nil {
		return nil, err
	}
	return &project, nil
}

func (r *repository) Update(ctx context.Context, project *Project) error {
	return r.db.WithContext(ctx).Save(project).Error
}

func (r *repository) GetPendingEvents(ctx context.Context, limit int) ([]OutboxEvent, error) {
	var events []OutboxEvent
	err := r.db.WithContext(ctx).Where("status = ?", "pending").Order("id asc").Limit(limit).Find(&events).Error
	return events, err
}

func (r *repository) DeleteEvent(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&OutboxEvent{}, id).Error
}

func (r *repository) ListByStudent(ctx context.Context, studentID int64) ([]*Project, error) {
	var projects []*Project
	if err := r.db.WithContext(ctx).Where("student_id = ?", studentID).Find(&projects).Error; err != nil {
		return nil, err
	}
	return projects, nil
}

func (r *repository) AddHistory(ctx context.Context, history *StateHistory) error {
	return r.db.WithContext(ctx).Create(history).Error
}
func (r *repository) MarkEventProcessed(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Model(&OutboxEvent{}).
		Where("id = ?", id).
		Update("status", "processed").Error
}
