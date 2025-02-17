package sim_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/go/pkg/sim"
)

func TestSimulation(t *testing.T) {

	sim := sim.New[any](t)

	sim.Run([]sim.Event[any]{})
	t.Fail()
	require.GreaterOrEqual(t, len(sim.Errors), 10000)
}
