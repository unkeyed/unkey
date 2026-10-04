package static_test

import (
	"context"
	"testing"

	"github.com/open-feature/go-sdk/openfeature"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/featureflag"
	"github.com/unkeyed/unkey/pkg/featureflag/static"
)

func TestStaticResolvesListedFlagsAndRejectsUnlisted(t *testing.T) {
	api, err := featureflag.New(t.Context(), static.New(static.Values{"on": true, "off": false}))
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, api.Shutdown(context.Background())) })
	client := api.NewClient()

	on, err := client.BooleanValueDetails(t.Context(), "on", false, featureflag.TeamContext("org_1"))
	require.NoError(t, err)
	require.True(t, on.Value)

	off, err := client.BooleanValueDetails(t.Context(), "off", true, featureflag.TeamContext("org_1"))
	require.NoError(t, err)
	require.False(t, off.Value)

	missing, err := client.BooleanValueDetails(t.Context(), "missing", false, featureflag.TeamContext("org_1"))
	require.Error(t, err)
	require.Equal(t, openfeature.FlagNotFoundCode, missing.ErrorCode)
}

func TestValuesValidate(t *testing.T) {
	require.NoError(t, static.Values{"enabled": false}.Validate())
	require.Error(t, static.Values{}.Validate())
	require.Error(t, static.Values(nil).Validate())
}
