package util

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	unkey "github.com/unkeyed/sdks/api/go/v3"
	"github.com/unkeyed/unkey/pkg/cli"
	"github.com/unkeyed/unkey/pkg/cli/credentials"
	"github.com/unkeyed/unkey/pkg/cli/session"
)

// CreateClient builds an SDK client. Credentials are used in this order:
// the --root-key flag, then UNKEY_ROOT_KEY, then the stored CLI session, then
// a root key saved with unkey login --root-key=-.
func CreateClient(cmd *cli.Command) (*unkey.Unkey, error) {
	return createClient(cmd, credentials.DefaultRootKeyStore())
}

func createClient(cmd *cli.Command, keys credentials.RootKeyStore) (*unkey.Unkey, error) {
	key, err := credential(cmd, keys)
	if err != nil {
		return nil, err
	}

	opts := []unkey.SDKOption{
		unkey.WithSecurity(key),
	}
	if url := cmd.String("api-url"); url != "" {
		opts = append(opts, unkey.WithServerURL(url))
	}

	return unkey.New(opts...), nil
}

// Credential resolves the bearer for a command. Deploy uses this because its
// flag is named the same way as the API commands.
func Credential(cmd *cli.Command) (string, error) {
	return credential(cmd, credentials.DefaultRootKeyStore())
}

func credential(cmd *cli.Command, keys credentials.RootKeyStore) (string, error) {
	if cmd.FlagIsSet("root-key") {
		key := strings.TrimSpace(cmd.String("root-key"))
		if key == "" {
			return "", fmt.Errorf("root key cannot be empty")
		}
		return key, nil
	}
	if env := strings.TrimSpace(os.Getenv("UNKEY_ROOT_KEY")); env != "" {
		return env, nil
	}
	apiURL := cmd.String("api-url")
	if apiURL == "" {
		apiURL = cmd.String("api-base-url")
	}
	if apiURL == "" {
		apiURL = "https://api.unkey.com"
	}
	token, err := session.AccessToken(context.Background(), func(ctx context.Context, refreshToken string) (session.Session, error) {
		return session.Refresh(ctx, apiURL, refreshToken)
	})
	if err == nil {
		return token, nil
	}
	if !errors.Is(err, session.ErrNotFound) {
		return "", err
	}
	loaded, err := credentials.LoadRootKey(keys, cmd.String("config"))
	if err != nil {
		if errors.Is(err, credentials.ErrRootKeyNotFound) {
			return "", fmt.Errorf("no credentials provided\n\nProvide them via:\n  --root-key flag\n  UNKEY_ROOT_KEY environment variable\n  unkey login")
		}
		return "", fmt.Errorf("failed to load credentials: %w", err)
	}
	return loaded, nil
}
