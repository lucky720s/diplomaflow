package builtin

import (
	"github.com/lucky720s/diplomaflow/internal/workflow/plugins"
	notificationv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/notification/v1"
	teamv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/team/v1"
)

// RegisterAll регистрирует все встроенные плагины
func RegisterAll(notifClient notificationv1.NotificationServiceClient, teamClient teamv1.TeamServiceClient) {
	// Notifications
	if notifClient != nil {
		plugins.Register(NewNotificationPlugin(notifClient))
	}
	plugins.Register(NewEmailPlugin())
	plugins.Register(NewReminderPlugin())

	// Team
	if teamClient != nil {
		plugins.Register(NewTeamLockPlugin(teamClient))
	}

	// External checks
	plugins.Register(NewAntiplagiatPlugin())
	plugins.Register(NewTurnitinPlugin())

	// Validation
	plugins.Register(NewFileValidationPlugin())
	plugins.Register(NewFormValidationPlugin())

	// Grading
	plugins.Register(NewGradeCalculationPlugin())

	// Documents
	plugins.Register(NewDocumentGeneratorPlugin())

	// Webhooks
	plugins.Register(NewWebhookPlugin())
}

func RegisteredPlugins() []string {
	list := plugins.List()
	ids := make([]string, len(list))
	for i, p := range list {
		ids[i] = p.ID()
	}
	return ids
}
