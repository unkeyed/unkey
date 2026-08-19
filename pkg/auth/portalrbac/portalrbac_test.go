package portalrbac_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/unkeyed/unkey/pkg/auth/portalrbac"
	"github.com/unkeyed/unkey/pkg/rbac"
)

func TestCapabilitiesUseExactMatching(t *testing.T) {
	granted := []string{
		portalrbac.CapKeysRead,
		portalrbac.CapKeysCreate,
		portalrbac.CapAnalyticsRead,
	}

	require.NoError(t, rbac.Check(rbac.S(portalrbac.CapKeysRead), granted))
	require.NoError(t, rbac.Check(rbac.S(portalrbac.CapKeysCreate), granted))
	require.NoError(t, rbac.Check(rbac.S(portalrbac.CapAnalyticsRead), granted))
	require.Error(t, rbac.Check(rbac.S(portalrbac.CapKeysReroll), granted),
		"keys:create must not imply keys:reroll")
}
