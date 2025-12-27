package project

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

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
		// Сравниваем trigger как строку
		if action.Trigger.String() != trigger {
			continue
		}

		if !action.IsEnabled {
			continue
		}

		if err := e.executeAction(ctx, action, project); err != nil {
			e.logger.Error("Failed to execute action",
				zap.Int64("action_id", action.Id),
				zap.String("type", action.Type.String()),
				zap.Error(err))
			// Продолжаем выполнение других actions
		}
	}
	return nil
}

func (e *StateActionExecutor) executeAction(ctx context.Context, action *workflowv1.StateAction, project *Project) error {
	// Парсим конфигурацию action
	config := make(map[string]interface{})
	if action.Config != nil {
		configBytes, _ := action.Config.MarshalJSON()
		if err := json.Unmarshal(configBytes, &config); err != nil {
			return fmt.Errorf("failed to unmarshal action config: %w", err)
		}
	}

	// ✅ ИСПРАВЛЕНО: используем ActionType_*, а не StateAction_*
	switch action.Type {
	case workflowv1.ActionType_SEND_NOTIFICATION:
		return e.sendNotification(ctx, config, project)

	case workflowv1.ActionType_SEND_EMAIL:
		return e.sendEmail(ctx, config, project)

	case workflowv1.ActionType_ASSIGN_TASK:
		return e.assignTask(ctx, config, project)

	case workflowv1.ActionType_CALL_WEBHOOK:
		return e.callWebhook(ctx, config, project)

	case workflowv1.ActionType_SCHEDULE_REMINDER:
		return e.scheduleReminder(ctx, config, project)

	case workflowv1.ActionType_VALIDATE_DATA:
		return e.validateData(ctx, config, project)

	case workflowv1.ActionType_UPDATE_PROJECT:
		return e.updateProject(ctx, config, project)

	default:
		e.logger.Warn("Unknown or unhandled action type",
			zap.String("type", action.Type.String()),
			zap.Int64("action_id", action.Id))
		return nil // Не возвращаем ошибку для неизвестных типов
	}
}

func (e *StateActionExecutor) sendNotification(ctx context.Context, config map[string]interface{}, project *Project) error {
	title, _ := config["title"].(string)
	message, _ := config["message"].(string)

	// Подставляем переменные в message
	message = e.interpolateMessage(message, project)

	// Определяем получателей
	recipients := e.getRecipients(config, project)

	for _, userID := range recipients {
		_, err := e.notifClient.SendNotification(ctx, &notificationv1.SendNotificationRequest{
			UserId:  userID,
			Title:   title,
			Message: message,
			Link:    fmt.Sprintf("/projects/%d", project.ID),
			Type:    "WORKFLOW_UPDATE",
		})
		if err != nil {
			e.logger.Warn("Failed to send notification",
				zap.Int64("user_id", userID),
				zap.Error(err))
		}
	}

	return nil
}

func (e *StateActionExecutor) sendEmail(ctx context.Context, config map[string]interface{}, project *Project) error {
	// TODO: Интеграция с email сервисом
	e.logger.Info("Sending email (not implemented)",
		zap.Uint("project_id", project.ID),
		zap.Any("config", config))
	return nil
}

func (e *StateActionExecutor) assignTask(ctx context.Context, config map[string]interface{}, project *Project) error {
	// TODO: Создать задачу в системе задач
	e.logger.Info("Assigning task",
		zap.Uint("project_id", project.ID),
		zap.Any("config", config))
	return nil
}

func (e *StateActionExecutor) callWebhook(ctx context.Context, config map[string]interface{}, project *Project) error {
	url, _ := config["url"].(string)
	if url == "" {
		return fmt.Errorf("webhook url is required")
	}

	// TODO: Реализовать HTTP вызов
	e.logger.Info("Calling webhook",
		zap.String("url", url),
		zap.Uint("project_id", project.ID))
	return nil
}

func (e *StateActionExecutor) scheduleReminder(ctx context.Context, config map[string]interface{}, project *Project) error {
	// TODO: Создать отложенное напоминание
	e.logger.Info("Scheduling reminder",
		zap.Uint("project_id", project.ID))
	return nil
}

func (e *StateActionExecutor) validateData(ctx context.Context, config map[string]interface{}, project *Project) error {
	// TODO: Валидация данных проекта
	e.logger.Info("Validating project data",
		zap.Uint("project_id", project.ID))
	return nil
}

func (e *StateActionExecutor) updateProject(ctx context.Context, config map[string]interface{}, project *Project) error {
	// TODO: Обновление полей проекта
	e.logger.Info("Updating project",
		zap.Uint("project_id", project.ID))
	return nil
}

// interpolateMessage подставляет переменные в сообщение
func (e *StateActionExecutor) interpolateMessage(message string, project *Project) string {
	// Простая замена переменных
	replacements := map[string]string{
		"{{project_title}}": project.Title,
		"{{project_id}}":    fmt.Sprintf("%d", project.ID),
		"{{current_state}}": project.CurrentState,
		"{{workflow_name}}": project.WorkflowName,
		"%s":                project.Title, // для обратной совместимости
	}

	result := message
	for placeholder, value := range replacements {
		result = strings.ReplaceAll(result, placeholder, value)
	}
	return result
}

// getRecipients определяет получателей уведомления
func (e *StateActionExecutor) getRecipients(config map[string]interface{}, project *Project) []int64 {
	var recipients []int64

	// По умолчанию отправляем студенту
	recipients = append(recipients, project.StudentID)

	// Проверяем дополнительных получателей в конфиге
	if recipientsList, ok := config["recipients"].([]interface{}); ok {
		for _, r := range recipientsList {
			switch v := r.(type) {
			case float64:
				recipients = append(recipients, int64(v))
			case string:
				// Специальные значения
				switch v {
				case "student":
					// Уже добавлен
				case "supervisor":
					// TODO: Получить ID руководителя из данных проекта
				case "team":
					// TODO: Получить ID всех членов команды
				}
			}
		}
	}

	return recipients
}
