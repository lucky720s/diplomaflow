package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/lucky720s/diplomaflow/internal/workflow/plugins"
	notificationv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/notification/v1"
	"google.golang.org/grpc/metadata"
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
func (p *NotificationPlugin) IsReversible() bool { return false }

func (p *NotificationPlugin) ConfigSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"required": ["title", "message"],
		"properties": {
			"title": {"type":"string"},
			"message": {"type":"string"},
			"link": {"type":"string"},
			"type": {"type":"string", "default":"WORKFLOW"},
			"recipients": {
				"type":"string",
				"enum":["student","team","supervisor","explicit"],
				"default":"student"
			},
			"explicit_user_ids": {
				"type":"array",
				"items":{"type":"integer"}
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
	if p.client == nil {
		return &plugins.ActionResult{Success: true, Data: map[string]interface{}{"skipped": true, "reason": "notification client is nil"}}
	}

	title := toString(actx.Config["title"])
	message := toString(actx.Config["message"])
	if title == "" || message == "" {
		// do not fail whole workflow; it's a configuration issue
		return &plugins.ActionResult{Success: true, Data: map[string]interface{}{"skipped": true, "reason": "empty title/message"}}
	}

	title = interpolate(title, actx)
	message = interpolate(message, actx)

	nType := toString(actx.Config["type"])
	if nType == "" {
		nType = "WORKFLOW"
	}

	recType := toString(actx.Config["recipients"])
	if recType == "" {
		recType = "student"
	}

	recipients := p.getRecipients(actx, recType)

	// If explicit list configured
	if recType == "explicit" {
		if ids := toInt64Slice(actx.Config["explicit_user_ids"]); len(ids) > 0 {
			recipients = append(recipients, ids...)
		}
	}
	link := toString(actx.Config["link"])
	if link == "" {
		// фронтовые роуты:
		// student/team -> /diplom
		// supervisor -> /teacher/diploms/[projectId]
		if recType == "supervisor" {
			link = fmt.Sprintf("/teacher/diploms/%d", actx.ProjectID)
		} else {
			link = "/diplom"
		}
	} else {
		link = interpolate(link, actx)
	}

	recipients = uniqueInt64(recipients)

	if len(recipients) == 0 {
		// best-effort: don't fail post-commit pipeline
		return &plugins.ActionResult{
			Success: true,
			Data: map[string]interface{}{
				"skipped": true,
				"reason":  "no recipients resolved",
				"mode":    recType,
			},
		}
	}

	// mark internal call for observability / future auth hardening
	nctx := metadata.AppendToOutgoingContext(ctx, "x-internal-service", "workflow_service")

	for _, userID := range recipients {
		_, err := p.client.SendNotification(nctx, &notificationv1.SendNotificationRequest{
			UserId:  userID,
			Title:   title,
			Message: message,
			Link:    link,
			Type:    nType,
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

	return &plugins.ActionResult{
		Success: true,
		Data: map[string]interface{}{
			"sent":       true,
			"recipients": len(recipients),
		},
	}
}

func (p *NotificationPlugin) Rollback(ctx context.Context, actx *plugins.ActionContext) error {
	return nil
}

// Recipients resolution strategy:
// - student: prefer project_data.student_id; else fallback to actx.UserID (initiator) to not be empty
// - supervisor: project_data.supervisor_id if present
// - team: requires project_data.team_members OR explicit_user_ids (team list not available in this plugin without team_client)
func (p *NotificationPlugin) getRecipients(actx *plugins.ActionContext, recipientType string) []int64 {
	var out []int64
	switch recipientType {
	case "student":
		if id, ok := readInt64(actx.ProjectData, "student_id"); ok && id > 0 {
			out = append(out, id)
		} else if actx.UserID > 0 {
			out = append(out, actx.UserID)
		}
	case "supervisor":
		if id, ok := readInt64(actx.ProjectData, "supervisor_id"); ok && id > 0 {
			out = append(out, id)
		}
	case "team":
		// expects team_members in project_data as array
		if raw, ok := actx.ProjectData["team_members"]; ok {
			out = append(out, toInt64Slice(raw)...)
		}
	}
	return out
}

func interpolate(template string, actx *plugins.ActionContext) string {
	s := template
	s = strings.ReplaceAll(s, "{{project_id}}", fmt.Sprint(actx.ProjectID))
	s = strings.ReplaceAll(s, "{{team_id}}", fmt.Sprint(actx.TeamID))
	s = strings.ReplaceAll(s, "{{user_id}}", fmt.Sprint(actx.UserID))
	s = strings.ReplaceAll(s, "{{state_name}}", actx.NewState)
	s = strings.ReplaceAll(s, "{{previous_state}}", actx.PreviousState)
	s = strings.ReplaceAll(s, "{{trigger}}", actx.Trigger)

	if t, ok := actx.ProjectData["title"].(string); ok && t != "" {
		s = strings.ReplaceAll(s, "{{project_title}}", t)
	}
	return s
}

func toString(v interface{}) string {
	if v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return t
	default:
		return fmt.Sprint(v)
	}
}

func readInt64(m map[string]interface{}, key string) (int64, bool) {
	if m == nil {
		return 0, false
	}
	v, ok := m[key]
	if !ok || v == nil {
		return 0, false
	}
	switch t := v.(type) {
	case int64:
		return t, true
	case int:
		return int64(t), true
	case float64:
		return int64(t), true
	case string:
		id, err := strconv.ParseInt(t, 10, 64)
		if err == nil {
			return id, true
		}
		return 0, false
	default:
		return 0, false
	}
}

func toInt64Slice(v interface{}) []int64 {
	if v == nil {
		return nil
	}
	switch t := v.(type) {
	case []int64:
		return t
	case []interface{}:
		out := make([]int64, 0, len(t))
		for _, x := range t {
			switch n := x.(type) {
			case int64:
				out = append(out, n)
			case int:
				out = append(out, int64(n))
			case float64:
				out = append(out, int64(n))
			case string:
				if id, err := strconv.ParseInt(n, 10, 64); err == nil {
					out = append(out, id)
				}
			}
		}
		return out
	default:
		return nil
	}
}

func uniqueInt64(in []int64) []int64 {
	seen := make(map[int64]struct{}, len(in))
	out := make([]int64, 0, len(in))
	for _, v := range in {
		if v <= 0 {
			continue
		}
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	return out
}
