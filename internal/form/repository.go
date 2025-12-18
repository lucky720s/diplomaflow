package form

import (
	"context"

	"gorm.io/gorm"
)

type Repository interface {
	Create(ctx context.Context, submission *FormSubmission) error
	GetByID(ctx context.Context, id string) (*FormSubmission, error)
	ListByProject(ctx context.Context, projectID int64) ([]*FormSubmission, error)
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	_ = db.AutoMigrate(&FormSubmission{})
	return &repository{db: db}
}

func (r *repository) Create(ctx context.Context, submission *FormSubmission) error {
	return r.db.WithContext(ctx).Create(submission).Error
}

func (r *repository) GetByID(ctx context.Context, id string) (*FormSubmission, error) {
	var sub FormSubmission
	if err := r.db.WithContext(ctx).First(&sub, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &sub, nil
}

func (r *repository) ListByProject(ctx context.Context, projectID int64) ([]*FormSubmission, error) {
	var subs []*FormSubmission
	if err := r.db.WithContext(ctx).Where("project_id = ?", projectID).Order("created_at desc").Find(&subs).Error; err != nil {
		return nil, err
	}
	return subs, nil
}
