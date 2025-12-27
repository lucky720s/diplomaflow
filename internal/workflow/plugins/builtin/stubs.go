package builtin

import (
	"context"
	"encoding/json"

	"github.com/lucky720s/diplomaflow/internal/workflow/plugins"
)

// =============== Email Plugin ===============
type EmailPlugin struct{}

func NewEmailPlugin() *EmailPlugin { return &EmailPlugin{} }

func (p *EmailPlugin) ID() string   { return "SEND_EMAIL" }
func (p *EmailPlugin) Name() string { return "Отправить email" }
func (p *EmailPlugin) Description() string {
	return "Отправляет email пользователям"
}
func (p *EmailPlugin) Category() string   { return plugins.CategoryNotification }
func (p *EmailPlugin) IsReversible() bool { return false }

func (p *EmailPlugin) ConfigSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"required": ["subject", "body"],
		"properties": {
			"subject": {"type": "string", "title": "Тема письма"},
			"body": {"type": "string", "title": "Текст письма"},
			"recipients": {"type": "string", "enum": ["student", "team", "supervisor"]}
		}
	}`)
}

func (p *EmailPlugin) Validate(config map[string]interface{}) error { return nil }
func (p *EmailPlugin) Execute(ctx context.Context, actx *plugins.ActionContext) *plugins.ActionResult {
	return &plugins.ActionResult{Success: true, Data: map[string]interface{}{"status": "sent"}}
}
func (p *EmailPlugin) Rollback(ctx context.Context, actx *plugins.ActionContext) error { return nil }

// =============== Reminder Plugin ===============
type ReminderPlugin struct{}

func NewReminderPlugin() *ReminderPlugin { return &ReminderPlugin{} }

func (p *ReminderPlugin) ID() string   { return "SCHEDULE_REMINDER" }
func (p *ReminderPlugin) Name() string { return "Запланировать напоминание" }
func (p *ReminderPlugin) Description() string {
	return "Создаёт отложенное напоминание"
}
func (p *ReminderPlugin) Category() string   { return plugins.CategoryNotification }
func (p *ReminderPlugin) IsReversible() bool { return true }

func (p *ReminderPlugin) ConfigSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"days_before_deadline": {"type": "integer", "default": 3},
			"message": {"type": "string"}
		}
	}`)
}

func (p *ReminderPlugin) Validate(config map[string]interface{}) error { return nil }
func (p *ReminderPlugin) Execute(ctx context.Context, actx *plugins.ActionContext) *plugins.ActionResult {
	return &plugins.ActionResult{Success: true}
}
func (p *ReminderPlugin) Rollback(ctx context.Context, actx *plugins.ActionContext) error { return nil }

// =============== Turnitin Plugin ===============
type TurnitinPlugin struct{}

func NewTurnitinPlugin() *TurnitinPlugin { return &TurnitinPlugin{} }

func (p *TurnitinPlugin) ID() string   { return "CHECK_TURNITIN" }
func (p *TurnitinPlugin) Name() string { return "Проверка Turnitin" }
func (p *TurnitinPlugin) Description() string {
	return "Отправляет документ на проверку в Turnitin"
}
func (p *TurnitinPlugin) Category() string   { return plugins.CategoryExternal }
func (p *TurnitinPlugin) IsReversible() bool { return false }

func (p *TurnitinPlugin) ConfigSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"api_key": {"type": "string"},
			"min_score": {"type": "number", "default": 70}
		}
	}`)
}

func (p *TurnitinPlugin) Validate(config map[string]interface{}) error { return nil }
func (p *TurnitinPlugin) Execute(ctx context.Context, actx *plugins.ActionContext) *plugins.ActionResult {
	return &plugins.ActionResult{Success: true}
}
func (p *TurnitinPlugin) Rollback(ctx context.Context, actx *plugins.ActionContext) error { return nil }

// =============== File Validation Plugin ===============
type FileValidationPlugin struct{}

func NewFileValidationPlugin() *FileValidationPlugin { return &FileValidationPlugin{} }

func (p *FileValidationPlugin) ID() string   { return "VALIDATE_FILES" }
func (p *FileValidationPlugin) Name() string { return "Валидация файлов" }
func (p *FileValidationPlugin) Description() string {
	return "Проверяет загруженные файлы"
}
func (p *FileValidationPlugin) Category() string   { return plugins.CategoryValidation }
func (p *FileValidationPlugin) IsReversible() bool { return false }

func (p *FileValidationPlugin) ConfigSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"required_files": {"type": "array", "items": {"type": "string"}},
			"max_size_mb": {"type": "number", "default": 50},
			"allowed_extensions": {"type": "array", "items": {"type": "string"}}
		}
	}`)
}

func (p *FileValidationPlugin) Validate(config map[string]interface{}) error { return nil }
func (p *FileValidationPlugin) Execute(ctx context.Context, actx *plugins.ActionContext) *plugins.ActionResult {
	return &plugins.ActionResult{Success: true}
}
func (p *FileValidationPlugin) Rollback(ctx context.Context, actx *plugins.ActionContext) error {
	return nil
}

// =============== Form Validation Plugin ===============
type FormValidationPlugin struct{}

