package builtin

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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
	return "Отправляет документ на проверку оригинальности"
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
				"title": "API ключ",
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

func (p *AntiplagiatPlugin) Execute(ctx context.Context, actx *plugins.ActionContext) *plugins.ActionResult {
	// Получаем конфиг
	serviceURL := "https://api.antiplagiat.ru"
	if url, ok := actx.Config["service_url"].(string); ok && url != "" {
		serviceURL = url
	}

	documentField := "diploma_file_id"
	if df, ok := actx.Config["document_field"].(string); ok && df != "" {
		documentField = df
	}

	minScore := actx.Config["min_score"].(float64)

	// Получаем файл из данных проекта
	fileID, ok := actx.ProjectData[documentField].(string)
	if !ok {
		// Пробуем как int
		if fid, ok := actx.ProjectData[documentField].(float64); ok {
			fileID = fmt.Sprint(int64(fid))
		} else {
			return &plugins.ActionResult{
				Success: false,
				Error:   fmt.Errorf("document not found in project data (field: %s)", documentField),
			}
		}
	}

	// Формируем запрос
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
			RetryAfter:  300, // 5 минут
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
			"project_id": actx.ProjectID,
		},
	}
}

func (p *AntiplagiatPlugin) Rollback(ctx context.Context, actx *plugins.ActionContext) error {
	return nil // Нельзя откатить внешнюю проверку
}
