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

func TestHandler_CreateStateAction(t *testing.T) {
	mockSvc := new(MockWorkflowService)
	handler := workflow.NewHandler(mockSvc)

	cfg, _ := structpb.NewStruct(map[string]interface{}{})
	action := &workflow.StateAction{ID: 99, StateID: 1, Config: []byte("{}"), Type: "EMAIL", Trigger: "ON_ENTER"}

	mockSvc.On("CreateStateAction", mock.Anything, mock.Anything).Return(action, nil)

	req := &workflowv1.CreateStateActionRequest{
		StateId: 1,
		Config:  cfg,
		Type:    workflowv1.StateAction_SEND_NOTIFICATION,
		Trigger: workflowv1.StateAction_ON_ENTER,
	}

	resp, err := handler.CreateStateAction(context.Background(), req)

	require.NoError(t, err)
	require.Equal(t, int64(99), resp.Id)
}
