package realtime

import (
	"context"
	"encoding/json"

	"github.com/redis/go-redis/v9"
)

// Publisher публикует realtime-события. Реализации: Redis и no-op.
type Publisher interface {
	Publish(ctx context.Context, ev Event) error
}

type redisPublisher struct {
	rdb *redis.Client
}

func (p *redisPublisher) Publish(ctx context.Context, ev Event) error {
	data, err := json.Marshal(ev)
	if err != nil {
		return err
	}
	return p.rdb.Publish(ctx, Channel, data).Err()
}

// nopPublisher используется, когда Redis не сконфигурирован (локально/тесты/CI):
// publish превращается в no-op, чтобы основная операция не зависела от realtime.
type nopPublisher struct{}

func (nopPublisher) Publish(context.Context, Event) error { return nil }

// NewPublisher создаёт Redis-публишер по адресу. Если addr пуст — no-op.
// Возвращает cleanup для закрытия клиента.
func NewPublisher(addr string) (Publisher, func(), error) {
	if addr == "" {
		return nopPublisher{}, func() {}, nil
	}
	rdb := redis.NewClient(&redis.Options{Addr: addr})
	return &redisPublisher{rdb: rdb}, func() { _ = rdb.Close() }, nil
}
