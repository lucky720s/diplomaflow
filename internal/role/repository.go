package role

import (
	"context"
	"errors"

	"gorm.io/gorm"
)

type Repository interface {
	Create(ctx context.Context, role *Role) error
	GetByID(ctx context.Context, id int64) (*Role, error)
	Delete(ctx context.Context, id int64) error
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	_ = db.AutoMigrate(&Role{})
	return &repository{db: db}
}

func (r *repository) Create(ctx context.Context, role *Role) error {
	return r.db.WithContext(ctx).Create(role).Error
}

func (r *repository) GetByID(ctx context.Context, id int64) (*Role, error) {
	var role Role
	if err := r.db.WithContext(ctx).First(&role, id).Error; err != nil {
		return nil, err
	}
	return &role, nil
}

func (r *repository) Delete(ctx context.Context, id int64) error {
	res := r.db.WithContext(ctx).Delete(&Role{}, id)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return errors.New("role not found")
	}
	return nil
}
