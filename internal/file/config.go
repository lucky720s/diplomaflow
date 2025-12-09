package file

type Config struct {
	Env      string `mapstructure:"env"`
	GRPCPort string `mapstructure:"grpc_port"`
	Storage  struct {
		Path    string `mapstructure:"path"`
		BaseURL string `mapstructure:"base_url"`
	} `mapstructure:"storage"`
}
