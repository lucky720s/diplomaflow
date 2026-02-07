package task

// Config - конфигурация task_service
type Config struct {
	Env      string `mapstructure:"env"`
	GRPCPort string `mapstructure:"grpc_port"`

	Database struct {
		DSN string `mapstructure:"dsn"`
	} `mapstructure:"database"`

	Services struct {
		AuthAddr         string `mapstructure:"auth_addr"`
		TeamAddr         string `mapstructure:"team_addr"`
		NotificationAddr string `mapstructure:"notification_addr"`
		FileAddr         string `mapstructure:"file_addr"`
	} `mapstructure:"services"`
}
