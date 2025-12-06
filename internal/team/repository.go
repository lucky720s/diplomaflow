package team

import (
	"context"
	"errors"

	"gorm.io/gorm"
)

type Repository interface {
	Create(ctx context.Context, team *Team) error
	GetByID(ctx context.Context, id uint64) (*Team, error)
	AddMember(ctx context.Context, member *TeamMember) error
	RemoveMember(ctx context.Context, teamID uint64, userID int64) error
	Update(ctx context.Context, team *Team) error
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	_ = db.AutoMigrate(&Team{}, &TeamMember{})
	return &repository{db: db}
}

func (r *repository) Create(ctx context.Context, team *Team) error {
	return r.db.WithContext(ctx).Create(team).Error
}

func (r *repository) GetByID(ctx context.Context, id uint64) (*Team, error) {
	var team Team
	if err := r.db.WithContext(ctx).Preload("Members").First(&team, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("team not found")
		}
		return nil, err
	}
	return &team, nil
}

func (r *repository) AddMember(ctx context.Context, member *TeamMember) error {
	return r.db.WithContext(ctx).Create(member).Error
}

func (r *repository) RemoveMember(ctx context.Context, teamID uint64, userID int64) error {
	return r.db.WithContext(ctx).
		Where("team_id = ? AND user_id = ?", teamID, userID).
		Delete(&TeamMember{}).Error
}
func (r *repository) Update(ctx context.Context, team *Team) error {
	return r.db.WithContext(ctx).Save(team).Error
}
