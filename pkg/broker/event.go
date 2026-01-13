package broker

import "encoding/json"

// Event — единый envelope для Kafka.
// Payload — json.RawMessage, чтобы:
// 1) не было двойного marshal/unmarshal,
// 2) consumer мог валидировать схему,
// 3) не терялись типы при повторном marshal.
type Event struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`

	// optional metadata (backward compatible)
	Version    int    `json:"version,omitempty"`
	EventID    string `json:"event_id,omitempty"`
	OccurredAt string `json:"occurred_at,omitempty"`
}
