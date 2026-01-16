package team

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	authv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/auth/v1"
	notificationv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/notification/v1"
	workflowv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/workflow/v1"
	"go.uber.org/zap"
)

type Service struct {
	repo           Repository
	authClient     authv1.AuthServiceClient
	notifClient    notificationv1.NotificationServiceClient
	workflowClient workflowv1.WorkflowServiceClient
	logger         *zap.Logger
}

func NewService(
	repo Repository,
	authClient authv1.AuthServiceClient,
	notifClient notificationv1.NotificationServiceClient,
	workflowClient workflowv1.WorkflowServiceClient,
	logger *zap.Logger,
) *Service {
	return &Service{
		repo:           repo,
		authClient:     authClient,
		notifClient:    notifClient,
		workflowClient: workflowClient,
		logger:         logger,
	}
}

// ===== TeamConfigResult =====

type TeamConfigResult struct {
	MinSize          int32
	MaxSize          int32
	AllowSolo        bool
	InviteExpireDays int32
}

func (c TeamConfigResult) normalize() TeamConfigResult {
	out := c

	// defaults
	if out.MinSize <= 0 {
		out.MinSize = 1
	}
	if out.MaxSize <= 0 {
		out.MaxSize = out.MinSize
	}
	if out.MaxSize < out.MinSize {
		out.MaxSize = out.MinSize
	}
	if out.InviteExpireDays <= 0 {
		out.InviteExpireDays = 3
	}

	// если solo запрещён — минимальный размер не может быть 1
	if !out.AllowSolo && out.MinSize < 2 {
		out.MinSize = 2
	}
	if out.MaxSize < out.MinSize {
		out.MaxSize = out.MinSize
	}

	return out
}

// ===== CreateTeam =====

func (s *Service) CreateTeam(
	ctx context.Context,
	name string,
	projectID *int64,
	memberIDs []int64,
	leaderID int64,
	departmentID int64,
) (int64, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return 0, fmt.Errorf("name is required")
	}
	if leaderID <= 0 {
		return 0, fmt.Errorf("leader_id is required")
	}

	// 1) config (workflow-driven) + нормализация
	cfg, err := s.getTeamConfiguration(ctx, departmentID)
	if err != nil {
		s.logger.Warn("failed to get team config from workflow, using defaults", zap.Error(err))
		cfg = &TeamConfigResult{
			MinSize:          1,
			MaxSize:          3,
			AllowSolo:        true,
			InviteExpireDays: 3,
		}
	}
	c := cfg.normalize()

	// 2) нормализуем memberIDs: убираем leader, 0/neg, дубликаты
	uniqueMembers := make(map[int64]struct{}, len(memberIDs))
	cleanMembers := make([]int64, 0, len(memberIDs))
	for _, id := range memberIDs {
		if id <= 0 || id == leaderID {
			continue
		}
		if _, ok := uniqueMembers[id]; ok {
			continue
		}
		uniqueMembers[id] = struct{}{}
		cleanMembers = append(cleanMembers, id)
	}
	// стабильность логов/поведения
	sort.Slice(cleanMembers, func(i, j int) bool { return cleanMembers[i] < cleanMembers[j] })

	totalMembers := int32(1 + len(cleanMembers)) // leader + unique members

	// 3) валидация min/max
	if totalMembers < c.MinSize {
		return 0, fmt.Errorf("minimum team size is %d", c.MinSize)
	}
	if totalMembers > c.MaxSize {
		return 0, fmt.Errorf("team size exceeds maximum allowed (%d)", c.MaxSize)
	}

	// 4) бизнес-инварианты: пользователь не может быть в двух командах
	inTeam, err := s.repo.IsUserInAnyTeam(ctx, leaderID)
	if err != nil {
		return 0, fmt.Errorf("failed to validate leader team membership: %w", err)
	}
	if inTeam {
		return 0, fmt.Errorf("leader already belongs to a team")
	}
	for _, memberID := range cleanMembers {
		alreadyInTeam, err := s.repo.IsUserInAnyTeam(ctx, memberID)
		if err != nil {
			return 0, fmt.Errorf("failed to validate member team membership (user_id=%d): %w", memberID, err)
		}
		if alreadyInTeam {
			return 0, fmt.Errorf("user %d already belongs to a team", memberID)
		}
	}

	// 5) создаём Team + Invites (атомарно, в транзакции репозитория)
	now := time.Now().UTC()

	team := &Team{
		Name:      name,
		ProjectID: projectID,
		Members: []TeamMember{
			{
				UserID:    leaderID,
				Role:      "leader",
				CreatedAt: now,
			},
		},
		CreatedAt: now,
		UpdatedAt: now,
	}

	invites := make([]*TeamInvite, 0, len(cleanMembers))
	for _, memberID := range cleanMembers {
		invites = append(invites, &TeamInvite{
			UserID:    memberID,
			InviterID: leaderID,
			Status:    "PENDING",
			ExpiresAt: now.AddDate(0, 0, int(c.InviteExpireDays)),
			CreatedAt: now,
			UpdatedAt: now,
		})
	}

	if err := s.repo.CreateTeamWithInvites(ctx, team, invites); err != nil {
		return 0, fmt.Errorf("failed to create team: %w", err)
	}

	// 6) уведомления (best-effort, не роняют создание)
	for _, memberID := range cleanMembers {
		_, notifErr := s.notifClient.SendNotification(ctx, &notificationv1.SendNotificationRequest{
			UserId:  memberID,
			Title:   "Приглашение в команду",
			Message: fmt.Sprintf("Вас пригласили в команду '%s'. Приглашение действительно %d дней.", name, c.InviteExpireDays),
			Link:    "/teams/invites",
			Type:    "TEAM_INVITE",
		})
		if notifErr != nil {
			s.logger.Warn("failed to send invite notification",
				zap.Int64("user_id", memberID),
				zap.Int64("team_id", int64(team.ID)),
				zap.Error(notifErr),
			)
		}
	}

	s.logger.Info("team created",
		zap.Int64("team_id", int64(team.ID)),
		zap.String("name", name),
		zap.Int64("leader_id", leaderID),
		zap.Int("members_total", int(totalMembers)),
		zap.Int("invites", len(invites)),
	)

	return int64(team.ID), nil
}

