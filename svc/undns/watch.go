package undns

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/unkeyed/unkey/pkg/logger"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
)

type trackedInformer struct {
	cache.SharedIndexInformer
	lastContact atomic.Int64
	timeout     time.Duration
}

func newTrackedInformer(name string, object runtime.Object, indexes cache.Indexers, timeout time.Duration, labelSelector string, list cache.ListWithContextFunc, watchFunc cache.WatchFuncWithContext) (*trackedInformer, error) {
	t := &trackedInformer{SharedIndexInformer: nil, lastContact: atomic.Int64{}, timeout: timeout}
	lw := &cache.ListWatch{
		ListFunc: nil, WatchFunc: nil, DisableChunking: false,
		ListWithContextFunc: func(ctx context.Context, options metav1.ListOptions) (runtime.Object, error) {
			options.LabelSelector = labelSelector
			result, err := list(ctx, options)
			if err == nil {
				t.lastContact.Store(time.Now().UnixNano())
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
				t.lastContact.Store(time.Now().UnixNano())
			}
			return result, err
		},
	}

	t.SharedIndexInformer = cache.NewSharedIndexInformer(lw, object, 0, indexes)
	err := t.SetWatchErrorHandlerWithContext(func(_ context.Context, _ *cache.Reflector, err error) {
		t.lastContact.Store(0)
		logger.Warn("DNS discovery watch failed", "resource", name, "error", err)
	})
	if err != nil {
		return nil, fmt.Errorf("configure %s watch: %w", name, err)
	}
	return t, nil
}

func (t *trackedInformer) healthy() bool {
	// HasSynced never becomes false after a lost watch. Short server-side
	// watch leases also bound stale serving when a connection silently stalls.
	return t.HasSynced() && !t.IsStopped() && time.Since(time.Unix(0, t.lastContact.Load())) < 2*t.timeout
}
