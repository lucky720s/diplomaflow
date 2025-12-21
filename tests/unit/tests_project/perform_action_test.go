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
	"google.golang.org/protobuf/types/known/structpb"
)

func TestService_PerformAction(t *testing.T) {
	repo := new(MockRepository)
	wfClient := new(MockWorkflowClient)

	registry := project.NewProcessorRegistry()
	registry.Register("submit_doc", &project.UploadTaskHandler{})

	svc := project.NewService(repo, wfClient, registry, nil, zap.NewNop())

	proj := &project.Project{ID: 1, CurrentStepID: "10", CurrentState: "Start"}
	repo.On("GetByID", mock.Anything, uint64(1)).Return(proj, nil)

	state := &workflowv1.State{Id: 10, Config: nil}
	wfClient.On("GetState", mock.Anything, mock.Anything).Return(state, nil)

	nextState := &workflowv1.State{Id: 20, Name: "Review"}
	wfClient.On("GetNextState", mock.Anything, mock.Anything).Return(nextState, nil)

	repo.On("AddHistory", mock.Anything, mock.Anything).Return(nil)
	repo.On("Update", mock.Anything, mock.MatchedBy(func(p *project.Project) bool {
		return p.CurrentState == "Review"
	})).Return(nil)

	payload := map[string]interface{}{"file_id": "file123"}

	res, err := svc.PerformAction(context.Background(), 1, "submit_doc", payload)

	require.NoError(t, err)
	require.Equal(t, "Review", res.CurrentState)
}

func TestHandler_PerformAction(t *testing.T) {
	mockSvc := new(MockProjectService)
	handler := project.NewHandler(mockSvc)

	proj := &project.Project{ID: 1, CurrentState: "Review"}
	mockSvc.On("PerformAction", mock.Anything, int64(1), "submit", mock.Anything).Return(proj, nil)

	pl, _ := structpb.NewStruct(map[string]interface{}{})
	req := &projectv1.PerformActionRequest{ProjectId: 1, ActionName: "submit", Payload: pl}

	resp, err := handler.PerformAction(context.Background(), req)

	require.NoError(t, err)
	require.Equal(t, "Review", resp.NewState)
}
