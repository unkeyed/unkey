package vercel

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestNewValidatesConfig(t *testing.T) {
	_, err := New(Config{SDKKey: "vf_server_test", RefreshInterval: 0, HTTPTimeout: 0, MaxStaleness: 0})
	require.NoError(t, err)

	for _, config := range []Config{
		{SDKKey: "", RefreshInterval: 0, HTTPTimeout: 0, MaxStaleness: 0},
		{SDKKey: "vf_client_test", RefreshInterval: 0, HTTPTimeout: 0, MaxStaleness: 0},
		{SDKKey: "vf_server_test", RefreshInterval: time.Millisecond, HTTPTimeout: 0, MaxStaleness: 0},
		{SDKKey: "vf_server_test", RefreshInterval: 0, HTTPTimeout: 2 * time.Minute, MaxStaleness: 0},
		{SDKKey: "vf_server_test", RefreshInterval: time.Minute, HTTPTimeout: 0, MaxStaleness: time.Second},
	} {
		_, err := New(config)
		require.Error(t, err)
	}

	for _, endpoint := range []string{"http://example.invalid/data", "https://user@example.invalid/data", "/data"} {
		_, err := newProvider(Config{SDKKey: "vf_server_test", RefreshInterval: 0, HTTPTimeout: 0, MaxStaleness: 0}, endpoint, &http.Client{})
		require.Error(t, err)
	}
}
