// Package deploytarget resolves the project, app, environment, settings, and
// secrets a deployment create targets.
//
// A target is loaded when the create runs, not when it was requested, so the
// row records the settings current at execution. Callers that only validate a
// target load it without secrets; the create worker needs the blob for the row
// and loads it with them.
package deploytarget

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/validation"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
	"google.golang.org/protobuf/encoding/protojson"
)

// Target bundles what a deployment create needs to know about where it deploys
// to. The embedded row carries only the columns a create writes or branches on.
type Target struct {
	db.FindDeployTargetRow

	// SecretsBlob holds the environment's variables marshaled as a
	// SecretsConfig. Empty when loaded with WithoutSecrets.
	SecretsBlob []byte
}

// TerminalError marks a load failure no retry can fix: the request names
// resources that do not exist or do not belong together. Transient failures, a
// broken read or a marshal error, come back as plain errors instead. Code is
// the connect code ctrl answers with; the create worker ignores it and fails
// terminally on any TerminalError.
type TerminalError struct {
	Code    connect.Code
	Message string
}

func (e *TerminalError) Error() string {
	return e.Message
}

func terminal(code connect.Code, format string, args ...any) error {
	return &TerminalError{Code: code, Message: fmt.Sprintf(format, args...)}
}

// SecretsMode selects whether Load fetches the environment's variables and
// marshals the secrets blob. Only the create worker consumes the blob, on the
// deployment row; ctrl validates targets without it.
type SecretsMode bool

const (
	WithSecrets    SecretsMode = true
	WithoutSecrets SecretsMode = false
)

// Load resolves the target for a (project, app, environment) triple, where the
// environment is named by id or slug. SecretsBlob is empty when loaded with
// WithoutSecrets.
//
// The query requires the app and environment to sit under the given project,
// so a Target never mixes records from different projects.
func Load(ctx context.Context, database db.Database, projectID, appID, env string, secrets SecretsMode) (Target, error) {
	row, err := database.FindDeployTarget(ctx, db.FindDeployTargetParams{
		ProjectID:   projectID,
		AppID:       appID,
		Environment: env,
	})
	if err != nil {
		if db.IsNotFound(err) {
			return Target{}, terminal(connect.CodeNotFound,
				"no deploy target for project '%s', app '%s', environment '%s'", projectID, appID, env)
		}
		return Target{}, fmt.Errorf("failed to lookup deploy target: %w", err)
	}

	secretsBlob := []byte{}
	if secrets == WithSecrets {
		secretsBlob, err = loadSecretsBlob(ctx, database, row.AppID, row.EnvironmentID)
		if err != nil {
			return Target{}, err
		}
	}

	return Target{FindDeployTargetRow: row, SecretsBlob: secretsBlob}, nil
}

// loadSecretsBlob fetches the environment's variables and marshals them into
// the SecretsConfig blob stored on the deployment row.
func loadSecretsBlob(ctx context.Context, database db.Database, appID, environmentID string) ([]byte, error) {
	appEnvVars, err := database.FindAppEnvVarsByAppAndEnv(ctx, db.FindAppEnvVarsByAppAndEnvParams{
		AppID:         appID,
		EnvironmentID: environmentID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch app environment variables: %w", err)
	}

	secretsBlob := []byte{}
	if len(appEnvVars) > 0 {
		secretsConfig := &ctrlv1.SecretsConfig{
			Secrets: make(map[string]string, len(appEnvVars)),
		}
		for _, ev := range appEnvVars {
			if !validation.IsValidEnvVarKey(ev.Key) {
				return nil, terminal(connect.CodeInvalidArgument,
					"environment variable key %q is invalid: %s", ev.Key, validation.ErrMsgInvalidEnvVarKey)
			}
			secretsConfig.Secrets[ev.Key] = ev.Value
		}

		secretsBlob, err = protojson.Marshal(secretsConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal secrets config: %w", err)
		}
	}

	return secretsBlob, nil
}
