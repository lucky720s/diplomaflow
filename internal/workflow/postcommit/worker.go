package postcommit

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgconn"
	"github.com/lucky720s/diplomaflow/internal/workflow"
	"github.com/lucky720s/diplomaflow/internal/workflow/plugins"
	projectv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/project/v1"
	teamv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/team/v1"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Worker struct {
	db            *gorm.DB
	projectClient projectv1.ProjectServiceClient
	teamClient    teamv1.TeamServiceClient
	logger        *zap.Logger

	projectRPCTimeout time.Duration
	teamRPCTimeout    time.Duration
	runningLease      time.Duration
}

func NewWorker(db *gorm.DB, projectClient projectv1.ProjectServiceClient, teamClient teamv1.TeamServiceClient, logger *zap.Logger) *Worker {
	return &Worker{
		db:                db,
		projectClient:     projectClient,
		teamClient:        teamClient,
		logger:            logger,
		projectRPCTimeout: 5 * time.Second,
		teamRPCTimeout:    5 * time.Second,
		runningLease:      10 * time.Minute,
	}
}

type postCommitPayload struct {
	ProjectID    int64   `json:"project_id"`
	StateID      int64   `json:"state_id"`
	TransitionID int64   `json:"transition_id"`
	Trigger      string  `json:"trigger"` // ON_ENTER/ON_EXIT
	ActionIDs    []int64 `json:"action_ids"`
	UserID       int64   `json:"user_id"`
	DepartmentID int64   `json:"department_id"`
}

type deadlinePayload struct {
	ProjectID    int64  `json:"project_id"`
	StateID      int64  `json:"state_id"`
	DepartmentID int64  `json:"department_id"`
	Trigger      string `json:"trigger"`     // ON_DEADLINE
	DeadlineAt   string `json:"deadline_at"` // RFC3339
}

// Handle is called by poller or gRPC server.
func (w *Worker) Handle(ctx context.Context, eventType string, payload []byte) error {
	switch eventType {
	case "WorkflowPostCommitActions":
		var p postCommitPayload
		if err := json.Unmarshal(payload, &p); err != nil {
			return status.Errorf(codes.InvalidArgument, "bad payload: %v", err)
		}
		return w.handlePostCommit(ctx, p)

	case "WorkflowDeadlineReached":
		var p deadlinePayload
		if err := json.Unmarshal(payload, &p); err != nil {
			return status.Errorf(codes.InvalidArgument, "bad payload: %v", err)
		}
		return w.handleDeadline(ctx, p)

	default:
		return status.Errorf(codes.InvalidArgument, "unknown event_type: %s", eventType)
	}
}

func (w *Worker) handlePostCommit(ctx context.Context, p postCommitPayload) error {
	if p.ProjectID <= 0 || p.Trigger == "" || len(p.ActionIDs) == 0 {
		return status.Error(codes.InvalidArgument, "project_id/trigger/action_ids required")
	}

	actions, err := w.loadActionsByIDs(ctx, p.ActionIDs)
	if err != nil {
		return status.Errorf(codes.Internal, "load actions: %v", err)
	}

	internalCtx := metadata.AppendToOutgoingContext(ctx, "x-internal-service", "workflow_service")

	ctxRPC, cancel := context.WithTimeout(internalCtx, w.projectRPCTimeout)
	snap, err := w.projectClient.GetProjectRuntime(ctxRPC, &projectv1.GetProjectRuntimeRequest{ProjectId: p.ProjectID})
	cancel()
	if err != nil {
		return status.Errorf(codes.Unavailable, "get project runtime: %v", err)
	}

	projectData := w.buildProjectData(ctx, internalCtx, snap)

	for _, a := range actions {
		if !a.IsEnabled {
			continue
		}

		dedup := makeDedupKey("postcommit", p.ProjectID, p.TransitionID, p.Trigger, a.ID, "")
		started, err := w.tryStartOrResumeRun(ctx, dedup, "WorkflowPostCommitActions", p.ProjectID, p.StateID, p.TransitionID, p.Trigger, &a)
		if err != nil {
			return status.Errorf(codes.Internal, "tryStartOrResumeRun: %v", err)
		}
		if !started {
			continue
		}

		actx := &plugins.ActionContext{
			ProjectID:    p.ProjectID,
			StateID:      p.StateID,
			UserID:       p.UserID,
			TeamID:       snap.TeamId,
			DepartmentID: snap.DepartmentId,
			UniversityID: snap.UniversityId,
			Trigger:      p.Trigger,
			Config:       parseConfig(a.Config),
			ProjectData:  projectData,
			NewState:     snap.CurrentStateName,
			TransitionID: p.TransitionID,
			Metadata:     map[string]interface{}{},
			Payload:      map[string]interface{}{},
		}

		if err := w.executeOne(ctx, dedup, &a, actx); err != nil {
			return err
		}
	}

	return nil
}

