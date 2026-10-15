package cluster

import (
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
)

func TestDesiredStateDistinguishesUnknownClusterFromMissingTopology(t *testing.T) {
	f := newDeletionFixture(t, 1)
	for _, test := range []struct {
		name    string
		cluster *ctrlv1.ClusterKey
		want    connect.Code
	}{
		{name: "unknown cluster", cluster: &ctrlv1.ClusterKey{CellId: "missing-cell", Platform: f.clusterKey.Platform, Region: f.clusterKey.Region}, want: connect.CodeFailedPrecondition},
		{name: "known cluster missing topology", cluster: f.clusterKey, want: connect.CodeNotFound},
	} {
		t.Run(test.name, func(t *testing.T) {
			req := connect.NewRequest(&ctrlv1.GetDesiredDeploymentStateRequest{Cluster: test.cluster, DeploymentId: "missing-deployment"})
			req.Header().Set("Authorization", "Bearer test-token")
			_, err := f.service.GetDesiredDeploymentState(t.Context(), req)
			require.Equal(t, test.want, connect.CodeOf(err))
		})
	}
}

func TestValidateClusterKey(t *testing.T) {
	t.Run("accepts complete key", func(t *testing.T) {
		err := validateClusterKey(&ctrlv1.ClusterKey{CellId: "cell001", Platform: "aws", Region: "us-east-1"})
		require.NoError(t, err)
	})

	t.Run("rejects nil key as InvalidArgument", func(t *testing.T) {
		err := validateClusterKey(nil)
		require.Error(t, err)
		require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
	})

	t.Run("rejects blank cell ID as InvalidArgument", func(t *testing.T) {
		err := validateClusterKey(&ctrlv1.ClusterKey{Platform: "aws", Region: "us-east-1"})
		require.Error(t, err)
		require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
	})

	t.Run("rejects blank platform as InvalidArgument", func(t *testing.T) {
		err := validateClusterKey(&ctrlv1.ClusterKey{CellId: "cell001", Region: "us-east-1"})
		require.Error(t, err)
		require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
	})

	t.Run("rejects blank region as InvalidArgument", func(t *testing.T) {
		err := validateClusterKey(&ctrlv1.ClusterKey{CellId: "cell001", Platform: "aws"})
		require.Error(t, err)
		require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
	})
}
