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
	CreateInvite(ctx context.Context, invite *TeamInvite) error
	GetInvitesByUserID(ctx context.Context, userID int64) ([]*TeamInvite, error)
	GetInviteByID(ctx context.Context, inviteID uint64) (*TeamInvite, error)
	UpdateInviteStatus(ctx context.Context, inviteID uint64, status string) error
	IsUserInAnyTeam(ctx context.Context, userID int64) (bool, error)
	DeletePendingInvitesForUser(ctx context.Context, userID int64) error
	CreateTeamWithInvites(ctx context.Context, team *Team, invites []*TeamInvite) error
	GetTeamByUserID(ctx context.Context, userID int64) (*Team, string, error)
	CountPendingInvitesByTeam(ctx context.Context, teamID uint64) (int64, error)
	List(ctx context.Context, departmentID, projectID int64, limit, offset int) ([]*Team, int64, error)
	Delete(ctx context.Context, id uint64) error
	GetMember(ctx context.Context, teamID uint64, userID int64) (*TeamMember, error)
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	_ = db.AutoMigrate(&Team{}, &TeamMember{}, &TeamInvite{})
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
func (r *repository) CreateInvite(ctx context.Context, invite *TeamInvite) error {
	return r.db.WithContext(ctx).Create(invite).Error
}

func (r *repository) GetInvitesByUserID(ctx context.Context, userID int64) ([]*TeamInvite, error) {
	var invites []*TeamInvite
	err := r.db.WithContext(ctx).Preload("Team").Where("user_id = ? AND status = ?", userID, "PENDING").Find(&invites).Error
	return invites, err
}

func (r *repository) GetInviteByID(ctx context.Context, inviteID uint64) (*TeamInvite, error) {
	var invite TeamInvite
	err := r.db.WithContext(ctx).First(&invite, inviteID).Error
	return &invite, err
}

func (r *repository) UpdateInviteStatus(ctx context.Context, inviteID uint64, status string) error {
	return r.db.WithContext(ctx).Model(&TeamInvite{}).Where("id = ?", inviteID).Update("status", status).Error
}

func (r *repository) IsUserInAnyTeam(ctx context.Context, userID int64) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&TeamMember{}).Where("user_id = ?", userID).Count(&count).Error
	return count > 0, err
}

func (r *repository) DeletePendingInvitesForUser(ctx context.Context, userID int64) error {
	return r.db.WithContext(ctx).Model(&TeamInvite{}).
		Where("user_id = ? AND status = ?", userID, "PENDING").
		Update("status", "AUTO_DECLINED").Error
}
func (r *repository) CreateTeamWithInvites(ctx context.Context, team *Team, invites []*TeamInvite) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(team).Error; err != nil {
			return err
		}
		for _, inv := range invites {
			inv.TeamID = int64(team.ID)
			if err := tx.Create(inv).Error; err != nil {
				return err
			}
		}
		return nil
	})
}
func (r *repository) GetTeamByUserID(ctx context.Context, userID int64) (*Team, string, error) {
	var member TeamMember
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).First(&member).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, "", nil
		}
		return nil, "", err
	}

	var team Team
	if err := r.db.WithContext(ctx).Preload("Members").First(&team, member.TeamID).Error; err != nil {
		return nil, "", err
	}

	return &team, member.Role, nil
}
func (r *repository) CountPendingInvitesByTeam(ctx context.Context, teamID uint64) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&TeamInvite{}).
		Where("team_id = ? AND status = ?", teamID, "PENDING").
		Count(&count).Error
	return count, err
}
func (r *repository) List(ctx context.Context, departmentID, projectID int64, limit, offset int) ([]*Team, int64, error) {
	var teams []*Team
	var total int64

	query := r.db.WithContext(ctx).Model(&Team{})

	if projectID > 0 {
		query = query.Where("project_id = ?", projectID)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := query.Preload("Members").
		Limit(limit).
		Offset(offset).
		Order("created_at DESC").
		Find(&teams).Error

	return teams, total, err
}

func (r *repository) Delete(ctx context.Context, id uint64) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("team_id = ?", id).Delete(&TeamInvite{}).Error; err != nil {
			return err
		}
		if err := tx.Where("team_id = ?", id).Delete(&TeamMember{}).Error; err != nil {
			return err
		}
		if err := tx.Delete(&Team{}, id).Error; err != nil {
			return err
		}
		return nil
	})
}

func (r *repository) GetMember(ctx context.Context, teamID uint64, userID int64) (*TeamMember, error) {
	var member TeamMember
	err := r.db.WithContext(ctx).
		Where("team_id = ? AND user_id = ?", teamID, userID).
		First(&member).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &member, nil
}
