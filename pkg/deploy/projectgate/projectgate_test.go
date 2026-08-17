package projectgate_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/deploy/projectgate"
	"github.com/unkeyed/unkey/pkg/fault"
)

func TestCheckSlug(t *testing.T) {
	require.NoError(t, projectgate.CheckSlug("default-project"))

	for _, slug := range []string{projectgate.DefaultSlug, "Default"} {
		t.Run(slug, func(t *testing.T) {
			err := projectgate.CheckSlug(slug)
			require.Error(t, err)

			code, ok := fault.GetCode(err)
			require.True(t, ok)
			require.Equal(t, codes.App.Validation.InvalidInput.URN(), code)
			require.Equal(t, "The project slug '"+slug+"' is reserved.", fault.UserFacingMessage(err))
		})
	}
}
