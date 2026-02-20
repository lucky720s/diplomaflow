package builtin

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/lucky720s/diplomaflow/internal/workflow/plugins"
)

type AntiplagiatPlugin struct {
	httpClient *http.Client
}

func NewAntiplagiatPlugin() *AntiplagiatPlugin {
	return &AntiplagiatPlugin{
		httpClient: &http.Client{Timeout: 60 * time.Second},
	}
}

func (p *AntiplagiatPlugin) ID() string   { return "CHECK_ANTIPLAGIAT" }
func (p *AntiplagiatPlugin) Name() string { return "Проверка на антиплагиат" }
func (p *AntiplagiatPlugin) Description() string {
	return "Отправляет документ на проверку оригинальности. При отсутствии API-ключа возвращает mock-результат."
}
func (p *AntiplagiatPlugin) Category() string   { return plugins.CategoryExternal }
func (p *AntiplagiatPlugin) IsReversible() bool { return false }

func (p *AntiplagiatPlugin) ConfigSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"required": ["min_score"],
		"properties": {
			"service_url": {
				"type": "string",
				"title": "URL сервиса",
				"default": "https://api.antiplagiat.ru"
			},
			"api_key": {
				"type": "string",
				"title": "API ключ (пусто = mock режим)",
				"format": "password"
			},
			"min_score": {
				"type": "number",
				"title": "Минимальный % оригинальности",
				"minimum": 0,
				"maximum": 100,
				"default": 70
			},
			"auto_reject": {
				"type": "boolean",
				"title": "Авто-отклонение при низком %",
				"default": false
			},
			"document_field": {
				"type": "string",
				"title": "Поле с ID документа в данных проекта",
				"default": "diploma_file_id"
			},
			"callback_url": {
				"type": "string",
				"title": "URL для callback (опционально)"
			},
			"mock_mode": {
				"type": "string",
				"title": "Режим mock: auto (если нет ключа), always, never",
				"enum": ["auto", "always", "never"],
				"default": "auto"
			}
		}
	}`)
}

func (p *AntiplagiatPlugin) Validate(config map[string]interface{}) error {
	if _, ok := config["min_score"]; !ok {
		return fmt.Errorf("min_score is required")
	}
	minScore, ok := config["min_score"].(float64)
	if !ok || minScore < 0 || minScore > 100 {
		return fmt.Errorf("min_score must be between 0 and 100")
	}
	return nil
}

// shouldMock определяет, нужно ли использовать mock-режим.
// Три стратегии через config["mock_mode"]:
//   - "auto"   (default): mock если api_key пустой
//   - "always": всегда mock (для dev/staging)
//   - "never":  никогда mock (для prod, упадёт если нет ключа)
func (p *AntiplagiatPlugin) shouldMock(actx *plugins.ActionContext) bool {
	mode := "auto"
	if m, ok := actx.Config["mock_mode"].(string); ok && m != "" {
		mode = strings.ToLower(strings.TrimSpace(m))
	}

	switch mode {
	case "always":
		return true
	case "never":
		return false
	default: // "auto"
		apiKey, _ := actx.Config["api_key"].(string)
		return strings.TrimSpace(apiKey) == ""
	}
}

func (p *AntiplagiatPlugin) executeMock(actx *plugins.ActionContext) *plugins.ActionResult {
	minScore := 70.0
	if ms, ok := actx.Config["min_score"].(float64); ok {
		minScore = ms
	}

	// Детерминированный seed на основе project_id для воспроизводимости в тестах,
	// но с добавлением времени для реалистичности
	src := rand.NewSource(actx.ProjectID*31 + time.Now().UnixNano()%1000)
	rng := rand.New(src)

	// Score в диапазоне 70-95 (реалистичный для дипломных работ)
	score := 70 + rng.Intn(26)

	checkStatus := "pass"
	if float64(score) < minScore {
		checkStatus = "fail"
	}

	autoReject, _ := actx.Config["auto_reject"].(bool)

	return &plugins.ActionResult{
		Success: checkStatus == "pass" || !autoReject,
		Data: map[string]interface{}{
			"score":      score,
			"min_score":  minScore,
			"status":     checkStatus,
			"report_url": fmt.Sprintf("http://mock-antiplagiat/reports/project-%d", actx.ProjectID),
			"check_id":   fmt.Sprintf("mock-%d-%d", actx.ProjectID, time.Now().Unix()),
			"service":    "antiplagiat",
			"mock":       true,
			"project_id": actx.ProjectID,
			"checked_at": time.Now().UTC().Format(time.RFC3339),
			"details": map[string]interface{}{
				"sources_found":   rng.Intn(5),
				"internet_score":  score - rng.Intn(10),
				"database_score":  score + rng.Intn(6),
				"processing_time": fmt.Sprintf("%dms", 200+rng.Intn(800)),
			},
		},
	}
}

func (p *AntiplagiatPlugin) Execute(ctx context.Context, actx *plugins.ActionContext) *plugins.ActionResult {
	// ===== MOCK CHECK (гибкий, конфигурируемый) =====
	if p.shouldMock(actx) {
		return p.executeMock(actx)
	}

	// ===== REAL API CALL =====
	serviceURL := "https://api.antiplagiat.ru"
	if url, ok := actx.Config["service_url"].(string); ok && url != "" {
		serviceURL = url
	}

	documentField := "diploma_file_id"
	if df, ok := actx.Config["document_field"].(string); ok && df != "" {
		documentField = df
	}

	minScore, _ := actx.Config["min_score"].(float64)

	// Получаем файл из данных проекта
	fileID, ok := actx.ProjectData[documentField].(string)
	if !ok {
		if fid, ok := actx.ProjectData[documentField].(float64); ok {
			fileID = fmt.Sprint(int64(fid))
		} else {
			return &plugins.ActionResult{
				Success: false,
				Error:   fmt.Errorf("document not found in project data (field: %s)", documentField),
			}
		}
	}

	callbackURL := fmt.Sprintf("/api/v1/webhooks/antiplagiat/projects/%d", actx.ProjectID)
	if cb, ok := actx.Config["callback_url"].(string); ok && cb != "" {
		callbackURL = cb
	}

	requestBody := map[string]interface{}{
		"file_id":      fileID,
		"project_id":   actx.ProjectID,
		"callback_url": callbackURL,
		"min_score":    minScore,
	}

	jsonBody, _ := json.Marshal(requestBody)
	req, err := http.NewRequestWithContext(ctx, "POST", serviceURL+"/api/v1/check", bytes.NewBuffer(jsonBody))
	if err != nil {
		return &plugins.ActionResult{Success: false, Error: err, ShouldRetry: true, RetryAfter: 300}
	}

	req.Header.Set("Content-Type", "application/json")
	if apiKey, ok := actx.Config["api_key"].(string); ok {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return &plugins.ActionResult{
			Success:     false,
			Error:       err,
			ShouldRetry: true,
			RetryAfter:  300,
		}
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		return &plugins.ActionResult{
			Success:     false,
			Error:       fmt.Errorf("antiplagiat API error: %s (status: %d)", string(body), resp.StatusCode),
			ShouldRetry: resp.StatusCode >= 500,
			RetryAfter:  300,
		}
	}

	var result map[string]interface{}
	json.Unmarshal(body, &result)

	return &plugins.ActionResult{
		Success: true,
		Data: map[string]interface{}{
			"check_id":   result["check_id"],
			"status":     "pending",
			"min_score":  minScore,
			"service":    "antiplagiat",
			"mock":       false,
			"project_id": actx.ProjectID,
		},
	}
}

func (p *AntiplagiatPlugin) Rollback(ctx context.Context, actx *plugins.ActionContext) error {
	return nil
}
