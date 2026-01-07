package config

import (
	"time"
)

// Config represents the application configuration
type Config struct {
	Server      ServerConfig      `yaml:"server"`
	Logging     LoggingConfig     `yaml:"logging"`
	MongoDB     MongoDBConfig     `yaml:"mongodb"`
	CSMS        CSMSConfig        `yaml:"csms"`
	Application ApplicationConfig `yaml:"application"`
	Auth        AuthConfig        `yaml:"auth"`
}

// AuthConfig holds authentication configuration
type AuthConfig struct {
	Enabled   bool           `yaml:"enabled" env:"OCPP_EMU_AUTH_ENABLED" env-default:"false"`
	JWTSecret string         `yaml:"jwt_secret" env:"OCPP_EMU_AUTH_JWT_SECRET"`
	JWTExpiry time.Duration  `yaml:"jwt_expiry" env:"OCPP_EMU_AUTH_JWT_EXPIRY" env-default:"24h"`
	Users     []UserConfig   `yaml:"users"`
	APIKeys   []APIKeyConfig `yaml:"api_keys"`
}

// UserConfig represents a user in configuration
type UserConfig struct {
	Username     string `yaml:"username"`
	PasswordHash string `yaml:"password_hash"`
	Role         string `yaml:"role"`
	Enabled      bool   `yaml:"enabled"`
}

// APIKeyConfig represents an API key in configuration
type APIKeyConfig struct {
	Name      string `yaml:"name"`
	KeyHash   string `yaml:"key_hash"`
	Role      string `yaml:"role"`
	Enabled   bool   `yaml:"enabled"`
	ExpiresAt string `yaml:"expires_at,omitempty"`
}

// ServerConfig holds HTTP server configuration
type ServerConfig struct {
	Port int       `yaml:"port" env:"SERVER_PORT" env-default:"8080"`
	Host string    `yaml:"host" env:"SERVER_HOST" env-default:"0.0.0.0"`
	TLS  TLSConfig `yaml:"tls"`
}

// TLSConfig holds TLS configuration
type TLSConfig struct {
	Enabled  bool   `yaml:"enabled" env:"SERVER_TLS_ENABLED" env-default:"false"`
	CertFile string `yaml:"cert_file" env:"SERVER_TLS_CERT_FILE"`
	KeyFile  string `yaml:"key_file" env:"SERVER_TLS_KEY_FILE"`
}

// LoggingConfig holds logging configuration
type LoggingConfig struct {
	Level  string `yaml:"level" env:"OCPP_EMU_LOG_LEVEL" env-default:"info"`     // debug, info, warn, error
	Format string `yaml:"format" env:"OCPP_EMU_LOG_FORMAT" env-default:"text"`   // json or text
	Output string `yaml:"output" env:"OCPP_EMU_LOG_OUTPUT" env-default:"stdout"` // stdout, stderr, or file path
}

// MongoDBConfig holds MongoDB connection configuration
type MongoDBConfig struct {
	URI               string                   `yaml:"uri" env:"MONGODB_URI" env-default:"mongodb://localhost:27017"`
	Database          string                   `yaml:"database" env:"MONGODB_DATABASE" env-default:"ocpp_emu"`
	ConnectionTimeout time.Duration            `yaml:"connection_timeout" env:"MONGODB_CONNECTION_TIMEOUT" env-default:"10s"`
	MaxPoolSize       uint64                   `yaml:"max_pool_size" env:"MONGODB_MAX_POOL_SIZE" env-default:"100"`
	Collections       MongoDBCollectionsConfig `yaml:"collections"`
	TimeSeries        MongoDBTimeSeriesConfig  `yaml:"timeseries"`
}

// MongoDBCollectionsConfig holds collection names
type MongoDBCollectionsConfig struct {
	Messages     string `yaml:"messages" env-default:"messages"`
	Transactions string `yaml:"transactions" env-default:"transactions"`
	Stations     string `yaml:"stations" env-default:"stations"`
	Sessions     string `yaml:"sessions" env-default:"sessions"`
	MeterValues  string `yaml:"meter_values" env-default:"meter_values"`
}

// MongoDBTimeSeriesConfig holds time-series configuration
type MongoDBTimeSeriesConfig struct {
	Enabled     bool   `yaml:"enabled" env-default:"true"`
	Granularity string `yaml:"granularity" env-default:"seconds"` // seconds, minutes, hours
}

// CSMSConfig holds CSMS connection configuration
type CSMSConfig struct {
	DefaultURL           string        `yaml:"default_url" env:"CSMS_DEFAULT_URL" env-default:"ws://localhost:9000"`
	ConnectionTimeout    time.Duration `yaml:"connection_timeout" env:"CSMS_CONNECTION_TIMEOUT" env-default:"30s"`
	HeartbeatInterval    time.Duration `yaml:"heartbeat_interval" env:"CSMS_HEARTBEAT_INTERVAL" env-default:"60s"`
	MaxReconnectAttempts int           `yaml:"max_reconnect_attempts" env:"CSMS_MAX_RECONNECT_ATTEMPTS" env-default:"5"`
	ReconnectBackoff     time.Duration `yaml:"reconnect_backoff" env:"CSMS_RECONNECT_BACKOFF" env-default:"10s"`
	TLS                  TLSCSMSConfig `yaml:"tls"`
}

// TLSCSMSConfig holds TLS configuration for CSMS connections
type TLSCSMSConfig struct {
	Enabled            bool   `yaml:"enabled" env:"CSMS_TLS_ENABLED" env-default:"false"`
	CACert             string `yaml:"ca_cert" env:"CSMS_TLS_CA_CERT"`
	ClientCert         string `yaml:"client_cert" env:"CSMS_TLS_CLIENT_CERT"`
	ClientKey          string `yaml:"client_key" env:"CSMS_TLS_CLIENT_KEY"`
	InsecureSkipVerify bool   `yaml:"insecure_skip_verify" env:"CSMS_TLS_INSECURE_SKIP_VERIFY" env-default:"false"`
}

// ApplicationConfig holds application-level configuration
type ApplicationConfig struct {
	MaxStations         int           `yaml:"max_stations" env:"OCPP_EMU_MAX_STATIONS" env-default:"10"`
	CacheTTL            time.Duration `yaml:"cache_ttl" env:"OCPP_EMU_CACHE_TTL" env-default:"3600s"`
	DebugMode           bool          `yaml:"debug_mode" env:"OCPP_EMU_DEBUG_MODE" env-default:"true"`
	MessageBufferSize   int           `yaml:"message_buffer_size" env:"OCPP_EMU_MESSAGE_BUFFER_SIZE" env-default:"1000"`
	BatchInsertInterval time.Duration `yaml:"batch_insert_interval" env:"OCPP_EMU_BATCH_INSERT_INTERVAL" env-default:"5s"`
}
