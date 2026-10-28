package collector

import (
	"context"
	"errors"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/batch"
	"github.com/unkeyed/unkey/pkg/clickhouse/schema"
	"github.com/unkeyed/unkey/pkg/prometheus/lazy"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/heimdall/internal/network"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
)

type blockingPodLister struct {
	corelisters.PodLister
	entered chan struct{}
	release chan struct{}
	once    sync.Once
	calls   chan struct{}
}

func (l *blockingPodLister) List(selector labels.Selector) ([]*corev1.Pod, error) {
	l.once.Do(func() { close(l.entered) })
	select {
	case l.calls <- struct{}{}:
	default:
	}
	<-l.release
	return l.PodLister.List(selector)
}

type delayedCancellationContext struct {
	context.Context
	release <-chan struct{}
}

func (c delayedCancellationContext) Value(any) any { return nil }

func (c delayedCancellationContext) AfterFunc(f func()) func() bool {
	return context.AfterFunc(c.Context, func() {
		<-c.release
		f()
	})
}

func TestRunStopsBeforeChildContextCancellation(t *testing.T) {
	parent, cancel := context.WithCancel(t.Context())
	defer cancel()
	propagate := make(chan struct{})
	defer close(propagate)
	ctx := delayedCancellationContext{Context: parent, release: propagate}
	index := cache.NewIndexer(cache.MetaNamespaceKeyFunc, cache.Indexers{})
	lister := &blockingPodLister{
		PodLister: corelisters.NewPodLister(index), entered: make(chan struct{}),
		release: make(chan struct{}), calls: make(chan struct{}, 1),
	}
	c := New(Config{PodLister: lister})
	done := make(chan error, 1)
	go func() { done <- c.Run(ctx, time.Millisecond) }()
	<-lister.entered
	<-lister.calls
	cancel()
	close(lister.release)
	require.ErrorIs(t, <-done, context.Canceled)
	select {
	case <-lister.calls:
		t.Fatal("collection continued after Run returned while child cancellation was delayed")
	case <-time.After(50 * time.Millisecond):
	}
}

func TestRunDrainsPeriodicCheckpoint(t *testing.T) {
	t.Run("in-flight tick", func(t *testing.T) {
		pod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{UID: types.UID(uid.New(uid.InstancePrefix)), Labels: map[string]string{
				LabelManagedBy: "krane", LabelComponent: "deployment", LabelWorkspace: uid.New(uid.WorkspacePrefix),
			}},
			Spec: corev1.PodSpec{NodeName: "test", Containers: []corev1.Container{{Name: "primary"}}},
			Status: corev1.PodStatus{Phase: corev1.PodRunning, ContainerStatuses: []corev1.ContainerStatus{
				{Name: "primary", RestartCount: 3},
			}},
		}
		index := cache.NewIndexer(cache.MetaNamespaceKeyFunc, cache.Indexers{})
		require.NoError(t, index.Add(pod))
		lister := &blockingPodLister{PodLister: corelisters.NewPodLister(index), entered: make(chan struct{}), release: make(chan struct{})}
		rows := make(chan schema.InstanceCheckpoint, 10)
		buffer := batch.New(batch.Config[schema.InstanceCheckpoint]{
			Name: "shutdown", BatchSize: 1, BufferSize: 10, FlushInterval: time.Hour,
			Flush: func(_ context.Context, batch []schema.InstanceCheckpoint) {
				for _, row := range batch {
					rows <- row
				}
			},
		})
		c := New(Config{CH: buffer, PodLister: lister, NodeName: "test", Collectors: CollectorSetFrom([]string{"disk"})})
		ctx, cancel := context.WithCancel(t.Context())
		done := make(chan error, 1)
		go func() {
			err := c.Run(ctx, time.Hour)
			buffer.Close()
			done <- err
		}()
		<-lister.entered
		cancel()
		returnedEarly := false
		select {
		case err := <-done:
			require.ErrorIs(t, err, context.Canceled)
			returnedEarly = true
		case <-time.After(100 * time.Millisecond):
		}
		close(lister.release)
		if !returnedEarly {
			require.ErrorIs(t, <-done, context.Canceled)
		}
		c.mu.Lock()
		c.mu.Unlock()
		t.Logf("returned before tick completed=%t flushed rows=%d", returnedEarly, len(rows))
		require.False(t, returnedEarly)
		require.Len(t, rows, 1)
		row := <-rows
		require.Equal(t, string(pod.UID), row.PodUID)
		require.EqualValues(t, 3, row.RestartCount)
	})
}

var metricsRegistry = prometheus.NewRegistry()

