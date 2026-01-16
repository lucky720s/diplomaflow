package team

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
)

type Repository interface {
	Create(ctx context.Context, team *Team) error
	GetByID(ctx context.Context, id uint64) (*Team, error)

	AddMember(ctx context.Context, member *TeamMember) error
	RemoveMember(ctx context.Context, teamID uint64, userID int64) error
	GetMember(ctx context.Context, teamID uint64, userID int64) (*TeamMember, error)

	Update(ctx context.Context, team *Team) error
	Delete(ctx context.Context, id uint64) error

	CreateInvite(ctx context.Context, invite *TeamInvite) error
	GetInvitesByUserID(ctx context.Context, userID int64) ([]*TeamInvite, error)
	GetInviteByID(ctx context.Context, inviteID uint64) (*TeamInvite, error)
	UpdateInviteStatus(ctx context.Context, inviteID uint64, status string) error
	DeletePendingInvitesForUser(ctx context.Context, userID int64) error

	IsUserInAnyTeam(ctx context.Context, userID int64) (bool, error)

	CreateTeamWithInvites(ctx context.Context, team *Team, invites []*TeamInvite) error
	GetTeamByUserID(ctx context.Context, userID int64) (*Team, string, error)
	CountPendingInvitesByTeam(ctx context.Context, teamID uint64) (int64, error)

	List(ctx context.Context, departmentID, projectID int64, limit, offset int) ([]*Team, int64, error)
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

func (r *repository) Create(ctx context.Context, team *Team) error {
	return r.db.WithContext(ctx).Create(team).Error
}

func (r *repository) GetByID(ctx context.Context, id uint64) (*Team, error) {
	var team Team
	if err := r.db.WithContext(ctx).
		Preload("Members").
		First(&team, id).Error; err != nil {

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

func (r *repository) Update(ctx context.Context, team *Team) error {
	return r.db.WithContext(ctx).Save(team).Error
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

func (r *repository) CreateInvite(ctx context.Context, invite *TeamInvite) error {
	return r.db.WithContext(ctx).Create(invite).Error
}

func (r *repository) GetInvitesByUserID(ctx context.Context, userID int64) ([]*TeamInvite, error) {
	var invites []*TeamInvite
	err := r.db.WithContext(ctx).
		Preload("Team").
		Where("user_id = ? AND status = ?", userID, "PENDING").
		Order("created_at DESC").
		Find(&invites).Error
	return invites, err
}

func (r *repository) GetInviteByID(ctx context.Context, inviteID uint64) (*TeamInvite, error) {
	var invite TeamInvite
	err := r.db.WithContext(ctx).
		Preload("Team").
		First(&invite, inviteID).Error
	if err != nil {
		return nil, err
	}
	return &invite, nil
}

func (r *repository) UpdateInviteStatus(ctx context.Context, inviteID uint64, status string) error {
	return r.db.WithContext(ctx).
		Model(&TeamInvite{}).
		Where("id = ?", inviteID).
		Updates(map[string]interface{}{
			"status":     status,
			"updated_at": time.Now().UTC(),
		}).Error
}

func (r *repository) DeletePendingInvitesForUser(ctx context.Context, userID int64) error {
	return r.db.WithContext(ctx).
		Model(&TeamInvite{}).
		Where("user_id = ? AND status = ?", userID, "PENDING").
		Updates(map[string]interface{}{
			"status":     "AUTO_DECLINED",
			"updated_at": time.Now().UTC(),
		}).Error
}

// Senior fix: учитывать soft-delete команд.
// Иначе пользователь, который был в удалённой команде, останется "занятым".
func (r *repository) IsUserInAnyTeam(ctx context.Context, userID int64) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Table("team_members tm").
		Joins("JOIN teams t ON t.id = tm.team_id AND t.deleted_at IS NULL").
		Where("tm.user_id = ?", userID).
		Count(&count).Error
	return count > 0, err
}

func (r *repository) CreateTeamWithInvites(ctx context.Context, team *Team, invites []*TeamInvite) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(team).Error; err != nil {
			return err
		}
		if len(invites) == 0 {
			return nil
		}
		now := time.Now().UTC()
		for _, inv := range invites {
			inv.TeamID = int64(team.ID)
			if inv.CreatedAt.IsZero() {
				inv.CreatedAt = now
			}
			inv.UpdatedAt = now
		}
		return tx.Create(&invites).Error
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
	if err := r.db.WithContext(ctx).
		Preload("Members").
		First(&team, member.TeamID).Error; err != nil {
		return nil, "", err
	}

	return &team, member.Role, nil
}

func (r *repository) CountPendingInvitesByTeam(ctx context.Context, teamID uint64) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&TeamInvite{}).
		Where("team_id = ? AND status = ?", teamID, "PENDING").
		Count(&count).Error
	return count, err
}

// ✅ Главная часть шага 3: department filter через JOIN projects.department_id
func (r *repository) List(ctx context.Context, departmentID, projectID int64, limit, offset int) ([]*Team, int64, error) {
	var teams []*Team
	var total int64

	q := r.db.WithContext(ctx).Model(&Team{})

	// filter by project
	if projectID > 0 {
		q = q.Where("teams.project_id = ?", projectID)
	}

	// filter by department (через projects)
	// Основание: department_id находится в projects, а teams связаны по teams.project_id [[1]].
	if departmentID > 0 {
		q = q.Joins("JOIN projects p ON p.id = teams.project_id").
			Where("p.department_id = ?", departmentID)
	}

	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := q.Preload("Members").
		Order("teams.created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&teams).Error

	return teams, total, err
}
