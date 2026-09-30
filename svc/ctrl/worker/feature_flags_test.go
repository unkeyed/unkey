package worker

import (
	"context"
	"testing"

	promclient "github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/featureflag"
	"github.com/unkeyed/unkey/pkg/featureflag/vercel"
)

func TestNewFeatureFlags(t *testing.T) {
	evaluate := func(t *testing.T, cfg FeatureFlagsConfig) (bool, error) {
		t.Helper()
		api, err := newFeatureFlags(t.Context(), cfg, promclient.NewRegistry())
		require.NoError(t, err)
		t.Cleanup(func() { require.NoError(t, api.Shutdown(context.Background())) })
		detail, err := api.NewClient().BooleanValueDetails(t.Context(), featureflag.PrivateNetworking, false, featureflag.TeamContext("org_test"))
		return detail.Value, err
	}

	t.Run("no provider resolves defaults without error", func(t *testing.T) {
		value, err := evaluate(t, FeatureFlagsConfig{Provider: FeatureFlagProviderNone, Static: nil, Vercel: VercelFlagsConfig{}})
		require.NoError(t, err)
		require.False(t, value)
	})

	t.Run("static provider returns configured values", func(t *testing.T) {
		value, err := evaluate(t, FeatureFlagsConfig{Provider: FeatureFlagProviderStatic, Static: map[string]bool{featureflag.PrivateNetworking: true}, Vercel: VercelFlagsConfig{}})
		require.NoError(t, err)
		require.True(t, value)
	})

	t.Run("static provider reports unlisted flags as errors", func(t *testing.T) {
		_, err := evaluate(t, FeatureFlagsConfig{Provider: FeatureFlagProviderStatic, Static: map[string]bool{"other": true}, Vercel: VercelFlagsConfig{}})
		require.Error(t, err)
	})

	t.Run("vercel provider rejects a client key", func(t *testing.T) {
		_, err := newFeatureFlags(t.Context(), FeatureFlagsConfig{
			Provider: FeatureFlagProviderVercel,
			Static:   nil,
			Vercel:   VercelFlagsConfig{SDKKey: "vf_client_test", RefreshInterval: 0, HTTPTimeout: 0, MaxStaleness: 0},
		}, promclient.NewRegistry())
		require.Error(t, err)
	})

	t.Run("vercel provider registers health metrics", func(t *testing.T) {
		reg := promclient.NewRegistry()
		provider, err := vercel.New(vercel.Config{SDKKey: "vf_server_test", RefreshInterval: 0, HTTPTimeout: 0, MaxStaleness: 0})
		require.NoError(t, err)
		require.NoError(t, registerVercelFlagMetrics(reg, provider))
		families, err := reg.Gather()
		require.NoError(t, err)
		names := make([]string, 0, len(families))
		for _, family := range families {
			names = append(names, family.GetName())
			for _, metric := range family.GetMetric() {
				require.Empty(t, metric.GetLabel(), "provider health metrics carry no labels")
			}
		}
		require.ElementsMatch(t, []string{
			"unkey_control_feature_flags_ready",
			"unkey_control_feature_flags_snapshot_age_seconds",
			"unkey_control_feature_flags_refresh_failures_total",
		}, names)
	})
}
