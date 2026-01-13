package broker

import (
	"encoding/json"
	"fmt"

	"github.com/IBM/sarama"
)

type Producer struct {
	producer sarama.SyncProducer
}

func NewProducer(brokers []string) (*Producer, error) {
	cfg := sarama.NewConfig()

	// Важно: одинаковая версия у producer/consumer снижает сюрпризы.
	cfg.Version = sarama.V2_8_0_0

	// Надёжная доставка
	cfg.Producer.Return.Successes = true
	cfg.Producer.RequiredAcks = sarama.WaitForAll
	cfg.Producer.Retry.Max = 10

	// Idempotent producer (уменьшает риск дублей при ретраях)
	// Требования Sarama: WaitForAll + Retry.Max>0 + MaxOpenRequests=1
	cfg.Producer.Idempotent = true
	cfg.Net.MaxOpenRequests = 1

	p, err := sarama.NewSyncProducer(brokers, cfg)
	if err != nil {
		return nil, err
	}

	return &Producer{producer: p}, nil
}

// Publish — совместимый метод: payload может быть struct/map/[]byte/json.RawMessage.
func (p *Producer) Publish(topic string, eventType string, payload interface{}) error {
	return p.PublishWithKey(topic, "", eventType, payload)
}

// PublishWithKey — полезно для order per aggregate (project_id как key).
func (p *Producer) PublishWithKey(topic string, key string, eventType string, payload interface{}) error {
	raw, err := toRawJSON(payload)
	if err != nil {
		return err
	}

	val, err := json.Marshal(Event{
		Type:    eventType,
		Payload: raw,
	})
	if err != nil {
		return fmt.Errorf("failed to marshal event envelope: %w", err)
	}

	msg := &sarama.ProducerMessage{
		Topic: topic,
		Value: sarama.ByteEncoder(val),
	}
	if key != "" {
		msg.Key = sarama.StringEncoder(key)
	}

	_, _, err = p.producer.SendMessage(msg)
	return err
}

func (p *Producer) Close() error {
	return p.producer.Close()
}

func toRawJSON(v interface{}) (json.RawMessage, error) {
	switch t := v.(type) {
	case nil:
		return json.RawMessage("null"), nil
	case json.RawMessage:
		if len(t) == 0 {
			return json.RawMessage("null"), nil
		}
		return t, nil
	case []byte:
		if len(t) == 0 {
			return json.RawMessage("null"), nil
		}
		return json.RawMessage(t), nil
	case string:
		if t == "" {
			return json.RawMessage(`""`), nil
		}
		// строка — это строка JSON, а не “готовый JSON”
		b, err := json.Marshal(t)
		if err != nil {
			return nil, fmt.Errorf("marshal string payload: %w", err)
		}
		return json.RawMessage(b), nil
	default:
		b, err := json.Marshal(v)
		if err != nil {
			return nil, fmt.Errorf("marshal payload: %w", err)
		}
		return json.RawMessage(b), nil
	}
}
