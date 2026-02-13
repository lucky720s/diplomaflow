package outboxpoller

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/lucky720s/diplomaflow/internal/workflow/postcommit"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type Config struct {
	Enabled bool

	Topic           string
	IntervalSeconds int
	BatchSize       int

	Table             string
	IDColumn          string
	TopicColumn       string
	StatusColumn      string
	EventTypeColumn   string
	PayloadColumn     string
	ProcessedAtColumn string // optional

	PendingStatus    string
	ProcessingStatus string // NEW (optional). Default "processing"
	ProcessedStatus  string
}

type Poller struct {
	db     *gorm.DB
	worker *postcommit.Worker
	log    *zap.Logger
	cfg    Config
}

func New(db *gorm.DB, worker *postcommit.Worker, log *zap.Logger, cfg Config) (*Poller, error) {
	if db == nil || worker == nil || log == nil {
		return nil, fmt.Errorf("db/worker/log required")
	}
	if cfg.Topic == "" {
		cfg.Topic = "workflow-actions"
	}
	if cfg.IntervalSeconds <= 0 {
		cfg.IntervalSeconds = 2
	}
	if cfg.BatchSize <= 0 {
		cfg.BatchSize = 20
	}
	if cfg.Table == "" {
		cfg.Table = "outbox_events"
	}
	if cfg.IDColumn == "" {
		cfg.IDColumn = "id"
	}
	if cfg.TopicColumn == "" {
		cfg.TopicColumn = "topic"
	}
	if cfg.StatusColumn == "" {
		cfg.StatusColumn = "status"
	}
	if cfg.EventTypeColumn == "" {
		cfg.EventTypeColumn = "event_type"
	}
	if cfg.PayloadColumn == "" {
		cfg.PayloadColumn = "payload"
	}
	if cfg.PendingStatus == "" {
		cfg.PendingStatus = "pending"
	}
	if cfg.ProcessingStatus == "" {
		cfg.ProcessingStatus = "processing"
	}
	if cfg.ProcessedStatus == "" {
		cfg.ProcessedStatus = "processed"
	}

	// minimal injection protection for identifiers (from config.yaml)
	for _, ident := range []string{
		cfg.Table, cfg.IDColumn, cfg.TopicColumn, cfg.StatusColumn, cfg.EventTypeColumn, cfg.PayloadColumn, cfg.ProcessedAtColumn,
	} {
		if ident == "" {
			continue
		}
		if !isSafeIdent(ident) {
			return nil, fmt.Errorf("unsafe identifier in poller config: %q", ident)
		}
	}

	return &Poller{db: db, worker: worker, log: log, cfg: cfg}, nil
}

var safeIdentRe = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

func isSafeIdent(s string) bool {
	return safeIdentRe.MatchString(s)
}

type outboxRow struct {
	ID        int64
	EventType string
	Payload   []byte
}

func (p *Poller) Start(ctx context.Context) {
	if !p.cfg.Enabled {
		p.log.Info("Outbox poller disabled")
		return
	}

	ticker := time.NewTicker(time.Duration(p.cfg.IntervalSeconds) * time.Second)
	defer ticker.Stop()

	p.log.Info("Outbox poller started",
		zap.String("topic", p.cfg.Topic),
		zap.Int("batch_size", p.cfg.BatchSize),
		zap.Int("interval_seconds", p.cfg.IntervalSeconds),
	)

	for {
		select {
		case <-ctx.Done():
			p.log.Info("Outbox poller stopped (context done)")
			return
		case <-ticker.C:
			if err := p.tick(ctx); err != nil {
				p.log.Warn("Outbox poller tick failed", zap.Error(err))
			}
		}
	}
}

