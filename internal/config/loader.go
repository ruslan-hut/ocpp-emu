package config

import (
	"fmt"
	"os"

	"github.com/ilyakaznacheev/cleanenv"
)

// Load loads the configuration from the config file and environment variables
func Load(configPath string) (*Config, error) {
	var cfg Config

	// Determine config file path
	path := configPath
	if path == "" {
		// Try default locations
		defaultPaths := []string{
			"./configs/config.yaml",
			"./config.yaml",
		}
		for _, p := range defaultPaths {
			if _, err := os.Stat(p); err == nil {
				path = p
				break
			}
		}
	}

	// Load configuration
	if path != "" {
		// Load from file with environment variable overrides
		if err := cleanenv.ReadConfig(path, &cfg); err != nil {
			return nil, fmt.Errorf("failed to read config file %s: %w", path, err)
		}
	} else {
		// Load from environment variables only (with defaults)
		if err := cleanenv.ReadEnv(&cfg); err != nil {
			return nil, fmt.Errorf("failed to read environment config: %w", err)
		}
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
