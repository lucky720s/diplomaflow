package notification

type Config struct {
	Env      string `mapstructure:"env"`
	GRPCPort string `mapstructure:"grpc_port"`
	Database struct {
		DSN string `mapstructure:"dsn"`
	} `mapstructure:"database"`
	// FCM — конфигурация push-уведомлений. Пусто => push отключён (noop).
	FCM struct {
		ServerKey string `mapstructure:"server_key"`
		Endpoint  string `mapstructure:"endpoint"`
	} `mapstructure:"fcm"`
	// Realtime через Redis Pub/Sub. Пусто => публикация отключена (no-op).
	RedisAddr string `mapstructure:"redis_addr"`
}
