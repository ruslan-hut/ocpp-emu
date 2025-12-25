package scenario

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

// Loader loads builtin scenarios from files.
type Loader struct {
	storage *Storage
	logger  *slog.Logger
}

// NewLoader creates a new scenario loader.
func NewLoader(storage *Storage, logger *slog.Logger) *Loader {
	return &Loader{
		storage: storage,
		logger:  logger,
	}
}

// LoadBuiltinScenarios loads all builtin scenarios from the specified directory.
func (l *Loader) LoadBuiltinScenarios(ctx context.Context, scenarioDir string) error {
	l.logger.Info("Loading builtin scenarios", "directory", scenarioDir)

	// Check if directory exists
	info, err := os.Stat(scenarioDir)
	if err != nil {
		if os.IsNotExist(err) {
			l.logger.Warn("Scenario directory does not exist", "directory", scenarioDir)
			return nil
		}
		return fmt.Errorf("failed to access scenario directory: %w", err)
	}

	if !info.IsDir() {
		return fmt.Errorf("scenario path is not a directory: %s", scenarioDir)
	}

	// Walk through directory and load JSON files
	loaded := 0
	skipped := 0

	err = filepath.WalkDir(scenarioDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if d.IsDir() {
			return nil
		}

		// Only process JSON files
		if !strings.HasSuffix(strings.ToLower(d.Name()), ".json") {
			return nil
		}

		// Load scenario from file
		scenario, err := l.loadScenarioFromFile(path)
		if err != nil {
			l.logger.Error("Failed to load scenario file",
				"file", path,
				"error", err,
			)
			return nil // Continue with other files
		}

		// Check if scenario already exists
		existing, _ := l.storage.GetScenario(ctx, scenario.ScenarioID)
		if existing != nil {
			l.logger.Debug("Scenario already exists, skipping",
				"scenario_id", scenario.ScenarioID,
				"name", scenario.Name,
			)
			skipped++
			return nil
		}

		// Create scenario
		if err := l.storage.CreateScenario(ctx, scenario); err != nil {
			l.logger.Error("Failed to save scenario",
				"scenario_id", scenario.ScenarioID,
				"error", err,
			)
			return nil
		}

		l.logger.Info("Loaded builtin scenario",
			"scenario_id", scenario.ScenarioID,
			"name", scenario.Name,
		)
		loaded++

		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to walk scenario directory: %w", err)
	}

	l.logger.Info("Finished loading builtin scenarios",
		"loaded", loaded,
		"skipped", skipped,
	)

	return nil
}

// loadScenarioFromFile loads a single scenario from a JSON file.
func (l *Loader) loadScenarioFromFile(path string) (*Scenario, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var scenario Scenario
	if err := json.Unmarshal(data, &scenario); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	// Validate required fields
	if scenario.ScenarioID == "" {
		return nil, fmt.Errorf("scenarioId is required")
	}
	if scenario.Name == "" {
		return nil, fmt.Errorf("name is required")
	}
	if len(scenario.Steps) == 0 {
		return nil, fmt.Errorf("at least one step is required")
	}

	// Set builtin flag
	scenario.IsBuiltin = true

	return &scenario, nil
}

// GetBuiltinScenarioCount returns the count of builtin scenarios in storage.
func (l *Loader) GetBuiltinScenarioCount(ctx context.Context) (int64, error) {
	scenarios, err := l.storage.ListScenarios(ctx, &ScenarioFilter{BuiltinOnly: true})
	if err != nil {
		return 0, err
	}
	return int64(len(scenarios)), nil
}
