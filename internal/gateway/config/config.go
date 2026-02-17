package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	Env                     string   `mapstructure:"env"`
	Port                    string   `mapstructure:"port"`
	JWTSecret               string   `mapstructure:"jwt_secret"`
	AuthServiceAddr         string   `mapstructure:"services_auth_addr"`
	ProjectServiceAddr      string   `mapstructure:"services_project_addr"`
	TeamServiceAddr         string   `mapstructure:"services_team_addr"`
	UniversityServiceAddr   string   `mapstructure:"services_university_addr"`
	WorkflowServiceAddr     string   `mapstructure:"services_workflow_addr"`
	NotificationServiceAddr string   `mapstructure:"services_notification_addr"`
	FileServiceAddr         string   `mapstructure:"services_file_addr"`
	FormServiceAddr         string   `mapstructure:"services_form_addr"`
	AdminServiceAddr        string   `mapstructure:"services_admin_addr"`
	RedisAddr               string   `mapstructure:"redis_addr"`
	TaskServiceAddr         string   `mapstructure:"services_task_addr"`
	AllowedOrigins          []string `mapstructure:"allowed_origins"`
}

func Load(path string, cfg interface{}) error {
	v := viper.New()
	v.SetConfigFile(path)
	v.SetConfigType("yaml")

	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return fmt.Errorf("read config: %w", err)
		}
	}

	// env overrides (явно)
	_ = v.BindEnv("env", "ENV")
	_ = v.BindEnv("port", "PORT")
	_ = v.BindEnv("jwt_secret", "JWT_SECRET")

	_ = v.BindEnv("services_auth_addr", "SERVICES_AUTH_ADDR")
	_ = v.BindEnv("services_project_addr", "SERVICES_PROJECT_ADDR")
	_ = v.BindEnv("services_team_addr", "SERVICES_TEAM_ADDR")
	_ = v.BindEnv("services_task_addr", "SERVICES_TASK_ADDR")
	_ = v.BindEnv("services_university_addr", "SERVICES_UNIVERSITY_ADDR")
	_ = v.BindEnv("services_workflow_addr", "SERVICES_WORKFLOW_ADDR")
	_ = v.BindEnv("services_notification_addr", "SERVICES_NOTIFICATION_ADDR")
	_ = v.BindEnv("services_file_addr", "SERVICES_FILE_ADDR")
	_ = v.BindEnv("services_form_addr", "SERVICES_FORM_ADDR")
	_ = v.BindEnv("services_admin_addr", "SERVICES_ADMIN_ADDR")

	_ = v.BindEnv("redis_addr", "REDIS_ADDR")

	if err := v.Unmarshal(cfg); err != nil {
		return fmt.Errorf("unmarshal config: %w", err)
	}
	return nil
}
