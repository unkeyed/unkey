package deploy

import (
	"flag"
	"fmt"
)

var commandName = "deploy"

// parseDeployFlags parses flags for deploy/version create commands
func parseDeployFlags(args []string, env map[string]string) (*DeployOptions, error) {
	fs := flag.NewFlagSet(commandName, flag.ExitOnError)
	opts := &DeployOptions{}

	defaultWorkspaceID := env["UNKEY_WORKSPACE_ID"]
	defaultProjectID := env["UNKEY_PROJECT_ID"]
	defaultRegistry := env["UNKEY_DOCKER_REGISTRY"]
	if defaultRegistry == "" {
		defaultRegistry = "ghcr.io/unkeyed/deploy"
	}

	// Required flags
	fs.StringVar(&opts.WorkspaceID, "workspace-id", defaultWorkspaceID, "Workspace ID (required)")
	fs.StringVar(&opts.ProjectID, "project-id", defaultProjectID, "Project ID (required)")

	// Optional flags with defaults
	fs.StringVar(&opts.Context, "context", ".", "Docker context path")
	fs.StringVar(&opts.Branch, "branch", "main", "Git branch")
	fs.StringVar(&opts.DockerImage, "docker-image", "", "Pre-built docker image")
	fs.StringVar(&opts.Dockerfile, "dockerfile", "Dockerfile", "Path to Dockerfile")
	fs.StringVar(&opts.Commit, "commit", "", "Git commit SHA")
	fs.StringVar(&opts.Registry, "registry", defaultRegistry, "Docker registry")
	fs.BoolVar(&opts.SkipPush, "skip-push", false, "Skip pushing to registry (for local testing)")

	// Control plane flags (internal)
	fs.StringVar(&opts.ControlPlaneURL, "control-plane-url", "http://localhost:7091", "Control plane URL")
	fs.StringVar(&opts.AuthToken, "auth-token", "ctrl-secret-token", "Control plane auth token")

	if err := fs.Parse(args); err != nil {
		return nil, fmt.Errorf("failed to parse %s flags: %w", commandName, err)
	}

	// Validate required fields
	if opts.WorkspaceID == "" {
		return nil, fmt.Errorf("--workspace-id is required (or set UNKEY_WORKSPACE_ID)")
	}
	if opts.ProjectID == "" {
		return nil, fmt.Errorf("--project-id is required (or set UNKEY_PROJECT_ID)")
	}

	return opts, nil
}