// ===== Workflow config =====

func (s *Service) getTeamConfiguration(ctx context.Context, departmentID int64) (*TeamConfigResult, error) {
	if s.workflowClient == nil {
		return nil, errors.New("workflow client not configured")
	}
	if departmentID <= 0 {
		return nil, errors.New("department_id is required to resolve team configuration")
	}

	// 1) prefer explicit runtime API
	resp, err := s.workflowClient.GetTeamConfiguration(ctx, &workflowv1.GetTeamConfigurationRequest{
		DepartmentId: departmentID,
		WorkflowId:   0, // active workflow
		StateId:      0, // initial state
	})
	if err == nil && resp != nil {
		out := &TeamConfigResult{
			MinSize:          1,
			MaxSize:          3,
			AllowSolo:        true,
			InviteExpireDays: resp.InviteExpireDays,
		}
		if resp.TeamConfig != nil {
			out.MinSize = resp.TeamConfig.MinSize
			out.MaxSize = resp.TeamConfig.MaxSize
			out.AllowSolo = resp.TeamConfig.AllowSolo
		}
		n := out.normalize()
		return &n, nil
	}

	// 2) fallback: active workflow settings JSON
	wf, wfErr := s.workflowClient.GetActiveWorkflowByDepartment(ctx, &workflowv1.GetActiveWorkflowByDepartmentRequest{
		DepartmentId: departmentID,
	})
	if wfErr != nil {
		return nil, fmt.Errorf("failed to get team configuration: %w (fallback wf also failed: %v)", err, wfErr)
	}
	if wf.Settings == nil {
		return nil, fmt.Errorf("workflow settings are empty")
	}

	settings := wf.Settings.AsMap()
	out := TeamConfigResult{
		MinSize:          getInt32(settings, "min_team_size", 1),
		MaxSize:          getInt32(settings, "max_team_size", 3),
		AllowSolo:        getBool(settings, "allow_solo_project", true),
		InviteExpireDays: getInt32(settings, "invite_expire_days", 3),
	}
	out = out.normalize()
	return &out, nil
}

// ===== Other service methods =====

func (s *Service) GetTeam(ctx context.Context, teamID uint64) (*Team, error) {
	return s.repo.GetByID(ctx, teamID)
}

