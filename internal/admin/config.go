package admin

type Config struct {
	Env      string `mapstructure:"env"`
	GRPCPort string `mapstructure:"grpc_port"`
	Database struct {
		DSN string `mapstructure:"dsn"`
	} `mapstructure:"database"`
	Services struct {
		AuthAddr     string `mapstructure:"auth_addr"`
		ProjectAddr  string `mapstructure:"project_addr"`
		TeamAddr     string `mapstructure:"team_addr"`
		WorkflowAddr string `mapstructure:"workflow_addr"`
	} `mapstructure:"services"`
}
