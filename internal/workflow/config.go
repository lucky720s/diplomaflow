package workflow

type Config struct {
	Env      string `mapstructure:"env"`
	GRPCPort string `mapstructure:"grpc_port"`

	Database struct {
		DSN string `mapstructure:"dsn"`
	} `mapstructure:"database"`

	Services struct {
		NotificationAddr string `mapstructure:"notification_addr"`
		ProjectAddr      string `mapstructure:"project_addr"`
		TeamAddr         string `mapstructure:"team_addr"`
	} `mapstructure:"services"`

	OutboxPoller struct {
		Enabled bool   `mapstructure:"enabled"`
		Topic   string `mapstructure:"topic"`

		IntervalSeconds int `mapstructure:"interval_seconds"`
		BatchSize       int `mapstructure:"batch_size"`

		Table             string `mapstructure:"table"`
		IDColumn          string `mapstructure:"id_column"`
		TopicColumn       string `mapstructure:"topic_column"`
		StatusColumn      string `mapstructure:"status_column"`
		EventTypeColumn   string `mapstructure:"event_type_column"`
		PayloadColumn     string `mapstructure:"payload_column"`
		ProcessedAtColumn string `mapstructure:"processed_at_column"`

		PendingStatus    string `mapstructure:"pending_status"`
		ProcessingStatus string `mapstructure:"processing_status"` // NEW
		ProcessedStatus  string `mapstructure:"processed_status"`
	} `mapstructure:"outbox_poller"`
}
