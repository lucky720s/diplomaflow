package team

import (
	"context"
	"errors"
	"fmt"
	"time"

	authv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/auth/v1"
	workflowv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/workflow/v1"
	"go.uber.org/zap"
)

var (
	ErrUnauthorized         = errors.New("unauthorized")
	ErrNotLeader            = errors.New("only team leader can perform this action")
	ErrCannotRemoveLeader   = errors.New("cannot remove team leader")
	ErrNewLeaderNotInTeam   = errors.New("new leader must be a team member")
	ErrCannotLeaveAsLeader  = errors.New("leader cannot leave without transferring leadership first")
	ErrTeamHasActiveProject = errors.New("cannot delete team with active project")
)

type Service struct {
	repo           Repository
	authClient     authv1.AuthServiceClient
	workflowClient workflowv1.WorkflowServiceClient
	logger         *zap.Logger
}

func NewService(
	repo Repository,
	authClient authv1.AuthServiceClient,
	workflowClient workflowv1.WorkflowServiceClient,
	logger *zap.Logger,
) *Service {
	return &Service{
		repo:           repo,
		authClient:     authClient,
		workflowClient: workflowClient,
		logger:         logger,
	}
}

func (s *Service) CreateTeam(ctx context.Context, name string, leaderID int64, memberIDs []int64, departmentID int64) (int64, error) {
	// Проверяем, что лидер не состоит в другой команде
	inTeam, err := s.repo.IsUserInTeam(ctx, leaderID)
	if err != nil {
		return 0, fmt.Errorf("check user team: %w", err)
	}
	if inTeam {
		return 0, ErrAlreadyInTeam
	}

	// Создаём команду
	team := &Team{
		Name: name,
	}
	if err := s.repo.Create(ctx, team); err != nil {
		return 0, fmt.Errorf("create team: %w", err)
	}

	// Добавляем лидера
	leaderMember := &TeamMember{
		TeamID: team.ID,
		UserID: leaderID,
		Role:   RoleLeader,
	}
	if err := s.repo.AddMember(ctx, leaderMember); err != nil {
		return 0, fmt.Errorf("add leader: %w", err)
	}

	// Создаём приглашения для остальных участников
	for _, memberID := range memberIDs {
		if memberID == leaderID {
			continue
		}
		// Проверяем, не в команде ли уже участник
		memberInTeam, _ := s.repo.IsUserInTeam(ctx, memberID)
		if memberInTeam {
			continue
		}

		invite := &TeamInvite{
			TeamID:    team.ID,
			UserID:    memberID,
			InviterID: leaderID,
			Status:    InviteStatusPending,
			ExpiresAt: time.Now().Add(72 * time.Hour), // 3 дня на ответ
		}
		_ = s.repo.CreateInvite(ctx, invite)
	}

	s.logger.Info("Team created",
		zap.Int64("team_id", team.ID),
		zap.String("name", name),
		zap.Int64("leader_id", leaderID))

	return team.ID, nil
}

func (s *Service) GetTeam(ctx context.Context, teamID int64) (*Team, []*TeamMember, error) {
	team, err := s.repo.GetByID(ctx, teamID)
	if err != nil {
		return nil, nil, err
	}

	members, err := s.repo.GetMembers(ctx, teamID)
	if err != nil {
		return nil, nil, err
	}

	return team, members, nil
}

func (s *Service) GetMyTeam(ctx context.Context, userID int64) (*Team, *TeamMember, []*TeamMember, int64, error) {
	team, membership, err := s.repo.GetUserTeam(ctx, userID)
	if err != nil {
		return nil, nil, nil, 0, err
	}
	if team == nil {
		return nil, nil, nil, 0, nil
	}

	members, err := s.repo.GetMembers(ctx, team.ID)
	if err != nil {
		return nil, nil, nil, 0, err
	}

	pendingCount, _ := s.repo.GetPendingInvitesCount(ctx, team.ID)

	return team, membership, members, pendingCount, nil
}

func (s *Service) UpdateTeam(ctx context.Context, teamID int64, name string, requesterID int64) (*Team, error) {
	// Проверяем права: только лидер может редактировать команду
	if err := s.validateLeader(ctx, teamID, requesterID); err != nil {
		return nil, err
	}

	team, err := s.repo.GetByID(ctx, teamID)
	if err != nil {
		return nil, err
	}

	if name != "" {
		team.Name = name
	}

	if err := s.repo.Update(ctx, team); err != nil {
		return nil, fmt.Errorf("update team: %w", err)
	}

	s.logger.Info("Team updated",
		zap.Int64("team_id", teamID),
		zap.String("new_name", name),
		zap.Int64("by_user", requesterID))

	return team, nil
}

