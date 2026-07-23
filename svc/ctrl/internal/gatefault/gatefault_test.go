package gatefault_test

import (
	"errors"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/svc/ctrl/internal/gatefault"
)

func TestConnect(t *testing.T) {
	require.NoError(t, gatefault.Connect(nil))

	f := fault.New("internal detail", fault.Public("The deployment is not ready."))
	err := gatefault.Connect(f)

	var ce *connect.Error
	require.True(t, errors.As(err, &ce))
	require.Equal(t, connect.CodeFailedPrecondition, ce.Code())
	require.Equal(t, "The deployment is not ready.", ce.Message())
	require.NotContains(t, ce.Message(), "internal detail")
}

func TestTerminal(t *testing.T) {
	require.NoError(t, gatefault.Terminal(nil))

	f := fault.New("internal detail", fault.Public("The deployment is not ready."))
	err := gatefault.Terminal(f)

	require.Error(t, err)
	require.Contains(t, err.Error(), "The deployment is not ready.")
	require.NotContains(t, err.Error(), "internal detail")
}
