package tests_project

import (
	"context"
	"testing"

	"github.com/lucky720s/diplomaflow/internal/project"
	notificationv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/notification/v1"
	workflowv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/workflow/v1"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestStateActionExecutor_SendNotification(t *testing.T) {
	wfClient := new(MockWorkflowClient)
	notifClient := new(MockNotificationClient)
	logger := zap.NewNop()

	executor := project.NewStateActionExecutor(wfClient, notifClient, logger)

	cfgMap := map[string]interface{}{
		"title":   "Status Update",
		"message": "Project %s updated",
	}
	cfgStruct, _ := structpb.NewStruct(cfgMap)

	action := &workflowv1.StateAction{
		Id: 1, Type: workflowv1.StateAction_SEND_NOTIFICATION, Trigger: workflowv1.StateAction_ON_ENTER,
		Config: cfgStruct,
	}

	wfClient.On("ListStateActions", mock.Anything, mock.Anything).
		Return(&workflowv1.ListStateActionsResponse{Actions: []*workflowv1.StateAction{action}}, nil)

	notifClient.On("SendNotification", mock.Anything, mock.Anything).
		Return(&notificationv1.SendNotificationResponse{}, nil)

	proj := &project.Project{ID: 10, Title: "My Dipl", StudentID: 5}

	err := executor.ExecuteActions(context.Background(), 100, "ON_ENTER", proj)

	require.NoError(t, err)
	notifClient.AssertExpectations(t)
}
