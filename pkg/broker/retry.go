package broker

import (
	"fmt"
	"math/rand"
	"time"

	"go.uber.org/zap"
)

// RetryConfig defines exponential backoff retry settings for Kafka init.
type RetryConfig struct {
	MaxWait      time.Duration
	StartBackoff time.Duration
	MaxBackoff   time.Duration
	Jitter       float64 // 0..1, e.g. 0.2 means +/-20% jitter
}

func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxWait:      90 * time.Second,
		StartBackoff: 1 * time.Second,
		MaxBackoff:   10 * time.Second,
		Jitter:       0.2,
	}
}

func (c RetryConfig) normalize() RetryConfig {
	out := c
	if out.MaxWait <= 0 {
		out.MaxWait = 90 * time.Second
	}
	if out.StartBackoff <= 0 {
		out.StartBackoff = 1 * time.Second
	}
	if out.MaxBackoff <= 0 {
		out.MaxBackoff = 10 * time.Second
	}
	if out.MaxBackoff < out.StartBackoff {
		out.MaxBackoff = out.StartBackoff
	}
	if out.Jitter < 0 {
		out.Jitter = 0
	}
	if out.Jitter > 1 {
		out.Jitter = 1
	}
	return out
}

func applyJitter(d time.Duration, jitter float64) time.Duration {
	if d <= 0 || jitter <= 0 {
		return d
	}
	// jitter range: [1-jitter, 1+jitter]
	factor := 1 - jitter + (rand.Float64() * 2 * jitter)
	return time.Duration(float64(d) * factor)
}

// NewProducerWithRetry creates Kafka producer with exponential backoff.
// It keeps NewProducer() behavior intact and adds resilient startup option.
func NewProducerWithRetry(brokers []string, log *zap.Logger, cfg RetryConfig) (*Producer, error) {
	cfg = cfg.normalize()
	deadline := time.Now().Add(cfg.MaxWait)

	backoff := cfg.StartBackoff
	var lastErr error

	for time.Now().Before(deadline) {
		p, err := NewProducer(brokers)
		if err == nil {
			if lastErr != nil && log != nil {
				log.Info("Kafka producer created after retries")
			}
			return p, nil
		}
		lastErr = err

		if log != nil {
			log.Warn("Kafka producer not ready yet, retrying",
				zap.Error(err),
				zap.Strings("brokers", brokers),
				zap.Duration("backoff", backoff),
			)
		}

		time.Sleep(applyJitter(backoff, cfg.Jitter))
		backoff *= 2
		if backoff > cfg.MaxBackoff {
			backoff = cfg.MaxBackoff
		}
	}

	return nil, fmt.Errorf("failed to create kafka producer after %s: %w", cfg.MaxWait, lastErr)
}

// NewConsumerWithRetry creates Kafka consumer with exponential backoff.
// It keeps NewConsumer() behavior intact and adds resilient startup option.
func NewConsumerWithRetry(brokers []string, groupID string, log *zap.Logger, cfg RetryConfig) (*Consumer, error) {
	cfg = cfg.normalize()
	deadline := time.Now().Add(cfg.MaxWait)

	backoff := cfg.StartBackoff
	var lastErr error

	for time.Now().Before(deadline) {
		c, err := NewConsumer(brokers, groupID, log)
		if err == nil {
			if lastErr != nil && log != nil {
				log.Info("Kafka consumer created after retries", zap.String("group_id", groupID))
			}
			return c, nil
		}
		lastErr = err

		if log != nil {
			log.Warn("Kafka consumer not ready yet, retrying",
				zap.Error(err),
				zap.Strings("brokers", brokers),
				zap.String("group_id", groupID),
				zap.Duration("backoff", backoff),
			)
		}

		time.Sleep(applyJitter(backoff, cfg.Jitter))
		backoff *= 2
		if backoff > cfg.MaxBackoff {
			backoff = cfg.MaxBackoff
		}
	}

	return nil, fmt.Errorf("failed to create kafka consumer after %s: %w", cfg.MaxWait, lastErr)
}

func init() {
}
