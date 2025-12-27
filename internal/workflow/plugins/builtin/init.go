package builtin

import (
	"github.com/lucky720s/diplomaflow/internal/workflow/plugins"
	notificationv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/notification/v1"
)

// RegisterAll регистрирует все встроенные плагины
func RegisterAll(notifClient notificationv1.NotificationServiceClient) {
	// Уведомления
	if notifClient != nil {
		plugins.Register(NewNotificationPlugin(notifClient))
	}
	plugins.Register(NewEmailPlugin())
	plugins.Register(NewReminderPlugin())

	// Внешние проверки
	plugins.Register(NewAntiplagiatPlugin())
	plugins.Register(NewTurnitinPlugin())

	// Валидация
	plugins.Register(NewFileValidationPlugin())
	plugins.Register(NewFormValidationPlugin())

	// Оценивание
	plugins.Register(NewGradeCalculationPlugin())

	// Документы
	plugins.Register(NewDocumentGeneratorPlugin())

	// Webhooks
	plugins.Register(NewWebhookPlugin())
}

// RegisterWithoutNotification регистрирует плагины без notification client
func RegisterWithoutNotification() {
	plugins.Register(NewEmailPlugin())
	plugins.Register(NewReminderPlugin())
	plugins.Register(NewAntiplagiatPlugin())
	plugins.Register(NewTurnitinPlugin())
	plugins.Register(NewFileValidationPlugin())
	plugins.Register(NewFormValidationPlugin())
	plugins.Register(NewGradeCalculationPlugin())
	plugins.Register(NewDocumentGeneratorPlugin())
	plugins.Register(NewWebhookPlugin())
}

// RegisteredPlugins возвращает список ID зарегистрированных плагинов
func RegisteredPlugins() []string {
	list := plugins.List()
	ids := make([]string, len(list))
	for i, p := range list {
		ids[i] = p.ID()
	}
	return ids
}
