package postcommit

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/lucky720s/diplomaflow/internal/workflow"
	"github.com/lucky720s/diplomaflow/internal/workflow/plugins"
	projectv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/project/v1"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

const workflowActionsTopic = "workflow-actions"

type Worker struct {
	db            *gorm.DB
	projectClient projectv1.ProjectServiceClient
	logger        *zap.Logger
}

func NewWorker(db *gorm.DB, projectClient projectv1.ProjectServiceClient, logger *zap.Logger) *Worker {
	return &Worker{db: db, projectClient: projectClient, logger: logger}
}

type postCommitPayload struct {
	ProjectID    int64   `json:"project_id"`
	StateID      int64   `json:"state_id"`
	TransitionID int64   `json:"transition_id"`
	Trigger      string  `json:"trigger"`
	ActionIDs    []int64 `json:"action_ids"`
	UserID       int64   `json:"user_id"`
	DepartmentID int64   `json:"department_id"`
}

type deadlinePayload struct {
	ProjectID    int64  `json:"project_id"`
	StateID      int64  `json:"state_id"`
	DepartmentID int64  `json:"department_id"`
	Trigger      string `json:"trigger"`
	DeadlineAt   string `json:"deadline_at"`
}

// Handle — обработчик события (без Kafka).
// eventType ожидается: "WorkflowPostCommitActions" | "WorkflowDeadlineReached"
// payload — JSON (как в outbox payload).
func (w *Worker) Handle(ctx context.Context, eventType string, payload []byte) error {
	switch eventType {
	case "WorkflowPostCommitActions":
		var p postCommitPayload
		if err := json.Unmarshal(payload, &p); err != nil {
			return status.Errorf(codes.InvalidArgument, "bad payload: %v", err)
		}
		return w.handlePostCommit(ctx, eventType, p)

	case "WorkflowDeadlineReached":
		var p deadlinePayload
		if err := json.Unmarshal(payload, &p); err != nil {
			return status.Errorf(codes.InvalidArgument, "bad payload: %v", err)
		}
		return w.handleDeadline(ctx, eventType, p)

	default:
		return status.Errorf(codes.InvalidArgument, "unknown event_type: %s", eventType)
	}
}

func (w *Worker) handlePostCommit(ctx context.Context, eventType string, p postCommitPayload) error {
	if p.ProjectID == 0 || p.Trigger == "" || len(p.ActionIDs) == 0 {
		return status.Error(codes.InvalidArgument, "invalid payload: project_id/trigger/action_ids required")
	}

	actions, err := w.loadActionsByIDs(ctx, p.ActionIDs)
	if err != nil {
		return status.Errorf(codes.Internal, "load actions: %v", err)
	}

	// internal call to project_service
	internalCtx := metadata.AppendToOutgoingContext(ctx, "x-internal-service", "workflow_service")

	// тянем актуальные данные проекта ПОСЛЕ commit
	snap, err := w.projectClient.GetProjectRuntime(internalCtx, &projectv1.GetProjectRuntimeRequest{ProjectId: p.ProjectID})
	if err != nil {
		return status.Errorf(codes.Unavailable, "get project runtime: %v", err)
	}

	projectData := map[string]interface{}{}
	if snap.Data != nil {
		projectData = snap.Data.AsMap()
	}

	for _, a := range actions {
		if !a.IsEnabled {
			continue
		}

		dedup := dedupKey("postcommit", p.ProjectID, p.TransitionID, p.Trigger, a.ID, "")
		started, err := w.tryStartRun(ctx, dedup, eventType, workflowActionsTopic, p.ProjectID, p.StateID, p.TransitionID, p.Trigger, a.ID)
		if err != nil {
			return status.Errorf(codes.Internal, "tryStartRun: %v", err)
		}
		if !started {
			continue
		}

		actx := &plugins.ActionContext{
			ProjectID:     p.ProjectID,
			StateID:       p.StateID,
			UserID:        p.UserID,
			TeamID:        snap.TeamId,
			DepartmentID:  snap.DepartmentId,
			UniversityID:  snap.UniversityId,
			Trigger:       p.Trigger,
			Config:        parseConfig(a.Config),
			ProjectData:   projectData,
			PreviousState: "",
			NewState:      snap.CurrentStateName,
			TransitionID:  p.TransitionID,
			Metadata:      map[string]interface{}{},
			Payload:       map[string]interface{}{},
		}

		if err := w.executeOne(ctx, dedup, &a, actx); err != nil {
			return err
		}
	}

	return nil
}