func (w *Worker) handleDeadline(ctx context.Context, p deadlinePayload) error {
	if p.ProjectID <= 0 || p.StateID <= 0 || p.Trigger == "" {
		return status.Error(codes.InvalidArgument, "project_id/state_id/trigger required")
	}

	actions, err := w.loadActionsByStateTrigger(ctx, p.StateID, p.Trigger)
	if err != nil {
		return status.Errorf(codes.Internal, "load actions: %v", err)
	}

	internalCtx := metadata.AppendToOutgoingContext(ctx, "x-internal-service", "workflow_service")

	ctxRPC, cancel := context.WithTimeout(internalCtx, w.projectRPCTimeout)
	snap, err := w.projectClient.GetProjectRuntime(ctxRPC, &projectv1.GetProjectRuntimeRequest{ProjectId: p.ProjectID})
	cancel()
	if err != nil {
		return status.Errorf(codes.Unavailable, "get project runtime: %v", err)
	}

	projectData := w.buildProjectData(ctx, internalCtx, snap)

	for _, a := range actions {
		if !a.IsEnabled {
			continue
		}

		dedup := makeDedupKey("deadline", p.ProjectID, 0, p.Trigger, a.ID, p.DeadlineAt)
		started, err := w.tryStartOrResumeRun(ctx, dedup, "WorkflowDeadlineReached", p.ProjectID, p.StateID, 0, p.Trigger, &a)
		if err != nil {
			return status.Errorf(codes.Internal, "tryStartOrResumeRun: %v", err)
		}
		if !started {
			continue
		}

		actx := &plugins.ActionContext{
			ProjectID:    p.ProjectID,
			StateID:      p.StateID,
			UserID:       0,
			TeamID:       snap.TeamId,
			DepartmentID: snap.DepartmentId,
			UniversityID: snap.UniversityId,
			Trigger:      p.Trigger,
			Config:       parseConfig(a.Config),
			ProjectData:  projectData,
			NewState:     snap.CurrentStateName,
			TransitionID: 0,
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

func (w *Worker) buildProjectData(ctx context.Context, internalCtx context.Context, snap *projectv1.GetProjectRuntimeResponse) map[string]interface{} {
	projectData := map[string]interface{}{}
	if snap != nil && snap.Data != nil {
		projectData = snap.Data.AsMap()
	}

	if snap != nil {
		projectData["project_id"] = snap.ProjectId
		projectData["student_id"] = snap.StudentId
		projectData["team_id"] = snap.TeamId
		projectData["department_id"] = snap.DepartmentId
		projectData["university_id"] = snap.UniversityId
		projectData["current_state_id"] = snap.CurrentStateId
		projectData["current_state_name"] = snap.CurrentStateName
	}

	if w.teamClient != nil && snap != nil && snap.TeamId > 0 {
		ctxTeam, cancel := context.WithTimeout(internalCtx, w.teamRPCTimeout)
		teamResp, err := w.teamClient.GetTeam(ctxTeam, &teamv1.GetTeamRequest{TeamId: snap.TeamId})
		cancel()
		if err != nil {
			w.logger.Warn("GetTeam failed (skip team_members enrichment)", zap.Int64("team_id", snap.TeamId), zap.Error(err))
			return projectData
		}
		var members []int64
		for _, m := range teamResp.Members {
			if m != nil && m.UserId > 0 {
				members = append(members, m.UserId)
			}
		}
		projectData["team_members"] = members
	}

	return projectData
}

// ---- DB models & helpers (workflow_action_runs) ----

type actionRun struct {
	ID           int64      `gorm:"primaryKey"`
	DedupKey     string     `gorm:"column:dedup_key;uniqueIndex"`
	EventType    string     `gorm:"column:event_type"`
	Topic        string     `gorm:"column:topic"`
	ProjectID    int64      `gorm:"column:project_id"`
	StateID      *int64     `gorm:"column:state_id"`
	TransitionID *int64     `gorm:"column:transition_id"`
	Trigger      string     `gorm:"column:trigger"`
	ActionID     int64      `gorm:"column:action_id"`
	Status       string     `gorm:"column:status"` // running|failed|succeeded
	Attempts     int        `gorm:"column:attempts"`
	LastError    *string    `gorm:"column:last_error"`
	CreatedAt    time.Time  `gorm:"column:created_at"`
	UpdatedAt    time.Time  `gorm:"column:updated_at"`
	SucceededAt  *time.Time `gorm:"column:succeeded_at"`
}

func (actionRun) TableName() string { return "workflow_action_runs" }

func (w *Worker) tryStartOrResumeRun(
	ctx context.Context,
	dedupKey string,
	eventType string,
	projectID int64,
	stateID int64,
	transitionID int64,
	trigger string,
	action *workflow.StateAction,
) (bool, error) {
	now := time.Now().UTC()

	maxRetries := int(action.MaxRetries)
	if maxRetries <= 0 {
		maxRetries = 3
	}

	var sid *int64
	if stateID > 0 {
		sid = &stateID
	}
	var tid *int64
	if transitionID > 0 {
		tid = &transitionID
	}

	return w.withTx(ctx, func(tx *gorm.DB) (bool, error) {
		// fast-path: create new run
		r := &actionRun{
			DedupKey:     dedupKey,
			EventType:    eventType,
			Topic:        "workflow-actions",
			ProjectID:    projectID,
			StateID:      sid,
			TransitionID: tid,
			Trigger:      trigger,
			ActionID:     action.ID,
			Status:       "running",
			Attempts:     1,
			CreatedAt:    now,
			UpdatedAt:    now,
		}
		if err := tx.WithContext(ctx).Create(r).Error; err == nil {
			return true, nil
		} else if !isUniqueViolation(err) {
			return false, err
		}

		// exists: lock row
		var ex actionRun
		if err := tx.WithContext(ctx).
			Clauses(clause.Locking{Strength: "UPDATE"}).
			First(&ex, "dedup_key = ?", dedupKey).Error; err != nil {
			return false, err
		}

		switch ex.Status {
		case "succeeded":
			return false, nil
		case "running":
			// if not stale: another worker is running
			if ex.UpdatedAt.After(now.Add(-w.runningLease)) {
				return false, nil
			}
			// stale -> allow retry
		case "failed":
			// allow retry if attempts < maxRetries
		default:
			// unknown -> treat as retryable
		}

		if ex.Attempts >= maxRetries {
			// give up
			return false, nil
		}

		// resume
		update := map[string]interface{}{
			"status":     "running",
			"attempts":   ex.Attempts + 1,
			"updated_at": now,
			"last_error": nil,
		}
		if err := tx.WithContext(ctx).
			Model(&actionRun{}).
			Where("id = ?", ex.ID).
			Updates(update).Error; err != nil {
			return false, err
		}
		return true, nil
	})
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

	msg := "action failed"
	if res.Error != nil {
		msg = res.Error.Error()
	}
	_ = w.markFailed(ctx, dedupKey, msg)

	if res.ShouldRetry {
		return status.Errorf(codes.Unavailable, "retryable action failed: %s", msg)
	}
	w.logger.Warn("Non-retryable action failed (will not requeue outbox)",
		zap.String("dedup_key", dedupKey),
		zap.Int64("action_id", action.ID),
		zap.String("action_type", action.Type),
		zap.String("reason", msg),
	)
	return nil

}

func (w *Worker) markSucceeded(ctx context.Context, dedupKey string) error {
	now := time.Now().UTC()
	return w.db.WithContext(ctx).
		Model(&actionRun{}).
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
		Model(&actionRun{}).
		Where("dedup_key = ?", dedupKey).
		Updates(map[string]interface{}{
			"status":     "failed",
			"updated_at": now,
			"last_error": &errMsg,
		}).Error
}

func (w *Worker) withTx(ctx context.Context, fn func(tx *gorm.DB) (bool, error)) (bool, error) {
	var ok bool
	err := w.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		v, e := fn(tx)
		if e != nil {
			return e
		}
		ok = v
		return nil
	})
	return ok, err
}

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

func parseConfig(b []byte) map[string]interface{} {
	if len(b) == 0 {
		return map[string]interface{}{}
	}
	var m map[string]interface{}
	if err := json.Unmarshal(b, &m); err != nil || m == nil {
		return map[string]interface{}{}
	}
	return m
}

func makeDedupKey(kind string, projectID, transitionID int64, trigger string, actionID int64, extra string) string {
	raw := fmt.Sprintf("%s|p=%d|t=%d|tr=%s|a=%d|x=%s", kind, projectID, transitionID, trigger, actionID, extra)
	sum := sha1.Sum([]byte(raw))
	return hex.EncodeToString(sum[:])
}

func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505"
	}
	s := err.Error()
	return strings.Contains(s, "duplicate key") ||
		strings.Contains(s, "unique constraint") ||
		strings.Contains(s, "UNIQUE constraint")
}
