package tests_project

import (
	"context"
	"testing"

	"github.com/lucky720s/diplomaflow/internal/project"
	projectv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/project/v1"
	workflowv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/workflow/v1"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestService_CreateProject(t *testing.T) {
	repo := new(MockRepository)
	wfClient := new(MockWorkflowClient)

	svc := project.NewService(repo, wfClient, nil, nil, zap.NewNop())

	wf := &workflowv1.Workflow{
		Id: 10, Name: "Diploma",
		Steps: []*workflowv1.State{{Id: 1, Name: "Start"}},
	}
	wfClient.On("GetActiveWorkflowByDepartment", mock.Anything, mock.Anything).Return(wf, nil)

	repo.On("CreateWithOutbox", mock.Anything, mock.MatchedBy(func(p *project.Project) bool {
		return p.Title == "My Project" && p.CurrentState == "Start"
	}), "ProjectCreated", "project-events", mock.Anything).Return(nil)

	req := &projectv1.CreateProjectRequest{
		Title: "My Project", StudentId: 1, DepartmentId: 5, UniversityId: 1,
	}

	res, err := svc.CreateProject(context.Background(), req)

	require.NoError(t, err)
	require.Equal(t, int64(100), res.ProjectId)
	repo.AssertExpectations(t)
}

func TestHandler_CreateProject(t *testing.T) {
	mockSvc := new(MockProjectService)
	handler := project.NewHandler(mockSvc)

	respMock := &projectv1.CreateProjectResponse{ProjectId: 555}

	mockSvc.On("CreateProject", mock.Anything, mock.Anything).Return(respMock, nil)

	req := &projectv1.CreateProjectRequest{Title: "P1", StudentId: 1, DepartmentId: 1, UniversityId: 1}
	resp, err := handler.CreateProject(context.Background(), req)

	require.NoError(t, err)
	require.Equal(t, int64(555), resp.ProjectId)
}
