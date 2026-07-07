package app

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSpinnersAdvanceOnlyOnOwnTicks(t *testing.T) {
	a := NewSpinner()
	b := NewSpinner()
	require.NotEqual(t, a.ID(), b.ID(), "each spinner gets a distinct id")

	start := a.View()
	// A tick addressed to b must not advance a.
	a, _ = a.Update(SpinnerTickMsg{ID: b.ID()})
	require.Equal(t, start, a.View(), "foreign tick is ignored")

	// A tick addressed to a advances it and re-arms.
	next, cmd := a.Update(SpinnerTickMsg{ID: a.ID()})
	require.NotEqual(t, start, next.View(), "own tick advances the frame")
	require.NotNil(t, cmd, "own tick re-arms the next tick")
}
