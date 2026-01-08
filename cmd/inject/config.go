package main

import (
	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/secrets/provider"
)

// allowedUnkeyVars are environment variables that should NOT be cleared before exec.
// All other UNKEY_* vars are removed to prevent leaking sensitive config to the child process.
var allowedUnkeyVars = map[string]bool{
	"UNKEY_DEPLOYMENT_ID":  true,
	"UNKEY_ENVIRONMENT_ID": true,
	"UNKEY_REGION":         true,
	"UNKEY_INSTANCE_ID":    true,
}

type config struct {
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

func (c *config) hasSecrets() bool {
	return c.Encrypted != ""
}

func (c *config) validate() error {
	if err := assert.True(len(c.Args) > 0, "command is required"); err != nil {
		return err
	}

	if !c.hasSecrets() {
		return nil
	}

	return assert.All(
		assert.NotEmpty(c.Endpoint, "endpoint is required when secrets-blob is provided"),
		assert.NotEmpty(c.DeploymentID, "deployment-id is required when secrets-blob is provided"),
		assert.True(c.Token != "" || c.TokenPath != "", "either token or token-path is required when secrets-blob is provided"),
	)
}
