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
	return &OutboxProcessor{
		repo:     repo,
		producer: producer,
		logger:   logger,
		stopCh:   make(chan struct{}),
	}
}

func (p *OutboxProcessor) Start(ctx context.Context) {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	p.logger.Info("Outbox processor started")
	for {
		select {
		case <-ctx.Done():
			p.logger.Info("Stopping outbox processor via context")
			return
		case <-p.stopCh:
			p.logger.Info("Stopping outbox processor via channel")
			return
		case <-ticker.C:
			p.processEvents(ctx)
		}
	}
}

func (p *OutboxProcessor) processEvents(ctx context.Context) {
	events, err := p.repo.GetPendingEvents(ctx, 10)
	if err != nil {
		p.logger.Error("Failed to fetch pending events", zap.Error(err))
		return
	}
	if len(events) == 0 {
		return
	}

	for _, event := range events {
		payload := json.RawMessage(event.Payload)

		if err := p.producer.Publish(event.Topic, event.EventType, payload); err != nil {
			p.logger.Error("Failed to publish event to kafka",
				zap.Int64("event_id", event.ID),
				zap.String("topic", event.Topic),
				zap.String("event_type", event.EventType),
				zap.Error(err),
			)
			continue
		}

		if err := p.repo.MarkEventProcessed(ctx, event.ID); err != nil {
			p.logger.Error("Failed to mark event as processed",
				zap.Int64("event_id", event.ID),
				zap.Error(err),
			)
		}
	}
}

func (p *OutboxProcessor) Stop() {
	close(p.stopCh)
}
