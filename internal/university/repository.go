package university

import (
	"context"

	"gorm.io/gorm"
)

type Repository interface {
	CreateUniversity(ctx context.Context, uni *University) error
	GetUniversity(ctx context.Context, id int64) (*University, error)
	ListUniversities(ctx context.Context) ([]*University, error)
	CreateDepartment(ctx context.Context, dep *Department) error
	ListDepartments(ctx context.Context, uniID int64) ([]*Department, error)
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	_ = db.AutoMigrate(&University{}, &Department{})
	return &repository{db: db}
}

func (r *repository) CreateUniversity(ctx context.Context, uni *University) error {
	return r.db.WithContext(ctx).Create(uni).Error
}

func (r *repository) GetUniversity(ctx context.Context, id int64) (*University, error) {
	var uni University
	if err := r.db.WithContext(ctx).First(&uni, id).Error; err != nil {
		return nil, err
	}
	return &uni, nil
}

func (r *repository) ListUniversities(ctx context.Context) ([]*University, error) {
	var unis []*University
	if err := r.db.WithContext(ctx).Find(&unis).Error; err != nil {
		return nil, err
	}
	return unis, nil
}

func (r *repository) CreateDepartment(ctx context.Context, dep *Department) error {
	return r.db.WithContext(ctx).Create(dep).Error
}

func (r *repository) ListDepartments(ctx context.Context, uniID int64) ([]*Department, error) {
	var deps []*Department
	if err := r.db.WithContext(ctx).Where("university_id = ?", uniID).Find(&deps).Error; err != nil {
		return nil, err
	}
	return deps, nil
}
