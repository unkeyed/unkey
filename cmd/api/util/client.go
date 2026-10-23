package util

import (
	"context"

	unkey "github.com/unkeyed/sdks/api/go/v3"
	"github.com/unkeyed/unkey/pkg/cli"
	"github.com/unkeyed/unkey/pkg/cli/credentials"
)

// CreateClient builds an SDK client with the bearer selected by [Credential].
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