func (p *Poller) tick(ctx context.Context) error {
	_ = p.requeueStuck(ctx, 10*time.Minute)
	rows, err := p.fetchAndClaimBatch(ctx)
	if err != nil {
		return err
	}
	if len(rows) == 0 {
		return nil
	}

	for _, r := range rows {
		if r.EventType == "" || len(r.Payload) == 0 {
			p.log.Warn("Skipping bad outbox row", zap.Int64("id", r.ID))
			// return row to pending so it can be inspected/repaired
			_ = p.setStatus(ctx, r.ID, p.cfg.PendingStatus)
			continue
		}

		if err := p.worker.Handle(ctx, r.EventType, r.Payload); err != nil {
			// return to pending to retry later
			p.log.Warn("Workflow action event handle failed (will retry)",
				zap.Int64("outbox_id", r.ID),
				zap.String("event_type", r.EventType),
				zap.Error(err),
			)
			_ = p.setStatus(ctx, r.ID, p.cfg.PendingStatus)
			continue
		}

		if err := p.markProcessed(ctx, r.ID); err != nil {
			p.log.Warn("Failed to mark outbox processed", zap.Int64("outbox_id", r.ID), zap.Error(err))
		}
	}

	return nil
}

// fetchAndClaimBatch selects rows with SKIP LOCKED AND immediately updates status->processing inside same TX.
// This prevents duplicates across multiple workflow_service instances.
func (p *Poller) fetchAndClaimBatch(ctx context.Context) ([]outboxRow, error) {
	var out []outboxRow

	err := p.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		q := fmt.Sprintf(
			`SELECT %s AS id, %s AS event_type, %s AS payload
			   FROM %s
			  WHERE %s = ? AND %s = ?
			  ORDER BY %s ASC
			  LIMIT %d
			  FOR UPDATE SKIP LOCKED`,
			p.cfg.IDColumn,
			p.cfg.EventTypeColumn,
			p.cfg.PayloadColumn,
			p.cfg.Table,
			p.cfg.TopicColumn,
			p.cfg.StatusColumn,
			p.cfg.IDColumn,
			p.cfg.BatchSize,
		)

		if err := tx.Raw(q, p.cfg.Topic, p.cfg.PendingStatus).Scan(&out).Error; err != nil {
			return err
		}
		if len(out) == 0 {
			return nil
		}

		ids := make([]int64, 0, len(out))
		for _, r := range out {
			ids = append(ids, r.ID)
		}

		uq := fmt.Sprintf(
			`UPDATE %s SET %s = ?, processing_started_at = NOW() WHERE %s IN ?`,
			p.cfg.Table, p.cfg.StatusColumn, p.cfg.IDColumn,
		)
		return tx.Exec(uq, p.cfg.ProcessingStatus, ids).Error

	})

	return out, err
}

func (p *Poller) setStatus(ctx context.Context, id int64, statusVal string) error {
	q := fmt.Sprintf(`UPDATE %s SET %s = ? WHERE %s = ?`, p.cfg.Table, p.cfg.StatusColumn, p.cfg.IDColumn)
	return p.db.WithContext(ctx).Exec(q, statusVal, id).Error
}

func (p *Poller) markProcessed(ctx context.Context, id int64) error {
	if strings.TrimSpace(p.cfg.ProcessedAtColumn) != "" {
		q := fmt.Sprintf(`UPDATE %s SET %s = ?, %s = NOW() WHERE %s = ?`,
			p.cfg.Table, p.cfg.StatusColumn, p.cfg.ProcessedAtColumn, p.cfg.IDColumn)
		if err := p.db.WithContext(ctx).Exec(q, p.cfg.ProcessedStatus, id).Error; err == nil {
			return nil
		}
	}
	q := fmt.Sprintf(`UPDATE %s SET %s = ? WHERE %s = ?`, p.cfg.Table, p.cfg.StatusColumn, p.cfg.IDColumn)
	return p.db.WithContext(ctx).Exec(q, p.cfg.ProcessedStatus, id).Error
}
func (p *Poller) requeueStuck(ctx context.Context, maxAge time.Duration) error {
	q := fmt.Sprintf(
		`UPDATE %s
		    SET %s = ?, processing_started_at = NULL
		  WHERE %s = ?
		    AND processing_started_at IS NOT NULL
		    AND processing_started_at < NOW() - INTERVAL '%d seconds'`,
		p.cfg.Table,
		p.cfg.StatusColumn,
		p.cfg.StatusColumn,
		int(maxAge.Seconds()),
	)
	return p.db.WithContext(ctx).Exec(q, p.cfg.PendingStatus, p.cfg.ProcessingStatus).Error
}
