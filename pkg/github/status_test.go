package github

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDeployAuthorizationStatusContextIsolatesTargets(t *testing.T) {
	context := DeployAuthorizationStatusContext("app_1", "environment_1")

	require.Equal(t, context, DeployAuthorizationStatusContext("app_1", "environment_1"))
	require.NotEqual(t, context, DeployAuthorizationStatusContext("app_2", "environment_1"))
	require.NotEqual(t, context, DeployAuthorizationStatusContext("app_1", "environment_2"))
}
