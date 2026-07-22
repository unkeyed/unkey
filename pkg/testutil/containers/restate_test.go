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

func TestRestateResetsServiceBetweenRegistrations(t *testing.T) {
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
			cfg := Restate(t, service(marker))
			ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
			defer cancel()

			response, err := ingress.Object[string, string](
				ingress.NewClient(cfg.IngressURL), serviceName, "same-key", handlerName,
			).Request(ctx, "")
			require.NoError(t, err)
			require.Equal(t, marker+":1", response)
		})
	}
}

func TestRestateDrainsPendingInvocations(t *testing.T) {
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
		ingress.NewClient(cfg.IngressURL), serviceName, "key", "sleep",
	).Request(ctx, "")
	require.ErrorIs(t, err, context.DeadlineExceeded)
}

func TestRestateServiceLeasesSerializeOverlappingRegistrations(t *testing.T) {
	firstRelease, err := acquireRestateServiceLeases([]string{"unkey.test.Lease"})
	require.NoError(t, err)
	firstReleased := false
	defer func() {
		if !firstReleased {
			_ = firstRelease()
		}
	}()

	type leaseResult struct {
		release func() error
		err     error
	}
	second := make(chan leaseResult, 1)
	go func() {
		release, acquireErr := acquireRestateServiceLeases([]string{"unkey.test.Lease"})
		second <- leaseResult{release: release, err: acquireErr}
	}()

	select {
	case result := <-second:
		if result.release != nil {
			_ = result.release()
		}
		require.Fail(t, "overlapping Restate service lease was acquired before its owner released it")
	case <-time.After(100 * time.Millisecond):
	}

	require.NoError(t, firstRelease())
	firstReleased = true
	select {
	case result := <-second:
		require.NoError(t, result.err)
		require.NoError(t, result.release())
	case <-time.After(5 * time.Second):
		require.Fail(t, "overlapping Restate service lease was not acquired after its owner released it")
	}
}

func TestDeploymentOverlapsMultipleServices(t *testing.T) {
	deployment := restateDeployment{
		ID:       "deployment",
		Services: []restateDeploymentService{{Name: "service.b"}},
	}
	require.True(t, deploymentOverlaps(deployment, []string{"service.a", "service.b", "service.c"}))
	require.False(t, deploymentOverlaps(deployment, []string{"service.c", "service.d"}))
}
