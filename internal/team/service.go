package team

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"time"

	adminv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/admin/v1"
	authv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/auth/v1"
	notificationv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/notification/v1"
	workflowv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/workflow/v1"
	"go.uber.org/zap"
	"google.golang.org/grpc/metadata"
)

var (
	ErrUnauthorized          = errors.New("unauthorized")
	ErrNotLeader             = errors.New("only team leader can perform this action")
	ErrCannotRemoveLeader    = errors.New("cannot remove team leader")
	ErrNewLeaderNotInTeam    = errors.New("new leader must be a team member")
	ErrCannotLeaveAsLeader   = errors.New("leader cannot leave without transferring leadership first")
	ErrTeamHasActiveProject  = errors.New("cannot delete team with active project")
	ErrInvalidInviteCode     = errors.New("invalid invite code")
	ErrTeamCompositionLocked = errors.New("team composition is locked")
	ErrTeamFull              = errors.New("team is full")
	ErrForbiddenDepartment   = errors.New("user department does not match team department")
	ErrForbiddenRole         = errors.New("only students can join by code")
)

type Service struct {
	repo               Repository
	authClient         authv1.AuthServiceClient
	workflowClient     workflowv1.WorkflowServiceClient
	notificationClient notificationv1.NotificationServiceClient
	adminClient        adminv1.AdminServiceClient // ДОБАВИТЬ
	logger             *zap.Logger
}

func NewService(
	repo Repository,
	authClient authv1.AuthServiceClient,
	workflowClient workflowv1.WorkflowServiceClient,
	notificationClient notificationv1.NotificationServiceClient,
	adminClient adminv1.AdminServiceClient,
	logger *zap.Logger,
) *Service {
	return &Service{
		repo:               repo,
		authClient:         authClient,
		workflowClient:     workflowClient,
		notificationClient: notificationClient,
		adminClient:        adminClient,
		logger:             logger,
	}
}

func (s *Service) notifyBestEffort(ctx context.Context, userID int64, title, message, link, nType string) {
	if s.notificationClient == nil || userID <= 0 {
		return
	}
	nctx := metadata.AppendToOutgoingContext(ctx, "x-internal-service", "team_service")
	_, err := s.notificationClient.SendNotification(nctx, &notificationv1.SendNotificationRequest{
		UserId:  userID,
		Title:   title,
		Message: message,
		Link:    link,
		Type:    nType,
	})
	if err != nil {
		s.logger.Warn("send notification failed", zap.Int64("user_id", userID), zap.Error(err))
	}
}

func (s *Service) workflowInternalCtx(ctx context.Context) context.Context {
	return metadata.AppendToOutgoingContext(ctx, "x-internal-service", "team_service")
}

func (s *Service) authInternalCtx(ctx context.Context) context.Context {
	return metadata.AppendToOutgoingContext(ctx, "x-internal-service", "team_service")
}