func (w *Worker) handleDeadline(ctx context.Context, eventType string, p deadlinePayload) error {
	if p.ProjectID == 0 || p.StateID == 0 || p.Trigger == "" {
		return status.Error(codes.InvalidArgument, "invalid payload: project_id/state_id/trigger required")
	}

	actions, err := w.loadActionsByStateTrigger(ctx, p.StateID, p.Trigger)
	if err != nil {
		return status.Errorf(codes.Internal, "load actions: %v", err)
	}

	internalCtx := metadata.AppendToOutgoingContext(ctx, "x-internal-service", "workflow_service")

	snap, err := w.projectClient.GetProjectRuntime(internalCtx, &projectv1.GetProjectRuntimeRequest{ProjectId: p.ProjectID})
	if err != nil {
		return status.Errorf(codes.Unavailable, "get project runtime: %v", err)
	}

	projectData := map[string]interface{}{}
	if snap.Data != nil {
		projectData = snap.Data.AsMap()
	}

	for _, a := range actions {
		if !a.IsEnabled {
			continue
		}

		dedup := dedupKey("deadline", p.ProjectID, 0, p.Trigger, a.ID, p.DeadlineAt)
		started, err := w.tryStartRun(ctx, dedup, eventType, workflowActionsTopic, p.ProjectID, p.StateID, 0, p.Trigger, a.ID)
		if err != nil {
			return status.Errorf(codes.Internal, "tryStartRun: %v", err)
		}
		if !started {
			continue
		}

		actx := &plugins.ActionContext{
			ProjectID:     p.ProjectID,
			StateID:       p.StateID,
			UserID:        0,
			TeamID:        snap.TeamId,
			DepartmentID:  snap.DepartmentId,
			UniversityID:  snap.UniversityId,
			Trigger:       p.Trigger,
			Config:        parseConfig(a.Config),
			ProjectData:   projectData,
			PreviousState: "",
			NewState:      snap.CurrentStateName,
			TransitionID:  0,
			Metadata: map[string]interface{}{
				"deadline_at": p.DeadlineAt,
			},
			Payload: map[string]interface{}{},
		}

		if err := w.executeOne(ctx, dedup, &a, actx); err != nil {
			return err
		}
	}

	return nil
}

// --- DB + execution helpers ---

func (w *Worker) loadActionsByIDs(ctx context.Context, ids []int64) ([]workflow.StateAction, error) {
	var actions []workflow.StateAction
	if err := w.db.WithContext(ctx).Where("id IN ?", ids).Find(&actions).Error; err != nil {
		return nil, err
	}
	return actions, nil
}

func (w *Worker) loadActionsByStateTrigger(ctx context.Context, stateID int64, trigger string) ([]workflow.StateAction, error) {
	var actions []workflow.StateAction
	if err := w.db.WithContext(ctx).
		Where("state_id = ? AND trigger = ? AND is_enabled = ?", stateID, trigger, true).
		Order("order_index ASC, id ASC").
		Find(&actions).Error; err != nil {
		return nil, err
	}
	return actions, nil
}

