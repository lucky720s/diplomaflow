package team

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	ErrTeamNotFound           = errors.New("team not found")
	ErrMemberNotFound         = errors.New("member not found")
	ErrNotTeamMember          = errors.New("user is not a team member")
	ErrNotTeamLeader          = errors.New("user is not the team leader")
	ErrAlreadyInTeam          = errors.New("user is already in a team")
	ErrCannotLeaveAsLast      = errors.New("cannot leave team as last member without deleting")
	ErrPendingInviteExists    = errors.New("pending invite already exists")
	ErrInviteAlreadyProcessed = errors.New("invite already processed")
	ErrInviteExpired          = errors.New("invite expired")
)

type TeamInviteWithTeam struct {
	TeamInvite
	TeamName string
}
type Repository interface {
	Create(ctx context.Context, team *Team) error
	GetByID(ctx context.Context, id int64) (*Team, error)
	Update(ctx context.Context, team *Team) error
	Delete(ctx context.Context, id int64) error

	// Members
	AddMember(ctx context.Context, member *TeamMember) error
	RemoveMember(ctx context.Context, teamID, userID int64) error
	GetMembers(ctx context.Context, teamID int64) ([]*TeamMember, error)
	GetMember(ctx context.Context, teamID, userID int64) (*TeamMember, error)
	GetMemberCount(ctx context.Context, teamID int64) (int64, error)
	UpdateMemberRole(ctx context.Context, teamID, userID int64, role string) error
	GetTeamLeader(ctx context.Context, teamID int64) (*TeamMember, error)

	// User's team
	GetUserTeam(ctx context.Context, userID int64) (*Team, *TeamMember, error)
	IsUserInTeam(ctx context.Context, userID int64) (bool, error)
	IsUserInSpecificTeam(ctx context.Context, userID, teamID int64) (bool, error)

	// Invites
	CreateInvite(ctx context.Context, invite *TeamInvite) error
	GetInvite(ctx context.Context, id int64) (*TeamInvite, error)
	GetPendingInvites(ctx context.Context, userID int64) ([]*TeamInvite, error)
	UpdateInvite(ctx context.Context, invite *TeamInvite) error
	GetPendingInvitesCount(ctx context.Context, teamID int64) (int64, error)

	// List
	ListTeams(ctx context.Context, departmentID int64, limit, offset int) ([]*Team, int64, error)
	// Invite code
	GetByInviteCode(ctx context.Context, universityID int64, code string) (*Team, error)
	UpdateInviteCode(ctx context.Context, teamID int64, code string) error

	// Lock
	SetCompositionLocked(ctx context.Context, teamID int64, locked bool, lockedAt *time.Time) error
	GetPendingInvitesWithTeams(ctx context.Context, userID int64) ([]*TeamInviteWithTeam, error)
	GetMembersByTeamIDs(ctx context.Context, teamIDs []int64) (map[int64][]*TeamMember, error)
	GetUsersInTeams(ctx context.Context, userIDs []int64) (map[int64]bool, error)

	GetSupervisorAssignment(ctx context.Context, teamID int64) (*SupervisorAssignment, error)

	HasPendingInvite(ctx context.Context, teamID, userID int64) (bool, error)
	CreateInviteSafe(ctx context.Context, invite *TeamInvite) error
	AcceptInvite(ctx context.Context, inviteID, userID int64, maxTeamSize int32) (*TeamInvite, error)
	RejectInvite(ctx context.Context, inviteID, userID int64) (*TeamInvite, error)
}

type repository struct {
	db *gorm.DB
}

type SupervisorAssignment struct {
	ID           int64     `gorm:"primaryKey"`
	TeamID       int64     `gorm:"column:team_id"`
	SupervisorID int64     `gorm:"column:supervisor_id"`
	AssignedBy   int64     `gorm:"column:assigned_by"`
	CreatedAt    time.Time `gorm:"column:created_at"`
	UpdatedAt    time.Time `gorm:"column:updated_at"`
}

func (SupervisorAssignment) TableName() string {
	return "admin_supervisor_assignments"
}

