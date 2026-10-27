package collector

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/batch"
	"github.com/unkeyed/unkey/pkg/clickhouse/schema"
	"github.com/unkeyed/unkey/pkg/uid"
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

func TestRunRejectsNonpositiveInterval(t *testing.T) {
	for _, interval := range []time.Duration{0, -time.Second} {
		t.Run(interval.String(), func(t *testing.T) {
			c := New(Config{})
			require.ErrorContains(t, c.Run(t.Context(), interval), "collection interval must be positive")
		})
	}
}
