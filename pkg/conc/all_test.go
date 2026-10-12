package conc

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
)

func TestAll_RunsEveryFunction(t *testing.T) {
	var count int
	var names []string
	err := All(t.Context(),
		func(context.Context) error {
			count = 3
			return nil
		},
		func(context.Context) error {
			names = []string{"KEBAP"}
			return nil
		},
	)

	require.NoError(t, err)
	require.Equal(t, 3, count)
	require.Equal(t, []string{"KEBAP"}, names)
}

func TestAll_ReturnsErrorAndCancelsTheOthers(t *testing.T) {
	failed := errors.New("read failed")
	err := All(t.Context(),
		func(context.Context) error { return failed },
		func(ctx context.Context) error {
			<-ctx.Done()
			return ctx.Err()
		},
	)

	require.ErrorIs(t, err, failed)
}

// A panic in a goroutine stops the test binary when nothing recovers it
func TestAll_ChangesPanicIntoError(t *testing.T) {
	err := All(t.Context(), func(context.Context) error { panic("KEBAP") })

	require.Error(t, err)
	code, ok := fault.GetCode(err)
	require.True(t, ok)
	require.Equal(t, codes.App.Internal.UnexpectedError.URN(), code)
}
