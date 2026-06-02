package notification

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/lucky720s/diplomaflow/pkg/logger"
	"go.uber.org/zap"
)

// Pusher отправляет push-уведомления на устройства.
//
// Абстракция намеренно простая: один метод. Реальная доставка (FCM) и заглушка
// (noop) реализуют один интерфейс, что позволяет сервису работать одинаково
// и в проде с ключом, и локально/в CI без секретов.
type Pusher interface {
	Push(ctx context.Context, tokens []string, title, body, link string) error
}

// noopPusher используется, когда FCM не сконфигурирован: только логирует.
// Это сохраняет сервис рабочим без секретов (локальная разработка, тесты, CI).
type noopPusher struct {
	log *logger.Logger
}

func (p *noopPusher) Push(_ context.Context, tokens []string, title, _, _ string) error {
	if len(tokens) == 0 {
		return nil
	}
	p.log.Debug("push skipped: FCM not configured",
		zap.Int("tokens", len(tokens)), zap.String("title", title))
	return nil
}

// fcmPusher шлёт уведомления через FCM HTTP API по server key.
// Зависимостей не добавляет — только стандартная библиотека.
type fcmPusher struct {
	endpoint  string
	serverKey string
	client    *http.Client
	log       *logger.Logger
}

func (p *fcmPusher) Push(ctx context.Context, tokens []string, title, body, link string) error {
	// FCM legacy не принимает массовую отправку >1000; для дипломного объёма
	// шлём по одному токену — проще и достаточно.
	for _, tok := range tokens {
		if tok == "" {
			continue
		}
		payload := map[string]interface{}{
			"to": tok,
			"notification": map[string]string{
				"title": title,
				"body":  body,
			},
			"data": map[string]string{
				"link": link,
			},
		}
		buf, err := json.Marshal(payload)
		if err != nil {
			return err
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.endpoint, bytes.NewReader(buf))
		if err != nil {
			return err
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "key="+p.serverKey)

		resp, err := p.client.Do(req)
		if err != nil {
			// Сетевые ошибки логируем, но не валим всю рассылку.
			p.log.Warn("fcm push failed", zap.String("token", tok), zap.Error(err))
			continue
		}
		_ = resp.Body.Close()
		if resp.StatusCode >= 300 {
			p.log.Warn("fcm push non-2xx", zap.String("token", tok), zap.Int("status", resp.StatusCode))
		}
	}
	return nil
}

// NewPusher выбирает реализацию по конфигу: если задан server key — FCM,
// иначе безопасный noop.
func NewPusher(cfg *Config, log *logger.Logger) Pusher {
	if cfg.FCM.ServerKey == "" {
		log.Info("FCM server key not set — push notifications disabled (noop)")
		return &noopPusher{log: log}
	}
	endpoint := cfg.FCM.Endpoint
	if endpoint == "" {
		endpoint = "https://fcm.googleapis.com/fcm/send"
	}
	log.Info("FCM push enabled", zap.String("endpoint", endpoint))
	return &fcmPusher{
		endpoint:  endpoint,
		serverKey: cfg.FCM.ServerKey,
		client:    &http.Client{Timeout: 10 * time.Second},
		log:       log,
	}
}