func (r *repository) GetSupervisorAssignment(ctx context.Context, teamID int64) (*SupervisorAssignment, error) {
	var assignment SupervisorAssignment

	err := r.db.WithContext(ctx).
		Where("team_id = ?", teamID).
		First(&assignment).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil // Supervisor не назначен
		}
		return nil, err
	}

	return &assignment, nil
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

func (r *repository) Create(ctx context.Context, team *Team) error {
	team.CreatedAt = time.Now()
	team.UpdatedAt = time.Now()
	return r.db.WithContext(ctx).Create(team).Error
}

func (r *repository) GetByID(ctx context.Context, id int64) (*Team, error) {
	var team Team
	if err := r.db.WithContext(ctx).First(&team, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrTeamNotFound
		}
		return nil, err
	}
	return &team, nil
}

func (r *repository) Update(ctx context.Context, team *Team) error {
	team.UpdatedAt = time.Now()
	return r.db.WithContext(ctx).Save(team).Error
}

func (r *repository) Delete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Удаляем приглашения
		if err := tx.Where("team_id = ?", id).Delete(&TeamInvite{}).Error; err != nil {
			return err
		}
		if err := tx.Where("team_id = ?", id).Delete(&TeamMember{}).Error; err != nil {
			return err
		}
		return tx.Delete(&Team{}, id).Error
	})
}

func (r *repository) AddMember(ctx context.Context, member *TeamMember) error {
	member.CreatedAt = time.Now()
	return r.db.WithContext(ctx).Create(member).Error
}

func (r *repository) RemoveMember(ctx context.Context, teamID, userID int64) error {
	result := r.db.WithContext(ctx).
		Where("team_id = ? AND user_id = ?", teamID, userID).
		Delete(&TeamMember{})
	if result.RowsAffected == 0 {
		return ErrMemberNotFound
	}
	return result.Error
}

func (r *repository) GetMembers(ctx context.Context, teamID int64) ([]*TeamMember, error) {
	var members []*TeamMember
	err := r.db.WithContext(ctx).
		Where("team_id = ?", teamID).
		Order("role DESC, created_at ASC").
		Find(&members).Error
	return members, err
}

func (r *repository) GetMember(ctx context.Context, teamID, userID int64) (*TeamMember, error) {
	var member TeamMember
	err := r.db.WithContext(ctx).
		Where("team_id = ? AND user_id = ?", teamID, userID).
		First(&member).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrMemberNotFound
	}
	return &member, err
}

func (r *repository) GetMemberCount(ctx context.Context, teamID int64) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&TeamMember{}).
		Where("team_id = ?", teamID).
		Count(&count).Error
	return count, err
}

func (r *repository) UpdateMemberRole(ctx context.Context, teamID, userID int64, role string) error {
	result := r.db.WithContext(ctx).
		Model(&TeamMember{}).
		Where("team_id = ? AND user_id = ?", teamID, userID).
		Update("role", role)
	if result.RowsAffected == 0 {
		return ErrMemberNotFound
	}
	return result.Error
}

func (r *repository) GetTeamLeader(ctx context.Context, teamID int64) (*TeamMember, error) {
	var member TeamMember
	err := r.db.WithContext(ctx).
		Where("team_id = ? AND role = ?", teamID, RoleLeader).
		First(&member).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrMemberNotFound
	}
	return &member, err
}

func (r *repository) GetUserTeam(ctx context.Context, userID int64) (*Team, *TeamMember, error) {
	var member TeamMember
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		First(&member).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil, nil // Пользователь не в команде
	}
	if err != nil {
		return nil, nil, err
	}

	var team Team
	if err := r.db.WithContext(ctx).First(&team, member.TeamID).Error; err != nil {
		return nil, nil, err
	}

	return &team, &member, nil
}

func (r *repository) IsUserInTeam(ctx context.Context, userID int64) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&TeamMember{}).
		Where("user_id = ?", userID).
		Count(&count).Error
	return count > 0, err
}

func (r *repository) IsUserInSpecificTeam(ctx context.Context, userID, teamID int64) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&TeamMember{}).
		Where("user_id = ? AND team_id = ?", userID, teamID).
		Count(&count).Error
	return count > 0, err
}

func (r *repository) CreateInvite(ctx context.Context, invite *TeamInvite) error {
	invite.CreatedAt = time.Now()
	invite.UpdatedAt = time.Now()
	return r.db.WithContext(ctx).Create(invite).Error
}

