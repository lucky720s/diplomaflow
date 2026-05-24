package handler

import (
	"context"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	projectv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/project/v1"
	workflowv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/workflow/v1"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func (h *Handler) GetProjectDetails(c *gin.Context) {
	projectID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || projectID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid project id"})
		return
	}

	userID := c.GetInt64("userId")
	userRole := c.GetString("role")
	if userRole == "" {
		userRole = "user"
	}

	var ctx context.Context

	// teacher/admin → internal ctx
	if userRole == "teacher" || userRole == "admin" {
		ctx = metadata.NewOutgoingContext(
			c.Request.Context(),
			metadata.Pairs(
				"x-internal-service", "admin_service",
				"x-user-id", strconv.FormatInt(userID, 10),
				"x-user-role", userRole,
			),
		)
	} else {
		ctx = outgoingCtx(c)
	}

	projectResp, err := h.projectClient.GetProject(ctx, &projectv1.GetProjectRequest{
		ProjectId: projectID,
	})
	if err != nil {
		MapGRPCError(c, err)
		return
	}

	runtimeResp, err := h.projectClient.GetProjectRuntime(ctx, &projectv1.GetProjectRuntimeRequest{
		ProjectId: projectID,
	})
	if err != nil {
		runtimeResp = nil
	}

	if runtimeResp == nil || runtimeResp.CurrentStateId == 0 {
		history := h.buildHistory(projectResp)
		c.JSON(http.StatusOK, gin.H{
			"project":           projectResp,
			"stages":            []interface{}{},
			"available_actions": []interface{}{},
			"history":           history,
			"viewer":            gin.H{"id": userID, "role": userRole},
		})
		return
	}

	var workflowFull *workflowv1.WorkflowFull
	var transitions *workflowv1.GetAvailableTransitionsResponse

	g, gCtx := errgroup.WithContext(ctx)

	g.Go(func() error {
		var e error
		workflowFull, e = h.workflowClient.GetWorkflowFull(gCtx, &workflowv1.GetWorkflowFullRequest{
			WorkflowId: runtimeResp.WorkflowId,
		})
		return e
	})

	g.Go(func() error {
		var e error
		transitions, e = h.workflowClient.GetAvailableTransitions(gCtx, &workflowv1.GetAvailableTransitionsRequest{
			ProjectId:      projectID,
			CurrentStateId: runtimeResp.CurrentStateId,
			UserId:         userID,
			UserRole:       userRole,
		})
		return e
	})

	if err := g.Wait(); err != nil {
		MapGRPCError(c, err)
		return
	}

	// Загружаем review-сводки для стэйтов с review_config (best-effort)
	reviewSummaries := h.loadReviewSummaries(ctx, projectID, workflowFull)

	stages := h.buildStages(workflowFull, runtimeResp, projectResp, reviewSummaries)
	availableActions := h.buildAvailableActions(transitions)
	history := h.buildHistory(projectResp)

	c.JSON(http.StatusOK, gin.H{
		"project": gin.H{
			"id":                  projectResp.ProjectId,
			"title":               projectResp.Title,
			"description":         projectResp.Description,
			"student_id":          projectResp.StudentId,
			"team_id":             runtimeResp.TeamId,
			"workflow_id":         runtimeResp.WorkflowId,
			"workflow_name":       projectResp.WorkflowName,
			"workflow_version":    runtimeResp.WorkflowVersion,
			"current_state_id":    runtimeResp.CurrentStateId,
			"current_state_name":  runtimeResp.CurrentStateName,
			"status":              projectResp.Status,
			"deadline_at":         formatTimestamp(runtimeResp.DeadlineAt),
			"topic_registered_at": formatTimestamp(projectResp.TopicRegisteredAt),
		},
		"stages":            stages,
		"available_actions": availableActions,
		"history":           history,
		"viewer":            gin.H{"id": userID, "role": userRole},
	})
}