func (s *Service) CreateTeam(
	ctx context.Context,
	name string,
	leaderID int64,
	memberIDs []int64,
	departmentID, universityID int64,
) (int64, error) {
	inTeam, err := s.repo.IsUserInTeam(ctx, leaderID)
	if err != nil {
		return 0, fmt.Errorf("check user team: %w", err)
	}
	if inTeam {
		return 0, ErrAlreadyInTeam
	}

	// team config from workflow (internal)
	cfg, cfgErr := s.workflowClient.GetTeamConfiguration(s.workflowInternalCtx(ctx), &workflowv1.GetTeamConfigurationRequest{
		DepartmentId: departmentID,
		WorkflowId:   0,
		StateId:      0,
	})
	if cfgErr == nil && cfg != nil && cfg.TeamConfig != nil && cfg.TeamConfig.MaxSize > 0 {
		requested := int32(1 + len(memberIDs)) // leader + invited
		if requested > cfg.TeamConfig.MaxSize {
			return 0, ErrTeamFull
		}
	}

	// create team with unique code
	var team *Team
	for i := 0; i < 10; i++ {
		code, err := generateInviteCode()
		if err != nil {
			return 0, fmt.Errorf("generate invite code: %w", err)
		}

		t := &Team{
			Name:         name,
			UniversityID: universityID,
			DepartmentID: departmentID,
			InviteCode:   code,
		}

		if err := s.repo.Create(ctx, t); err != nil {
			if isUniqueViolation(err) {
				continue
			}
			return 0, fmt.Errorf("create team: %w", err)
		}
		team = t
		break
	}
	if team == nil {
		return 0, errors.New("failed to generate unique invite code")
	}

	// add leader
	if err := s.repo.AddMember(ctx, &TeamMember{
		TeamID: team.ID,
		UserID: leaderID,
		Role:   RoleLeader,
	}); err != nil {
		_ = s.repo.Delete(ctx, team.ID)
		return 0, fmt.Errorf("add leader: %w", err)
	}

	s.notifyBestEffort(ctx, leaderID,
		"Команда создана",
		fmt.Sprintf("Команда \"%s\" создана. Код для вступления: %s", team.Name, team.InviteCode),
		"/dashboard/team",
		"team_created",
	)

	// invite expiry days
	inviteDays := int32(3)
	if cfgErr == nil && cfg != nil && cfg.InviteExpireDays > 0 {
		inviteDays = cfg.InviteExpireDays
	}

	// create invites
	for _, memberID := range memberIDs {
		if memberID == leaderID {
			continue
		}

		memberInTeam, err := s.repo.IsUserInTeam(ctx, memberID)
		if err != nil {
			s.logger.Warn("check member team failed", zap.Int64("user_id", memberID), zap.Error(err))
			continue
		}
		if memberInTeam {
			continue
		}

		invite := &TeamInvite{
			TeamID:    team.ID,
			UserID:    memberID,
			InviterID: leaderID,
			Status:    InviteStatusPending,
			ExpiresAt: time.Now().UTC().Add(time.Duration(inviteDays) * 24 * time.Hour),
		}
		if err := s.repo.CreateInvite(ctx, invite); err != nil {
			s.logger.Warn("create invite failed", zap.Int64("team_id", team.ID), zap.Int64("user_id", memberID), zap.Error(err))
			continue
		}

		s.notifyBestEffort(ctx, memberID,
			"Приглашение в команду",
			fmt.Sprintf("Вас пригласили в команду \"%s\". Откройте приглашения, чтобы принять/отклонить.", team.Name),
			"/dashboard/team",
			"team_invite",
		)
	}

	s.logger.Info("Team created", zap.Int64("team_id", team.ID), zap.String("name", name), zap.Int64("leader_id", leaderID))
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
	s.logger.Info("Team updated", zap.Int64("team_id", teamID), zap.String("new_name", name), zap.Int64("by_user", requesterID))
	return team, nil
}

func (s *Service) DeleteTeam(ctx context.Context, teamID int64, requesterID int64) error {
	if err := s.validateLeader(ctx, teamID, requesterID); err != nil {
		return err
	}
	if err := s.repo.Delete(ctx, teamID); err != nil {
		return fmt.Errorf("delete team: %w", err)
	}
	s.logger.Info("Team deleted", zap.Int64("team_id", teamID), zap.Int64("by_user", requesterID))
	return nil
}

type LeaveTeamResult struct {
	Success     bool
	Message     string
	TeamDeleted bool
	NewLeaderID int64
}

func (s *Service) LeaveTeam(ctx context.Context, teamID int64, userID int64) (*LeaveTeamResult, error) {
	if err := s.ensureCompositionUnlocked(ctx, teamID); err != nil {
		return nil, err
	}

	member, err := s.repo.GetMember(ctx, teamID, userID)
	if err != nil {
		if errors.Is(err, ErrMemberNotFound) {
			return nil, ErrNotTeamMember
		}
		return nil, err
	}

	result := &LeaveTeamResult{Success: true, Message: "Вы успешно вышли из команды"}

	memberCount, err := s.repo.GetMemberCount(ctx, teamID)
	if err != nil {
		return nil, err
	}

	if memberCount == 1 {
		if err := s.repo.Delete(ctx, teamID); err != nil {
			return nil, fmt.Errorf("delete team: %w", err)
		}
		result.TeamDeleted = true
		result.Message = "Вы вышли из команды. Команда удалена, так как вы были последним участником."
		s.logger.Info("Last member left, team deleted", zap.Int64("team_id", teamID), zap.Int64("user_id", userID))
		return result, nil
	}

	if member.Role == RoleLeader {
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

		s.notifyBestEffort(ctx, newLeaderID,
			"Вы стали лидером команды",
			"Лидер команды вышел. Вам передано лидерство.",
			"/dashboard/team",
			"leadership_changed",
		)

		s.logger.Info("Leader left, leadership transferred", zap.Int64("team_id", teamID), zap.Int64("old_leader", userID), zap.Int64("new_leader", newLeaderID))
	}

	if err := s.repo.RemoveMember(ctx, teamID, userID); err != nil {
		return nil, fmt.Errorf("remove member: %w", err)
	}
	if member.Role != RoleLeader {
		leader, lerr := s.repo.GetTeamLeader(ctx, teamID)
		if lerr == nil && leader != nil && leader.UserID != userID {
			s.notifyBestEffort(ctx, leader.UserID,
				"Участник вышел из команды",
				fmt.Sprintf("Пользователь %d вышел из команды.", userID),
				"/dashboard/team",
				"team_member_left",
			)
		}
	}

	s.logger.Info("Member left team", zap.Int64("team_id", teamID), zap.Int64("user_id", userID))
	return result, nil
}

