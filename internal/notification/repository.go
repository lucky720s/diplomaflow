package notification

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
)

var ErrNotificationNotFound = errors.New("notification not found")

type Repository interface {
	Create(ctx context.Context, n *Notification) error
	List(ctx context.Context, userID int64, onlyUnread bool, limit, offset int) ([]*Notification, int64, error)
	MarkAsRead(ctx context.Context, id int64, userID int64) error
	MarkAllAsRead(ctx context.Context, userID int64) (int64, error)
	Delete(ctx context.Context, id int64, userID int64) error
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

	// IMPORTANT: Notification has gorm.DeletedAt, so soft-deleted rows are filtered automatically
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
	res := r.db.WithContext(ctx).
		Model(&Notification{}).
		Where("id = ? AND user_id = ?", id, userID).
		Updates(map[string]interface{}{
			"is_read":    true,
			"updated_at": time.Now().UTC(),
		})

	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrNotificationNotFound
	}
	return nil
}

func (r *repository) Delete(ctx context.Context, id int64, userID int64) error {
	res := r.db.WithContext(ctx).
		Where("id = ? AND user_id = ?", id, userID).
		Delete(&Notification{})

	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrNotificationNotFound
	}
	return nil
}
func (r *repository) MarkAllAsRead(ctx context.Context, userID int64) (int64, error) {
	result := r.db.WithContext(ctx).
		Model(&Notification{}).
		Where("user_id = ? AND is_read = ? AND deleted_at IS NULL", userID, false).
		Update("is_read", true)
	return result.RowsAffected, result.Error
}
