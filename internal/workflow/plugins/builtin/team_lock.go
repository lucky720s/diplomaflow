package builtin

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/lucky720s/diplomaflow/internal/workflow/plugins"
	teamv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/team/v1"
	"google.golang.org/grpc/metadata"
)

type TeamLockPlugin struct {
	client teamv1.TeamServiceClient
}

func NewTeamLockPlugin(client teamv1.TeamServiceClient) *TeamLockPlugin {
	return &TeamLockPlugin{client: client}
}

func (p *TeamLockPlugin) ID() string { return "LOCK_TEAM_COMPOSITION" }
func (p *TeamLockPlugin) Name() string {
	return "Заблокировать состав команды"
}
func (p *TeamLockPlugin) Description() string {
	return "Запрещает любые изменения состава команды после этапа формирования"
}
func (p *TeamLockPlugin) Category() string   { return plugins.CategoryExternal }
func (p *TeamLockPlugin) IsReversible() bool { return false }

func (p *TeamLockPlugin) ConfigSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"reason": { "type": "string", "default": "workflow_lock_after_team_formed" }
		}
	}`)
}

func (p *TeamLockPlugin) Validate(config map[string]interface{}) error { return nil }

func (p *TeamLockPlugin) Execute(ctx context.Context, actx *plugins.ActionContext) *plugins.ActionResult {
	if actx.TeamID <= 0 {
		// если team_id не проброшен — это ошибка интеграции (починим во 2 пакете файлов)
		return &plugins.ActionResult{Success: false, Error: fmt.Errorf("team_id is required in action context")}
	}

	reason := "workflow_lock_after_team_formed"
	if v, ok := actx.Config["reason"].(string); ok && v != "" {
		reason = v
	}

	internalCtx := metadata.AppendToOutgoingContext(ctx, "x-internal-service", "workflow_service")

	_, err := p.client.LockTeamComposition(internalCtx, &teamv1.LockTeamCompositionRequest{
		TeamId: actx.TeamID,
		Reason: reason,
	})
	if err != nil {
		return &plugins.ActionResult{Success: false, Error: err, ShouldRetry: true, RetryAfter: 60}
	}

	return &plugins.ActionResult{Success: true}
}

func (p *TeamLockPlugin) Rollback(ctx context.Context, actx *plugins.ActionContext) error {
	return nil
}
