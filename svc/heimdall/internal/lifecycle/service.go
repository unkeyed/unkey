package lifecycle

import (
	"context"
	"fmt"
	"sync/atomic"

	"github.com/unkeyed/unkey/pkg/clickhouse"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/svc/heimdall/internal/collector"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/tools/cache"
)

// Config holds the configuration for the lifecycle tracker.
type Config struct {
	CH        clickhouse.Bufferer
	Collector *collector.Collector
	Factory   informers.SharedInformerFactory
	Region    string
	Platform  string
}

// Tracker watches pod lifecycle events and emits billing events to ClickHouse.
type Tracker struct {
	ch        clickhouse.Bufferer
	collector *collector.Collector
	factory   informers.SharedInformerFactory
	region    string
	platform  string

	// synced is set to true after the initial informer cache replay is done.
	// AddFunc fires for every existing pod during the initial list — we skip
	// emitting "started" events for those since the pods already existed
	// before heimdall started.
	synced atomic.Bool
}

// New creates a new lifecycle tracker.
func New(cfg Config) *Tracker {
	return &Tracker{
		ch:        cfg.CH,
		collector: cfg.Collector,
		factory:   cfg.Factory,
		region:    cfg.Region,
		platform:  cfg.Platform,
	}
}

// Run registers pod event handlers and blocks until the context is cancelled.
// The informer factory must already be started (via factory.Start) before calling Run.
func (t *Tracker) Run(ctx context.Context) error {
	podInformer := t.factory.Core().V1().Pods().Informer()

	_, err := podInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			pod := toKranePod(obj)
			if pod == nil {
				return
			}

			// Skip pods that already existed before heimdall started.
			// The informer replays the full cache as AddFunc calls on registration.
			if !t.synced.Load() {
				return
			}

			t.emitEvent(pod, "started")
			t.collector.CollectPod(ctx, pod.Name)
		},
		DeleteFunc: func(obj interface{}) {
			pod := toKranePod(obj)
			if pod == nil {
				return
			}

			t.emitEvent(pod, "stopped")
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			oldPod, newPod := toKranePod(oldObj), toKranePod(newObj)
			if oldPod == nil || newPod == nil {
				return
			}

			if oldPod.DeletionTimestamp == nil && newPod.DeletionTimestamp != nil {
				t.emitEvent(newPod, "stopping")
				t.collector.CollectPod(ctx, newPod.Name)
			}
		},
	})
	if err != nil {
		return fmt.Errorf("adding pod event handler: %w", err)
	}

	// Wait for the initial cache replay to finish, then emit "observed"
	// events for all pre-existing pods so the billing service has a baseline.
	// These are NOT "started" events — the pod may have been running for hours.
	// The billing service uses these to reconcile gaps if heimdall restarted.
	cache.WaitForCacheSync(ctx.Done(), podInformer.HasSynced)

	var observed int
	for _, obj := range podInformer.GetStore().List() {
		pod := toKranePod(obj)
		if pod == nil {
			continue
		}
		t.emitEvent(pod, "observed")
		observed++
	}
	logger.Info("lifecycle tracker started", "observed_krane_pods", observed)

	t.synced.Store(true)
	<-ctx.Done()
	return ctx.Err()
}

// toKranePod casts obj to a krane-managed deployment pod, or returns nil.
func toKranePod(obj interface{}) *corev1.Pod {
	pod, ok := obj.(*corev1.Pod)
	if !ok || !isKraneManagedDeployment(pod) {
		return nil
	}

	return pod
}
