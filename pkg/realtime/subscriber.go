package realtime

import (
	"context"
	"encoding/json"

	"github.com/redis/go-redis/v9"
)

// Subscriber читает realtime-события из Redis Pub/Sub. Используется gateway.
type Subscriber struct {
	rdb    *redis.Client
	pubsub *redis.PubSub
}

// NewSubscriber подписывается на realtime-канал.
func NewSubscriber(rdb *redis.Client) *Subscriber {
	return &Subscriber{rdb: rdb}
}

// Events запускает чтение сообщений и отдаёт распарсенные события в канал.
// Канал закрывается при отмене ctx или закрытии подписки.
func (s *Subscriber) Events(ctx context.Context) <-chan Event {
	s.pubsub = s.rdb.Subscribe(ctx, Channel)
	raw := s.pubsub.Channel()
	out := make(chan Event, 64)

	go func() {
		defer close(out)
		for {
			select {
			case <-ctx.Done():
				return
			case msg, ok := <-raw:
				if !ok {
					return
				}
				var ev Event
				if err := json.Unmarshal([]byte(msg.Payload), &ev); err != nil {
					continue
				}
				select {
				case out <- ev:
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	return out
}

func (s *Subscriber) Close() error {
	if s.pubsub != nil {
		return s.pubsub.Close()
	}
	return nil
}
