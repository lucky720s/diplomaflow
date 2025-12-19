package tests_workflow

import (
	"context"
	"testing"

	"github.com/lucky720s/diplomaflow/internal/workflow"
	workflowv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/workflow/v1"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestService_CreateState(t *testing.T) {
	repo := new(MockRepository)
	svc := workflow.NewService(repo)

	repo.On("CreateState", mock.Anything, mock.MatchedBy(func(s *workflow.State) bool {
		return s.Name == "Start" && s.WorkflowID == 100
	})).Return(nil)

	cfg, _ := structpb.NewStruct(map[string]interface{}{})
	req := &workflowv1.CreateStateRequest{
		WorkflowId: 100,
		Name:       "Start",
		Type:       workflowv1.StateType_TEAM_FORMATION,
		Config:     cfg,
	}

	res, err := svc.CreateState(context.Background(), req)

	require.NoError(t, err)
	require.Equal(t, int64(200), res.ID)
}

func TestHandler_GetState(t *testing.T) {
	mockSvc := new(MockWorkflowService)
	handler := workflow.NewHandler(mockSvc)

	state := &workflow.State{ID: 50, Name: "Draft", Config: []byte("{}"), Type: "IN_PROGRESS"}
	mockSvc.On("GetState", mock.Anything, int64(50)).Return(state, nil)

	req := &workflowv1.GetStateRequest{StateId: 50}
	resp, err := handler.GetState(context.Background(), req)

	require.NoError(t, err)
	require.Equal(t, "Draft", resp.Name)
}
