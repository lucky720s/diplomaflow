package auth

import (
	"context"

	"gorm.io/gorm"
)

type Repository interface {
	Create(ctx context.Context, user *User) error
	GetByEmail(ctx context.Context, email string) (*User, error)
	GetByID(ctx context.Context, id int64) (*User, error)
	ListUsers(ctx context.Context, filter UserFilter) ([]*User, int64, error)
}

type UserFilter struct {
	UniversityID int64
	Role         string
	Limit        int
	Offset       int
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	_ = db.AutoMigrate(&User{})
	return &repository{db: db}
}

func (r *repository) Create(ctx context.Context, user *User) error {
	return r.db.WithContext(ctx).Create(user).Error
}

func (r *repository) GetByEmail(ctx context.Context, email string) (*User, error) {
	var user User
	if err := r.db.WithContext(ctx).Where("email = ?", email).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *repository) GetByID(ctx context.Context, id int64) (*User, error) {
	var user User
	if err := r.db.WithContext(ctx).First(&user, id).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *repository) ListUsers(ctx context.Context, filter UserFilter) ([]*User, int64, error) {
	var users []*User
	var total int64

	query := r.db.WithContext(ctx).Model(&User{})

	if filter.UniversityID != 0 {
		query = query.Where("university_id = ?", filter.UniversityID)
	}
	if filter.Role != "" {
		query = query.Where("role = ?", filter.Role)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := query.Limit(filter.Limit).Offset(filter.Offset).Find(&users).Error; err != nil {
		return nil, 0, err
	}

	return users, total, nil
}
