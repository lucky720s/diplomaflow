package notification

import (
	"context"

	"gorm.io/gorm"
)

type Repository interface {
	Create(ctx context.Context, n *Notification) error
	List(ctx context.Context, userID int64, onlyUnread bool, limit, offset int) ([]*Notification, int64, error)
	MarkAsRead(ctx context.Context, id int64, userID int64) error
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

func (r *repository) Create(ctx context.Context, n *Notification) error {
	return r.db.WithContext(ctx).Create(n).Error
}

func (r *repository) List(ctx context.Context, userID int64, onlyUnread bool, limit, offset int) ([]*Notification, int64, error) {
	var notifications []*Notification
	var total int64

	query := r.db.WithContext(ctx).Model(&Notification{}).Where("user_id = ?", userID)

	if onlyUnread {
		query = query.Where("is_read = ?", false)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := query.Order("created_at DESC").Limit(limit).Offset(offset).Find(&notifications).Error
	return notifications, total, err
}

func (r *repository) MarkAsRead(ctx context.Context, id int64, userID int64) error {
	return r.db.WithContext(ctx).
		Model(&Notification{}).
		Where("id = ? AND user_id = ?", id, userID).
		Update("is_read", true).Error
}
