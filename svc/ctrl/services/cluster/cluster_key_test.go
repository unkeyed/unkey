package cluster

import (
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
)

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
