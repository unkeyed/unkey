package featureflag

import (
	"context"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/featureflag/static"
	"github.com/unkeyed/unkey/pkg/featureflag/vercel"
	"github.com/unkeyed/unkey/pkg/runner"
)

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{name: "none", config: Config{Provider: ProviderNone}, wantErr: false},
		{name: "static with flags", config: Config{Provider: ProviderStatic, Static: static.Values{PrivateNetworking: true}}, wantErr: false},
		{name: "static without flags", config: Config{Provider: ProviderStatic}, wantErr: true},
		{name: "vercel with settings", config: Config{Provider: ProviderVercel, Vercel: &vercel.Config{SDKKey: "vf_server_test"}}, wantErr: false},
		{name: "vercel without settings", config: Config{Provider: ProviderVercel}, wantErr: true},
		{name: "unknown provider", config: Config{Provider: "launchdarkly"}, wantErr: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := test.config.Validate()
			if test.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestNewFromConfig(t *testing.T) {
	evaluate := func(t *testing.T, cfg Config) (bool, error) {
		t.Helper()
		api, err := NewFromConfig(t.Context(), runner.New(), cfg, prometheus.NewRegistry())
		require.NoError(t, err)
		t.Cleanup(func() { require.NoError(t, api.Shutdown(context.Background())) })
		detail, err := api.NewClient().BooleanValueDetails(t.Context(), PrivateNetworking, false, TeamContext("org_test"))
		return detail.Value, err
	}

	t.Run("no provider resolves defaults without error", func(t *testing.T) {
		value, err := evaluate(t, Config{Provider: ProviderNone})
		require.NoError(t, err)
		require.False(t, value)
	})

	t.Run("static provider returns configured values", func(t *testing.T) {
		value, err := evaluate(t, Config{Provider: ProviderStatic, Static: static.Values{PrivateNetworking: true}})
		require.NoError(t, err)
		require.True(t, value)
	})

	t.Run("static provider reports unlisted flags as errors", func(t *testing.T) {
		_, err := evaluate(t, Config{Provider: ProviderStatic, Static: static.Values{"other": true}})
		require.Error(t, err)
	})

	t.Run("vercel provider rejects a client key before any request", func(t *testing.T) {
		_, err := NewFromConfig(t.Context(), runner.New(), Config{Provider: ProviderVercel, Vercel: &vercel.Config{SDKKey: "vf_client_test"}}, prometheus.NewRegistry())
		require.Error(t, err)
	})

	t.Run("invalid config is rejected", func(t *testing.T) {
		_, err := NewFromConfig(t.Context(), runner.New(), Config{Provider: ProviderStatic}, prometheus.NewRegistry())
		require.Error(t, err)
	})
}
