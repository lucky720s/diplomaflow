package broker

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/IBM/sarama"
	"go.uber.org/zap"
)

// ErrSkip — “poison pill / невалидное событие”: отметить offset и продолжать.
var ErrSkip = errors.New("skip message")

// Permanent(err) — обёртка для ошибок, которые НЕ надо ретраить (ack + log).
type PermanentError struct{ err error }

func (e PermanentError) Error() string { return e.err.Error() }
func (e PermanentError) Unwrap() error { return e.err }

func Permanent(err error) error {
	if err == nil {
		return nil
	}
	return PermanentError{err: err}
}

func isPermanent(err error) bool {
	var pe PermanentError
	return errors.As(err, &pe) || errors.Is(err, ErrSkip)
}

type ConsumerHandler func(ctx context.Context, event Event) error

type Consumer struct {
	logger  *zap.Logger
	client  sarama.ConsumerGroup
	handler ConsumerHandler
	topics  []string

	ready chan struct{}
}

func NewConsumer(brokers []string, groupID string, log *zap.Logger) (*Consumer, error) {
	cfg := sarama.NewConfig()
	cfg.Version = sarama.V2_8_0_0

	cfg.Consumer.Group.Rebalance.Strategy = sarama.NewBalanceStrategyRoundRobin()
	cfg.Consumer.Offsets.Initial = sarama.OffsetOldest
	cfg.Consumer.Return.Errors = true

	// чуть более “production-friendly” таймауты
	cfg.Consumer.Group.Session.Timeout = 30 * time.Second
	cfg.Consumer.Group.Heartbeat.Interval = 3 * time.Second

	client, err := sarama.NewConsumerGroup(brokers, groupID, cfg)
	if err != nil {
		return nil, err
	}

	return &Consumer{
		logger: log,
		client: client,
		ready:  make(chan struct{}),
	}, nil
}

// Start запускает consume loop в горутине и блокируется до первого успешного Setup.
// Важно: если ctx отменён — вернёт управление.
func (c *Consumer) Start(ctx context.Context, topics []string, handler ConsumerHandler) {
	c.handler = handler
	c.topics = topics

	go c.consumeLoop(ctx)

	select {
	case <-c.ready:
		c.logger.Info("Sarama consumer ready", zap.Strings("topics", topics))
	case <-ctx.Done():
		c.logger.Warn("Sarama consumer start canceled", zap.Error(ctx.Err()))
		return
	}
}

func (c *Consumer) consumeLoop(ctx context.Context) {
	backoff := 2 * time.Second

	for {
		if err := c.client.Consume(ctx, c.topics, c); err != nil {
			if errors.Is(err, sarama.ErrClosedConsumerGroup) {
				return
			}
			c.logger.Error("Consumer error", zap.Error(err))
			time.Sleep(backoff)
		}

		if ctx.Err() != nil {
			return
		}

		// новый ready для следующей сессии (переребаланс и т.п.)
		c.ready = make(chan struct{})
	}
}

func (c *Consumer) Close() error {
	return c.client.Close()
}

func (c *Consumer) Setup(sarama.ConsumerGroupSession) error {
	// сигнализируем, что consumer готов
	select {
	case <-c.ready:
		// уже закрыт
	default:
		close(c.ready)
	}
	return nil
}

func (c *Consumer) Cleanup(sarama.ConsumerGroupSession) error {
	return nil
}

func (c *Consumer) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for {
		select {
		case msg, ok := <-claim.Messages():
			if !ok {
				return nil
			}

			var ev Event
			if err := json.Unmarshal(msg.Value, &ev); err != nil {
				// невалидный envelope — это poison pill, его надо ack, иначе будет вечный цикл
				c.logger.Error("Failed to unmarshal event envelope (skip)", zap.Error(err))
				session.MarkMessage(msg, "")
				continue
			}

			c.logger.Info("Message claimed",
				zap.String("topic", msg.Topic),
				zap.Int32("partition", msg.Partition),
				zap.Int64("offset", msg.Offset),
				zap.String("type", ev.Type),
			)

			if c.handler == nil {
				c.logger.Error("Consumer handler is nil (skip)")
				session.MarkMessage(msg, "")
				continue
			}

			err := c.handler(session.Context(), ev)
			if err == nil {
				session.MarkMessage(msg, "")
				continue
			}

			if isPermanent(err) {
				c.logger.Error("Permanent handler error (ack+skip)", zap.Error(err))
				session.MarkMessage(msg, "")
				continue
			}

			// retryable error: НЕ отмечаем offset, Kafka будет ретраить
			c.logger.Error("Handler failed (will retry)", zap.Error(err))

		case <-session.Context().Done():
			return nil
		}
	}
}