// loadReviewSummaries загружает review-сводки для всех стэйтов workflow у которых есть review_config.
// Запросы параллельны; ошибки игнорируются — не ломаем весь ответ.
func (h *Handler) loadReviewSummaries(
	ctx context.Context,
	projectID int64,
	wf *workflowv1.WorkflowFull,
) map[int64]*workflowv1.GetStateReviewsResponse {
	if wf == nil {
		return nil
	}

	// Определяем стэйты с review_config
	var reviewStateIDs []int64
	for _, state := range wf.States {
		if state == nil || state.Config == nil {
			continue
		}
		cfg := state.Config.AsMap()
		if _, hasReview := cfg["review_config"]; hasReview {
			reviewStateIDs = append(reviewStateIDs, state.Id)
		}
	}
	if len(reviewStateIDs) == 0 {
		return nil
	}

	result := make(map[int64]*workflowv1.GetStateReviewsResponse, len(reviewStateIDs))
	var mu = struct{ sync interface{} }{}
	_ = mu

	// Параллельная загрузка через errgroup (игнорируем ошибки — best-effort)
	type pair struct {
		stateID int64
		resp    *workflowv1.GetStateReviewsResponse
	}
	pairs := make([]pair, len(reviewStateIDs))
	var eg errgroup.Group
	for i, sid := range reviewStateIDs {
		i, sid := i, sid
		eg.Go(func() error {
			resp, err := h.workflowClient.GetStateReviews(ctx, &workflowv1.GetStateReviewsRequest{
				ProjectId: projectID,
				StateId:   sid,
			})
			if err == nil {
				pairs[i] = pair{stateID: sid, resp: resp}
			}
			return nil // best-effort
		})
	}
	_ = eg.Wait()

	for _, p := range pairs {
		if p.resp != nil {
			result[p.stateID] = p.resp
		}
	}
	return result
}

func (h *Handler) buildStages(
	wf *workflowv1.WorkflowFull,
	runtime *projectv1.GetProjectRuntimeResponse,
	project *projectv1.GetProjectResponse,
	reviewSummaries map[int64]*workflowv1.GetStateReviewsResponse,
) []gin.H {
	if wf == nil || wf.States == nil {
		return []gin.H{}
	}
	var currentOrderIndex int32 = -1
	for _, state := range wf.States {
		if state != nil && state.Id == runtime.CurrentStateId {
			currentOrderIndex = state.OrderIndex
			break
		}
	}

	completedStateNames := make(map[string]bool)
	if project != nil && project.History != nil {
		for _, entry := range project.History {
			if entry != nil && entry.StateName != "" {
				completedStateNames[entry.StateName] = true
			}
		}
	}

	stages := make([]gin.H, 0, len(wf.States))
	for _, state := range wf.States {
		if state == nil {
			continue
		}
		stageStatus := "pending"
		if state.Id == runtime.CurrentStateId {
			stageStatus = "current"
		} else if currentOrderIndex >= 0 && state.OrderIndex < currentOrderIndex {
			stageStatus = "completed"
		} else if completedStateNames[state.Name] {
			stageStatus = "completed"
		}

		stage := gin.H{
			"id":            state.Id,
			"name":          state.Name,
			"display_name":  state.DisplayName,
			"description":   state.Description,
			"order_index":   state.OrderIndex,
			"type":          state.Type.String(),
			"is_initial":    state.IsInitial,
			"is_final":      state.IsFinal,
			"is_optional":   state.IsOptional,
			"duration_days": state.DurationDays,
			"duration_mode": state.DurationMode,
			"color":         state.Color,
			"icon":          state.Icon,
			"status":        stageStatus,
		}

		if state.Config != nil {
			cfg := state.Config.AsMap()
			stage["config"] = cfg
			// Выносим review_config на верхний уровень для удобства фронта
			if rc, ok := cfg["review_config"]; ok {
				stage["review_config"] = rc
			}
		}

		if state.FixedDeadline != nil {
			stage["fixed_deadline"] = state.FixedDeadline.AsTime().Format("2006-01-02T15:04:05Z")
		}

		// Исходящие переходы
		var outTransitions []gin.H
		if wf.Transitions != nil {
			for _, tr := range wf.Transitions {
				if tr != nil && tr.FromStateId == state.Id {
					outTransitions = append(outTransitions, gin.H{
						"id":           tr.Id,
						"event_name":   tr.EventName,
						"display_name": tr.DisplayName,
						"to_state_id":  tr.ToStateId,
						"button_label": tr.ButtonLabel,
						"button_color": tr.ButtonColor,
						"confirm_text": tr.ConfirmText,
					})
				}
			}
		}
		if len(outTransitions) > 0 {
			stage["transitions"] = outTransitions
		}

		// Действия стэйта
		if state.Actions != nil {
			var actions []gin.H
			for _, a := range state.Actions {
				if a == nil {
					continue
				}
				actions = append(actions, gin.H{
					"id":      a.Id,
					"name":    a.Name,
					"type":    a.Type.String(),
					"trigger": a.Trigger.String(),
				})
			}
			if len(actions) > 0 {
				stage["actions"] = actions
			}
		}

		// Review-сводка (оценки/допуски) если есть
		if reviewSummaries != nil {
			if rv, ok := reviewSummaries[state.Id]; ok && rv != nil {
				var reviews []gin.H
				for _, r := range rv.Reviews {
					if r == nil {
						continue
					}
					item := gin.H{
						"id":          r.Id,
						"reviewer_id": r.ReviewerId,
						"role_slug":   r.RoleSlug,
						"decision":    r.Decision,
						"comment":     r.Comment,
						"created_at":  formatTimestamp(r.CreatedAt),
					}
					if r.HasScore {
						item["score"] = r.Score
					}
					reviews = append(reviews, item)
				}
				if len(reviews) > 0 {
					stage["reviews"] = reviews
				}
				if rv.Summary != nil {
					s := rv.Summary
					summary := gin.H{
						"total_reviews":  s.TotalReviews,
						"all_voted":      s.AllVoted,
						"result":         s.Result,
						"approved_count": s.ApprovedCount,
						"rejected_count": s.RejectedCount,
						"average_score":  s.AverageScore,
					}
					if s.HasFinalScore {
						summary["final_score"] = s.FinalScore
					}
					stage["review_summary"] = summary
				}
			}
		}

		stages = append(stages, stage)
	}

	return stages
}

