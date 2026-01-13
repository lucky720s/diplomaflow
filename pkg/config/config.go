package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

// Load читает YAML (если есть) + env overrides.
// Без глобального viper: это критично для тестов, повторных вызовов, и чтобы разные сервисы не конфликтовали.
func Load(path string, cfg interface{}) error {
	v := viper.New()

	v.SetConfigFile(path)
	v.SetConfigType("yaml")

	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Конфиг-файл опциональный (k8s обычно только env)
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return fmt.Errorf("read config: %w", err)
		}
	}

	// Явные биндинги: меньше сюрпризов и проще искать, откуда поле взялось.
	_ = v.BindEnv("env", "ENV")
	_ = v.BindEnv("database.dsn", "DATABASE_DSN")

	_ = v.BindEnv("grpc_port", "GRPC_PORT")
	_ = v.BindEnv("port", "PORT")

	_ = v.BindEnv("kafka.brokers", "KAFKA_BROKERS")
	_ = v.BindEnv("kafka.group_id", "KAFKA_GROUP_ID")

	_ = v.BindEnv("jwt.secret", "JWT_SECRET")
	_ = v.BindEnv("jwt.access_token_ttl", "JWT_ACCESS_TOKEN_TTL")
	_ = v.BindEnv("jwt.refresh_token_ttl", "JWT_REFRESH_TOKEN_TTL")

	_ = v.BindEnv("redis_addr", "REDIS_ADDR")

	_ = v.BindEnv("services.workflow_addr", "SERVICES_WORKFLOW_ADDR")
	_ = v.BindEnv("services.notification_addr", "SERVICES_NOTIFICATION_ADDR")
	_ = v.BindEnv("services.auth_addr", "SERVICES_AUTH_ADDR")
	_ = v.BindEnv("services.project_addr", "SERVICES_PROJECT_ADDR")
	_ = v.BindEnv("services.team_addr", "SERVICES_TEAM_ADDR")
	_ = v.BindEnv("services.university_addr", "SERVICES_UNIVERSITY_ADDR")
	_ = v.BindEnv("services.role_addr", "SERVICES_ROLE_ADDR")
	_ = v.BindEnv("services.file_addr", "SERVICES_FILE_ADDR")
	_ = v.BindEnv("services.form_addr", "SERVICES_FORM_ADDR")
	_ = v.BindEnv("services.admin_addr", "SERVICES_ADMIN_ADDR")

	if err := v.Unmarshal(cfg); err != nil {
		return fmt.Errorf("unmarshal config: %w", err)
	}

	return nil
}
