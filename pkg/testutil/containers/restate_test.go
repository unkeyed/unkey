package containers

import (
	"context"
	"fmt"
	"testing"
	"time"

	restate "github.com/restatedev/sdk-go"
	"github.com/restatedev/sdk-go/ingress"
	"github.com/stretchr/testify/require"
)

// TestRestateIsolatesConcurrentRegistrations pins the property the helper
// exists for. Both subtests register the same service name, under the same
// object key, at the same time. Sharing one Restate server would route both
// through whichever worker registered last and share its state, so a marker
// mismatch or a count above one means isolation broke.
func TestRestateIsolatesConcurrentRegistrations(t *testing.T) {
	t.Parallel()

	const (
		serviceName = "unkey.test.Isolation"
		handlerName = "increment"
	)

	service := func(marker string) restate.ServiceDefinition {
		return restate.NewObject(serviceName).
			Handler(handlerName, restate.NewObjectHandler(
				func(ctx restate.ObjectContext, _ string) (string, error) {
					count, err := restate.Get[int](ctx, "count")
					if err != nil {
						return "", err
					}
					restate.Set(ctx, "count", count+1)
					return fmt.Sprintf("%s:%d", marker, count+1), nil
				}))
	}

	for _, marker := range []string{"first", "second"} {
		t.Run(marker, func(t *testing.T) {
			t.Parallel()

			cfg := Restate(t, service(marker))
			ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
			defer cancel()

			response, err := ingress.Object[string, string](
				cfg.IngressClient, serviceName, "same-key", handlerName,
			).Request(ctx, "")
			require.NoError(t, err)
			require.Equal(t, marker+":1", response)
		})
	}
}

// TestRestateCleansUpWithInvocationInFlight leaves a handler parked for an hour
// when the test ends. Cleanup has to remove the container before closing the
// worker, otherwise it waits on the invocation's still-open request and the
// test binary hangs instead of finishing.
func TestRestateCleansUpWithInvocationInFlight(t *testing.T) {
	t.Parallel()

	const serviceName = "unkey.test.PendingInvocation"
	service := restate.NewObject(serviceName).
		Handler("sleep", restate.NewObjectHandler(
			func(ctx restate.ObjectContext, _ string) (string, error) {
				if err := restate.Sleep(ctx, time.Hour); err != nil {
					return "", err
				}
				return "done", nil
			}))

	cfg := Restate(t, service)
	ctx, cancel := context.WithTimeout(t.Context(), 250*time.Millisecond)
	defer cancel()
	_, err := ingress.Object[string, string](
		cfg.IngressClient, serviceName, "key", "sleep",
	).Request(ctx, "")
	require.ErrorIs(t, err, context.DeadlineExceeded)
}

func TestIsolatedProjectOwner(t *testing.T) {
	t.Parallel()

	const prefix = "unkey-test-abc123-restate-"

	owner, ok := isolatedProjectOwner(prefix+"4242-7", prefix)
	require.True(t, ok)
	require.Equal(t, 4242, owner)

	_, ok = isolatedProjectOwner("unkey-test-abc123-mysql-1", prefix)
	require.False(t, ok, "a shared service project must never be reaped")

	_, ok = isolatedProjectOwner("unkey-test-abc123", prefix)
	require.False(t, ok)

	_, ok = isolatedProjectOwner(prefix+"notapid-1", prefix)
	require.False(t, ok)

	_, ok = isolatedProjectOwner(prefix+"4242", prefix)
	require.False(t, ok, "a name without the sequence suffix is not one of ours")
}
