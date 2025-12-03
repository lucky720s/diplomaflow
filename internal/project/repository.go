package project

import (
	"context"
	"errors"

	"gorm.io/gorm"
)

type Repository interface {
	Create(ctx context.Context, project *Project) error
	GetByID(ctx context.Context, id uint64) (*Project, error)
	Update(ctx context.Context, project *Project) error
	Delete(ctx context.Context, id uint64) error
	ListByStudent(ctx context.Context, studentID int64) ([]*Project, error)
	AddHistory(ctx context.Context, history *StateHistory) error
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

func (r *repository) Create(ctx context.Context, project *Project) error {
	return r.db.WithContext(ctx).Create(project).Error
}

func (r *repository) GetByID(ctx context.Context, id uint64) (*Project, error) {
	var project Project
	if err := r.db.WithContext(ctx).Preload("History").First(&project, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("project not found")
		}
		return nil, err
	}
	return &project, nil
}

func (r *repository) Update(ctx context.Context, project *Project) error {
	return r.db.WithContext(ctx).Save(project).Error
}

func (r *repository) Delete(ctx context.Context, id uint64) error {
	return r.db.WithContext(ctx).Delete(&Project{}, id).Error
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
