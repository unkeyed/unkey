package cluster

import (
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
)

func TestValidateRegionKey(t *testing.T) {
	t.Run("accepts populated key", func(t *testing.T) {
		err := validateRegionKey(&ctrlv1.RegionKey{Platform: "aws", Name: "us-east-1"})
		require.NoError(t, err)
	})

	t.Run("rejects nil key as InvalidArgument", func(t *testing.T) {
		err := validateRegionKey(nil)
		require.Error(t, err)
		require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
	})

	t.Run("rejects blank platform as InvalidArgument", func(t *testing.T) {
		err := validateRegionKey(&ctrlv1.RegionKey{Platform: "", Name: "us-east-1"})
		require.Error(t, err)
		require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
	})

	t.Run("rejects blank name as InvalidArgument", func(t *testing.T) {
		err := validateRegionKey(&ctrlv1.RegionKey{Platform: "aws", Name: ""})
		require.Error(t, err)
		require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
	})
}
