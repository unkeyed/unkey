package defaults_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/defaults"
)

func TestOr(t *testing.T) {
	t.Parallel()

	require.Equal(t, 10, defaults.Or(0, 10))
	require.Equal(t, 5, defaults.Or(5, 10))
	require.Equal(t, time.Hour, defaults.Or(time.Duration(0), time.Hour))
	require.Equal(t, "set", defaults.Or("set", "fallback"))
	require.Equal(t, "fallback", defaults.Or("", "fallback"))
}

func TestOrFunc_OnlyEvaluatesFallbackOnZero(t *testing.T) {
	t.Parallel()

	// The whole point of OrFunc is that the fallback does not run when
	// the value is already set. Verify by making the fallback observable.
	called := 0
	fb := func() int { called++; return 99 }

	require.Equal(t, 7, defaults.OrFunc(7, fb))
	require.Equal(t, 0, called, "fallback must not run when first is non-zero")

	require.Equal(t, 99, defaults.OrFunc(0, fb))
	require.Equal(t, 1, called)
}

func TestOr_NilableInterface(t *testing.T) {
	t.Parallel()

	type fooer interface{ Foo() }
	var nilFooer fooer
	concrete := concreteFooer{}
	require.Equal(t, fooer(concrete), defaults.Or[fooer](nilFooer, concrete))
}

type concreteFooer struct{}

func (concreteFooer) Foo() {}