func (r *repository) GetInvite(ctx context.Context, id int64) (*TeamInvite, error) {
	var invite TeamInvite
	if err := r.db.WithContext(ctx).First(&invite, id).Error; err != nil {
		return nil, err
	}
	return &invite, nil
}

func (r *repository) GetPendingInvites(ctx context.Context, userID int64) ([]*TeamInvite, error) {
	var invites []*TeamInvite
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND status = ?", userID, InviteStatusPending).
		Find(&invites).Error
	return invites, err
}

func (r *repository) UpdateInvite(ctx context.Context, invite *TeamInvite) error {
	invite.UpdatedAt = time.Now()
	return r.db.WithContext(ctx).Save(invite).Error
}

func (r *repository) GetPendingInvitesCount(ctx context.Context, teamID int64) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&TeamInvite{}).
		Where("team_id = ? AND status = ?", teamID, InviteStatusPending).
		Count(&count).Error
	return count, err
}

func (r *repository) ListTeams(ctx context.Context, departmentID int64, limit, offset int) ([]*Team, int64, error) {
	var teams []*Team
	var total int64

	query := r.db.WithContext(ctx).Model(&Team{})

	if departmentID > 0 {
		query = query.Where("department_id = ?", departmentID)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := query.
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&teams).Error

	return teams, total, err
}

func (r *repository) GetByInviteCode(ctx context.Context, universityID int64, code string) (*Team, error) {
	var team Team
	err := r.db.WithContext(ctx).
		Where("university_id = ? AND invite_code = ?", universityID, code).
		First(&team).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrTeamNotFound
	}
	return &team, err
}

func (r *repository) UpdateInviteCode(ctx context.Context, teamID int64, code string) error {
	return r.db.WithContext(ctx).
		Model(&Team{}).
		Where("id = ?", teamID).
		Updates(map[string]any{
			"invite_code": code,
			"updated_at":  time.Now(),
		}).Error
}

func (r *repository) SetCompositionLocked(ctx context.Context, teamID int64, locked bool, lockedAt *time.Time) error {
	return r.db.WithContext(ctx).
		Model(&Team{}).
		Where("id = ?", teamID).
		Updates(map[string]any{
			"composition_locked":    locked,
			"composition_locked_at": lockedAt,
			"updated_at":            time.Now(),
		}).Error
}
func (r *repository) GetPendingInvitesWithTeams(ctx context.Context, userID int64) ([]*TeamInviteWithTeam, error) {
	var results []*TeamInviteWithTeam
	err := r.db.WithContext(ctx).
		Table("team_invites").
		Select("team_invites.*, teams.name as team_name").
		Joins("LEFT JOIN teams ON teams.id = team_invites.team_id").
		Where("team_invites.user_id = ? AND team_invites.status = ?", userID, InviteStatusPending).
		Scan(&results).Error
	return results, err
}
func (r *repository) GetMembersByTeamIDs(ctx context.Context, teamIDs []int64) (map[int64][]*TeamMember, error) {
	var members []*TeamMember
	err := r.db.WithContext(ctx).
		Where("team_id IN ?", teamIDs).
		Order("role DESC, created_at ASC").
		Find(&members).Error
	if err != nil {
		return nil, err
	}

	result := make(map[int64][]*TeamMember)
	for _, m := range members {
		result[m.TeamID] = append(result[m.TeamID], m)
	}
	return result, nil
}
func (r *repository) GetUsersInTeams(ctx context.Context, userIDs []int64) (map[int64]bool, error) {
	var members []*TeamMember
	err := r.db.WithContext(ctx).Where("user_id IN ?", userIDs).Find(&members).Error
	result := make(map[int64]bool)
	for _, m := range members {
		result[m.UserID] = true
	}
	return result, err
}

