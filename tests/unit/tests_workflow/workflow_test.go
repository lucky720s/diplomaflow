package tests_workflow

import (
	"context"
	"testing"

	"github.com/lucky720s/diplomaflow/internal/workflow"
	workflowv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/workflow/v1"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestService_CreateWorkflow(t *testing.T) {
	repo := new(MockRepository)
	svc := workflow.NewService(repo)

	repo.On("CreateWorkflow", mock.Anything, mock.MatchedBy(func(w *workflow.Workflow) bool {
		return w.Name == "Diploma" && w.DepartmentID == 1
	})).Return(nil)

	res, err := svc.CreateWorkflow(context.Background(), "Diploma", 1)

	require.NoError(t, err)
	require.Equal(t, int64(100), res.ID)
}

func TestHandler_CreateWorkflow(t *testing.T) {
	mockSvc := new(MockWorkflowService)
	handler := workflow.NewHandler(mockSvc)

	wf := &workflow.Workflow{ID: 10, Name: "Diploma", DepartmentID: 5}
	mockSvc.On("CreateWorkflow", mock.Anything, "Diploma", int64(5)).Return(wf, nil)

	req := &workflowv1.CreateWorkflowRequest{Name: "Diploma", DepartmentId: 5}
	resp, err := handler.CreateWorkflow(context.Background(), req)

	require.NoError(t, err)
	require.Equal(t, int64(10), resp.Id)
}

func TestHandler_GetWorkflow(t *testing.T) {
	mockSvc := new(MockWorkflowService)
	handler := workflow.NewHandler(mockSvc)

	wf := &workflow.Workflow{ID: 10, Name: "Diploma"}
	mockSvc.On("GetWorkflow", mock.Anything, int64(10)).Return(wf, nil)

	req := &workflowv1.GetWorkflowRequest{Criteria: &workflowv1.GetWorkflowRequest_WorkflowId{WorkflowId: 10}}
	resp, err := handler.GetWorkflow(context.Background(), req)

	require.NoError(t, err)
	require.Equal(t, "Diploma", resp.Name)
}
