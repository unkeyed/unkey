//go:build linux

package collector

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/batch"
	"github.com/unkeyed/unkey/pkg/clickhouse/schema"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/heimdall/internal/network"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
)

type lifecycleNetwork struct {
	network.Reader
	bytes int64
	err   error
}

func (n *lifecycleNetwork) Attach(types.UID) error { return nil }
func (n *lifecycleNetwork) Read(types.UID) (network.Counters, error) {
	return network.Counters{EgressPublicBytes: n.bytes}, n.err
}
func (n *lifecycleNetwork) Detach(types.UID) { n.bytes = 0 }

func lifecyclePod() *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "primary", ResourceVersion: "1", UID: types.UID(uid.New(uid.InstancePrefix)), Labels: map[string]string{
			LabelManagedBy: "krane", LabelComponent: "deployment", LabelWorkspace: uid.New(uid.WorkspacePrefix),
		}},
		Spec: corev1.PodSpec{NodeName: "node", Containers: []corev1.Container{{Name: "primary"}}},
		Status: corev1.PodStatus{Phase: corev1.PodRunning, QOSClass: corev1.PodQOSBurstable, PodIP: "10.0.0.1", ContainerStatuses: []corev1.ContainerStatus{{
			Name: "primary", RestartCount: 2, ContainerID: "containerd://" + uid.New(uid.InstancePrefix),
			State: corev1.ContainerState{Running: &corev1.ContainerStateRunning{}},
		}}},
	}
}

func lifecycleBuffer(t *testing.T) (*batch.BatchProcessor[schema.InstanceCheckpoint], *[]schema.InstanceCheckpoint) {
	t.Helper()
	rows := []schema.InstanceCheckpoint{}
	bp := batch.New(batch.Config[schema.InstanceCheckpoint]{Name: "lifecycle-test", BatchSize: 1, BufferSize: 32, FlushInterval: time.Hour,
		Flush: func(_ context.Context, batch []schema.InstanceCheckpoint) { rows = append(rows, batch...) },
	})
	t.Cleanup(bp.Close)
	return bp, &rows
}

func TestLifecycleNetworkMeasurement(t *testing.T) {
	for _, tc := range []struct {
		name     string
		bytes    int64
		err      error
		measured bool
	}{
		{name: "existing zero", measured: true},
		{name: "retained counter", bytes: 791, measured: true},
		{name: "unattached", bytes: 791, err: network.ErrNotAttached},
		{name: "unobserved", bytes: 791, err: network.ErrCounterUnobserved},
		{name: "lost", bytes: 791, err: network.ErrCounterLost},
		{name: "read failure", bytes: 791, err: os.ErrPermission},
	} {
		t.Run(tc.name, func(t *testing.T) {
			pod := lifecyclePod()
			index := cache.NewIndexer(cache.MetaNamespaceKeyFunc, cache.Indexers{})
			require.NoError(t, index.Add(pod))
			bp, rows := lifecycleBuffer(t)
			net := &lifecycleNetwork{bytes: tc.bytes, err: tc.err}
			c := New(Config{CH: bp, PodLister: corelisters.NewPodLister(index), NodeName: "node", Network: net, Collectors: CollectorSetFrom([]string{"network"})})
			id := pod.Status.ContainerStatuses[0].ContainerID[len("containerd://"):]
			c.OnStart(t.Context(), string(pod.UID), id)
			c.OnExit(t.Context(), string(pod.UID), id)
			bp.Close()
			require.Len(t, *rows, 2)
			for _, row := range *rows {
				var attrs schema.InstanceCheckpointAttributes
				require.NoError(t, json.Unmarshal([]byte(row.Attributes), &attrs))
				require.Equal(t, tc.measured, attrs.NetworkAttached)
				if tc.measured {
					require.Equal(t, tc.bytes, row.NetworkEgressPublicBytes)
				} else {
					require.Zero(t, row.NetworkEgressPublicBytes)
				}
			}
		})
	}
}

type countingNetwork struct {
	lifecycleNetwork
	calls int
}

func (n *countingNetwork) Attach(uid types.UID) error {
	n.calls++
	return n.lifecycleNetwork.Attach(uid)
}

func (n *countingNetwork) Read(uid types.UID) (network.Counters, error) {
	n.calls++
	return n.lifecycleNetwork.Read(uid)
}

