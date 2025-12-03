package project

import (
	"context"
	"fmt"
	"time"

	teamv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/team/v1"
	workflowv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/workflow/v1"
)

type Service struct {
	repo           Repository
	workflowClient workflowv1.WorkflowServiceClient
	teamClient     teamv1.TeamServiceClient
}

func NewService(repo Repository, wfClient workflowv1.WorkflowServiceClient, tClient teamv1.TeamServiceClient) *Service {
	return &Service{
		repo:           repo,
		workflowClient: wfClient,
		teamClient:     tClient,
	}
}

func (s *Service) CreateProject(ctx context.Context, title, description string, studentID int64, workflowName string) (*Project, error) {
	wfResp, err := s.workflowClient.GetWorkflow(ctx, &workflowv1.GetWorkflowRequest{
		Criteria: &workflowv1.GetWorkflowRequest_Name{
			Name: workflowName,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get workflow: %w", err)
	}
	if len(wfResp.Steps) == 0 {
		return nil, fmt.Errorf("workflow %s has no steps", workflowName)
	}

	project := &Project{
		Title:        title,
		Description:  description,
		StudentID:    studentID,
		WorkflowName: workflowName,
		CurrentState: wfResp.Steps[0].Name,
		Status:       "CREATED",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if err := s.repo.Create(ctx, project); err != nil {
		return nil, fmt.Errorf("failed to create project in db: %w", err)
	}

	teamResp, err := s.teamClient.CreateTeam(ctx, &teamv1.CreateTeamRequest{
		Name:      fmt.Sprintf("Team: %s", title),
		ProjectId: int64(project.ID),
		MemberIds: []int64{studentID},
	})

	if err != nil {
		if delErr := s.repo.Delete(ctx, project.ID); delErr != nil {
			fmt.Printf("CRITICAL: Failed to rollback project %d after team creation failure: %v\n", project.ID, delErr)
		}
		return nil, fmt.Errorf("failed to create team, project rolled back: %w", err)
	}

	project.TeamID = teamResp.TeamId
	if err := s.repo.Update(ctx, project); err != nil {
		return nil, fmt.Errorf("failed to link team to project: %w", err)
	}

	return project, nil
}

func (s *Service) GetProject(ctx context.Context, id uint64) (*Project, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *Service) GetStudentProjects(ctx context.Context, studentID int64) ([]*Project, error) {
	return s.repo.ListByStudent(ctx, studentID)
}
