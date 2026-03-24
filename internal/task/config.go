package task

type Config struct {
	Env      string         `mapstructure:"env"`
	GRPCPort string         `mapstructure:"grpc_port"`
	Database DatabaseConfig `mapstructure:"database"`
	Services ServicesConfig `mapstructure:"services"`
}

type DatabaseConfig struct {
	DSN string `mapstructure:"dsn"`
}

type ServicesConfig struct {
	AuthAddr         string `mapstructure:"auth_addr"`
	TeamAddr         string `mapstructure:"team_addr"`
	NotificationAddr string `mapstructure:"notification_addr"`
	FileAddr         string `mapstructure:"file_addr"`
}
