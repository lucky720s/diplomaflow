package project

import (
	"context"
	"encoding/json"
	"fmt"

	notificationv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/notification/v1"
	workflowv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/workflow/v1"
	"go.uber.org/zap"
)

type StateActionExecutor struct {
	workflowClient workflowv1.WorkflowServiceClient
	notifClient    notificationv1.NotificationServiceClient
	logger         *zap.Logger
}

func NewStateActionExecutor(
	wfClient workflowv1.WorkflowServiceClient,
	notifClient notificationv1.NotificationServiceClient,
	logger *zap.Logger,
) *StateActionExecutor {
	return &StateActionExecutor{
		workflowClient: wfClient,
		notifClient:    notifClient,
		logger:         logger,
	}
}

func (e *StateActionExecutor) ExecuteActions(ctx context.Context, stateID int64, trigger string, project *Project) error {
	actionsResp, err := e.workflowClient.ListStateActions(ctx, &workflowv1.ListStateActionsRequest{
		StateId: stateID,
	})
	if err != nil {
		return fmt.Errorf("failed to get state actions: %w", err)
	}

	for _, action := range actionsResp.Actions {
		if action.Trigger.String() != trigger {
			continue
		}

		if err := e.executeAction(ctx, action, project); err != nil {
			e.logger.Error("Failed to execute action",
				zap.Int64("action_id", action.Id),
				zap.String("type", action.Type.String()),
				zap.Error(err))

		}
	}

	return nil
}

func (e *StateActionExecutor) executeAction(ctx context.Context, action *workflowv1.StateAction, project *Project) error {
	configBytes, _ := action.Config.MarshalJSON()
	var config map[string]interface{}
	json.Unmarshal(configBytes, &config)

	switch action.Type {
	case workflowv1.StateAction_SEND_NOTIFICATION:
		return e.sendNotification(ctx, config, project)
	case workflowv1.StateAction_ASSIGN_TASK:
		return e.assignTask(ctx, config, project)
	case workflowv1.StateAction_CALL_WEBHOOK:
		return e.callWebhook(ctx, config, project)
	default:
		return fmt.Errorf("unknown action type: %s", action.Type)
	}
}

func (e *StateActionExecutor) sendNotification(ctx context.Context, config map[string]interface{}, project *Project) error {
	title, _ := config["title"].(string)
	message, _ := config["message"].(string)

	message = fmt.Sprintf(message, project.Title)

	_, err := e.notifClient.SendNotification(ctx, &notificationv1.SendNotificationRequest{
		UserId:  project.StudentID,
		Title:   title,
		Message: message,
		Link:    fmt.Sprintf("/projects/%d", project.ID),
		Type:    "WORKFLOW_UPDATE",
	})

	return err
}

func (e *StateActionExecutor) assignTask(ctx context.Context, config map[string]interface{}, project *Project) error {
	// TODO: Создать задачу в системе
	e.logger.Info("Assigning task", zap.Uint("project_id", project.ID))
	return nil
}

func (e *StateActionExecutor) callWebhook(ctx context.Context, config map[string]interface{}, project *Project) error {
	// TODO: Вызвать внешний webhook
	url, _ := config["url"].(string)
	e.logger.Info("Calling webhook", zap.String("url", url))
	return nil
}
