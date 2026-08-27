// Package deploytarget resolves the project, app, environment, settings, and
// secrets a deployment create targets. Ctrl loads a target without secrets to
// run its gates and resolve the build source; the DeploymentCreateService
// worker loads it again with secrets at execution time so the row records the
// settings current when the create runs, not a snapshot from when it was
// requested.
package deploytarget

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/validation"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
	"golang.org/x/sync/errgroup"
	"google.golang.org/protobuf/encoding/protojson"
)

// Target bundles everything a deployment create needs to know about where it
// deploys to.
type Target struct {
	Project            db.Project
	WorkspaceID        string
	Env                db.FindEnvironmentByAppIdAndSlugRow
	App                db.App
	AppBuildSettings   db.AppBuildSetting
	AppRuntimeSettings db.AppRuntimeSetting
	SecretsBlob        []byte
}

// TerminalError marks a load failure no retry can fix: the request names
// resources that do not exist or do not belong together. Transient failures
// (a broken read, a marshal error) come back as plain errors instead. Code is
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
// marshals the secrets blob. Only the create worker consumes the blob, on
// the deployment row; ctrl validates targets without it, which saves the
// env var query on every create.
type SecretsMode bool

const (
	WithSecrets    SecretsMode = true
	WithoutSecrets SecretsMode = false
)

// Load resolves the target for a (project, app, environment slug) triple.
// SecretsBlob is empty when loaded with WithoutSecrets.
func Load(ctx context.Context, database db.Database, projectID, appID, envSlug string, secrets SecretsMode) (Target, error) {
	var project db.Project
	var env db.FindEnvironmentByAppIdAndSlugRow
	var appWithSettings db.FindAppWithSettingsRow

	// The project lookup shares nothing with the environment and app lookups,
	// so the two chains run concurrently. The app lookup needs the environment
	// id, so those two stay sequential.
	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		var err error
		project, err = database.FindProjectById(gctx, projectID)
		if err != nil {
			if db.IsNotFound(err) {
				return terminal(connect.CodeNotFound, "project not found: %s", projectID)
			}
			return fmt.Errorf("failed to lookup project: %w", err)
		}
		return nil
	})
	g.Go(func() error {
		var err error
		env, err = database.FindEnvironmentByAppIdAndSlug(gctx, db.FindEnvironmentByAppIdAndSlugParams{
			AppID: appID,
			Slug:  envSlug,
		})
		if err != nil {
			if db.IsNotFound(err) {
				return terminal(connect.CodeNotFound, "environment '%s' not found for app '%s'", envSlug, appID)
			}
			return fmt.Errorf("failed to lookup environment: %w", err)
		}

		appWithSettings, err = database.FindAppWithSettings(gctx, db.FindAppWithSettingsParams{
			ID:            appID,
			EnvironmentID: env.Environment.ID,
		})
		if err != nil && db.IsNotFound(err) {
			return terminal(connect.CodeNotFound, "app '%s' not found or missing settings", appID)
		}
		if err != nil {
			return fmt.Errorf("failed to lookup app: %w", err)
		}
		return nil
	})
	if err := g.Wait(); err != nil {
		return Target{}, err
	}

	// The three records are resolved independently, so verify they belong to
	// one project: without this, an internal caller mixing ids would insert a
	// deployment row with a mismatched (project, app, environment) triple.
	if appWithSettings.App.ProjectID != project.ID {
		return Target{}, terminal(connect.CodeInvalidArgument, "app %q does not belong to project %q", appID, project.ID)
	}
	if env.Environment.ProjectID != project.ID {
		return Target{}, terminal(connect.CodeInvalidArgument, "environment %q does not belong to project %q", envSlug, project.ID)
	}

	secretsBlob := []byte{}
	if secrets == WithSecrets {
		var loadErr error
		secretsBlob, loadErr = loadSecretsBlob(ctx, database, appWithSettings.App.ID, env.Environment.ID)
		if loadErr != nil {
			return Target{}, loadErr
		}
	}

	return Target{
		Project:            project,
		WorkspaceID:        project.WorkspaceID,
		Env:                env,
		App:                appWithSettings.App,
		AppBuildSettings:   appWithSettings.AppBuildSetting,
		AppRuntimeSettings: appWithSettings.AppRuntimeSetting,
		SecretsBlob:        secretsBlob,
	}, nil
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