func (s *Service) TransferLeadership(ctx context.Context, teamID int64, currentLeaderID int64, newLeaderID int64) (*Team, []*TeamMember, error) {
	if err := s.ensureCompositionUnlocked(ctx, teamID); err != nil {
		return nil, nil, err
	}
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
		return s.GetTeam(ctx, teamID)
	}

	if err := s.repo.UpdateMemberRole(ctx, teamID, currentLeaderID, RoleMember); err != nil {
		return nil, nil, fmt.Errorf("demote current leader: %w", err)
	}
	if err := s.repo.UpdateMemberRole(ctx, teamID, newLeaderID, RoleLeader); err != nil {
		_ = s.repo.UpdateMemberRole(ctx, teamID, currentLeaderID, RoleLeader)
		return nil, nil, fmt.Errorf("promote new leader: %w", err)
	}

	s.notifyBestEffort(ctx, newLeaderID, "Вам передано лидерство", "Теперь вы лидер команды.", "/dashboard/team", "leadership_changed")
	s.notifyBestEffort(ctx, currentLeaderID,
		"Лидерство передано",
		fmt.Sprintf("Вы передали лидерство пользователю %d.", newLeaderID),
		"/dashboard/team",
		"leadership_transferred",
	)

	s.logger.Info("Leadership transferred", zap.Int64("team_id", teamID), zap.Int64("from_user", currentLeaderID), zap.Int64("to_user", newLeaderID))
	return s.GetTeam(ctx, teamID)
}

func (s *Service) AddMember(ctx context.Context, teamID int64, userID int64, role string, requesterID int64) error {
	if err := s.ensureCompositionUnlocked(ctx, teamID); err != nil {
		return err
	}
	if err := s.validateLeader(ctx, teamID, requesterID); err != nil {
		return err
	}
	if err := s.ensureTeamNotFull(ctx, teamID); err != nil {
		return err
	}

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

	if err := s.repo.AddMember(ctx, &TeamMember{TeamID: teamID, UserID: userID, Role: role}); err != nil {
		return err
	}

	s.notifyBestEffort(ctx, userID, "Вы добавлены в команду", "Вас добавили в команду. Откройте страницу команды.", "/dashboard/team", "team_member_added")
	s.notifyBestEffort(ctx, requesterID,
		"Участник добавлен в команду",
		fmt.Sprintf("Вы добавили пользователя %d в команду.", userID),
		"/dashboard/team",
		"team_member_added_leader",
	)

	return nil
}

