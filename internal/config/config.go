package config

import (
	"time"
)

// Config represents the application configuration
type Config struct {
	Server      ServerConfig      `mapstructure:"server"`
	Logging     LoggingConfig     `mapstructure:"logging"`
	MongoDB     MongoDBConfig     `mapstructure:"mongodb"`
	CSMS        CSMSConfig        `mapstructure:"csms"`
	Application ApplicationConfig `mapstructure:"application"`
}

// ServerConfig holds HTTP server configuration
type ServerConfig struct {
	Port int       `mapstructure:"port"`
	Host string    `mapstructure:"host"`
	TLS  TLSConfig `mapstructure:"tls"`
}

// TLSConfig holds TLS configuration
type TLSConfig struct {
	Enabled  bool   `mapstructure:"enabled"`
	CertFile string `mapstructure:"cert_file"`
	KeyFile  string `mapstructure:"key_file"`
}

// LoggingConfig holds logging configuration
type LoggingConfig struct {
	Level  string `mapstructure:"level"`  // debug, info, warn, error
	Format string `mapstructure:"format"` // json or text
	Output string `mapstructure:"output"` // stdout, stderr, or file path
}

// MongoDBConfig holds MongoDB connection configuration
type MongoDBConfig struct {
	URI               string                   `mapstructure:"uri"`
	Database          string                   `mapstructure:"database"`
	ConnectionTimeout time.Duration            `mapstructure:"connection_timeout"`
	MaxPoolSize       uint64                   `mapstructure:"max_pool_size"`
	Collections       MongoDBCollectionsConfig `mapstructure:"collections"`
	TimeSeries        MongoDBTimeSeriesConfig  `mapstructure:"timeseries"`
}

// MongoDBCollectionsConfig holds collection names
type MongoDBCollectionsConfig struct {
	Messages     string `mapstructure:"messages"`
	Transactions string `mapstructure:"transactions"`
	Stations     string `mapstructure:"stations"`
	Sessions     string `mapstructure:"sessions"`
	MeterValues  string `mapstructure:"meter_values"`
}

// MongoDBTimeSeriesConfig holds time-series configuration
type MongoDBTimeSeriesConfig struct {
	Enabled     bool   `mapstructure:"enabled"`
	Granularity string `mapstructure:"granularity"` // seconds, minutes, hours
}

// CSMSConfig holds CSMS connection configuration
type CSMSConfig struct {
	DefaultURL           string        `mapstructure:"default_url"`
	ConnectionTimeout    time.Duration `mapstructure:"connection_timeout"`
	HeartbeatInterval    time.Duration `mapstructure:"heartbeat_interval"`
	MaxReconnectAttempts int           `mapstructure:"max_reconnect_attempts"`
	ReconnectBackoff     time.Duration `mapstructure:"reconnect_backoff"`
	TLS                  TLSCSMSConfig `mapstructure:"tls"`
}

// TLSCSMSConfig holds TLS configuration for CSMS connections
type TLSCSMSConfig struct {
	Enabled            bool   `mapstructure:"enabled"`
	CACert             string `mapstructure:"ca_cert"`
	ClientCert         string `mapstructure:"client_cert"`
	ClientKey          string `mapstructure:"client_key"`
	InsecureSkipVerify bool   `mapstructure:"insecure_skip_verify"`
}

// ApplicationConfig holds application-level configuration
type ApplicationConfig struct {
	MaxStations         int           `mapstructure:"max_stations"`
	CacheTTL            time.Duration `mapstructure:"cache_ttl"`
	DebugMode           bool          `mapstructure:"debug_mode"`
	MessageBufferSize   int           `mapstructure:"message_buffer_size"`
	BatchInsertInterval time.Duration `mapstructure:"batch_insert_interval"`
}
