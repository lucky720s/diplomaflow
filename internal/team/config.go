package team

type Config struct {
	Env      string `mapstructure:"env"`
	GRPCPort string `mapstructure:"grpc_port"`
	Database struct {
		DSN string `mapstructure:"dsn"`
	} `mapstructure:"database"`
	Services struct {
		AuthAddr         string `mapstructure:"auth_addr"`
		NotificationAddr string `mapstructure:"notification_addr"`
		WorkflowAddr     string `mapstructure:"workflow_addr"`
	} `mapstructure:"services"`
	Kafka struct {
		Enabled bool   `mapstructure:"enabled"`
		Brokers string `mapstructure:"brokers"`
		GroupID string `mapstructure:"group_id"`
	} `mapstructure:"kafka"`
}