func (s *Service) DeleteTeam(ctx context.Context, teamID int64, requesterID int64) error {
	if err := s.validateLeader(ctx, teamID, requesterID); err != nil {
		return err
	}

	team, err := s.repo.GetByID(ctx, teamID)
	if err != nil {
		return err
	}

	// Проверяем, нет ли активного проекта
	if team.ProjectID > 0 {
		s.logger.Warn("Deleting team with project",
			zap.Int64("team_id", teamID),
			zap.Int64("project_id", team.ProjectID))
	}

	if err := s.repo.Delete(ctx, teamID); err != nil {
		return fmt.Errorf("delete team: %w", err)
	}

	s.logger.Info("Team deleted",
		zap.Int64("team_id", teamID),
		zap.Int64("by_user", requesterID))

	return nil
}

func (s *Service) LeaveTeam(ctx context.Context, teamID int64, userID int64) (*LeaveTeamResult, error) {
	// Проверяем, что пользователь состоит в этой команде
	member, err := s.repo.GetMember(ctx, teamID, userID)
	if err != nil {
		if errors.Is(err, ErrMemberNotFound) {
			return nil, ErrNotTeamMember
		}
		return nil, err
	}

	result := &LeaveTeamResult{
		Success: true,
		Message: "Вы успешно вышли из команды",
	}

	memberCount, err := s.repo.GetMemberCount(ctx, teamID)
	if err != nil {
		return nil, err
	}

	// Если это последний участник — удаляем команду
	if memberCount == 1 {
		if err := s.repo.Delete(ctx, teamID); err != nil {
			return nil, fmt.Errorf("delete team: %w", err)
		}
		result.TeamDeleted = true
		result.Message = "Вы вышли из команды. Команда удалена, так как вы были последним участником."

		s.logger.Info("Last member left, team deleted",
			zap.Int64("team_id", teamID),
			zap.Int64("user_id", userID))

		return result, nil
	}

	// Если выходит лидер — нужно передать лидерство
	if member.Role == RoleLeader {
		// Находим нового лидера (первый по времени вступления)
		members, err := s.repo.GetMembers(ctx, teamID)
		if err != nil {
			return nil, err
		}

		var newLeaderID int64
		for _, m := range members {
			if m.UserID != userID {
				newLeaderID = m.UserID
				break
			}
		}

		if newLeaderID == 0 {
			return nil, errors.New("no suitable member to become leader")
		}

		if err := s.repo.UpdateMemberRole(ctx, teamID, newLeaderID, RoleLeader); err != nil {
			return nil, fmt.Errorf("transfer leadership: %w", err)
		}

		result.NewLeaderID = newLeaderID
		result.Message = fmt.Sprintf("Вы вышли из команды. Лидерство передано участнику с ID %d.", newLeaderID)

		s.logger.Info("Leader left, leadership transferred",
			zap.Int64("team_id", teamID),
			zap.Int64("old_leader", userID),
			zap.Int64("new_leader", newLeaderID))
	}

	if err := s.repo.RemoveMember(ctx, teamID, userID); err != nil {
		return nil, fmt.Errorf("remove member: %w", err)
	}

	s.logger.Info("Member left team",
		zap.Int64("team_id", teamID),
		zap.Int64("user_id", userID))

	return result, nil
}

// LeaveTeamResult - результат выхода из команды
type LeaveTeamResult struct {
	Success     bool
	Message     string
	TeamDeleted bool
	NewLeaderID int64
}

func (s *Service) TransferLeadership(ctx context.Context, teamID int64, currentLeaderID int64, newLeaderID int64) (*Team, []*TeamMember, error) {
	// Проверяем, что текущий пользователь — лидер
	if err := s.validateLeader(ctx, teamID, currentLeaderID); err != nil {
		return nil, nil, err
	}

	newLeaderMember, err := s.repo.GetMember(ctx, teamID, newLeaderID)
	if err != nil {
		if errors.Is(err, ErrMemberNotFound) {
			return nil, nil, ErrNewLeaderNotInTeam
		}
		return nil, nil, err
	}

	if newLeaderMember.Role == RoleLeader {
		team, members, _ := s.GetTeam(ctx, teamID)
		return team, members, nil
	}

	if err := s.repo.UpdateMemberRole(ctx, teamID, currentLeaderID, RoleMember); err != nil {
		return nil, nil, fmt.Errorf("demote current leader: %w", err)
	}

	if err := s.repo.UpdateMemberRole(ctx, teamID, newLeaderID, RoleLeader); err != nil {
		_ = s.repo.UpdateMemberRole(ctx, teamID, currentLeaderID, RoleLeader)
		return nil, nil, fmt.Errorf("promote new leader: %w", err)
	}

	s.logger.Info("Leadership transferred",
		zap.Int64("team_id", teamID),
		zap.Int64("from_user", currentLeaderID),
		zap.Int64("to_user", newLeaderID))

	return s.GetTeam(ctx, teamID)
}

func (s *Service) AddMember(ctx context.Context, teamID int64, userID int64, role string) error {
	// Проверяем, не в команде ли уже
	inTeam, err := s.repo.IsUserInTeam(ctx, userID)
	if err != nil {
		return err
	}
	if inTeam {
		return ErrAlreadyInTeam
	}

	if role == "" {
		role = RoleMember
	}

	member := &TeamMember{
		TeamID: teamID,
		UserID: userID,
		Role:   role,
	}

	return s.repo.AddMember(ctx, member)
}