func (r *repository) HasPendingInvite(ctx context.Context, teamID, userID int64) (bool, error) {
	var count int64

	err := r.db.WithContext(ctx).
		Model(&TeamInvite{}).
		Where("team_id = ? AND user_id = ? AND status = ?", teamID, userID, InviteStatusPending).
		Count(&count).Error

	return count > 0, err
}
func (r *repository) AcceptInvite(ctx context.Context, inviteID, userID int64, maxTeamSize int32) (*TeamInvite, error) {
	now := time.Now().UTC()

	var invite TeamInvite

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			First(&invite, "id = ? AND user_id = ?", inviteID, userID).Error; err != nil {
			return err
		}

		if invite.Status != InviteStatusPending {
			return ErrInviteAlreadyProcessed
		}

		if now.After(invite.ExpiresAt) {
			invite.Status = InviteStatusExpired
			invite.UpdatedAt = now

			if err := tx.Save(&invite).Error; err != nil {
				return err
			}

			return ErrInviteExpired
		}

		var team Team
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			First(&team, "id = ?", invite.TeamID).Error; err != nil {
			return err
		}

		if team.CompositionLocked {
			return ErrTeamCompositionLocked
		}

		var existingMember int64
		if err := tx.Model(&TeamMember{}).
			Where("user_id = ?", userID).
			Count(&existingMember).Error; err != nil {
			return err
		}

		if existingMember > 0 {
			invite.Status = InviteStatusDeclined
			invite.UpdatedAt = now

			if err := tx.Save(&invite).Error; err != nil {
				return err
			}

			return ErrAlreadyInTeam
		}

		if maxTeamSize > 0 {
			var memberCount int64
			if err := tx.Model(&TeamMember{}).
				Where("team_id = ?", invite.TeamID).
				Count(&memberCount).Error; err != nil {
				return err
			}

			if int32(memberCount) >= maxTeamSize {
				return ErrTeamFull
			}
		}

		if err := tx.Create(&TeamMember{
			TeamID:    invite.TeamID,
			UserID:    userID,
			Role:      RoleMember,
			CreatedAt: now,
		}).Error; err != nil {
			return err
		}

		invite.Status = InviteStatusAccepted
		invite.UpdatedAt = now

		if err := tx.Save(&invite).Error; err != nil {
			return err
		}

		// После вступления отменяем остальные pending invites этого студента.
		if err := tx.Model(&TeamInvite{}).
			Where("user_id = ? AND id <> ? AND status = ?", userID, inviteID, InviteStatusPending).
			Updates(map[string]any{
				"status":     InviteStatusDeclined,
				"updated_at": now,
			}).Error; err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return &invite, nil
}
func (r *repository) RejectInvite(ctx context.Context, inviteID, userID int64) (*TeamInvite, error) {
	now := time.Now().UTC()

	var invite TeamInvite

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			First(&invite, "id = ? AND user_id = ?", inviteID, userID).Error; err != nil {
			return err
		}

		if invite.Status != InviteStatusPending {
			return ErrInviteAlreadyProcessed
		}

		if now.After(invite.ExpiresAt) {
			invite.Status = InviteStatusExpired
		} else {
			invite.Status = InviteStatusDeclined
		}

		invite.UpdatedAt = now
		return tx.Save(&invite).Error
	})

	if err != nil {
		return nil, err
	}

	return &invite, nil
}
func (r *repository) CreateInviteSafe(ctx context.Context, invite *TeamInvite) error {
	now := time.Now().UTC()

	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var existing TeamInvite

		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("team_id = ? AND user_id = ?", invite.TeamID, invite.UserID).
			Order("created_at DESC").
			First(&existing).Error

		if err == nil {
			if existing.Status == InviteStatusPending {
				return ErrPendingInviteExists
			}

			// Если раньше отклонили/истекло — разрешаем повторно отправить приглашение
			existing.Status = InviteStatusPending
			existing.InviterID = invite.InviterID
			existing.UpdatedAt = now

			if invite.ExpiresAt.IsZero() {
				existing.ExpiresAt = now.Add(3 * 24 * time.Hour)
			} else {
				existing.ExpiresAt = invite.ExpiresAt
			}

			return tx.Save(&existing).Error
		}

		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		invite.Status = InviteStatusPending
		invite.CreatedAt = now
		invite.UpdatedAt = now

		if invite.ExpiresAt.IsZero() {
			invite.ExpiresAt = now.Add(3 * 24 * time.Hour)
		}

		return tx.Create(invite).Error
	})
}
