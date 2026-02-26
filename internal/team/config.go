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
		AdminAddr        string `mapstructure:"admin_addr"`
	} `mapstructure:"services"`
}
