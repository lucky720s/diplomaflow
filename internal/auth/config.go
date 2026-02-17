package auth

type Config struct {
	Env      string `mapstructure:"env"`
	GRPCPort string `mapstructure:"grpc_port"`
	Database struct {
		DSN string `mapstructure:"dsn"`
	} `mapstructure:"database"`
	JWT struct {
		Secret          string `mapstructure:"secret"`
		AccessTokenTTL  string `mapstructure:"access_token_ttl"`
		RefreshTokenTTL string `mapstructure:"refresh_token_ttl"`
	} `mapstructure:"jwt"`
	Services struct {
		UniversityAddr string `mapstructure:"university_addr"`
	} `mapstructure:"services"`
}