type ActionRun struct {
	ID           int64      `gorm:"primaryKey"`
	DedupKey     string     `gorm:"column:dedup_key;uniqueIndex"`
	EventType    string     `gorm:"column:event_type"`
	Topic        string     `gorm:"column:topic"`
	ProjectID    int64      `gorm:"column:project_id"`
	StateID      *int64     `gorm:"column:state_id"`
	TransitionID *int64     `gorm:"column:transition_id"`
	Trigger      string     `gorm:"column:trigger"`
	ActionID     int64      `gorm:"column:action_id"`
	Status       string     `gorm:"column:status"`
	Attempts     int        `gorm:"column:attempts"`
	LastError    *string    `gorm:"column:last_error"`
	CreatedAt    time.Time  `gorm:"column:created_at"`
	UpdatedAt    time.Time  `gorm:"column:updated_at"`
	SucceededAt  *time.Time `gorm:"column:succeeded_at"`
}

func (ActionRun) TableName() string { return "workflow_action_runs" }

func (w *Worker) tryStartRun(
	ctx context.Context,
	dedupKey string,
	eventType string,
	topic string,
	projectID int64,
	stateID int64,
	transitionID int64,
	trigger string,
	actionID int64,
) (bool, error) {
	now := time.Now().UTC()

	var sid *int64
	if stateID != 0 {
		sid = &stateID
	}
	var tid *int64
	if transitionID != 0 {
		tid = &transitionID
	}

	run := &ActionRun{
		DedupKey:     dedupKey,
		EventType:    eventType,
		Topic:        topic,
		ProjectID:    projectID,
		StateID:      sid,
		TransitionID: tid,
		Trigger:      trigger,
		ActionID:     actionID,
		Status:       "running",
		Attempts:     1,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	err := w.db.WithContext(ctx).Create(run).Error
	if err == nil {
		return true, nil
	}
	if isUniqueViolation(err) {
		return false, nil
	}
	return false, err
}

func (w *Worker) executeOne(ctx context.Context, dedupKey string, action *workflow.StateAction, actx *plugins.ActionContext) error {
	pl, err := plugins.Get(action.Type)
	if err != nil {
		_ = w.markFailed(ctx, dedupKey, fmt.Sprintf("plugin not found: %v", err))
		return status.Errorf(codes.InvalidArgument, "plugin not found: %v", err)
	}

	res := pl.Execute(ctx, actx)
	if res == nil || res.Success {
		_ = w.markSucceeded(ctx, dedupKey)
		return nil
	}

	errMsg := "action failed"
	if res.Error != nil {
		errMsg = res.Error.Error()
	}
	_ = w.markFailed(ctx, dedupKey, errMsg)

	if res.ShouldRetry {
		return status.Errorf(codes.Unavailable, "action retryable: %s", errMsg)
	}
	return status.Errorf(codes.Internal, "action failed: %s", errMsg)
}

func (w *Worker) markSucceeded(ctx context.Context, dedupKey string) error {
	now := time.Now().UTC()
	return w.db.WithContext(ctx).
		Model(&ActionRun{}).
		Where("dedup_key = ?", dedupKey).
		Updates(map[string]interface{}{
			"status":       "succeeded",
			"updated_at":   now,
			"succeeded_at": &now,
			"last_error":   nil,
		}).Error
}

func (w *Worker) markFailed(ctx context.Context, dedupKey string, errMsg string) error {
	now := time.Now().UTC()
	return w.db.WithContext(ctx).
		Model(&ActionRun{}).
		Where("dedup_key = ?", dedupKey).
		Updates(map[string]interface{}{
			"status":     "failed",
			"updated_at": now,
			"last_error": &errMsg,
		}).Error
}

func parseConfig(b []byte) map[string]interface{} {
	var m map[string]interface{}
	if err := json.Unmarshal(b, &m); err != nil || m == nil {
		return map[string]interface{}{}
	}
	return m
}

func dedupKey(kind string, projectID, transitionID int64, trigger string, actionID int64, extra string) string {
	raw := fmt.Sprintf("%s|p=%d|t=%d|tr=%s|a=%d|x=%s", kind, projectID, transitionID, trigger, actionID, extra)
	sum := sha1.Sum([]byte(raw))
	return hex.EncodeToString(sum[:])
}

func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "duplicate key") ||
		strings.Contains(msg, "unique constraint") ||
		strings.Contains(msg, "UNIQUE constraint")
}
