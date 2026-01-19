package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"

	"github.com/unkeyed/unkey/pkg/otel/logging"
	"github.com/unkeyed/unkey/pkg/secrets/provider"
)

func run(ctx context.Context, cfg config) error {
	if len(cfg.Args) == 0 {
		return fmt.Errorf("no command specified")
	}

	logger := logging.New()

	// Clear sensitive UNKEY_* env vars first, before setting user secrets.
	// This prevents leaking config like UNKEY_TOKEN to the child process,
	// while allowing users to have secrets named UNKEY_* if they want.
	err := clearSensitiveEnvVars()
	if err != nil {
		return fmt.Errorf("failed to clear env vars: %w", err)
	}
	if cfg.hasSecrets() {
		secrets, err := fetchSecrets(ctx, cfg)
		if err != nil {
			return err
		}

		logger.Debug("fetched secrets", "count", len(secrets))

		for key, value := range secrets {
			if err := os.Setenv(key, value); err != nil {
				return fmt.Errorf("failed to set env var %s: %w", key, err)
			}
			if cfg.Debug {
				logger.Debug("set environment variable", "key", key)
			}
		}
	}

	return execCommand(cfg.Args, logger)
}

func fetchSecrets(ctx context.Context, cfg config) (map[string]string, error) {
	p, err := provider.New(provider.Config{
		Type:     cfg.Provider,
		Endpoint: cfg.Endpoint,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create provider: %w", err)
	}

	fetchCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	encrypted, err := base64.StdEncoding.DecodeString(cfg.Encrypted)
	if err != nil {
		return nil, fmt.Errorf("failed to decode secrets blob: %w", err)
	}

	secrets, err := p.FetchSecrets(fetchCtx, provider.FetchOptions{
		DeploymentID:  cfg.DeploymentID,
		EnvironmentID: cfg.EnvironmentID,
		Encrypted:     encrypted,
		Token:         cfg.Token,
		TokenPath:     cfg.TokenPath,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch secrets: %w", err)
	}

	return secrets, nil
}

func clearSensitiveEnvVars() error {
	for _, env := range os.Environ() {
		name, _, _ := strings.Cut(env, "=")
		if strings.HasPrefix(name, "UNKEY_") && !allowedUnkeyVars[name] {
			err := os.Unsetenv(name)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func execCommand(args []string, logger logging.Logger) error {
	binary, err := exec.LookPath(args[0])
	if err != nil {
		return fmt.Errorf("command not found: %s: %w", args[0], err)
	}

	logger.Debug("executing command", "binary", binary, "args", args)

	if err := syscall.Exec(binary, args, os.Environ()); err != nil {
		return fmt.Errorf("exec failed: %w", err)
	}

	return nil
}
