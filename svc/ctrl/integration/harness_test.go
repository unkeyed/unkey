package integration

import (
	"testing"

	"github.com/stretchr/testify/require"
	mysqltype "github.com/unkeyed/unkey/pkg/mysql/types"
	"github.com/unkeyed/unkey/pkg/uid"
)

func TestCreateDeploymentResolvedImage(t *testing.T) {
	h := New(t)
	deployment := h.CreateDeployment(h.Context(), CreateDeploymentRequest{
		Region:       uid.DNS1035(12),
		DesiredState: mysqltype.DeploymentsDesiredStateRunning,
	})

	require.True(t, deployment.ImageResolved.Valid)
	require.Equal(t, "nginx:1.19", deployment.ImageResolved.String)
}
