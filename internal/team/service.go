// internal/team/service.go — исправить конструктор и добавить workflow client

package team

import (
	"context"
	"errors"
	"fmt"
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
	workflowClient workflowv1.WorkflowServiceClient // ✅ Добавлено
	logger         *zap.Logger
}

func NewService(
	repo Repository,
	authClient authv1.AuthServiceClient,
	notifClient notificationv1.NotificationServiceClient,
	workflowClient workflowv1.WorkflowServiceClient, // ✅ Добавлено
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

// CreateTeam с получением конфигурации от Workflow Service
func (s *Service) CreateTeam(ctx context.Context, name string, projectID *int64, memberIDs []int64, leaderID int64, departmentID int64) (int64, error) {
	// ✅ Получаем конфигурацию команды от Workflow Service
	teamConfig, err := s.getTeamConfiguration(ctx, departmentID)
	if err != nil {
		s.logger.Warn("Failed to get team config from workflow, using defaults", zap.Error(err))
		teamConfig = &TeamConfigResult{
			MinSize:          1,
			MaxSize:          3,
			AllowSolo:        true,
			InviteExpireDays: 3,
		}
	}

	// ✅ Валидация размера команды
	totalMembers := 1 + len(memberIDs) // leader + members
	if !teamConfig.AllowSolo && totalMembers < 2 {
		return 0, fmt.Errorf("solo projects are not allowed, minimum team size is %d", teamConfig.MinSize)
	}
	if int32(totalMembers) > teamConfig.MaxSize {
		return 0, fmt.Errorf("team size exceeds maximum allowed (%d)", teamConfig.MaxSize)
	}

	team := &Team{
		Name:      name,
		ProjectID: projectID,
		Members: []TeamMember{
			{UserID: leaderID, Role: "leader", CreatedAt: time.Now()},
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	var invites []*TeamInvite
	for _, memberID := range memberIDs {
		if memberID == leaderID {
			continue
		}
		invites = append(invites, &TeamInvite{
			UserID:    memberID,
			InviterID: leaderID,
			Status:    "PENDING",
			ExpiresAt: time.Now().AddDate(0, 0, int(teamConfig.InviteExpireDays)),
			CreatedAt: time.Now(),
		})
	}

	if err := s.repo.CreateTeamWithInvites(ctx, team, invites); err != nil {
		return 0, fmt.Errorf("failed to create team: %w", err)
	}

	// Отправляем уведомления
	for _, memberID := range memberIDs {
		if memberID == leaderID {
			continue
		}
		_, notifErr := s.notifClient.SendNotification(ctx, &notificationv1.SendNotificationRequest{
			UserId:  memberID,
			Title:   "Приглашение в команду",
			Message: fmt.Sprintf("Вас пригласили в команду '%s'. Приглашение действительно %d дней.", name, teamConfig.InviteExpireDays),
			Link:    "/teams/invites",
			Type:    "TEAM_INVITE",
		})
		if notifErr != nil {
			s.logger.Error("Failed to send notification", zap.Int64("user_id", memberID), zap.Error(notifErr))
		}
	}

	s.logger.Info("Team created",
		zap.Int64("team_id", int64(team.ID)),
		zap.String("name", name),
		zap.Int("members", len(memberIDs)+1))

	return int64(team.ID), nil
}

// getTeamConfiguration получает конфигурацию команды от Workflow Service
func (s *Service) getTeamConfiguration(ctx context.Context, departmentID int64) (*TeamConfigResult, error) {
	if s.workflowClient == nil {
		return nil, errors.New("workflow client not configured")
	}

	resp, err := s.workflowClient.GetTeamConfiguration(ctx, &workflowv1.GetTeamConfigurationRequest{
		DepartmentId: departmentID,
		WorkflowId:   0, // Use active workflow
		StateId:      0, // Use initial state
	})
	if err != nil {
		// Fallback: пробуем получить активный workflow
		wfResp, wfErr := s.workflowClient.GetActiveWorkflowByDepartment(ctx, &workflowv1.GetActiveWorkflowByDepartmentRequest{
			DepartmentId: departmentID,
		})
		if wfErr != nil {
			return nil, wfErr
		}

		// Парсим настройки из workflow
		if wfResp.Settings != nil {
			settings := wfResp.Settings.AsMap()
			return &TeamConfigResult{
				MinSize:          getInt32(settings, "min_team_size", 1),
				MaxSize:          getInt32(settings, "max_team_size", 3),
				AllowSolo:        getBool(settings, "allow_solo_project", true),
				InviteExpireDays: getInt32(settings, "invite_expire_days", 3),
			}, nil
		}
		return nil, fmt.Errorf("no workflow settings found")
	}

	result := &TeamConfigResult{
		MinSize:          1,
		MaxSize:          3,
		AllowSolo:        true,
		InviteExpireDays: resp.InviteExpireDays,
	}

	if resp.TeamConfig != nil {
		result.MinSize = resp.TeamConfig.MinSize
		result.MaxSize = resp.TeamConfig.MaxSize
		result.AllowSolo = resp.TeamConfig.AllowSolo
	}

	return result, nil
}

// TeamConfigResult результат получения конфигурации команды
type TeamConfigResult struct {
	MinSize          int32
	MaxSize          int32
	AllowSolo        bool
	InviteExpireDays int32
}

// Helper functions
func getInt32(m map[string]interface{}, key string, def int32) int32 {
	if v, ok := m[key]; ok {
		switch val := v.(type) {
		case float64:
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

func (s *Service) GetTeam(ctx context.Context, teamID uint64) (*Team, error) {
	return s.repo.GetByID(ctx, teamID)
}

// internal/team/service.go — исправить возвращаемый тип
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
			ID:       u.Id,
			FullName: u.FirstName + " " + u.LastName,
			Email:    u.Email,
		})
	}
	return students, nil
}

func (s *Service) CreateTeamForProject(ctx context.Context, projectID int64, studentID int64) error {
	pID := projectID

	team := &Team{
		Name:      fmt.Sprintf("Team Project %d", projectID),
		ProjectID: &pID,
		Members: []TeamMember{
			{
				UserID:    studentID,
				Role:      "leader",
				CreatedAt: time.Now(),
			},
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
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
		CreatedAt: time.Now(),
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

	if name != "" {
		team.Name = name
	}
	team.UpdatedAt = time.Now()

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

	if role == "" {
		role = "member"
	}

	member := &TeamMember{
		TeamID:    uint(teamID),
		UserID:    userID,
		Role:      role,
		CreatedAt: time.Now(),
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
