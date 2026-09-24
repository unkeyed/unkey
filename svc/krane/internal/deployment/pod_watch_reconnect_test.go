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

func TestRunPodWatchLoopRecoversAfterFailedReconnect(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		client := fake.NewClientset()
		first := watch.NewFake()
		first.Stop()
		reconnected := watch.NewFake()
		defer reconnected.Stop()

		var attempts atomic.Int32
		client.PrependWatchReactor("pods", func(ktesting.Action) (bool, watch.Interface, error) {
			switch attempts.Add(1) {
			case 1:
				return true, first, nil
			case 2:
				return true, nil, errors.New("API server unavailable")
			default:
				return true, reconnected, nil
			}
		})

		ctrl := &Controller{clientSet: client}
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()

		go ctrl.runPodWatchLoop(ctx)

		time.Sleep(11 * time.Second)
		require.Equal(t, int32(3), attempts.Load())

		cancel()
		synctest.Wait()
	})
}
