package depotclient

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewValidatesAPIURL(t *testing.T) {
	for _, apiURL := range []string{
		"api.depot.dev",
		"ftp://api.depot.dev",
		"https://",
		"://invalid",
	} {
		t.Run(apiURL, func(t *testing.T) {
			client, err := New(Config{APIUrl: apiURL, Token: "token"})
			require.Error(t, err)
			require.Nil(t, client)
		})
	}
}

func TestNewAcceptsAbsoluteHTTPURL(t *testing.T) {
	client, err := New(Config{APIUrl: "https://api.depot.dev", Token: "token"})
	require.NoError(t, err)
	require.NotNil(t, client)
}
