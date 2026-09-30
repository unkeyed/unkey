package featureflag

import (
	"context"
	"testing"

	"github.com/open-feature/go-sdk/openfeature"
	"github.com/stretchr/testify/require"
)

func TestNewUsesIsolatedNoopByDefault(t *testing.T) {
	ctx := context.Background()
	first, err := New(ctx, nil)
	require.NoError(t, err)
	second, err := New(ctx, Static(map[string]bool{"enabled": true}))
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, first.Shutdown(ctx))
		require.NoError(t, second.Shutdown(ctx))
	})

	firstDetail, err := first.NewClient().BooleanValueDetails(ctx, "enabled", false, openfeature.EvaluationContext{})
	require.NoError(t, err)
	require.False(t, firstDetail.Value)
	secondDetail, err := second.NewClient().BooleanValueDetails(ctx, "enabled", false, openfeature.EvaluationContext{})
	require.NoError(t, err)
	require.True(t, secondDetail.Value)
}

func TestStaticResolvesListedFlagsAndRejectsUnlisted(t *testing.T) {
	ctx := context.Background()
	api, err := New(ctx, Static(map[string]bool{"on": true, "off": false}))
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, api.Shutdown(ctx)) })
	client := api.NewClient()

	on, err := client.BooleanValueDetails(ctx, "on", false, TeamContext("org_1"))
	require.NoError(t, err)
	require.True(t, on.Value)

	off, err := client.BooleanValueDetails(ctx, "off", true, TeamContext("org_1"))
	require.NoError(t, err)
	require.False(t, off.Value)

	missing, err := client.BooleanValueDetails(ctx, "missing", false, TeamContext("org_1"))
	require.Error(t, err)
	require.Equal(t, openfeature.FlagNotFoundCode, missing.ErrorCode)
}

func TestTeamContextUsesTeamID(t *testing.T) {
	evaluation := TeamContext("org_2")
	require.Empty(t, evaluation.TargetingKey())
	require.Equal(t, map[string]any{"team": map[string]any{"id": "org_2"}}, evaluation.Attributes())
}
