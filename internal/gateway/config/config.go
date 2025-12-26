package config

import (
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
	RedisAddr               string   `mapstructure:"redis_addr"`
	AdminServiceAddr        string   `mapstructure:"services_admin_addr"`
	AllowedOrigins          []string `mapstructure:"allowed_origins"`
}

func Load(path string, cfg interface{}) error {
	viper.SetConfigFile(path)
	viper.SetConfigType("yaml")
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	if err := viper.ReadInConfig(); err != nil {
		return err
	}
	return viper.Unmarshal(cfg)
}
