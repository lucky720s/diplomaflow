package admin

type Config struct {
	Env      string `mapstructure:"env"`
	GRPCPort string `mapstructure:"grpc_port"`
	Database struct {
		DSN string `mapstructure:"dsn"`
	} `mapstructure:"database"`
	Services struct {
		AuthAddr         string `mapstructure:"auth_addr"`
		ProjectAddr      string `mapstructure:"project_addr"`
		TeamAddr         string `mapstructure:"team_addr"`
		WorkflowAddr     string `mapstructure:"workflow_addr"`
		NotificationAddr string `mapstructure:"notification_addr"`
		// FileAddr is used by norm-control to resolve file metadata for the
		// unified file contract embedded in document responses.
		FileAddr string `mapstructure:"file_addr"`
	} `mapstructure:"services"`
}
