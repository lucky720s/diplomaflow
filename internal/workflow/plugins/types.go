package plugins

import (
	"context"
	"encoding/json"
)

// ActionPlugin - ЕДИНЫЙ интерфейс для всех плагинов действий
type ActionPlugin interface {
	// Идентификация
	ID() string
	Name() string
	Description() string
	Category() string // notification, validation, external, grading, document, webhook

	// JSON Schema для UI админки
	ConfigSchema() json.RawMessage

	// Валидация конфигурации перед сохранением
	Validate(config map[string]interface{}) error

	// Выполнение действия
	Execute(ctx context.Context, actx *ActionContext) *ActionResult

	// Можно ли отменить действие
	IsReversible() bool

	// Откат действия (если IsReversible = true)
	Rollback(ctx context.Context, actx *ActionContext) error
}

// ActionContext - контекст выполнения действия
type ActionContext struct {
	// Идентификаторы
	ProjectID    int64
	StateID      int64
	UserID       int64
	TeamID       int64
	DepartmentID int64
	UniversityID int64

	// Триггер выполнения
	Trigger string // ON_ENTER, ON_EXIT, ON_DEADLINE, ON_UPDATE, ON_CONDITION

	// Конфигурация действия из БД
	Config map[string]interface{}

	// Данные проекта
	ProjectData map[string]interface{}

	// Информация о переходе
	PreviousState string
	NewState      string
	TransitionID  int64

	// Дополнительные данные
	Metadata map[string]interface{}

	// Payload от пользователя (если есть)
	Payload map[string]interface{}
}

// ActionResult - результат выполнения действия
type ActionResult struct {
	Success bool
	Data    map[string]interface{}
	Error   error

	// Retry logic
	ShouldRetry bool
	RetryAfter  int // секунды до повтора

	// Chaining
	NextActions []string // ID действий которые нужно выполнить после
}

// PluginSchema - схема плагина для UI админки
type PluginSchema struct {
	ID           string          `json:"id"`
	Name         string          `json:"name"`
	Description  string          `json:"description"`
	Category     string          `json:"category"`
	ConfigSchema json.RawMessage `json:"config_schema"`
	IsReversible bool            `json:"is_reversible"`
}

// Trigger constants
const (
	TriggerOnEnter     = "ON_ENTER"
	TriggerOnExit      = "ON_EXIT"
	TriggerOnDeadline  = "ON_DEADLINE"
	TriggerOnUpdate    = "ON_UPDATE"
	TriggerOnCondition = "ON_CONDITION"
	TriggerScheduled   = "SCHEDULED"
)

// Category constants
const (
	CategoryNotification = "notification"
	CategoryValidation   = "validation"
	CategoryExternal     = "external"
	CategoryGrading      = "grading"
	CategoryDocument     = "document"
	CategoryWebhook      = "webhook"
)
