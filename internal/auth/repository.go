package auth

import (
	"context"
	"time"

	"gorm.io/gorm"
)

type Repository interface {
	Create(ctx context.Context, user *User) error
	GetByEmail(ctx context.Context, email string) (*User, error)
	GetByID(ctx context.Context, id int64) (*User, error)
	ListUsers(ctx context.Context, filter UserFilter) ([]*User, int64, error)

	CreateRefreshToken(ctx context.Context, token *RefreshToken) error
	GetRefreshTokenByID(ctx context.Context, id uint64) (*RefreshToken, error)
	GetRefreshToken(ctx context.Context, token string) (*RefreshToken, error)
	RevokeRefreshToken(ctx context.Context, id uint64) error
	RevokeAllUserTokens(ctx context.Context, userID int64) error
	ListActiveSessions(ctx context.Context, userID int64) ([]*RefreshToken, error)

	Update(ctx context.Context, user *User) error
	GetByIDs(ctx context.Context, ids []int64) ([]*User, error)

	// NEW: Delete user (soft delete)
	Delete(ctx context.Context, id int64) error
}

type UserFilter struct {
	UniversityID  int64
	DepartmentID  int64
	Role          string
	ExcludeUserID int64
	Limit         int
	Offset        int
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
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
	if filter.DepartmentID != 0 {
		query = query.Where("department_id = ?", filter.DepartmentID)
	}
	if filter.Role != "" {
		query = query.Where("role = ?", filter.Role)
	}
	if filter.ExcludeUserID != 0 {
		query = query.Where("id != ?", filter.ExcludeUserID)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if filter.Limit <= 0 {
		filter.Limit = 10
	}
	if filter.Offset < 0 {
		filter.Offset = 0
	}

	if err := query.
		Order("id ASC").
		Limit(filter.Limit).
		Offset(filter.Offset).
		Find(&users).Error; err != nil {
		return nil, 0, err
	}

	return users, total, nil
}

func (r *repository) CreateRefreshToken(ctx context.Context, token *RefreshToken) error {
	return r.db.WithContext(ctx).Create(token).Error
}

func (r *repository) GetRefreshToken(ctx context.Context, token string) (*RefreshToken, error) {
	var rt RefreshToken
	if err := r.db.WithContext(ctx).Where("token = ?", token).First(&rt).Error; err != nil {
		return nil, err
	}
	return &rt, nil
}

func (r *repository) RevokeRefreshToken(ctx context.Context, id uint64) error {
	return r.db.WithContext(ctx).
		Model(&RefreshToken{}).
		Where("id = ?", id).
		Update("revoked", true).Error
}

func (r *repository) RevokeAllUserTokens(ctx context.Context, userID int64) error {
	return r.db.WithContext(ctx).
		Model(&RefreshToken{}).
		Where("user_id = ?", userID).
		Update("revoked", true).Error
}

func (r *repository) GetRefreshTokenByID(ctx context.Context, id uint64) (*RefreshToken, error) {
	var rt RefreshToken
	if err := r.db.WithContext(ctx).First(&rt, id).Error; err != nil {
		return nil, err
	}
	return &rt, nil
}

func (r *repository) ListActiveSessions(ctx context.Context, userID int64) ([]*RefreshToken, error) {
	var tokens []*RefreshToken
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND revoked = ? AND expires_at > ?", userID, false, time.Now()).
		Order("created_at desc").
		Find(&tokens).Error
	return tokens, err
}

func (r *repository) Update(ctx context.Context, user *User) error {
	return r.db.WithContext(ctx).Save(user).Error
}

func (r *repository) GetByIDs(ctx context.Context, ids []int64) ([]*User, error) {
	var users []*User
	err := r.db.WithContext(ctx).
		Where("id IN ?", ids).
		Find(&users).Error
	return users, err
}

func (r *repository) Delete(ctx context.Context, id int64) error {
	result := r.db.WithContext(ctx).Delete(&User{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}
