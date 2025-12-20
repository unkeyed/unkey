package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"

	"github.com/unkeyed/unkey/go/pkg/assert"
	"github.com/unkeyed/unkey/go/pkg/cli"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"github.com/unkeyed/unkey/go/pkg/secrets/provider"
)

var allowedUnkeyVars = map[string]bool{
	"UNKEY_DEPLOYMENT_ID":  true,
	"UNKEY_ENVIRONMENT_ID": true,
	"UNKEY_REGION":         true,
	"UNKEY_INSTANCE_ID":    true,
}

type Config struct {
	Provider      provider.Type
	Endpoint      string
	DeploymentID  string
	EnvironmentID string
	Encrypted     string
	Token         string
	TokenPath     string
	Debug         bool
	Args          []string
}

func (c *Config) Validate() error {
	return assert.All(
		assert.NotEmpty(c.Endpoint, "endpoint is required"),
		assert.NotEmpty(c.DeploymentID, "deployment-id is required"),
		assert.True(c.Token != "" || c.TokenPath != "", "either token or token-path is required"),
		assert.True(len(c.Args) > 0, "command is required"),
	)
}

var Cmd = &cli.Command{
	Name:        "unkey-env",
	Usage:       "Fetch secrets and exec the given command",
	AcceptsArgs: true,
	Flags: []cli.Flag{
		cli.String("provider", "Secrets provider type",
			cli.Default(string(provider.KraneVault)),
			cli.EnvVar("UNKEY_PROVIDER")),
		cli.String("endpoint", "Provider API endpoint",
			cli.Required(),
			cli.EnvVar("UNKEY_PROVIDER_ENDPOINT")),
		cli.String("deployment-id", "Deployment ID",
			cli.Required(),
			cli.EnvVar("UNKEY_DEPLOYMENT_ID")),
		cli.String("environment-id", "Environment ID for decryption",
			cli.EnvVar("UNKEY_ENVIRONMENT_ID")),
		cli.String("secrets-blob", "Base64-encoded encrypted secrets blob",
			cli.EnvVar("UNKEY_ENCRYPTED_ENV")),
		cli.String("token", "Authentication token",
			cli.EnvVar("UNKEY_TOKEN")),
		cli.String("token-path", "Path to token file",
			cli.EnvVar("UNKEY_TOKEN_PATH")),
		cli.Bool("debug", "Enable debug logging",
			cli.EnvVar("UNKEY_DEBUG")),
	},
	Action: action,
}

func main() {
	if err := Cmd.Run(context.Background(), os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "unkey-env: %v\n", err)
		os.Exit(1)
	}
}

func action(ctx context.Context, cmd *cli.Command) error {
	cfg := Config{
		Provider:      provider.Type(cmd.String("provider")),
		Endpoint:      cmd.String("endpoint"),
		DeploymentID:  cmd.String("deployment-id"),
		EnvironmentID: cmd.String("environment-id"),
		Encrypted:     cmd.String("secrets-blob"),
		Token:         cmd.String("token"),
		TokenPath:     cmd.String("token-path"),
		Debug:         cmd.Bool("debug"),
		Args:          cmd.Args(),
	}

	if err := cfg.Validate(); err != nil {
		return cli.Exit(err.Error(), 1)
	}

	return run(ctx, cfg)
}

func run(ctx context.Context, cfg Config) error {
	logger := logging.New()
	if cfg.Debug {
		logger = logger.With(slog.String("deployment", cfg.DeploymentID))
	}

	logger.Debug("starting unkey-env",
		"provider", cfg.Provider,
		"endpoint", cfg.Endpoint,
		"deployment", cfg.DeploymentID,
	)

	p, err := provider.New(provider.Config{
		Type:     cfg.Provider,
		Endpoint: cfg.Endpoint,
	})
	if err != nil {
		return fmt.Errorf("failed to create provider: %w", err)
	}

	fetchCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	var Encrypted []byte
	if cfg.Encrypted != "" {
		var decodeErr error
		Encrypted, decodeErr = base64.StdEncoding.DecodeString(cfg.Encrypted)
		if decodeErr != nil {
			return fmt.Errorf("failed to decode secrets blob: %w", decodeErr)
		}
	}

	secrets, err := p.FetchSecrets(fetchCtx, provider.FetchOptions{
		DeploymentID:  cfg.DeploymentID,
		EnvironmentID: cfg.EnvironmentID,
		Encrypted:     Encrypted,
		Token:         cfg.Token,
		TokenPath:     cfg.TokenPath,
	})
	if err != nil {
		return fmt.Errorf("failed to fetch secrets: %w", err)
	}

	logger.Debug("fetched secrets", "count", len(secrets))

	for _, env := range os.Environ() {
		name, _, _ := strings.Cut(env, "=")
		if strings.HasPrefix(name, "UNKEY_") && !allowedUnkeyVars[name] {
			os.Unsetenv(name)
		}
	}

	for key, value := range secrets {
		if err := os.Setenv(key, value); err != nil {
			return fmt.Errorf("failed to set env var %s: %w", key, err)
		}
		logger.Debug("set environment variable", "key", key)
	}

	command := cfg.Args[0]
	binary, err := exec.LookPath(command)
	if err != nil {
		return fmt.Errorf("command not found: %s: %w", command, err)
	}

	logger.Debug("executing command", "binary", binary, "args", cfg.Args)

	err = syscall.Exec(binary, cfg.Args, os.Environ())
	if err != nil {
		return fmt.Errorf("exec failed: %w", err)
	}

	return nil
}
