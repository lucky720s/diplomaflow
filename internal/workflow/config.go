package workflow

type Config struct {
	Env      string `mapstructure:"env"`
	GRPCPort string `mapstructure:"grpc_port"`

	Database struct {
		DSN string `mapstructure:"dsn"`
	} `mapstructure:"database"`

	Services struct {
		NotificationAddr string `mapstructure:"notification_addr"`
		ProjectAddr      string `mapstructure:"project_addr"`
	} `mapstructure:"services"`
}