func TestDisabledNetworkIsUnmeasured(t *testing.T) {
	for _, supplied := range []bool{false, true} {
		t.Run(fmt.Sprintf("supplied reader=%t", supplied), func(t *testing.T) {
			pod := lifecyclePod()
			index := cache.NewIndexer(cache.MetaNamespaceKeyFunc, cache.Indexers{})
			require.NoError(t, index.Add(pod))
			bp, rows := lifecycleBuffer(t)
			net := &countingNetwork{lifecycleNetwork: lifecycleNetwork{bytes: 4096}}
			cfg := Config{CH: bp, PodLister: corelisters.NewPodLister(index), NodeName: "node", Collectors: CollectorSetFrom([]string{"disk"})}
			if supplied {
				cfg.Network = net
			}
			c := New(cfg)
			readErrors := metricValue(t, "unkey_heimdall_network_read_errors_total")
			attachFailures := metricValue(t, "unkey_heimdall_network_attach_failures_total")
			id := pod.Status.ContainerStatuses[0].ContainerID[len("containerd://"):]

			c.OnStart(t.Context(), string(pod.UID), id)
			require.NoError(t, c.collect(t.Context()))
			c.OnExit(t.Context(), string(pod.UID), id)
			bp.Close()

			require.Len(t, *rows, 3)
			for _, row := range *rows {
				var attrs schema.InstanceCheckpointAttributes
				require.NoError(t, json.Unmarshal([]byte(row.Attributes), &attrs))
				require.False(t, attrs.NetworkAttached, row.EventKind)
				require.Equal(t, []string{"disk"}, attrs.Collectors)
				require.Zero(t, row.NetworkEgressPublicBytes)
			}
			require.Zero(t, net.calls, "a disabled collector must not attach or read")
			require.Equal(t, readErrors, metricValue(t, "unkey_heimdall_network_read_errors_total"))
			require.Equal(t, attachFailures, metricValue(t, "unkey_heimdall_network_attach_failures_total"))
		})
	}
}

func TestLifecycleRetainsPodLedger(t *testing.T) {
	for _, unavailable := range []string{"restart unknown", "pod absent", "cgroup absent"} {
		t.Run(unavailable, func(t *testing.T) {
			pod := lifecyclePod()
			id := pod.Status.ContainerStatuses[0].ContainerID[len("containerd://"):]
			index := cache.NewIndexer(cache.MetaNamespaceKeyFunc, cache.Indexers{})
			if unavailable == "restart unknown" {
				pod.Status.ContainerStatuses = nil
			}
			if unavailable != "pod absent" {
				require.NoError(t, index.Add(pod))
			}
			bp, rows := lifecycleBuffer(t)
			net := &lifecycleNetwork{bytes: 12345}
			c := New(Config{CH: bp, PodLister: corelisters.NewPodLister(index), NodeName: "node", CgroupRoot: t.TempDir(), Network: net, Collectors: CollectorSetFrom([]string{"cpu", "network"})})
			c.OnExit(t.Context(), string(pod.UID), id)
			bp.Close()
			require.Empty(t, *rows)
			counters, err := net.Read(pod.UID)
			require.NoError(t, err)
			require.EqualValues(t, 12345, counters.EgressPublicBytes)
		})
	}
}