func (s *Service) RemoveMember(ctx context.Context, teamID int64, userID int64, requesterID int64) error {
	if err := s.ensureCompositionUnlocked(ctx, teamID); err != nil {
		return err
	}
	if err := s.validateLeader(ctx, teamID, requesterID); err != nil {
		return err
	}
	if userID == requesterID {
		return errors.New("use LeaveTeam to leave the team")
	}

	member, err := s.repo.GetMember(ctx, teamID, userID)
	if err != nil {
		return err
	}
	if member.Role == RoleLeader {
		return ErrCannotRemoveLeader
	}
	if err := s.repo.RemoveMember(ctx, teamID, userID); err != nil {
		return err
	}

	s.notifyBestEffort(ctx, userID, "Вы удалены из команды", "Вас удалили из команды.", "/dashboard", "team_member_removed")
	s.notifyBestEffort(ctx, requesterID,
		"Участник удалён из команды",
		fmt.Sprintf("Вы удалили пользователя %d из команды.", userID),
		"/dashboard/team",
		"team_member_removed_leader",
	)

	return nil
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
	if time.Now().UTC().After(invite.ExpiresAt) {
		invite.Status = InviteStatusExpired
		_ = s.repo.UpdateInvite(ctx, invite)
		return errors.New("invite expired")
	}

	if accept {
		if err := s.ensureCompositionUnlocked(ctx, invite.TeamID); err != nil {
			return err
		}
		if err := s.ensureTeamNotFull(ctx, invite.TeamID); err != nil {
			return err
		}

		inTeam, _ := s.repo.IsUserInTeam(ctx, userID)
		if inTeam {
			invite.Status = InviteStatusDeclined
			_ = s.repo.UpdateInvite(ctx, invite)
			return ErrAlreadyInTeam
		}

		if err := s.repo.AddMember(ctx, &TeamMember{TeamID: invite.TeamID, UserID: userID, Role: RoleMember}); err != nil {
			return err
		}
		invite.Status = InviteStatusAccepted

		s.notifyBestEffort(ctx, invite.InviterID,
			"Приглашение принято",
			fmt.Sprintf("Пользователь %d принял приглашение в команду.", userID),
			"/dashboard/team",
			"team_invite_accepted",
		)

	} else {
		invite.Status = InviteStatusDeclined
		s.notifyBestEffort(ctx, invite.InviterID,
			"Приглашение отклонено",
			fmt.Sprintf("Пользователь %d отклонил приглашение в команду.", userID),
			"/dashboard/team",
			"team_invite_declined",
		)
	}

	return s.repo.UpdateInvite(ctx, invite)
}

