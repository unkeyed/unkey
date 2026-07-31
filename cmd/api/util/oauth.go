package util

import (
	"os"

	"github.com/unkeyed/unkey/cmd/internal/oauth"
)

// NewOAuthClient builds a WorkOS OAuth client. The base URL can be overridden
// with UNKEY_WORKOS_BASE_URL for self-hosted WorkOS proxies or tests; otherwise
// it targets the WorkOS default host.
func NewOAuthClient() *oauth.Client {
	var opts []oauth.Option
	if base := os.Getenv("UNKEY_WORKOS_BASE_URL"); base != "" {
		opts = append(opts, oauth.WithBaseURL(base))
	}
	return oauth.New(opts...)
}
