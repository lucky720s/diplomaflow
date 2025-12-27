package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/lucky720s/diplomaflow/internal/workflow/plugins"
	notificationv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/notification/v1"
)

type NotificationPlugin struct {
	client notificationv1.NotificationServiceClient
}

func NewNotificationPlugin(client notificationv1.NotificationServiceClient) *NotificationPlugin {
	return &NotificationPlugin{client: client}
}

func (p *NotificationPlugin) ID() string   { return "SEND_NOTIFICATION" }
func (p *NotificationPlugin) Name() string { return "Отправить уведомление" }
func (p *NotificationPlugin) Description() string {
	return "Отправляет push-уведомление пользователям"
}
func (p *NotificationPlugin) Category() string   { return plugins.CategoryNotification }
func (p *NotificationPlugin) IsReversible() bool { return false } // ← ДОБАВЛЕНО!

func (p *NotificationPlugin) ConfigSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"required": ["title", "message"],
		"properties": {
			"title": {
				"type": "string",
				"title": "Заголовок",
				"description": "Поддерживает переменные: {{project_title}}, {{student_name}}, {{deadline}}"
			},
			"message": {
				"type": "string",
				"title": "Сообщение"
			},
			"link": {
				"type": "string",
				"title": "Ссылка"
			},
			"recipients": {
				"type": "string",
				"title": "Получатели",
				"enum": ["student", "team", "supervisor", "commission"],
				"default": "student"
			},
			"urgency": {
				"type": "string",
				"enum": ["low", "normal", "high", "urgent"],
				"default": "normal"
			}
		}
	}`)
}

func (p *NotificationPlugin) Validate(config map[string]interface{}) error {
	if _, ok := config["title"]; !ok {
		return fmt.Errorf("title is required")
	}
	if _, ok := config["message"]; !ok {
		return fmt.Errorf("message is required")
	}
	return nil
}

func (p *NotificationPlugin) Execute(ctx context.Context, actx *plugins.ActionContext) *plugins.ActionResult {
	title := p.interpolate(actx.Config["title"].(string), actx)
	message := p.interpolate(actx.Config["message"].(string), actx)

	recipients := p.getRecipients(actx)
	for _, userID := range recipients {
		_, err := p.client.SendNotification(ctx, &notificationv1.SendNotificationRequest{
			UserId:  userID,
			Title:   title,
			Message: message,
			Link:    fmt.Sprintf("/projects/%d", actx.ProjectID),
			Type:    "WORKFLOW",
		})
		if err != nil {
			return &plugins.ActionResult{
				Success:     false,
				Error:       err,
				ShouldRetry: true,
				RetryAfter:  60,
			}
		}
	}
	return &plugins.ActionResult{Success: true}
}

// ДОБАВЛЕНО: реализация Rollback
func (p *NotificationPlugin) Rollback(ctx context.Context, actx *plugins.ActionContext) error {
	return nil // Уведомления нельзя отменить
}

func (p *NotificationPlugin) interpolate(template string, actx *plugins.ActionContext) string {
	result := template
	result = strings.ReplaceAll(result, "{{project_id}}", fmt.Sprint(actx.ProjectID))
	if title, ok := actx.ProjectData["title"].(string); ok {
		result = strings.ReplaceAll(result, "{{project_title}}", title)
	}
	return result
}

func (p *NotificationPlugin) getRecipients(actx *plugins.ActionContext) []int64 {
	recipientType, _ := actx.Config["recipients"].(string)
	var recipients []int64

	switch recipientType {
	case "student":
		if id, ok := actx.ProjectData["student_id"].(float64); ok {
			recipients = append(recipients, int64(id))
		}
	case "team":
		if members, ok := actx.ProjectData["team_members"].([]interface{}); ok {
			for _, m := range members {
				if id, ok := m.(float64); ok {
					recipients = append(recipients, int64(id))
				}
			}
		}
	case "supervisor":
		if id, ok := actx.ProjectData["supervisor_id"].(float64); ok {
			recipients = append(recipients, int64(id))
		}
	}
	return recipients
}
