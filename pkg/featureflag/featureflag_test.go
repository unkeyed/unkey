package featureflag

import (
	"context"
	"testing"

	"github.com/open-feature/go-sdk/openfeature"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/featureflag/static"
)

func TestNewUsesIsolatedNoopByDefault(t *testing.T) {
	ctx := context.Background()
	first, err := New(ctx, nil)
	require.NoError(t, err)
	second, err := New(ctx, static.New(static.Values{"enabled": true}))
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

func TestTeamContextUsesTeamID(t *testing.T) {
	evaluation := TeamContext("org_2")
	require.Empty(t, evaluation.TargetingKey())
	require.Equal(t, map[string]any{"team": map[string]any{"id": "org_2"}}, evaluation.Attributes())
}
