package pprof

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/config"
)

func TestNewRequiresCredentials(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		cfg  *config.PprofConfig
	}{
		{
			name: "nil config",
			cfg:  nil,
		},
		{
			name: "missing username",
			cfg: &config.PprofConfig{
				Password: "password",
			},
		},
		{
			name: "missing password",
			cfg: &config.PprofConfig{
				Username: "username",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			srv, err := New(tc.cfg, "/debug")
			require.Error(t, err)
			require.Nil(t, srv)
		})
	}
}

func TestNewWithCredentials(t *testing.T) {
	t.Parallel()

	srv, err := New(&config.PprofConfig{
		Username: "username",
		Password: "password",
	}, "/debug")
	require.NoError(t, err)
	require.NotNil(t, srv)
}
