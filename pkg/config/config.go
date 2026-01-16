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
	RoleServiceAddr         string   `mapstructure:"services_role_addr"`
	WorkflowServiceAddr     string   `mapstructure:"services_workflow_addr"`
	NotificationServiceAddr string   `mapstructure:"services_notification_addr"`
	FileServiceAddr         string   `mapstructure:"services_file_addr"`
	FormServiceAddr         string   `mapstructure:"services_form_addr"`
	AdminServiceAddr        string   `mapstructure:"services_admin_addr"`
	RedisAddr               string   `mapstructure:"redis_addr"`
	AllowedOrigins          []string `mapstructure:"allowed_origins"`
}

func Load(path string, cfg interface{}) error {
	v := viper.New()
	v.SetConfigFile(path)
	v.SetConfigType("yaml")

	// env support
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	if err := v.ReadInConfig(); err != nil {
		// для gateway конфиг файл обычно существует, но сделаем поведение мягким
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return fmt.Errorf("read config: %w", err)
		}
	}

	// env overrides (явно)
	_ = v.BindEnv("env", "ENV")
	_ = v.BindEnv("port", "PORT")
	_ = v.BindEnv("jwt_secret", "JWT_SECRET")

	// ===== Services: поддерживаем новый стандарт SERVICES_* и старый *_SERVICE_ADDR =====
	_ = v.BindEnv("services_auth_addr", "SERVICES_AUTH_ADDR", "AUTH_SERVICE_ADDR")
	_ = v.BindEnv("services_project_addr", "SERVICES_PROJECT_ADDR", "PROJECT_SERVICE_ADDR")
	_ = v.BindEnv("services_team_addr", "SERVICES_TEAM_ADDR", "TEAM_SERVICE_ADDR")
	_ = v.BindEnv("services_university_addr", "SERVICES_UNIVERSITY_ADDR", "UNIVERSITY_SERVICE_ADDR")
	_ = v.BindEnv("services_role_addr", "SERVICES_ROLE_ADDR", "ROLE_SERVICE_ADDR")
	_ = v.BindEnv("services_workflow_addr", "SERVICES_WORKFLOW_ADDR", "WORKFLOW_SERVICE_ADDR")
	_ = v.BindEnv("services_notification_addr", "SERVICES_NOTIFICATION_ADDR", "NOTIFICATION_SERVICE_ADDR")
	_ = v.BindEnv("services_file_addr", "SERVICES_FILE_ADDR", "FILE_SERVICE_ADDR")
	_ = v.BindEnv("services_form_addr", "SERVICES_FORM_ADDR", "FORM_SERVICE_ADDR")
	_ = v.BindEnv("services_admin_addr", "SERVICES_ADMIN_ADDR", "ADMIN_SERVICE_ADDR")

	_ = v.BindEnv("redis_addr", "REDIS_ADDR")

	if err := v.Unmarshal(cfg); err != nil {
		return fmt.Errorf("unmarshal config: %w", err)
	}
	return nil
}