func (s *Service) GetAvailableStudents(ctx context.Context, universityID int64, excludeUserID int64) ([]*StudentPreview, error) {
	resp, err := s.authClient.ListUsers(ctx, &authv1.ListUsersRequest{
		UniversityId:  universityID,
		Role:          "student",
		Page:          1,
		PageSize:      100,
		ExcludeUserId: excludeUserID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch students from auth service: %w", err)
	}

	var students []*StudentPreview
	for _, u := range resp.Users {
		students = append(students, &StudentPreview{
			ID:        u.Id,
			FullName:  u.FirstName + " " + u.LastName,
			Email:     u.Email,
			FirstName: u.FirstName,
			LastName:  u.LastName,
		})
	}
	return students, nil
}

func (s *Service) CreateTeamForProject(ctx context.Context, projectID int64, studentID int64) error {
	pID := projectID
	now := time.Now().UTC()

	// защитимся от повторного создания лидером, который уже в команде
	inTeam, err := s.repo.IsUserInAnyTeam(ctx, studentID)
	if err != nil {
		return fmt.Errorf("failed to validate student team membership: %w", err)
	}
	if inTeam {
		return fmt.Errorf("student already belongs to a team")
	}

	team := &Team{
		Name:      fmt.Sprintf("Team Project %d", projectID),
		ProjectID: &pID,
		Members: []TeamMember{
			{
				UserID:    studentID,
				Role:      "leader",
				CreatedAt: now,
			},
		},
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := s.repo.Create(ctx, team); err != nil {
		return fmt.Errorf("failed to create team for project: %w", err)
	}
	return nil
}

func (s *Service) AssignProject(ctx context.Context, teamID int64, projectID int64) error {
	team, err := s.repo.GetByID(ctx, uint64(teamID))
	if err != nil {
		return err
	}
	team.ProjectID = &projectID
	team.UpdatedAt = time.Now().UTC()
	if err := s.repo.Update(ctx, team); err != nil {
		return fmt.Errorf("failed to assign project: %w", err)
	}
	return nil
}

func (s *Service) GetMyInvites(ctx context.Context, userID int64) ([]*TeamInvite, error) {
	return s.repo.GetInvitesByUserID(ctx, userID)
}

func (s *Service) RespondToInvite(ctx context.Context, inviteID int64, userID int64, accept bool) error {
	invite, err := s.repo.GetInviteByID(ctx, uint64(inviteID))
	if err != nil {
		return fmt.Errorf("invite not found")
	}
	if invite.UserID != userID {
		return fmt.Errorf("permission denied")
	}
	if invite.Status != "PENDING" {
		return fmt.Errorf("invite is already %s", invite.Status)
	}
	if time.Now().UTC().After(invite.ExpiresAt.UTC()) {
		_ = s.repo.UpdateInviteStatus(ctx, uint64(inviteID), "EXPIRED")
		return fmt.Errorf("invite expired")
	}
	if !accept {
		return s.repo.UpdateInviteStatus(ctx, uint64(inviteID), "DECLINED")
	}

	inTeam, err := s.repo.IsUserInAnyTeam(ctx, userID)
	if err != nil {
		return err
	}
	if inTeam {
		return fmt.Errorf("user already belongs to a team")
	}

	member := &TeamMember{
		TeamID:    uint(invite.TeamID),
		UserID:    userID,
		Role:      "member",
		CreatedAt: time.Now().UTC(),
	}
	if err := s.repo.AddMember(ctx, member); err != nil {
		return err
	}
	if err := s.repo.UpdateInviteStatus(ctx, uint64(inviteID), "ACCEPTED"); err != nil {
		return err
	}
	if err := s.repo.DeletePendingInvitesForUser(ctx, userID); err != nil {
		s.logger.Warn("failed to delete pending invites", zap.Error(err))
	}
	return nil
}

func (s *Service) GetMyTeam(ctx context.Context, userID int64) (*Team, string, int64, error) {
	team, role, err := s.repo.GetTeamByUserID(ctx, userID)
	if err != nil {
		return nil, "", 0, err
	}
	if team == nil {
		return nil, "", 0, nil
	}
	pendingCount, _ := s.repo.CountPendingInvitesByTeam(ctx, uint64(team.ID))
	return team, role, pendingCount, nil
}

func (s *Service) ListTeams(ctx context.Context, departmentID, projectID int64, page, pageSize int32) ([]*Team, int64, error) {
	if pageSize <= 0 {
		pageSize = 20
	}
	if page <= 0 {
		page = 1
	}
	offset := (page - 1) * pageSize
	return s.repo.List(ctx, departmentID, projectID, int(pageSize), int(offset))
}

func (s *Service) UpdateTeam(ctx context.Context, teamID uint64, name string) (*Team, error) {
	team, err := s.repo.GetByID(ctx, teamID)
	if err != nil {
		return nil, fmt.Errorf("team not found: %w", err)
	}
	if v := strings.TrimSpace(name); v != "" {
		team.Name = v
	}
	team.UpdatedAt = time.Now().UTC()
	if err := s.repo.Update(ctx, team); err != nil {
		return nil, fmt.Errorf("failed to update team: %w", err)
	}
	return team, nil
}

func (s *Service) DeleteTeam(ctx context.Context, teamID uint64) error {
	team, err := s.repo.GetByID(ctx, teamID)
	if err != nil {
		return fmt.Errorf("team not found: %w", err)
	}
	for _, member := range team.Members {
		_, _ = s.notifClient.SendNotification(ctx, &notificationv1.SendNotificationRequest{
			UserId:  member.UserID,
			Title:   "Команда удалена",
			Message: fmt.Sprintf("Команда '%s' была удалена", team.Name),
			Type:    "TEAM_DELETED",
		})
	}
	return s.repo.Delete(ctx, teamID)
}

func (s *Service) AddMember(ctx context.Context, teamID uint64, userID int64, role string) error {
	team, err := s.repo.GetByID(ctx, teamID)
	if err != nil {
		return fmt.Errorf("team not found: %w", err)
	}
	inTeam, err := s.repo.IsUserInAnyTeam(ctx, userID)
	if err != nil {
		return err
	}
	if inTeam {
		return fmt.Errorf("user already belongs to a team")
	}
	if strings.TrimSpace(role) == "" {
		role = "member"
	}

	member := &TeamMember{
		TeamID:    uint(teamID),
		UserID:    userID,
		Role:      role,
		CreatedAt: time.Now().UTC(),
	}
	if err := s.repo.AddMember(ctx, member); err != nil {
		return fmt.Errorf("failed to add member: %w", err)
	}
	_ = s.repo.DeletePendingInvitesForUser(ctx, userID)

	_, _ = s.notifClient.SendNotification(ctx, &notificationv1.SendNotificationRequest{
		UserId:  userID,
		Title:   "Добавлены в команду",
		Message: fmt.Sprintf("Вы были добавлены в команду '%s'", team.Name),
		Link:    "/teams/my",
		Type:    "TEAM_MEMBER_ADDED",
	})
	return nil
}

func (s *Service) RemoveMember(ctx context.Context, teamID uint64, userID int64) error {
	team, err := s.repo.GetByID(ctx, teamID)
	if err != nil {
		return fmt.Errorf("team not found: %w", err)
	}
	member, err := s.repo.GetMember(ctx, teamID, userID)
	if err != nil {
		return err
	}
	if member == nil {
		return fmt.Errorf("user is not a member of this team")
	}
	if member.Role == "leader" && len(team.Members) > 1 {
		return fmt.Errorf("cannot remove leader while team has other members")
	}
	if err := s.repo.RemoveMember(ctx, teamID, userID); err != nil {
		return fmt.Errorf("failed to remove member: %w", err)
	}
	_, _ = s.notifClient.SendNotification(ctx, &notificationv1.SendNotificationRequest{
		UserId:  userID,
		Title:   "Удалены из команды",
		Message: fmt.Sprintf("Вы были удалены из команды '%s'", team.Name),
		Type:    "TEAM_MEMBER_REMOVED",
	})
	return nil
}

// ===== Helpers =====

func getInt32(m map[string]interface{}, key string, def int32) int32 {
	if v, ok := m[key]; ok {
		switch val := v.(type) {
		case float64:
			return int32(val)
		case float32:
			return int32(val)
		case int:
			return int32(val)
		case int32:
			return val
		case int64:
			return int32(val)
		}
	}
	return def
}

func getBool(m map[string]interface{}, key string, def bool) bool {
	if v, ok := m[key]; ok {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return def
}