func TestMain(m *testing.M) {
	lazy.SetRegistry(metricsRegistry)
	os.Exit(m.Run())
}

func metricValue(t *testing.T, name string, labelPairs ...string) float64 {
	t.Helper()
	families, err := metricsRegistry.Gather()
	require.NoError(t, err)
	var total float64
	for _, family := range families {
		if family.GetName() != name {
			continue
		}
	series:
		for _, m := range family.GetMetric() {
			labels := map[string]string{}
			for _, label := range m.GetLabel() {
				labels[label.GetName()] = label.GetValue()
			}
			for i := 0; i+1 < len(labelPairs); i += 2 {
				if labels[labelPairs[i]] != labelPairs[i+1] {
					continue series
				}
			}
			total += m.GetCounter().GetValue() + m.GetGauge().GetValue()
		}
	}
	return total
}

type occupancyNetwork struct {
	network.Reader
	entries    int
	err        error
	reconciled int
}

func (n *occupancyNetwork) MapEntries() (int, error)         { return n.entries, n.err }
func (n *occupancyNetwork) Reconcile(map[types.UID]struct{}) { n.reconciled++ }

func TestCollectMapOccupancy(t *testing.T) {
	const failuresName = "unkey_heimdall_bpf_map_enumeration_errors_total"
	const entriesName = "unkey_heimdall_bpf_map_entries"
	lister := corelisters.NewPodLister(cache.NewIndexer(cache.MetaNamespaceKeyFunc, cache.Indexers{}))
	enabled := CollectorSetFrom([]string{"network"})

	t.Run("failure keeps last count and recovers", func(t *testing.T) {
		net := &occupancyNetwork{Reader: network.NewNoop(), entries: 7}
		c := New(Config{PodLister: lister, Network: net, Collectors: enabled})
		failures := metricValue(t, failuresName)

		require.NoError(t, c.collect(t.Context()))
		require.Equal(t, 7.0, metricValue(t, entriesName))
		require.Equal(t, failures, metricValue(t, failuresName))

		net.entries, net.err = 3, os.ErrPermission
		for i := 1; i <= 2; i++ {
			require.NoError(t, c.collect(t.Context()), "occupancy is diagnostic and must not fail the tick")
			require.Equal(t, 7.0, metricValue(t, entriesName), "a failed enumeration must not publish a partial count")
			require.Equal(t, failures+float64(i), metricValue(t, failuresName))
		}

		net.entries, net.err = 9, nil
		require.NoError(t, c.collect(t.Context()))
		require.Equal(t, 9.0, metricValue(t, entriesName))
		require.Equal(t, failures+2, metricValue(t, failuresName))
	})

	t.Run("disabled collector is not a failure", func(t *testing.T) {
		for name, supplied := range map[string]*occupancyNetwork{
			"nil reader":      nil,
			"supplied reader": {Reader: network.NewNoop(), err: os.ErrPermission},
		} {
			t.Run(name, func(t *testing.T) {
				cfg := Config{PodLister: lister, Collectors: CollectorSetFrom([]string{"disk"})}
				if supplied != nil {
					cfg.Network = supplied
				}
				c := New(cfg)
				failures := metricValue(t, failuresName)
				require.NoError(t, c.collect(t.Context()))
				require.Equal(t, failures, metricValue(t, failuresName))
				if supplied != nil {
					require.Zero(t, supplied.reconciled, "a disabled collector must not use the supplied reader")
				}
			})
		}
	})
}

type failingPodLister struct {
	corelisters.PodLister
}

func (failingPodLister) List(labels.Selector) ([]*corev1.Pod, error) {
	return nil, errors.New("pod cache unavailable")
}

func TestCollectPodListFailure(t *testing.T) {
	net := &occupancyNetwork{Reader: network.NewNoop(), entries: 5}
	c := New(Config{PodLister: failingPodLister{}, Network: net, Collectors: CollectorSetFrom([]string{"network"})})
	collectionErrors := metricValue(t, "unkey_heimdall_collection_total", "result", "error")

	c.collectWithMetrics(t.Context(), time.Hour)

	require.Equal(t, collectionErrors+1, metricValue(t, "unkey_heimdall_collection_total", "result", "error"))
	require.Zero(t, net.reconciled, "an unknown pod set must not evict attachments")
}

func TestRunRejectsNonpositiveInterval(t *testing.T) {
	for _, interval := range []time.Duration{0, -time.Second} {
		t.Run(interval.String(), func(t *testing.T) {
			c := New(Config{})
			require.ErrorContains(t, c.Run(t.Context(), interval), "collection interval must be positive")
		})
	}
}
