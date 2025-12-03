package config

import (
	"strings"

	"github.com/spf13/viper"
)

func Load(path string, cfg interface{}) error {
	viper.SetConfigFile(path)
	viper.SetConfigType("yaml")
	_ = viper.ReadInConfig()
	viper.SetEnvPrefix("APP")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()
	_ = viper.BindEnv("database.dsn", "APP_DATABASE_DSN")
	_ = viper.BindEnv("jwt.secret", "APP_JWT_SECRET")
	_ = viper.BindEnv("grpc_port", "APP_GRPC_PORT")
	_ = viper.BindEnv("port", "APP_PORT")

	_ = viper.BindEnv("services.auth_addr", "APP_SERVICES_AUTH_ADDR")
	_ = viper.BindEnv("services.project_addr", "APP_SERVICES_PROJECT_ADDR")
	_ = viper.BindEnv("services.team_addr", "APP_SERVICES_TEAM_ADDR")
	_ = viper.BindEnv("services.university_addr", "APP_SERVICES_UNIVERSITY_ADDR")
	_ = viper.BindEnv("services.role_addr", "APP_SERVICES_ROLE_ADDR")
	_ = viper.BindEnv("services.workflow_addr", "APP_SERVICES_WORKFLOW_ADDR")

	_ = viper.BindEnv("kafka.brokers", "APP_KAFKA_BROKERS")
	_ = viper.BindEnv("kafka.group_id", "APP_KAFKA_GROUP_ID")

	return viper.Unmarshal(cfg)
}
