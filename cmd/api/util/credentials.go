package util

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/unkeyed/unkey/pkg/cli"
	"github.com/unkeyed/unkey/pkg/cli/credentials"
	"github.com/unkeyed/unkey/pkg/cli/session"
)

// ResolvedCredential is the bearer a command will send and where it came from.
// APIURL, WorkspaceID, and SessionID are set only for CLI sessions.
type ResolvedCredential struct {
	Key         string
	Source      CredentialSource
	APIURL      string
	WorkspaceID string
	SessionID   string
}

// CredentialSource names where [Credential] found the bearer.
type CredentialSource string

const (
	SourceRootKeyFlag  CredentialSource = "--root-key"
	SourceRootKeyEnv   CredentialSource = "UNKEY_ROOT_KEY"
	SourceConfig       CredentialSource = "--config"
	SourceSession      CredentialSource = "browser session"
	SourceSavedRootKey CredentialSource = "saved root key"
)

// ErrNoCredentials means no credential source holds a bearer.
var ErrNoCredentials = errors.New("no credentials provided")

// Credential resolves the bearer for a command. Credentials are used in this order:
// the --root-key flag, then UNKEY_ROOT_KEY, then an explicit --config file,
// then the stored CLI session for this API, then a root key saved with
// unkey login --root-key=-. When the selected CLI session cannot be refreshed,
// the error comes with that session's source, API, workspace, and session ID
// but no key.
func Credential(ctx context.Context, cmd *cli.Command) (ResolvedCredential, error) {
	return credential(ctx, cmd, credentials.DefaultRootKeyStore())
}

func credential(ctx context.Context, cmd *cli.Command, keys credentials.RootKeyStore) (ResolvedCredential, error) {
	if cmd.FlagIsSet("root-key") {
		key := strings.TrimSpace(cmd.String("root-key"))
		if key == "" {
			return ResolvedCredential{}, fmt.Errorf("root key cannot be empty")
		}

		return rootKey(key, SourceRootKeyFlag), nil
	}

	if env := strings.TrimSpace(os.Getenv("UNKEY_ROOT_KEY")); env != "" {
		return rootKey(env, SourceRootKeyEnv), nil
	}

	if cmd.FlagIsSet("config") {
		return rootKeyCredential(keys, cmd.String("config"), SourceConfig)
	}

	commandURL, explicitAPI := commandAPIURL(cmd)
	current, err := session.Active()
	if err != nil && !errors.Is(err, session.ErrNotFound) {
		return ResolvedCredential{}, err
	}

	if err == nil && sessionMatchesAPI(current, commandURL, explicitAPI) {
		used, err := session.AccessToken(ctx, func(ctx context.Context, current session.Session) (session.Session, error) {
			return session.Refresh(ctx, storedAPIURL(current), current.RefreshToken)
		})
		if err != nil && !errors.Is(err, session.ErrNotFound) {
			return ResolvedCredential{
				Key:         "",
				Source:      SourceSession,
				APIURL:      storedAPIURL(current),
				WorkspaceID: current.WorkspaceID,
				SessionID:   current.SessionID,
			}, err
		}

		if err == nil && sessionMatchesAPI(used, commandURL, explicitAPI) {
			return ResolvedCredential{
				Key:         used.AccessToken,
				Source:      SourceSession,
				APIURL:      storedAPIURL(used),
				WorkspaceID: used.WorkspaceID,
				SessionID:   used.SessionID,
			}, nil
		}
	}

	return rootKeyCredential(keys, cmd.String("config"), SourceSavedRootKey)
}

func rootKey(key string, source CredentialSource) ResolvedCredential {
	return ResolvedCredential{
		Key:         key,
		Source:      source,
		APIURL:      "",
		WorkspaceID: "",
		SessionID:   "",
	}
}

func rootKeyCredential(keys credentials.RootKeyStore, configPath string, source CredentialSource) (ResolvedCredential, error) {
	if configPath == "" {
		path, err := cli.UserConfigPath()
		if err != nil {
			return ResolvedCredential{}, fmt.Errorf("failed to load credentials: %w", err)
		}

		configPath = path
	}

	loaded, err := credentials.LoadRootKey(keys, configPath)
	if err != nil {
		if errors.Is(err, credentials.ErrRootKeyNotFound) {
			return ResolvedCredential{}, fmt.Errorf("%w\n\nProvide them via:\n  --root-key flag\n  UNKEY_ROOT_KEY environment variable\n  unkey login", ErrNoCredentials)
		}

		return ResolvedCredential{}, fmt.Errorf("failed to load credentials: %w", err)
	}

	return rootKey(loaded, source), nil
}

func commandAPIURL(cmd *cli.Command) (string, bool) {
	if cmd.FlagIsSet("api-url") {
		return session.NormalizeAPIURL(cmd.String("api-url")), true
	}
	if cmd.FlagIsSet("api-base-url") {
		return session.NormalizeAPIURL(cmd.String("api-base-url")), true
	}
	if env := strings.TrimSpace(os.Getenv("UNKEY_API_BASE_URL")); env != "" {
		return session.NormalizeAPIURL(env), true
	}

	url := cmd.String("api-url")
	if url == "" {
		url = cmd.String("api-base-url")
	}
	if url == "" {
		url = session.DefaultAPIURL
	}

	return session.NormalizeAPIURL(url), false
}

func storedAPIURL(current session.Session) string {
	url := session.NormalizeAPIURL(current.APIURL)
	if url == "" {
		return session.DefaultAPIURL
	}

	return url
}

func sessionMatchesAPI(current session.Session, commandURL string, explicitAPI bool) bool {
	if !explicitAPI {
		return true
	}

	return storedAPIURL(current) == commandURL
}