func TestLifecycleRestartUsesRetainedBaseline(t *testing.T) {
	pod := lifecyclePod()
	index := cache.NewIndexer(cache.MetaNamespaceKeyFunc, cache.Indexers{})
	require.NoError(t, index.Add(pod))
	bp, rows := lifecycleBuffer(t)
	net := &lifecycleNetwork{bytes: 1000}
	root := t.TempDir()
	buildFakePodCgroup(t, root, pod.UID, pod.Status.QOSClass, 9000, 8000, 1000)
	c := New(Config{CH: bp, PodLister: corelisters.NewPodLister(index), CgroupRoot: root, CgroupDriver: CgroupDriverCgroupfs, NodeName: "node", Network: net, Collectors: CollectorSetFrom([]string{"cpu", "memory", "network"})})
	id := pod.Status.ContainerStatuses[0].ContainerID[len("containerd://"):]
	c.OnStart(t.Context(), string(pod.UID), id)
	net.bytes = 1400
	c.OnExit(t.Context(), string(pod.UID), id)
	stopped := pod.DeepCopy()
	stopped.Status.Phase = corev1.PodSucceeded
	stopped.Status.ContainerStatuses[0].State = corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{}}
	require.NoError(t, index.Update(stopped))
	c.HandlePodUpdate(pod, stopped)
	restarted := pod.DeepCopy()
	restarted.Status.ContainerStatuses[0].RestartCount++
	restarted.Status.ContainerStatuses[0].ContainerID = "containerd://" + uid.New(uid.InstancePrefix)
	require.NoError(t, index.Update(restarted))
	c.HandlePodUpdate(stopped, restarted)
	c.OnExit(t.Context(), string(pod.UID), id)
	c.HandlePodUpdate(pod, stopped)
	net.bytes = 1650
	c.OnExit(t.Context(), string(pod.UID), restarted.Status.ContainerStatuses[0].ContainerID[len("containerd://"):])
	bp.Close()
	require.Len(t, *rows, 5)
	for i, kind := range []string{"start", "stop", "stop", "start", "stop"} {
		require.Equal(t, kind, (*rows)[i].EventKind)
		require.EqualValues(t, 9000, (*rows)[i].CPUUsageUsec)
		require.EqualValues(t, 7000, (*rows)[i].MemoryBytes)
	}
	require.Equal(t, (*rows)[0].ContainerUID, (*rows)[2].ContainerUID)
	require.NotEqual(t, (*rows)[2].ContainerUID, (*rows)[3].ContainerUID)
	require.EqualValues(t, 1400, (*rows)[2].NetworkEgressPublicBytes)
	require.EqualValues(t, 1400, (*rows)[3].NetworkEgressPublicBytes)
	require.EqualValues(t, 400, (*rows)[1].NetworkEgressPublicBytes-(*rows)[0].NetworkEgressPublicBytes)
	require.EqualValues(t, 250, (*rows)[4].NetworkEgressPublicBytes-(*rows)[3].NetworkEgressPublicBytes)
}

func TestLifecycleDelayedInformerStop(t *testing.T) {
	pod := lifecyclePod()
	stream := watch.NewRaceFreeFake()
	lw := &cache.ListWatch{
		ListFunc: func(metav1.ListOptions) (runtime.Object, error) {
			return &corev1.PodList{ListMeta: metav1.ListMeta{ResourceVersion: "1"}, Items: []corev1.Pod{*pod}}, nil
		},
		WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
			if options.SendInitialEvents != nil && *options.SendInitialEvents {
				stream.Add(pod)
				stream.Action(watch.Bookmark, &corev1.Pod{ObjectMeta: metav1.ObjectMeta{ResourceVersion: "1", Annotations: map[string]string{metav1.InitialEventsAnnotationKey: "true"}}})
			}
			return stream, nil
		},
	}
	informer := cache.NewSharedIndexInformer(lw, &corev1.Pod{}, 0, cache.Indexers{})
	bp, rows := lifecycleBuffer(t)
	net := &lifecycleNetwork{bytes: 3200}
	c := New(Config{CH: bp, PodLister: corelisters.NewPodLister(informer.GetIndexer()), NodeName: "node", Network: net, Collectors: CollectorSetFrom([]string{"network"})})
	entered := make(chan struct{})
	release := make(chan struct{})
	processed := make(chan struct{})
	var releaseOnce sync.Once
	defer releaseOnce.Do(func() { close(release) })
	_, err := informer.AddEventHandler(cache.ResourceEventHandlerFuncs{UpdateFunc: func(oldObj, newObj any) {
		oldPod := oldObj.(*corev1.Pod)
		newPod := newObj.(*corev1.Pod)
		if newPod.ResourceVersion == "2" {
			close(entered)
			<-release
		}
		c.HandlePodUpdate(oldPod, newPod)
		if newPod.ResourceVersion == "3" {
			close(processed)
		}
	}})
	require.NoError(t, err)
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	done := make(chan struct{})
	go func() { defer close(done); informer.Run(ctx.Done()) }()
	t.Cleanup(func() { cancel(); <-done })
	require.True(t, cache.WaitForCacheSync(ctx.Done(), informer.HasSynced))
	stopped := pod.DeepCopy()
	stopped.ResourceVersion = "2"
	stopped.Status.ContainerStatuses[0].State = corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{}}
	stream.Modify(stopped)
	select {
	case <-entered:
	case <-ctx.Done():
		t.Fatal("stop callback not entered")
	}
	restarted := pod.DeepCopy()
	restarted.ResourceVersion = "3"
	restarted.Status.ContainerStatuses[0].RestartCount++
	restarted.Status.ContainerStatuses[0].ContainerID = "containerd://" + uid.New(uid.InstancePrefix)
	stream.Modify(restarted)
	require.Eventually(t, func() bool {
		obj, exists, err := informer.GetStore().Get(restarted)
		return err == nil && exists && obj.(*corev1.Pod).ResourceVersion == "3"
	}, time.Second, time.Millisecond)
	releaseOnce.Do(func() { close(release) })
	select {
	case <-processed:
	case <-ctx.Done():
		t.Fatal("restart callback not processed")
	}
	bp.Close()
	require.Len(t, *rows, 1)
	require.Equal(t, "start", (*rows)[0].EventKind)
	require.EqualValues(t, 3, (*rows)[0].RestartCount)
	require.EqualValues(t, 3200, (*rows)[0].NetworkEgressPublicBytes)
}

