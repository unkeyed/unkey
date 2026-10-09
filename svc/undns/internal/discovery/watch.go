package discovery

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/logger"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
)

type trackedInformer struct {
	cache.SharedIndexInformer
	resource    string
	lastContact atomic.Int64
	timeout     time.Duration
	clock       clock.Clock
}

func newTrackedInformer(resource string, object runtime.Object, indexes cache.Indexers, timeout time.Duration, clk clock.Clock, labelSelector string, list cache.ListWithContextFunc, watchFunc cache.WatchFuncWithContext) (*trackedInformer, error) {
	t := &trackedInformer{SharedIndexInformer: nil, resource: resource, lastContact: atomic.Int64{}, timeout: timeout, clock: clk}
	lw := &cache.ListWatch{
		ListFunc: nil, WatchFunc: nil, DisableChunking: false,
		ListWithContextFunc: func(ctx context.Context, options metav1.ListOptions) (runtime.Object, error) {
			options.LabelSelector = labelSelector
			result, err := list(ctx, options)
			if err == nil {
				t.lastContact.Store(clk.Now().UnixNano())
			}
			return result, err
		},
		WatchFuncWithContext: func(ctx context.Context, options metav1.ListOptions) (watch.Interface, error) {
			seconds := int64(timeout / time.Second)
			options.TimeoutSeconds = &seconds
			options.AllowWatchBookmarks = true
			options.LabelSelector = labelSelector
			result, err := watchFunc(ctx, options)
			if err == nil {
				t.lastContact.Store(clk.Now().UnixNano())
			}
			return result, err
		},
	}

	t.SharedIndexInformer = cache.NewSharedIndexInformer(lw, object, 0, indexes)
	err := t.SetWatchErrorHandlerWithContext(func(_ context.Context, _ *cache.Reflector, err error) {
		t.lastContact.Store(0)
		logger.Warn("DNS discovery watch failed", "resource", resource, "error", err)
	})
	if err != nil {
		return nil, fmt.Errorf("configure %s watch: %w", resource, err)
	}
	return t, nil
}

func (t *trackedInformer) healthy() bool {
	// HasSynced never becomes false after a lost watch. Short server-side
	// watch leases also bound stale serving when a connection silently stalls.
	return t.HasSynced() && !t.IsStopped() && t.clock.Now().Sub(time.Unix(0, t.lastContact.Load())) < 2*t.timeout
}
