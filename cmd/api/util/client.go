package util

import (
	"errors"
	"fmt"

	unkey "github.com/unkeyed/sdks/api/go/v3"
	"github.com/unkeyed/unkey/pkg/cli"
)

// CreateClient builds an SDK client using the root key from (in priority order):
// 1. --root-key flag or UNKEY_ROOT_KEY env var (handled by the flag's EnvVar option)
// 2. The system keychain, when --config is the default ~/.unkey/config.toml
// 3. The config file
func CreateClient(cmd *cli.Command) (*unkey.Unkey, error) {
	return createClient(cmd, cli.DefaultRootKeyStore())
}

func createClient(cmd *cli.Command, keys cli.RootKeyStore) (*unkey.Unkey, error) {
	key := cmd.String("root-key")

	if key == "" {
		loaded, err := cli.LoadRootKey(keys, cmd.String("config"))
		if err != nil {
			if errors.Is(err, cli.ErrRootKeyNotFound) {
				return nil, fmt.Errorf("no root key provided\n\nProvide one via:\n  --root-key flag\n  UNKEY_ROOT_KEY environment variable\n  unkey login")
			}
			return nil, fmt.Errorf("failed to load root key: %w", err)
		}
		key = loaded
	}

	opts := []unkey.SDKOption{
		unkey.WithSecurity(key),
	}
	if url := cmd.String("api-url"); url != "" {
		opts = append(opts, unkey.WithServerURL(url))
	}

	return unkey.New(opts...), nil
}
