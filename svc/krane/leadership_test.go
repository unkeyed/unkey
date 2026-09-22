package krane

import (
	"context"
	"strconv"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/require"
	coordinationv1 "k8s.io/api/coordination/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes/fake"
	ktesting "k8s.io/client-go/testing"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
)

func TestLeadershipHandoverWaitsForWorkers(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		client, _ := leaseTestClient(t)
		ctxA, cancelA := context.WithCancel(t.Context())
		defer cancelA()
		ctxB, cancelB := context.WithCancel(t.Context())
		defer cancelB()

		var active atomic.Int32
		var overlap atomic.Bool
		var startsB atomic.Int32
		drainA := make(chan struct{})
		resultA := make(chan error, 1)
		resultB := make(chan error, 1)

		go func() {
			resultA <- runWithLeadership(ctxA, client, "unkey", "a", func(ctx context.Context) {
				if active.Add(1) != 1 {
					overlap.Store(true)
				}
				<-ctx.Done()
				<-drainA
				active.Add(-1)
			})
		}()
		synctest.Wait()
		require.Equal(t, int32(1), active.Load())

		go func() {
			resultB <- runWithLeadership(ctxB, client, "unkey", "b", func(ctx context.Context) {
				startsB.Add(1)
				if active.Add(1) != 1 {
					overlap.Store(true)
				}
				<-ctx.Done()
				active.Add(-1)
			})
		}()
		time.Sleep(6 * time.Second)
		require.Zero(t, startsB.Load())

		cancelA()
		synctest.Wait()
		lease, err := client.CoordinationV1().Leases("unkey").Get(t.Context(), "krane", metav1.GetOptions{})
		require.NoError(t, err)
		require.Equal(t, "a", *lease.Spec.HolderIdentity)
		require.Empty(t, resultA)

		close(drainA)
		synctest.Wait()
		require.NoError(t, <-resultA)
		time.Sleep(5 * time.Second)
		require.Equal(t, int32(1), startsB.Load())
		require.False(t, overlap.Load())

		cancelB()
		synctest.Wait()
		require.NoError(t, <-resultB)
		require.Zero(t, active.Load())
	})
}

func TestLeadershipRecoversFromStartupAndRenewalOutages(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		client, unavailable := leaseTestClient(t)
		unavailable.Store(true)
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()
		var active atomic.Int32
		var starts atomic.Int32
		result := make(chan error, 1)

		go func() {
			result <- runWithLeadership(ctx, client, "unkey", "a", func(ctx context.Context) {
				starts.Add(1)
				active.Add(1)
				<-ctx.Done()
				active.Add(-1)
			})
		}()
		time.Sleep(20 * time.Second)
		require.Zero(t, starts.Load())
		require.Empty(t, result)

		unavailable.Store(false)
		time.Sleep(5 * time.Second)
		require.Equal(t, int32(1), active.Load())

		unavailable.Store(true)
		time.Sleep(14 * time.Second)
		require.Zero(t, active.Load())
		require.Empty(t, result)

		unavailable.Store(false)
		time.Sleep(5 * time.Second)
		require.Equal(t, int32(1), active.Load())
		require.Equal(t, int32(2), starts.Load())

		cancel()
		synctest.Wait()
		require.NoError(t, <-result)
		require.Zero(t, active.Load())
	})
}

func TestLeadershipCancellationBeforeAcquisition(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		client, unavailable := leaseTestClient(t)
		unavailable.Store(true)
		ctx, cancel := context.WithCancel(t.Context())
		var starts atomic.Int32
		result := make(chan error, 1)

		go func() {
			result <- runWithLeadership(ctx, client, "unkey", "a", func(context.Context) { starts.Add(1) })
		}()
		synctest.Wait()

		cancel()
		synctest.Wait()
		require.NoError(t, <-result)
		require.Zero(t, starts.Load())
	})
}

func TestReleaseLeadershipPreservesOtherHolder(t *testing.T) {
	client, _ := leaseTestClient(t)
	holder := "other"
	_, err := client.CoordinationV1().Leases("unkey").Create(t.Context(), &coordinationv1.Lease{
		ObjectMeta: metav1.ObjectMeta{Name: "krane", Namespace: "unkey"},
		Spec:       coordinationv1.LeaseSpec{HolderIdentity: &holder},
	}, metav1.CreateOptions{})
	require.NoError(t, err)

	lock := &resourcelock.LeaseLock{
		LeaseMeta:  metav1.ObjectMeta{Name: "krane", Namespace: "unkey"},
		Client:     client.CoordinationV1(),
		LockConfig: resourcelock.ResourceLockConfig{Identity: "a"},
	}
	releaseLeadership(t.Context(), lock)

	lease, err := client.CoordinationV1().Leases("unkey").Get(t.Context(), "krane", metav1.GetOptions{})
	require.NoError(t, err)
	require.Equal(t, holder, *lease.Spec.HolderIdentity)
	for _, action := range client.Actions() {
		require.NotEqual(t, "update", action.GetVerb())
	}
}

func leaseTestClient(t *testing.T) (*fake.Clientset, *atomic.Bool) {
	t.Helper()

	client := fake.NewClientset()
	unavailable := &atomic.Bool{}
	resource := schema.GroupVersionResource{Group: "coordination.k8s.io", Version: "v1", Resource: "leases"}
	client.PrependReactor("*", "leases", func(action ktesting.Action) (bool, runtime.Object, error) {
		if unavailable.Load() {
			return true, nil, apierrors.NewServiceUnavailable("test API outage")
		}

		switch action.GetVerb() {
		case "create":
			lease := action.(ktesting.CreateAction).GetObject().(*coordinationv1.Lease).DeepCopy()
			lease.ResourceVersion = "1"
			err := client.Tracker().Create(resource, lease, lease.Namespace)
			return true, lease, err
		case "update":
			lease := action.(ktesting.UpdateAction).GetObject().(*coordinationv1.Lease).DeepCopy()
			stored, err := client.Tracker().Get(resource, lease.Namespace, lease.Name)
			if err != nil {
				return true, nil, err
			}

			previous := stored.(*coordinationv1.Lease)
			if lease.ResourceVersion != previous.ResourceVersion {
				return true, nil, apierrors.NewConflict(resource.GroupResource(), lease.Name, nil)
			}

			version, err := strconv.Atoi(previous.ResourceVersion)
			if err != nil {
				return true, nil, err
			}
			lease.ResourceVersion = strconv.Itoa(version + 1)
			err = client.Tracker().Update(resource, lease, lease.Namespace)
			return true, lease, err
		default:
			return false, nil, nil
		}
	})

	return client, unavailable
}