func (h *Handler) buildAvailableActions(transitions *workflowv1.GetAvailableTransitionsResponse) []gin.H {
	if transitions == nil || transitions.Transitions == nil {
		return []gin.H{}
	}

	actions := make([]gin.H, 0, len(transitions.Transitions))
	for _, t := range transitions.Transitions {
		if t == nil || t.Transition == nil {
			continue
		}
		actions = append(actions, gin.H{
			"transition_id":        t.Transition.Id,
			"event_name":           t.Transition.EventName,
			"display_name":         t.Transition.DisplayName,
			"from_state_id":        t.Transition.FromStateId,
			"to_state_id":          t.Transition.ToStateId,
			"button_label":         t.Transition.ButtonLabel,
			"button_color":         t.Transition.ButtonColor,
			"confirm_text":         t.Transition.ConfirmText,
			"can_execute":          t.CanExecute,
			"blocked_reason":       t.BlockedReason,
			"missing_requirements": t.MissingRequirements,
		})
	}

	return actions
}

func (h *Handler) buildHistory(project *projectv1.GetProjectResponse) []gin.H {
	if project == nil || project.History == nil {
		return []gin.H{}
	}

	history := make([]gin.H, 0, len(project.History))
	for _, entry := range project.History {
		if entry == nil {
			continue
		}
		history = append(history, gin.H{
			"state_name": entry.StateName,
			"status":     entry.Status,
			"changed_by": entry.ChangedBy,
			"comment":    entry.Comment,
			"timestamp":  entry.Timestamp,
		})
	}

	return history
}

func formatTimestamp(ts *timestamppb.Timestamp) interface{} {
	if ts == nil || ts.Seconds == 0 {
		return nil
	}
	return ts.AsTime().Format("2006-01-02T15:04:05Z")
}
