package builtin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/lucky720s/diplomaflow/internal/workflow"
	"github.com/lucky720s/diplomaflow/internal/workflow/plugins"
)

const ReviewGatePluginID = "REVIEW_GATE"

// ReviewGatePlugin — ON_EXIT плагин для стэйтов с оценкой/допуском.
// Блокирует переход если:
//   - не все обязанные преподы проголосовали
//   - вызывающий не является finalizer_role
//
// При успехе патчит DataPatch: добавляет "review_ready", "review_result",
// "final_score" (для score-стэйтов) в project data.
type ReviewGatePlugin struct {
	reviewSvc *workflow.ReviewService
}

func NewReviewGatePlugin(reviewSvc *workflow.ReviewService) *ReviewGatePlugin {
	return &ReviewGatePlugin{reviewSvc: reviewSvc}
}

func (p *ReviewGatePlugin) ID() string          { return ReviewGatePluginID }
func (p *ReviewGatePlugin) Name() string        { return "Review Gate" }
func (p *ReviewGatePlugin) Description() string { return "Validates all reviewers voted before allowing transition" }
func (p *ReviewGatePlugin) Category() string    { return plugins.CategoryValidation }
func (p *ReviewGatePlugin) IsReversible() bool  { return false }

func (p *ReviewGatePlugin) ConfigSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"patch_key": {"type": "string", "description": "Key prefix in project data, e.g. 'predefense1'"}
		}
	}`)
}

func (p *ReviewGatePlugin) Validate(config map[string]interface{}) error { return nil }

func (p *ReviewGatePlugin) Rollback(ctx context.Context, actx *plugins.ActionContext) error {
	return nil
}

func (p *ReviewGatePlugin) Execute(ctx context.Context, actx *plugins.ActionContext) *plugins.ActionResult {
	// Достаём review_config из project_data (передаётся движком через actx.Metadata)
	// Ключ "state_review_config" заполняется движком перед вызовом ON_EXIT плагинов.
	rcRaw, ok := actx.Metadata["state_review_config"]
	if !ok {
		return &plugins.ActionResult{
			Success: false,
			Error:   errors.New("review_gate: state_review_config not in metadata"),
		}
	}

	rcBytes, err := json.Marshal(rcRaw)
	if err != nil {
		return &plugins.ActionResult{Success: false, Error: fmt.Errorf("review_gate: marshal rc: %w", err)}
	}
	var rc workflow.ReviewConfig
	if err := json.Unmarshal(rcBytes, &rc); err != nil {
		return &plugins.ActionResult{Success: false, Error: fmt.Errorf("review_gate: unmarshal rc: %w", err)}
	}

	// Проверяем caller role
	if rc.FinalizerRole != "" && actx.Metadata["user_role"] != rc.FinalizerRole {
		callerRole, _ := actx.Metadata["user_role"].(string)
		return &plugins.ActionResult{
			Success: false,
			Error:   fmt.Errorf("review_gate: only %q can finalize this state, got %q", rc.FinalizerRole, callerRole),
		}
	}

	// Вычисляем итог голосования
	summary, err := p.reviewSvc.ComputeSummary(ctx, actx.ProjectID, actx.StateID, &rc)
	if err != nil {
		return &plugins.ActionResult{Success: false, Error: fmt.Errorf("review_gate: compute summary: %w", err)}
	}

	if !summary.AllVoted {
		return &plugins.ActionResult{
			Success: false,
			Error:   fmt.Errorf("review_gate: not all reviewers voted (%d/%d)", summary.TotalReviews, rc.MinReviewers),
		}
	}

	// Патч в project data
	patchKey := "review"
	if pk, ok := actx.Config["patch_key"].(string); ok && pk != "" {
		patchKey = pk
	}

	patch := map[string]interface{}{
		"all_voted":    true,
		"result":       summary.Result,
		"total_reviews": summary.TotalReviews,
	}
	if summary.FinalScore != nil {
		patch["final_score"] = *summary.FinalScore
		patch["average_score"] = summary.AverageScore
	}
	patch["approved_count"] = summary.ApprovedCount
	patch["rejected_count"] = summary.RejectedCount

	// Флаг для условий перехода в transitions
	isPassed := summary.Result == "passed" || summary.Result == "approved"

	return &plugins.ActionResult{
		Success: true,
		Data: map[string]interface{}{
			patchKey + "_review": patch,
			// Плоские флаги для evaluateCondition в движке
			patchKey + "_all_voted": true,
			patchKey + "_passed":    isPassed,
			patchKey + "_failed":    !isPassed,
		},
	}
}
