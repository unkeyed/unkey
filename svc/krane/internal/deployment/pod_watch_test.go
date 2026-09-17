package deployment

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes/fake"
	ktesting "k8s.io/client-go/testing"
)

func TestRunPodWatchLoopRetriesStartupAndReconnectFailures(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		client := fake.NewClientset()
		first := &stubbornWatch{result: make(chan watch.Event)}
		second := &stubbornWatch{result: make(chan watch.Event)}
		var attempts atomic.Int32
		client.PrependWatchReactor("pods", func(ktesting.Action) (bool, watch.Interface, error) {
			switch attempts.Add(1) {
			case 1, 2, 4, 5:
				return true, nil, errors.New("watch unavailable")
			case 3:
				return true, first, nil
			default:
				return true, second, nil
			}
		})
		ctrl := &Controller{clientSet: client}
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()
		done := make(chan struct{})
		go func() {
			ctrl.runPodWatchLoop(ctx)
			close(done)
		}()
		time.Sleep(11 * time.Second)
		require.Equal(t, int32(3), attempts.Load())
		close(first.result)
		time.Sleep(16 * time.Second)
		require.Equal(t, int32(6), attempts.Load())
		require.True(t, first.stopped.Load())
		cancel()
		synctest.Wait()
		require.True(t, second.stopped.Load())
		select {
		case <-done:
		default:
			t.Fatal("watch loop did not stop after cancellation")
		}
	})
}

func TestRunPodWatchLoopCancellationInterruptsBackoff(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		client := fake.NewClientset()
		var attempts atomic.Int32
		client.PrependWatchReactor("pods", func(ktesting.Action) (bool, watch.Interface, error) {
			attempts.Add(1)
			return true, nil, errors.New("watch unavailable")
		})
		ctrl := &Controller{clientSet: client}
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()
		done := make(chan struct{})
		go func() {
			ctrl.runPodWatchLoop(ctx)
			close(done)
		}()
		synctest.Wait()
		require.Equal(t, int32(1), attempts.Load())
		cancel()
		synctest.Wait()
		select {
		case <-done:
		default:
			t.Fatal("watch loop did not interrupt reconnect backoff")
		}
		require.Equal(t, int32(1), attempts.Load())
	})
}

type stubbornWatch struct {
	result  chan watch.Event
	stopped atomic.Bool
}

func (w *stubbornWatch) Stop() { w.stopped.Store(true) }

func (w *stubbornWatch) ResultChan() <-chan watch.Event { return w.result }
