package broker

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/IBM/sarama"
	"go.uber.org/zap"
)

type Event struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}

type ConsumerHandler func(ctx context.Context, event Event) error

type Consumer struct {
	logger  *zap.Logger
	client  sarama.ConsumerGroup
	handler ConsumerHandler
	topics  []string
	ready   chan bool
}

func NewConsumer(brokers []string, groupID string, log *zap.Logger) (*Consumer, error) {
	config := sarama.NewConfig()
	config.Version = sarama.V2_8_0_0
	config.Consumer.Group.Rebalance.Strategy = sarama.NewBalanceStrategyRoundRobin()
	config.Consumer.Offsets.Initial = sarama.OffsetOldest
	config.Consumer.Return.Errors = true

	client, err := sarama.NewConsumerGroup(brokers, groupID, config)
	if err != nil {
		return nil, err
	}

	return &Consumer{
		logger: log,
		client: client,
		ready:  make(chan bool),
	}, nil
}

func (c *Consumer) Start(ctx context.Context, topics []string, handler ConsumerHandler) {
	c.handler = handler
	c.topics = topics

	go func() {
		for {
			if err := c.client.Consume(ctx, topics, c); err != nil {
				if errors.Is(err, sarama.ErrClosedConsumerGroup) {
					return
				}
				c.logger.Error("Error from consumer", zap.Error(err))
				time.Sleep(2 * time.Second)
			}
			if ctx.Err() != nil {
				return
			}
			c.ready = make(chan bool)
		}
	}()

	<-c.ready
	c.logger.Info("Sarama consumer up and running!...")
}

func (c *Consumer) Close() error {
	return c.client.Close()
}

func (c *Consumer) Setup(sarama.ConsumerGroupSession) error {
	close(c.ready)
	return nil
}

func (c *Consumer) Cleanup(sarama.ConsumerGroupSession) error {
	return nil
}

func (c *Consumer) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for {
		select {
		case message, ok := <-claim.Messages():
			if !ok {
				return nil
			}

			var event Event
			if err := json.Unmarshal(message.Value, &event); err != nil {
				c.logger.Error("Failed to unmarshal event", zap.Error(err))
				session.MarkMessage(message, "")
				continue
			}

			c.logger.Info("Message claimed", zap.String("topic", message.Topic), zap.Int64("offset", message.Offset))

			if err := c.handler(session.Context(), event); err != nil {
				c.logger.Error("Failed to handle event", zap.Error(err))
				continue
			}

			session.MarkMessage(message, "")

		case <-session.Context().Done():
			return nil
		}
	}
}
