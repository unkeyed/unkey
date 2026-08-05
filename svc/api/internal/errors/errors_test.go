package errors

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
)

func TestHumanizeBytes(t *testing.T) {
	cases := []struct {
		in   uint64
		want string
	}{
		{0, "0 B"},
		{5, "5 B"},
		{9, "9 B"},
		{10, "10 B"},
		{999, "999 B"},
		{1000, "1.0 kB"},
		{1500, "1.5 kB"},
		{16000, "16 kB"},
		{1000000, "1.0 MB"},
		{1048576, "1.0 MB"},
		{1500000, "1.5 MB"},
		{1000000000, "1.0 GB"},
	}
	for _, tc := range cases {
		require.Equal(t, tc.want, humanizeBytes(tc.in), "humanizeBytes(%d)", tc.in)
	}
}

func TestMaskInsufficientPermissionsAsNotFound(t *testing.T) {
	t.Run("masks insufficient permissions", func(t *testing.T) {
		authorizationErr := fault.New("missing permission",
			fault.Code(codes.Auth.Authorization.InsufficientPermissions.URN()),
		)

		err := MaskInsufficientPermissionsAsNotFound(
			authorizationErr,
			codes.Data.Identity.NotFound.URN(),
			"This identity does not exist.",
		)

		code, ok := fault.GetCode(err)
		require.True(t, ok)
		require.Equal(t, codes.Data.Identity.NotFound.URN(), code)
	})

	t.Run("preserves other coded errors", func(t *testing.T) {
		systemErr := fault.New("database unavailable",
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
		)

		err := MaskInsufficientPermissionsAsNotFound(
			systemErr,
			codes.Data.Identity.NotFound.URN(),
			"This identity does not exist.",
		)

		require.Equal(t, systemErr, err)
	})

	t.Run("preserves untagged errors", func(t *testing.T) {
		originalErr := errors.New("authorization evaluator failed")

		err := MaskInsufficientPermissionsAsNotFound(
			originalErr,
			codes.Data.Identity.NotFound.URN(),
			"This identity does not exist.",
		)

		require.ErrorIs(t, err, originalErr)
	})
}
