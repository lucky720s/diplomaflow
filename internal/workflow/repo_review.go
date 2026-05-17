package workflow

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var ErrReviewNotFound = errors.New("review not found")

// ReviewRepository — хранилище голосований преподов по стэйтам
type ReviewRepository interface {
	// UpsertReview сохраняет или обновляет голос препода (один препод = один голос за стэйт)
	UpsertReview(ctx context.Context, review *StateReview) error

	// GetReviews возвращает все голоса по стэйту проекта
	GetReviews(ctx context.Context, projectID, stateID int64) ([]StateReview, error)

	// GetReviewByReviewer возвращает голос конкретного препода (если есть)
	GetReviewByReviewer(ctx context.Context, projectID, stateID, reviewerID int64) (*StateReview, error)

	// CountReviewsByRole считает количество проголосовавших из конкретной роли
	CountReviewsByRole(ctx context.Context, projectID, stateID int64, roleSlug string) (int64, error)

	// DeleteReviews удаляет все голоса по стэйту (при откате проекта назад)
	DeleteReviews(ctx context.Context, projectID, stateID int64) error
}

type reviewRepository struct {
	db *gorm.DB
}

func NewReviewRepository(db *gorm.DB) ReviewRepository {
	return &reviewRepository{db: db}
}

func (r *reviewRepository) UpsertReview(ctx context.Context, review *StateReview) error {
	now := time.Now()
	review.UpdatedAt = now
	if review.CreatedAt.IsZero() {
		review.CreatedAt = now
	}

	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{
				{Name: "project_id"},
				{Name: "state_id"},
				{Name: "reviewer_id"},
			},
			DoUpdates: clause.AssignmentColumns([]string{
				"role_slug", "decision", "score", "comment", "updated_at",
			}),
		}).
		Create(review).Error
}

func (r *reviewRepository) GetReviews(ctx context.Context, projectID, stateID int64) ([]StateReview, error) {
	var reviews []StateReview
	err := r.db.WithContext(ctx).
		Where("project_id = ? AND state_id = ?", projectID, stateID).
		Order("created_at ASC").
		Find(&reviews).Error
	return reviews, err
}

func (r *reviewRepository) GetReviewByReviewer(ctx context.Context, projectID, stateID, reviewerID int64) (*StateReview, error) {
	var review StateReview
	err := r.db.WithContext(ctx).
		Where("project_id = ? AND state_id = ? AND reviewer_id = ?", projectID, stateID, reviewerID).
		First(&review).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrReviewNotFound
	}
	return &review, err
}

func (r *reviewRepository) CountReviewsByRole(ctx context.Context, projectID, stateID int64, roleSlug string) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&StateReview{}).
		Where("project_id = ? AND state_id = ? AND role_slug = ?", projectID, stateID, roleSlug).
		Count(&count).Error
	return count, err
}

func (r *reviewRepository) DeleteReviews(ctx context.Context, projectID, stateID int64) error {
	return r.db.WithContext(ctx).
		Where("project_id = ? AND state_id = ?", projectID, stateID).
		Delete(&StateReview{}).Error
}
