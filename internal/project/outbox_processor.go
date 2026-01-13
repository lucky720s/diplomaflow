package project

import (
	"context"
	"encoding/json"
	"time"

	"github.com/lucky720s/diplomaflow/pkg/broker"
	"go.uber.org/zap"
)

type OutboxProcessor struct {
	repo     Repository
	producer *broker.Producer
	logger   *zap.Logger
	stopCh   chan struct{}
}

func NewOutboxProcessor(repo Repository, producer *broker.Producer, logger *zap.Logger) *OutboxProcessor {
	return &OutboxProcessor{repo: repo, producer: producer, logger: logger, stopCh: make(chan struct{})}
}

func (p *OutboxProcessor) Start(ctx context.Context) {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-p.stopCh:
			return
		case <-ticker.C:
			p.process(ctx)
		}
	}
}

func (p *OutboxProcessor) process(ctx context.Context) {
	events, err := p.repo.GetPendingEvents(ctx, 10)
	if err != nil {
		p.logger.Error("fetch pending events failed", zap.Error(err))
		return
	}

	for _, ev := range events {
		// ev.Payload уже JSON
		raw := json.RawMessage(ev.Payload)

		// Совет: если у вас есть aggregate_id — публикуйте с key для order per aggregate.
		if err := p.producer.Publish(ev.Topic, ev.EventType, raw); err != nil {
			p.logger.Error("publish failed", zap.Int64("event_id", ev.ID), zap.Error(err))
			continue
		}

		if err := p.repo.MarkEventProcessed(ctx, ev.ID); err != nil {
			p.logger.Error("mark processed failed", zap.Int64("event_id", ev.ID), zap.Error(err))
		}
	}
}

func (p *OutboxProcessor) Stop() { close(p.stopCh) }
