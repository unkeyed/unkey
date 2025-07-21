package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

var (
	ErrWorkspaceIDRequired = errors.New("workspace ID is required (use --workspace-id flag or edit unkey.json)")
	ErrProjectIDRequired   = errors.New("project ID is required (use --project-id flag or edit unkey.json)")
	ErrConfigPathResolve   = errors.New("failed to resolve config path")
	ErrConfigDirNotExist   = errors.New("config directory does not exist")
	ErrConfigDirAccess     = errors.New("failed to access config directory")
	ErrConfigFileRead      = errors.New("failed to read config file")
	ErrConfigFileParse     = errors.New("failed to parse config file")
	ErrConfigFileWrite     = errors.New("failed to write config file")
	ErrConfigMarshal       = errors.New("failed to marshal config")
	ErrDirectoryCreate     = errors.New("failed to create directory")
)

type Config struct {
	WorkspaceID string `json:"workspace_id"`
	ProjectID   string `json:"project_id"`
	Context     string `json:"context"`
}

// LoadConfig loads configuration from unkey.json in the specified directory.
// If configPath is empty, uses current directory.
// If the file doesn't exist, it returns an empty config without error.
func LoadConfig(configPath string) (*Config, error) {
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

// CreateTemplate creates a new unkey.json template file in the specified directory
func CreateTemplate(configDir string) error {
	configPath := filepath.Join(configDir, "unkey.json")

	// Create template config with placeholder values
	config := &Config{
		WorkspaceID: "ws_your_workspace_id",
		ProjectID:   "proj_your_project_id",
		Context:     "./your_app_directory",
	}

	// Create directory if it doesn't exist
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("%w %s: %w", ErrDirectoryCreate, configDir, err)
	}

	// Write config file
	if err := writeConfig(configPath, config); err != nil {
		return fmt.Errorf("%w: %w", ErrConfigFileWrite, err)
	}

	return nil
}

// ConfigExists checks if unkey.json exists in the specified directory
func ConfigExists(configDir string) bool {
	configPath := filepath.Join(configDir, "unkey.json")
	_, err := os.Stat(configPath)
	return err == nil
}

// GetConfigFilePath returns the full path to unkey.json in the specified directory
func GetConfigFilePath(configDir string) string {
	if configDir == "" {
		configDir = "."
	}
	return filepath.Join(configDir, "unkey.json")
}

// writeConfig writes the config struct to a JSON file
func writeConfig(configPath string, config *Config) error {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("%w: %w", ErrConfigMarshal, err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// MergeWithFlags merges config values with command flags, with flags taking precedence
func (c *Config) MergeWithFlags(workspaceID, projectID, context string) *Config {
	merged := &Config{
		WorkspaceID: c.WorkspaceID,
		ProjectID:   c.ProjectID,
		Context:     c.Context,
	}

	// Flags override config values
	if workspaceID != "" {
		merged.WorkspaceID = workspaceID
	}
	if projectID != "" {
		merged.ProjectID = projectID
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

// Validate checks if required fields are present and not placeholder values
func (c *Config) Validate() error {
	if c.WorkspaceID == "" || c.WorkspaceID == "ws_your_workspace_id" {
		return ErrWorkspaceIDRequired
	}
	if c.ProjectID == "" || c.ProjectID == "proj_your_project_id" {
		return ErrProjectIDRequired
	}
	return nil
}

// GetConfigPath resolves the config directory path
func GetConfigPath(configFlag string) (string, error) {
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
