package config

import (
	"fmt"
	"os"

	"github.com/spf13/viper"
)

// Load loads the configuration from the config file
func Load(configPath string) (*Config, error) {
	// Set up viper
	v := viper.New()

	// Set config file path
	if configPath != "" {
		v.SetConfigFile(configPath)
	} else {
		// Default config locations
		v.SetConfigName("config")
		v.SetConfigType("yaml")
		v.AddConfigPath("./configs")
		v.AddConfigPath(".")
	}

	// Read environment variables
	v.AutomaticEnv()
	v.SetEnvPrefix("OCPP_EMU")

	// Allow environment variable overrides
	// e.g., OCPP_EMU_MONGODB_URI will override mongodb.uri
	v.BindEnv("mongodb.uri", "MONGODB_URI")
	v.BindEnv("mongodb.database", "MONGODB_DATABASE")
	v.BindEnv("server.port", "SERVER_PORT")
	v.BindEnv("server.host", "SERVER_HOST")

	// Read config file
	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse config into struct
	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Validate configuration
	if err := validate(&cfg); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &cfg, nil
}

// validate performs basic validation on the configuration
func validate(cfg *Config) error {
	// Validate server config
	if cfg.Server.Port <= 0 || cfg.Server.Port > 65535 {
		return fmt.Errorf("invalid server port: %d", cfg.Server.Port)
	}

	// Validate MongoDB config
	if cfg.MongoDB.URI == "" {
		return fmt.Errorf("mongodb.uri is required")
	}

	if cfg.MongoDB.Database == "" {
		return fmt.Errorf("mongodb.database is required")
	}

	// Validate collection names
	if cfg.MongoDB.Collections.Messages == "" {
		return fmt.Errorf("mongodb.collections.messages is required")
	}
	if cfg.MongoDB.Collections.Transactions == "" {
		return fmt.Errorf("mongodb.collections.transactions is required")
	}
	if cfg.MongoDB.Collections.Stations == "" {
		return fmt.Errorf("mongodb.collections.stations is required")
	}
	if cfg.MongoDB.Collections.Sessions == "" {
		return fmt.Errorf("mongodb.collections.sessions is required")
	}
	if cfg.MongoDB.Collections.MeterValues == "" {
		return fmt.Errorf("mongodb.collections.meter_values is required")
	}

	// Validate logging config
	validLevels := map[string]bool{"debug": true, "info": true, "warn": true, "error": true}
	if !validLevels[cfg.Logging.Level] {
		return fmt.Errorf("invalid logging level: %s", cfg.Logging.Level)
	}

	validFormats := map[string]bool{"json": true, "text": true}
	if !validFormats[cfg.Logging.Format] {
		return fmt.Errorf("invalid logging format: %s", cfg.Logging.Format)
	}

	return nil
}

// LoadOrExit loads the configuration and exits on error
func LoadOrExit(configPath string) *Config {
	cfg, err := Load(configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading configuration: %v\n", err)
		os.Exit(1)
	}
	return cfg
}
