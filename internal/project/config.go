package project

type Config struct {
	Env      string `mapstructure:"env"`
	GRPCPort string `mapstructure:"grpc_port"`
	Database struct {
		DSN string `mapstructure:"dsn"`
	} `mapstructure:"database"`
	Services struct {
		WorkflowAddr     string `mapstructure:"workflow_addr"`
		NotificationAddr string `mapstructure:"notification_addr"`
	} `mapstructure:"services"`
}
