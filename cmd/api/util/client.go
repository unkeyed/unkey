package util

import (
	"fmt"

	unkey "github.com/unkeyed/sdks/api/go/v2"
	"github.com/unkeyed/unkey/pkg/cli"
)

// CreateClient builds an SDK client using the root key from (in priority order):
// 1. --root-key flag or UNKEY_ROOT_KEY env var (handled by the flag's EnvVar option)
// 2. Config file at ~/.unkey/config.toml (from unkey auth login)
func CreateClient(cmd *cli.Command) (*unkey.Unkey, error) {
	key := cmd.String("root-key")

	if key == "" {
		cfg, err := cli.LoadUserConfig(cmd.String("config"))
		if err != nil {
			return nil, fmt.Errorf("no root key provided\n\nProvide one via:\n  --root-key flag\n  UNKEY_ROOT_KEY environment variable\n  unkey auth login")
		}
		key = cfg.RootKey
	}

	opts := []unkey.SDKOption{
		unkey.WithSecurity(key),
	}
	if url := cmd.String("api-url"); url != "" {
		opts = append(opts, unkey.WithServerURL(url))
	}

	return unkey.New(opts...), nil
}
