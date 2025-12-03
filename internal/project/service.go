package project

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/lucky720s/diplomaflow/pkg/broker"
	workflowv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/workflow/v1"
)

type Service struct {
	repo           Repository
	workflowClient workflowv1.WorkflowServiceClient
	kafkaProducer  *broker.Producer
}

func NewService(repo Repository, wfClient workflowv1.WorkflowServiceClient, producer *broker.Producer) *Service {
	return &Service{
		repo:           repo,
		workflowClient: wfClient,
		kafkaProducer:  producer,
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

	payload := map[string]interface{}{
		"project_id": project.ID,
		"title":      title,
		"student_id": studentID,
	}

	if err := s.kafkaProducer.Publish("project-events", "ProjectCreated", payload); err != nil {
		log.Printf("CRITICAL: Failed to publish ProjectCreated event: %v", err)
	}

	return project, nil
}

func (s *Service) GetProject(ctx context.Context, id uint64) (*Project, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *Service) GetStudentProjects(ctx context.Context, studentID int64) ([]*Project, error) {
	return s.repo.ListByStudent(ctx, studentID)
}
