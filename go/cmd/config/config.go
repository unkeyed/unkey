package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
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

	// Always look for unkey.json in the specified directory
	configFile := filepath.Join(configPath, "unkey.json")

	// Check if file exists
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		// Return empty config if file doesn't exist
		return &Config{}, nil
	}

	data, err := os.ReadFile(configFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %s: %w", configFile, err)
	}

	config := &Config{}
	if err := json.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("failed to parse config file %s: %w", configFile, err)
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
		return fmt.Errorf("failed to create directory %s: %w", configDir, err)
	}

	// Write config file
	if err := writeConfig(configPath, config); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
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
		return fmt.Errorf("failed to marshal config: %w", err)
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
		return fmt.Errorf("workspace ID is required (use --workspace-id flag or edit unkey.json)")
	}
	if c.ProjectID == "" || c.ProjectID == "proj_your_project_id" {
		return fmt.Errorf("project ID is required (use --project-id flag or edit unkey.json)")
	}
	return nil
}

// GetConfigPath resolves the config directory path
func GetConfigPath(configFlag string) string {
	if configFlag == "" {
		return "."
	}

	// Convert to absolute path if relative
	if !filepath.IsAbs(configFlag) {
		if abs, err := filepath.Abs(configFlag); err == nil {
			return abs
		}
	}

	return configFlag
}
