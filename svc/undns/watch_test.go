package undns

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/features"
	featuretesting "k8s.io/client-go/features/testing"
)

func TestTrackedInformerWatchFailureStallAndRecovery(t *testing.T) {
	featuretesting.SetFeatureDuringTest(t, features.WatchListClient, false)
	t.Run("watch error and recovery", func(t *testing.T) {
		first := watch.NewRaceFreeFake()
		recovered := watch.NewRaceFreeFake()
		var watchCalls atomic.Int64
		informer, err := newTrackedInformer("pods", &corev1.Pod{}, nil, time.Minute, "",
			func(context.Context, metav1.ListOptions) (runtime.Object, error) {
				return &corev1.PodList{ListMeta: metav1.ListMeta{ResourceVersion: "1"}}, nil
			},
			func(context.Context, metav1.ListOptions) (watch.Interface, error) {
				switch watchCalls.Add(1) {
				case 1:
					return first, nil
				case 2:
					return nil, errors.New("watch transport failed")
				default:
					return recovered, nil
				}
			})
		require.NoError(t, err)
		runTrackedInformer(t, informer)
		require.Eventually(t, informer.HasSynced, time.Second, time.Millisecond)
		require.True(t, informer.healthy())

		first.Stop()
		require.Eventually(t, func() bool { return watchCalls.Load() >= 2 }, 3*time.Second, time.Millisecond)
		require.Eventually(t, func() bool {
			return informer.HasSynced() && !informer.healthy()
		}, time.Second, time.Millisecond)
		require.Eventually(t, func() bool {
			return watchCalls.Load() >= 3 && informer.healthy()
		}, 6*time.Second, time.Millisecond)
	})

	t.Run("silent open watch expires", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			const timeout = time.Second
			stalled := watch.NewRaceFreeFake()
			informer, err := newTrackedInformer("pods", &corev1.Pod{}, nil, timeout, "",
				func(context.Context, metav1.ListOptions) (runtime.Object, error) {
					return &corev1.PodList{ListMeta: metav1.ListMeta{ResourceVersion: "1"}}, nil
				},
				func(context.Context, metav1.ListOptions) (watch.Interface, error) { return stalled, nil })
			require.NoError(t, err)
			runTrackedInformer(t, informer)
			synctest.Wait()
			require.True(t, informer.HasSynced())
			require.True(t, informer.healthy())
			time.Sleep(2*timeout - time.Nanosecond)
			require.True(t, informer.healthy(), "do not expire before two watch leases")
			time.Sleep(time.Nanosecond)
			require.False(t, informer.healthy(), "a silent watch must expire even though HasSynced stays true")
			require.True(t, informer.HasSynced())
		})
	})
}

func runTrackedInformer(t *testing.T, informer *trackedInformer) {
	t.Helper()
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan struct{})
	go func() {
		informer.RunWithContext(ctx)
		close(done)
	}()
	t.Cleanup(func() {
		cancel()
		select {
		case <-done:
		case <-time.After(time.Second):
			t.Error("tracked informer did not stop")
		}
	})
}
