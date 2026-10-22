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
// the --root-key flag, then UNKEY_ROOT_KEY, then an explicit --config file,
// then the stored CLI session for this API, then a root key saved with
// unkey login --root-key=-.
func CreateClient(ctx context.Context, cmd *cli.Command) (*unkey.Unkey, error) {
	return createClient(ctx, cmd, credentials.DefaultRootKeyStore())
}

func createClient(ctx context.Context, cmd *cli.Command, keys credentials.RootKeyStore) (*unkey.Unkey, error) {
	cred, err := credential(ctx, cmd, keys)
	if err != nil {
		return nil, err
	}

	opts := []unkey.SDKOption{
		unkey.WithSecurity(cred.Key),
	}
	url := cred.APIURL
	if url == "" {
		url, _ = commandAPIURL(cmd)
	}
	if url != "" {
		opts = append(opts, unkey.WithServerURL(url))
	}

	return unkey.New(opts...), nil
}

// ResolvedCredential is the bearer a command will send and, when it came from
// a CLI session, the API that issued that session.
type ResolvedCredential struct {
	Key    string
	APIURL string
}

// Credential resolves the bearer for a command. Deploy uses this because its
// flag is named the same way as the API commands. APIURL is set only when the
// bearer is a CLI session, so the caller keeps its own API flag otherwise.
func Credential(ctx context.Context, cmd *cli.Command) (ResolvedCredential, error) {
	return credential(ctx, cmd, credentials.DefaultRootKeyStore())
}

func credential(ctx context.Context, cmd *cli.Command, keys credentials.RootKeyStore) (ResolvedCredential, error) {
	if cmd.FlagIsSet("root-key") {
		key := strings.TrimSpace(cmd.String("root-key"))
		if key == "" {
			return ResolvedCredential{}, fmt.Errorf("root key cannot be empty")
		}
		return ResolvedCredential{Key: key}, nil
	}
	if env := strings.TrimSpace(os.Getenv("UNKEY_ROOT_KEY")); env != "" {
		return ResolvedCredential{Key: env}, nil
	}
	if cmd.FlagIsSet("config") {
		return rootKeyCredential(keys, cmd.String("config"))
	}
	commandURL, explicitAPI := commandAPIURL(cmd)
	current, err := session.Active()
	if err == nil && sessionMatchesAPI(current, commandURL, explicitAPI) {
		sessionURL := storedAPIURL(current)
		token, err := session.AccessToken(ctx, func(ctx context.Context, current session.Session) (session.Session, error) {
			return session.Refresh(ctx, storedAPIURL(current), current.RefreshToken)
		})
		if err != nil {
			return ResolvedCredential{}, err
		}
		return ResolvedCredential{Key: token, APIURL: sessionURL}, nil
	}
	if err != nil && !errors.Is(err, session.ErrNotFound) {
		return ResolvedCredential{}, err
	}
	return rootKeyCredential(keys, cmd.String("config"))
}

func rootKeyCredential(keys credentials.RootKeyStore, configPath string) (ResolvedCredential, error) {
	loaded, err := credentials.LoadRootKey(keys, configPath)
	if err != nil {
		if errors.Is(err, credentials.ErrRootKeyNotFound) {
			return ResolvedCredential{}, fmt.Errorf("no credentials provided\n\nProvide them via:\n  --root-key flag\n  UNKEY_ROOT_KEY environment variable\n  unkey login")
		}
		return ResolvedCredential{}, fmt.Errorf("failed to load credentials: %w", err)
	}
	return ResolvedCredential{Key: loaded}, nil
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
