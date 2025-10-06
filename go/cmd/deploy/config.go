package deploy

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

var (
	ErrProjectIDRequired = errors.New("project ID is required (use --project-id flag or edit unkey.json)")
	ErrConfigPathResolve = errors.New("failed to resolve config path")
	ErrConfigFileRead    = errors.New("failed to read config file")
	ErrConfigFileParse   = errors.New("failed to parse config file")
	ErrConfigFileWrite   = errors.New("failed to write config file")
	ErrConfigMarshal     = errors.New("failed to marshal config")
	ErrDirectoryCreate   = errors.New("failed to create directory")
)

type Config struct {
	KeyspaceID string `json:"keyspace_id"`
	ProjectID  string `json:"project_id"`
	Context    string `json:"context"`
}

// loadConfig loads configuration from unkey.json in the specified directory.
// If configPath is empty, uses current directory.
// If the file doesn't exist, it returns an empty config without error.
func loadConfig(configPath string) (*Config, error) {
	// If no config path specified, use current directory
	if configPath == "" {
		configPath = "."
	}

	// Check if directory exists first
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("config directory '%s' does not exist", configPath)
	} else if err != nil {
		return nil, fmt.Errorf("failed to access config directory '%s': %w", configPath, err)
	}

	// Always look for unkey.json in the specified directory
	configFile := filepath.Join(configPath, "unkey.json")

	// Check if file exists
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		// Return empty config if file doesn't exist but directory does
		return &Config{}, nil
	}

	data, err := os.ReadFile(configFile)
	if err != nil {
		return nil, fmt.Errorf("%w %s: %w", ErrConfigFileRead, configFile, err)
	}

	config := &Config{}
	if err := json.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("%w %s: %w", ErrConfigFileParse, configFile, err)
	}

	return config, nil
}

// configExists checks if unkey.json exists in the specified directory
func configExists(configDir string) bool {
	configPath := filepath.Join(configDir, "unkey.json")
	_, err := os.Stat(configPath)
	return err == nil
}

// getConfigFilePath returns the full path to unkey.json in the specified directory
func getConfigFilePath(configDir string) string {
	if configDir == "" {
		configDir = "."
	}
	return filepath.Join(configDir, "unkey.json")
}

// createConfigWithValues creates a new unkey.json file with the provided values
func createConfigWithValues(configDir, projectID, context string) error {
	// Create directory if it doesn't exist
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("%w %s: %w", ErrDirectoryCreate, configDir, err)
	}

	config := &Config{
		ProjectID: projectID,
		Context:   context,
	}

	configPath := filepath.Join(configDir, "unkey.json")
	if err := writeConfig(configPath, config); err != nil {
		return fmt.Errorf("%w: %w", ErrConfigFileWrite, err)
	}

	return nil
}

// writeConfig writes the config struct to a JSON file
func writeConfig(configPath string, config *Config) error {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("%w: %w", ErrConfigMarshal, err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("%w: %w", ErrConfigFileWrite, err)
	}

	return nil
}

// mergeWithFlags merges config values with command flags, with flags taking precedence
func (c *Config) mergeWithFlags(projectID, keyspaceID, context string) *Config {
	merged := &Config{
		KeyspaceID: c.KeyspaceID,
		ProjectID:  c.ProjectID,
		Context:    c.Context,
	}
	// Flags override config values
	if projectID != "" {
		merged.ProjectID = projectID
	}
	if keyspaceID != "" {
		merged.KeyspaceID = keyspaceID
	}
	if context != "" {
		merged.Context = context
	}
	// Set default context if empty
	if merged.Context == "" {
		merged.Context = "."
	}
	return merged
}

// validate checks if required fields are present and not placeholder values
func (c *Config) validate() error {
	if c.ProjectID == "" || c.ProjectID == "proj_your_project_id" {
		return ErrProjectIDRequired
	}
	return nil
}

// getConfigPath resolves the config directory path
func getConfigPath(configFlag string) (string, error) {
	if configFlag == "" {
		return ".", nil
	}

	// Convert to absolute path if relative
	if !filepath.IsAbs(configFlag) {
		abs, err := filepath.Abs(configFlag)
		if err != nil {
			return "", fmt.Errorf("%w '%s': %w", ErrConfigPathResolve, configFlag, err)
		}
		return abs, nil
	}

	return configFlag, nil
}