func (s *Service) GetAvailableStudents(ctx context.Context, universityID int64, departmentID int64, excludeUserID int64) ([]*authv1.UserPreview, error) {
	resp, err := s.authClient.ListUsers(s.authInternalCtx(ctx), &authv1.ListUsersRequest{
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

	// Batch check
	userIDs := make([]int64, 0, len(resp.Users))
	for _, u := range resp.Users {
		userIDs = append(userIDs, u.Id)
	}
	inTeamMap, err := s.repo.GetUsersInTeams(ctx, userIDs)
	if err != nil {
		return nil, err
	}

	available := make([]*authv1.UserPreview, 0, len(resp.Users))
	for _, user := range resp.Users {
		if !inTeamMap[user.Id] {
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

func generateInviteCode() (string, error) {
	const inviteAlphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"
	const alphaLen = byte(len(inviteAlphabet)) // 62
	const maxUnbiased = 248

	b := make([]byte, 6)
	buf := make([]byte, 8)
	idx := len(buf)

	for i := 0; i < 6; {
		if idx >= len(buf) {
			if _, err := rand.Read(buf); err != nil {
				return "", err
			}
			idx = 0
		}
		rb := buf[idx]
		idx++
		if rb >= maxUnbiased {
			continue
		}
		b[i] = inviteAlphabet[rb%alphaLen]
		i++
	}
	return string(b), nil
}

func (s *Service) LockTeamComposition(ctx context.Context, teamID int64) error {
	now := time.Now().UTC()
	return s.repo.SetCompositionLocked(ctx, teamID, true, &now)
}

func (s *Service) JoinTeamByCode(ctx context.Context, inviteCode string, userID, universityID, departmentID int64, role string) (int64, error) {
	if role != "student" {
		return 0, ErrForbiddenRole
	}
	if len(inviteCode) != 6 {
		return 0, ErrInvalidInviteCode
	}

	inTeam, err := s.repo.IsUserInTeam(ctx, userID)
	if err != nil {
		return 0, err
	}
	if inTeam {
		return 0, ErrAlreadyInTeam
	}

	team, err := s.repo.GetByInviteCode(ctx, universityID, inviteCode)
	if err != nil {
		return 0, err
	}

	if team.CompositionLocked {
		return 0, ErrTeamCompositionLocked
	}
	if team.DepartmentID != departmentID {
		return 0, ErrForbiddenDepartment
	}

	cfg, err := s.workflowClient.GetTeamConfiguration(s.workflowInternalCtx(ctx), &workflowv1.GetTeamConfigurationRequest{
		DepartmentId: team.DepartmentID,
		WorkflowId:   0,
		StateId:      0,
	})
	if err == nil && cfg != nil && cfg.TeamConfig != nil && cfg.TeamConfig.MaxSize > 0 {
		cnt, err := s.repo.GetMemberCount(ctx, team.ID)
		if err != nil {
			return 0, err
		}
		if int32(cnt) >= cfg.TeamConfig.MaxSize {
			return 0, ErrTeamFull
		}
	}

	if err := s.repo.AddMember(ctx, &TeamMember{TeamID: team.ID, UserID: userID, Role: RoleMember}); err != nil {
		return 0, err
	}

	leader, lerr := s.repo.GetTeamLeader(ctx, team.ID)
	if lerr == nil && leader != nil {
		s.notifyBestEffort(ctx, leader.UserID,
			"Новый участник в команде",
			fmt.Sprintf("Пользователь %d присоединился к команде по коду.", userID),
			"/dashboard/team",
			"team_member_joined",
		)
	}
	s.notifyBestEffort(ctx, userID,
		"Вы вступили в команду",
		"Вы успешно присоединились к команде по коду.",
		"/dashboard/team",
		"team_joined_by_code",
	)

	return team.ID, nil
}

func (s *Service) RegenerateInviteCode(ctx context.Context, teamID, requesterID int64) (string, error) {
	if err := s.validateLeader(ctx, teamID, requesterID); err != nil {
		return "", err
	}
	team, err := s.repo.GetByID(ctx, teamID)
	if err != nil {
		return "", err
	}
	if team.CompositionLocked {
		return "", ErrTeamCompositionLocked
	}

	for i := 0; i < 10; i++ {
		c, err := generateInviteCode()
		if err != nil {
			return "", err
		}
		err = s.repo.UpdateInviteCode(ctx, teamID, c)
		if err == nil {
			return c, nil
		}
		if isUniqueViolation(err) {
			continue
		}
		return "", err
	}
	return "", errors.New("failed to generate unique invite code")
}

func (s *Service) ensureCompositionUnlocked(ctx context.Context, teamID int64) error {
	team, err := s.repo.GetByID(ctx, teamID)
	if err != nil {
		return err
	}
	if team.CompositionLocked {
		return ErrTeamCompositionLocked
	}
	return nil
}

func (s *Service) getMaxTeamSizeFromWorkflow(ctx context.Context, departmentID int64) (int32, error) {
	cfg, err := s.workflowClient.GetTeamConfiguration(s.workflowInternalCtx(ctx), &workflowv1.GetTeamConfigurationRequest{
		DepartmentId: departmentID,
		WorkflowId:   0,
		StateId:      0,
	})
	if err != nil {
		return 0, err
	}
	if cfg == nil || cfg.TeamConfig == nil {
		return 0, nil
	}
	return cfg.TeamConfig.MaxSize, nil
}

func (s *Service) ensureTeamNotFull(ctx context.Context, teamID int64) error {
	team, err := s.repo.GetByID(ctx, teamID)
	if err != nil {
		return err
	}

	maxSize, err := s.getMaxTeamSizeFromWorkflow(ctx, team.DepartmentID)
	if err != nil {
		// fail-open
		return nil
	}
	if maxSize <= 0 {
		return nil
	}

	cnt, err := s.repo.GetMemberCount(ctx, teamID)
	if err != nil {
		return err
	}
	if int32(cnt) >= maxSize {
		return ErrTeamFull
	}
	return nil
}
func (s *Service) UpdateTeamFull(ctx context.Context, teamID int64, name string, requesterID int64) (*Team, []*TeamMember, error) {
	team, err := s.UpdateTeam(ctx, teamID, name, requesterID)
	if err != nil {
		return nil, nil, err
	}
	members, err := s.repo.GetMembers(ctx, team.ID)
	if err != nil {
		return nil, nil, err
	}
	return team, members, nil
}
func (s *Service) GetPendingInvitesWithTeams(ctx context.Context, userID int64) ([]*TeamInviteWithTeam, error) {
	return s.repo.GetPendingInvitesWithTeams(ctx, userID)
}
func (s *Service) IsUserInSpecificTeam(ctx context.Context, userID, teamID int64) (bool, error) {
	return s.repo.IsUserInSpecificTeam(ctx, userID, teamID)
}
func (s *Service) GetMembersByTeamIDs(ctx context.Context, teamIDs []int64) (map[int64][]*TeamMember, error) {
	return s.repo.GetMembersByTeamIDs(ctx, teamIDs)
}
