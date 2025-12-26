package config

import (
	"strings"

	"github.com/spf13/viper"
)

func Load(path string, cfg interface{}) error {
	viper.SetConfigFile(path)
	viper.SetConfigType("yaml")
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return err
		}
	}
	viper.BindEnv("database.dsn", "DATABASE_DSN")
	viper.BindEnv("jwt.secret", "JWT_SECRET")
	viper.BindEnv("kafka.brokers", "KAFKA_BROKERS")
	viper.BindEnv("grpc_port", "GRPC_PORT")
	viper.BindEnv("port", "PORT")
	viper.BindEnv("services.workflow_addr", "SERVICES_WORKFLOW_ADDR")
	viper.BindEnv("services.notification_addr", "SERVICES_NOTIFICATION_ADDR")
	viper.BindEnv("services.auth_addr", "SERVICES_AUTH_ADDR")
	viper.BindEnv("services.project_addr", "SERVICES_PROJECT_ADDR")
	viper.BindEnv("services.team_addr", "SERVICES_TEAM_ADDR")
	viper.BindEnv("services.university_addr", "SERVICES_UNIVERSITY_ADDR")
	viper.BindEnv("services.role_addr", "SERVICES_ROLE_ADDR")
	viper.BindEnv("services.file_addr", "SERVICES_FILE_ADDR")
	viper.BindEnv("services.form_addr", "SERVICES_FORM_ADDR")

	viper.BindEnv("jwt.access_token_ttl", "JWT_ACCESS_TOKEN_TTL")
	viper.BindEnv("jwt.refresh_token_ttl", "JWT_REFRESH_TOKEN_TTL")
	viper.BindEnv("redis_addr", "REDIS_ADDR")

	viper.BindEnv("services.admin_addr", "SERVICES_ADMIN_ADDR")

	return viper.Unmarshal(cfg)
}
