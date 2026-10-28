package cli

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOpenBrowserRejectsNonHTTPURLs(t *testing.T) {
	for _, rawURL := range []string{
		"file:///etc/passwd",
		"javascript:alert(1)",
		"app.unkey.com/cli/device",
		"http:///cli/device",
	} {
		t.Run(rawURL, func(t *testing.T) {
			require.ErrorContains(t, OpenBrowser(rawURL), "refusing to open non-http(s) url")
		})
	}
}
