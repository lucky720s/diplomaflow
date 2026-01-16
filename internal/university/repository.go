package university

import (
	"context"

	"gorm.io/gorm"
)

type Repository interface {
	CreateUniversity(ctx context.Context, uni *University) error
	GetUniversity(ctx context.Context, id int64) (*University, error)
	ListUniversities(ctx context.Context) ([]*University, error)
	UpdateUniversity(ctx context.Context, uni *University) error
	DeleteUniversity(ctx context.Context, id int64) error

	CreateDepartment(ctx context.Context, dep *Department) error
	GetDepartment(ctx context.Context, id int64) (*Department, error)
	ListDepartments(ctx context.Context, uniID int64) ([]*Department, error)
	UpdateDepartment(ctx context.Context, dep *Department) error
	DeleteDepartment(ctx context.Context, id int64) error
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
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

func (r *repository) UpdateUniversity(ctx context.Context, uni *University) error {
	return r.db.WithContext(ctx).Save(uni).Error
}

func (r *repository) DeleteUniversity(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&University{}, id).Error
}

func (r *repository) CreateDepartment(ctx context.Context, dep *Department) error {
	return r.db.WithContext(ctx).Create(dep).Error
}

func (r *repository) GetDepartment(ctx context.Context, id int64) (*Department, error) {
	var dep Department
	if err := r.db.WithContext(ctx).First(&dep, id).Error; err != nil {
		return nil, err
	}
	return &dep, nil
}

func (r *repository) ListDepartments(ctx context.Context, uniID int64) ([]*Department, error) {
	var deps []*Department
	if err := r.db.WithContext(ctx).Where("university_id = ?", uniID).Find(&deps).Error; err != nil {
		return nil, err
	}
	return deps, nil
}

func (r *repository) UpdateDepartment(ctx context.Context, dep *Department) error {
	return r.db.WithContext(ctx).Save(dep).Error
}

func (r *repository) DeleteDepartment(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&Department{}, id).Error
}