func TestLifecycleRunningIncarnationFallback(t *testing.T) {
	for _, change := range []string{"id and count", "id only", "count only"} {
		t.Run(change, func(t *testing.T) {
			pod := lifecyclePod()
			stream := watch.NewRaceFreeFake()
			lw := &cache.ListWatch{
				ListFunc: func(metav1.ListOptions) (runtime.Object, error) {
					return &corev1.PodList{ListMeta: metav1.ListMeta{ResourceVersion: "1"}, Items: []corev1.Pod{*pod}}, nil
				},
				WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
					if options.SendInitialEvents != nil && *options.SendInitialEvents {
						stream.Add(pod)
						stream.Action(watch.Bookmark, &corev1.Pod{ObjectMeta: metav1.ObjectMeta{ResourceVersion: "1", Annotations: map[string]string{metav1.InitialEventsAnnotationKey: "true"}}})
					}
					return stream, nil
				},
			}
			informer := cache.NewSharedIndexInformer(lw, &corev1.Pod{}, 0, cache.Indexers{})
			bp, rows := lifecycleBuffer(t)
			net := &lifecycleNetwork{bytes: 3200}
			c := New(Config{CH: bp, PodLister: corelisters.NewPodLister(informer.GetIndexer()), NodeName: "node", Network: net, Collectors: CollectorSetFrom([]string{"network"})})
			processed := make(chan string, 4)
			_, err := informer.AddEventHandler(cache.ResourceEventHandlerFuncs{UpdateFunc: func(oldObj, newObj any) {
				newPod := newObj.(*corev1.Pod)
				c.HandlePodUpdate(oldObj.(*corev1.Pod), newPod)
				processed <- newPod.ResourceVersion
			}})
			require.NoError(t, err)
			ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
			done := make(chan struct{})
			go func() { defer close(done); informer.Run(ctx.Done()) }()
			t.Cleanup(func() { cancel(); <-done })
			require.True(t, cache.WaitForCacheSync(ctx.Done(), informer.HasSynced))
			update := func(p *corev1.Pod) {
				t.Helper()
				stream.Modify(p)
				select {
				case version := <-processed:
					require.Equal(t, p.ResourceVersion, version)
				case <-ctx.Done():
					t.Fatal("running update not processed")
				}
			}
			restarted := pod.DeepCopy()
			restarted.ResourceVersion = "2"
			if change != "id only" {
				restarted.Status.ContainerStatuses[0].RestartCount++
			}
			if change != "count only" {
				id := uid.New(uid.InstancePrefix)
				restarted.Status.ContainerStatuses[0].ContainerID = "containerd://" + id
				c.OnStart(t.Context(), string(pod.UID), id)
			}
			net.bytes = 3450
			update(restarted)
			unchanged := restarted.DeepCopy()
			unchanged.ResourceVersion = "3"
			update(unchanged)
			successor := restarted.DeepCopy()
			successor.ResourceVersion = "4"
			successor.Status.ContainerStatuses[0].RestartCount++
			successor.Status.ContainerStatuses[0].ContainerID = "containerd://" + uid.New(uid.InstancePrefix)
			net.bytes = 3700
			update(successor)
			c.HandlePodUpdate(pod, restarted)
			bp.Close()
			require.Len(t, *rows, 2)
			for i, expected := range []*corev1.Pod{restarted, successor} {
				require.Equal(t, "start", (*rows)[i].EventKind)
				require.EqualValues(t, expected.Status.ContainerStatuses[0].RestartCount, (*rows)[i].RestartCount)
			}
			require.EqualValues(t, 3450, (*rows)[0].NetworkEgressPublicBytes)
			require.EqualValues(t, 3700, (*rows)[1].NetworkEgressPublicBytes)
		})
	}
}