func NewFormValidationPlugin() *FormValidationPlugin { return &FormValidationPlugin{} }

func (p *FormValidationPlugin) ID() string   { return "VALIDATE_FORM" }
func (p *FormValidationPlugin) Name() string { return "Валидация формы" }
func (p *FormValidationPlugin) Description() string {
	return "Проверяет данные формы"
}
func (p *FormValidationPlugin) Category() string   { return plugins.CategoryValidation }
func (p *FormValidationPlugin) IsReversible() bool { return false }

func (p *FormValidationPlugin) ConfigSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"form_id": {"type": "string"},
			"required_fields": {"type": "array", "items": {"type": "string"}}
		}
	}`)
}

func (p *FormValidationPlugin) Validate(config map[string]interface{}) error { return nil }
func (p *FormValidationPlugin) Execute(ctx context.Context, actx *plugins.ActionContext) *plugins.ActionResult {
	return &plugins.ActionResult{Success: true}
}
func (p *FormValidationPlugin) Rollback(ctx context.Context, actx *plugins.ActionContext) error {
	return nil
}

// =============== Grade Calculation Plugin ===============
type GradeCalculationPlugin struct{}

func NewGradeCalculationPlugin() *GradeCalculationPlugin { return &GradeCalculationPlugin{} }

func (p *GradeCalculationPlugin) ID() string   { return "CALCULATE_GRADE" }
func (p *GradeCalculationPlugin) Name() string { return "Расчёт оценки" }
func (p *GradeCalculationPlugin) Description() string {
	return "Вычисляет итоговую оценку"
}
func (p *GradeCalculationPlugin) Category() string   { return plugins.CategoryGrading }
func (p *GradeCalculationPlugin) IsReversible() bool { return true }

func (p *GradeCalculationPlugin) ConfigSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"scale": {"type": "string", "enum": ["5", "100", "ECTS"], "default": "5"},
			"passing_score": {"type": "number", "default": 50},
			"components": {
				"type": "array",
				"items": {
					"type": "object",
					"properties": {
						"name": {"type": "string"},
						"weight": {"type": "number"}
					}
				}
			}
		}
	}`)
}

func (p *GradeCalculationPlugin) Validate(config map[string]interface{}) error { return nil }
func (p *GradeCalculationPlugin) Execute(ctx context.Context, actx *plugins.ActionContext) *plugins.ActionResult {
	return &plugins.ActionResult{Success: true, Data: map[string]interface{}{"grade": 0}}
}
func (p *GradeCalculationPlugin) Rollback(ctx context.Context, actx *plugins.ActionContext) error {
	return nil
}

// =============== Document Generator Plugin ===============
type DocumentGeneratorPlugin struct{}

func NewDocumentGeneratorPlugin() *DocumentGeneratorPlugin { return &DocumentGeneratorPlugin{} }

func (p *DocumentGeneratorPlugin) ID() string   { return "GENERATE_DOCUMENT" }
func (p *DocumentGeneratorPlugin) Name() string { return "Генерация документа" }
func (p *DocumentGeneratorPlugin) Description() string {
	return "Генерирует документ по шаблону"
}
func (p *DocumentGeneratorPlugin) Category() string   { return plugins.CategoryDocument }
func (p *DocumentGeneratorPlugin) IsReversible() bool { return true }

func (p *DocumentGeneratorPlugin) ConfigSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"template_id": {"type": "string"},
			"output_format": {"type": "string", "enum": ["pdf", "docx"], "default": "pdf"}
		}
	}`)
}

func (p *DocumentGeneratorPlugin) Validate(config map[string]interface{}) error { return nil }
func (p *DocumentGeneratorPlugin) Execute(ctx context.Context, actx *plugins.ActionContext) *plugins.ActionResult {
	return &plugins.ActionResult{Success: true}
}
func (p *DocumentGeneratorPlugin) Rollback(ctx context.Context, actx *plugins.ActionContext) error {
	return nil
}

// =============== Webhook Plugin ===============
type WebhookPlugin struct{}

func NewWebhookPlugin() *WebhookPlugin { return &WebhookPlugin{} }

func (p *WebhookPlugin) ID() string   { return "CALL_WEBHOOK" }
func (p *WebhookPlugin) Name() string { return "Вызов webhook" }
func (p *WebhookPlugin) Description() string {
	return "Отправляет HTTP запрос на внешний URL"
}
func (p *WebhookPlugin) Category() string   { return plugins.CategoryWebhook }
func (p *WebhookPlugin) IsReversible() bool { return false }

func (p *WebhookPlugin) ConfigSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"required": ["url"],
		"properties": {
			"url": {"type": "string", "title": "URL"},
			"method": {"type": "string", "enum": ["GET", "POST", "PUT"], "default": "POST"},
			"headers": {"type": "object"},
			"body_template": {"type": "string"}
		}
	}`)
}

func (p *WebhookPlugin) Validate(config map[string]interface{}) error { return nil }
func (p *WebhookPlugin) Execute(ctx context.Context, actx *plugins.ActionContext) *plugins.ActionResult {
	return &plugins.ActionResult{Success: true}
}
func (p *WebhookPlugin) Rollback(ctx context.Context, actx *plugins.ActionContext) error { return nil }
