package team

import (
	"context"
	"fmt"
	"time"

	authv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/auth/v1"
)

type Service struct {
	repo       Repository
	authClient authv1.AuthServiceClient
}

func NewService(repo Repository, authClient authv1.AuthServiceClient) *Service {
	return &Service{
		repo:       repo,
		authClient: authClient,
	}
}

func (s *Service) CreateTeam(ctx context.Context, name string, projectID *int64, memberIDs []int64) (int64, error) {
	members := make([]TeamMember, len(memberIDs))
	for i, uid := range memberIDs {
		role := "member"
		if i == 0 {
			role = "leader"
		}
		members[i] = TeamMember{
			UserID:    uid,
			Role:      role,
			CreatedAt: time.Now(),
		}
	}

	team := &Team{
		Name:      name,
		ProjectID: projectID,
		Members:   members,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := s.repo.Create(ctx, team); err != nil {
		return 0, fmt.Errorf("failed to create team: %w", err)
	}

	return int64(team.ID), nil
}

func (s *Service) GetTeam(ctx context.Context, teamID uint64) (*Team, error) {
	return s.repo.GetByID(ctx, teamID)
}

func (s *Service) GetAvailableStudents(ctx context.Context, universityID int64) ([]*authv1.UserPreview, error) {
	resp, err := s.authClient.ListUsers(ctx, &authv1.ListUsersRequest{
		UniversityId: universityID,
		Role:         "student",
		Page:         1,
		PageSize:     100,
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
