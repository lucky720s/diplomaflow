package team

import (
	"context"
	"fmt"
	"time"

	authv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/auth/v1"
	notificationv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/notification/v1"
	"go.uber.org/zap"
)

type Service struct {
	repo        Repository
	authClient  authv1.AuthServiceClient
	notifClient notificationv1.NotificationServiceClient
	logger      *zap.Logger
}

func NewService(repo Repository, authClient authv1.AuthServiceClient, notifClient notificationv1.NotificationServiceClient, logger *zap.Logger) *Service {
	return &Service{
		repo:        repo,
		authClient:  authClient,
		notifClient: notifClient,
		logger:      logger,
	}
}

func (s *Service) CreateTeam(ctx context.Context, name string, projectID *int64, memberIDs []int64, leaderID int64) (int64, error) {
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
			CreatedAt: time.Now(),
		})
	}
	if err := s.repo.CreateTeamWithInvites(ctx, team, invites); err != nil {
		return 0, fmt.Errorf("failed to create team: %w", err)
	}
	for _, memberID := range memberIDs {
		if memberID == leaderID {
			continue
		}
		_, err := s.notifClient.SendNotification(ctx, &notificationv1.SendNotificationRequest{
			UserId:  memberID,
			Title:   "Приглашение в команду",
			Message: fmt.Sprintf("Вас пригласили в команду '%s'", name),
			Link:    "/teams/invites",
			Type:    "TEAM_INVITE",
		})
		if err != nil {
			s.logger.Error("Failed to send notification", zap.Int64("user_id", memberID), zap.Error(err))
		}
	}

	return int64(team.ID), nil
}

func (s *Service) GetTeam(ctx context.Context, teamID uint64) (*Team, error) {
	return s.repo.GetByID(ctx, teamID)
}

func (s *Service) GetAvailableStudents(ctx context.Context, universityID int64, excludeUserID int64) ([]*authv1.UserPreview, error) {
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

	return resp.Users, nil
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
		TeamID:    invite.TeamID,
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
	}

	return nil
}
