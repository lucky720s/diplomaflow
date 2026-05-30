package file

type Config struct {
	Env      string `mapstructure:"env"`
	GRPCPort string `mapstructure:"grpc_port"`
	Storage  struct {
		Path    string `mapstructure:"path"`
		BaseURL string `mapstructure:"base_url"`
		// MaxUploadBytes is the single upload size limit (default 50 MB).
		// Keep this aligned with nginx client_max_body_size and the gateway
		// MaxMultipartMemory so the limit is uniform across the product.
		MaxUploadBytes int64 `mapstructure:"max_upload_bytes"`
	} `mapstructure:"storage"`
	Database struct {
		DSN string `mapstructure:"dsn"`
	} `mapstructure:"database"`
}

// DefaultMaxUploadBytes is the product-wide upload size limit (50 MB).
const DefaultMaxUploadBytes int64 = 50 << 20

// MaxUploadBytes returns the configured limit or the default if unset.
func (c *Config) MaxUploadBytes() int64 {
	if c.Storage.MaxUploadBytes > 0 {
		return c.Storage.MaxUploadBytes
	}
	return DefaultMaxUploadBytes
}