func (s *Service) RemoveMember(ctx context.Context, teamID int64, userID int64, requesterID int64) error {
	// Проверяем права: только лидер может удалять участников
	if err := s.validateLeader(ctx, teamID, requesterID); err != nil {
		return err
	}

	if userID == requesterID {
		return errors.New("use LeaveTeam to leave the team")
	}

	// Проверяем, что удаляемый — участник команды
	member, err := s.repo.GetMember(ctx, teamID, userID)
	if err != nil {
		return err
	}

	// Нельзя удалить лидера
	if member.Role == RoleLeader {
		return ErrCannotRemoveLeader
	}

	return s.repo.RemoveMember(ctx, teamID, userID)
}

func (s *Service) GetMyInvites(ctx context.Context, userID int64) ([]*TeamInvite, error) {
	return s.repo.GetPendingInvites(ctx, userID)
}

func (s *Service) RespondToInvite(ctx context.Context, inviteID int64, userID int64, accept bool) error {
	invite, err := s.repo.GetInvite(ctx, inviteID)
	if err != nil {
		return err
	}

	if invite.UserID != userID {
		return ErrUnauthorized
	}

	if invite.Status != InviteStatusPending {
		return errors.New("invite already processed")
	}

	if time.Now().After(invite.ExpiresAt) {
		invite.Status = InviteStatusExpired
		_ = s.repo.UpdateInvite(ctx, invite)
		return errors.New("invite expired")
	}

	if accept {
		// Проверяем, не в команде ли уже
		inTeam, _ := s.repo.IsUserInTeam(ctx, userID)
		if inTeam {
			invite.Status = InviteStatusDeclined
			_ = s.repo.UpdateInvite(ctx, invite)
			return ErrAlreadyInTeam
		}

		member := &TeamMember{
			TeamID: invite.TeamID,
			UserID: userID,
			Role:   RoleMember,
		}
		if err := s.repo.AddMember(ctx, member); err != nil {
			return err
		}
		invite.Status = InviteStatusAccepted
	} else {
		invite.Status = InviteStatusDeclined
	}

	return s.repo.UpdateInvite(ctx, invite)
}

func (s *Service) GetAvailableStudents(ctx context.Context, universityID int64, departmentID int64, excludeUserID int64) ([]*authv1.UserPreview, error) {
	resp, err := s.authClient.ListUsers(ctx, &authv1.ListUsersRequest{
		UniversityId:  universityID,
		DepartmentId:  departmentID,
		Role:          "student",
		Page:          1,
		PageSize:      100,
		ExcludeUserId: excludeUserID,
	})
	if err != nil {
		return nil, err
	}

	// Фильтруем тех, кто уже в команде
	var available []*authv1.UserPreview
	for _, user := range resp.Users {
		inTeam, _ := s.repo.IsUserInTeam(ctx, user.Id)
		if !inTeam {
			available = append(available, user)
		}
	}

	return available, nil
}

func (s *Service) ListTeams(ctx context.Context, departmentID int64, page, pageSize int32) ([]*Team, int64, error) {
	if pageSize <= 0 {
		pageSize = 20
	}
	if page <= 0 {
		page = 1
	}
	offset := (page - 1) * pageSize

	return s.repo.ListTeams(ctx, departmentID, int(pageSize), int(offset))
}

func (s *Service) validateLeader(ctx context.Context, teamID int64, userID int64) error {
	member, err := s.repo.GetMember(ctx, teamID, userID)
	if err != nil {
		if errors.Is(err, ErrMemberNotFound) {
			return ErrNotTeamMember
		}
		return err
	}

	if member.Role != RoleLeader {
		return ErrNotLeader
	}
	return nil
}

// CreateTeamForProject - создаёт команду для проекта (вызывается из Kafka consumer)
func (s *Service) CreateTeamForProject(ctx context.Context, event ProjectCreatedEvent) error {
	s.logger.Info("Processing ProjectCreatedEvent",
		zap.Int64("project_id", event.ProjectID),
		zap.Int64("student_id", event.StudentID),
		zap.Int64("team_id", event.TeamID)) // Нужно поле TeamID в структуре!

	// Создаём новую команду
	team := &Team{
		Name:      fmt.Sprintf("Team for Project %d", event.ProjectID),
		ProjectID: event.ProjectID,
	}
	if err := s.repo.Create(ctx, team); err != nil {
		return fmt.Errorf("create team: %w", err)
	}

	// Добавляем студента как лидера
	member := &TeamMember{
		TeamID: team.ID,
		UserID: event.StudentID,
		Role:   RoleLeader,
	}
	if err := s.repo.AddMember(ctx, member); err != nil {
		return fmt.Errorf("add leader: %w", err)
	}

	s.logger.Info("Team created for project",
		zap.Int64("team_id", team.ID),
		zap.Int64("project_id", event.ProjectID))

	return nil
}
