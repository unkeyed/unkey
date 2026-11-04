package privatenetwork

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/svc/krane/internal/cilium"
	"github.com/unkeyed/unkey/svc/krane/internal/testutil"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	fakedynamic "k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/kubernetes/fake"
	clienttesting "k8s.io/client-go/testing"
)

func TestCleanupRequiresTwoIndependentSnapshotIDs(t *testing.T) {
	r, key, p, _ := cleanupFixture(t)

	require.NoError(t, cleanupWith(t, r, p, "one", nil, true))
	require.Contains(t, p.connections, key)
	require.NoError(t, cleanupWith(t, r, p, "one", nil, true))
	require.Contains(t, p.connections, key, "a repeated cached snapshot ID cannot confirm absence")
	require.NoError(t, cleanupWith(t, r, p, "two", nil, true))
	require.NotContains(t, p.connections, key)
	require.NotContains(t, p.policies, key)
}

func TestCleanupAbsenceConfirmationResets(t *testing.T) {
	r, key, p, spec := cleanupFixture(t)
	require.NoError(t, cleanupWith(t, r, p, "one", nil, true))
	require.NoError(t, cleanupWith(t, r, p, "two", []*ctrlv1.PrivateNetworkConnection{spec}, true))
	require.Empty(t, r.absences, "returning to wanted resets confirmation")
	require.NoError(t, cleanupWith(t, r, p, "three", nil, false))
	require.Empty(t, r.absences, "an invalid snapshot cannot preserve confirmation")
	require.NoError(t, cleanupWith(t, r, p, "four", nil, true))
	require.Contains(t, p.connections, key)
}

func TestCleanupRemovesOrphanPolicy(t *testing.T) {
	r, key, p, _ := cleanupFixture(t)
	p.connections = map[string][]*corev1.Service{}
	require.NoError(t, cleanupWith(t, r, p, "one", nil, true))
	require.NoError(t, cleanupWith(t, r, p, "two", nil, true))
	require.NotContains(t, p.policies, key)
}

func TestConfirmedAbsenceCleanupPrecedesEndpointFailure(t *testing.T) {
	r, key, p, spec := cleanupFixture(t)
	r.clusterKey = &ctrlv1.ClusterKey{Region: testRegion}
	r.cluster = &testutil.MockClusterClient{StreamPrivateNetworkStateFunc: chunkStream(t,
		&ctrlv1.PrivateNetworkStateChunk{Version: "v1", SnapshotId: "one", Complete: true, Certified: true},
	)}
	failing := true
	r.client.(*fake.Clientset).PrependReactor("create", "endpointslices", func(clienttesting.Action) (bool, runtime.Object, error) {
		if failing {
			return true, nil, fmt.Errorf("endpoint API unavailable")
		}
		return false, nil, nil
	})
	require.ErrorContains(t, converge(t, r), "endpoint API unavailable")
	connection := p.connections[key][0]
	_, err := r.client.CoreV1().Services(connection.Namespace).Get(t.Context(), connection.Name, metav1.GetOptions{})
	require.NoError(t, err, "one snapshot cannot confirm absence")
	r.cluster = &testutil.MockClusterClient{StreamPrivateNetworkStateFunc: chunkStream(t,
		&ctrlv1.PrivateNetworkStateChunk{Version: "v1", SnapshotId: "two", Complete: true, Certified: true},
	)}
	require.ErrorContains(t, converge(t, r), "endpoint API unavailable")
	_, err = r.client.CoreV1().Services(connection.Namespace).Get(t.Context(), connection.Name, metav1.GetOptions{})
	require.True(t, apierrors.IsNotFound(err))
	_, err = r.dynamic.Resource(cilium.NetworkPolicyResource).Namespace(connection.Namespace).Get(t.Context(), policyName(spec), metav1.GetOptions{})
	require.True(t, apierrors.IsNotFound(err))
	failing = false
	require.NoError(t, converge(t, r))
	_, err = r.client.CoreV1().Services(connection.Namespace).Get(t.Context(), connection.Name, metav1.GetOptions{})
	require.True(t, apierrors.IsNotFound(err))
}

func TestCleanupDeleteErrorRetainsDNSAndRetries(t *testing.T) {
	r, key, p, _ := cleanupFixture(t)
	failing := true
	r.dynamic.(interface {
		PrependReactor(string, string, clienttesting.ReactionFunc)
	}).PrependReactor("delete", cilium.NetworkPolicyResource.Resource, func(clienttesting.Action) (bool, runtime.Object, error) {
		if failing {
			return true, nil, apierrors.NewForbidden(schema.GroupResource{Resource: cilium.NetworkPolicyResource.Resource}, "", fmt.Errorf("denied"))
		}
		return false, nil, nil
	})
	require.NoError(t, cleanupWith(t, r, p, "one", nil, true))
	require.Error(t, cleanupWith(t, r, p, "two", nil, true))
	require.Contains(t, p.connections, key)
	require.Contains(t, p.policies, key)
	failing = false
	require.NoError(t, cleanupWith(t, r, p, "two", nil, true))
	require.NotContains(t, p.connections, key)
}

func TestUncertifiedSnapshotsNeverConfirmAbsence(t *testing.T) {
	r, key, p, spec := cleanupFixture(t)
	r.clusterKey = &ctrlv1.ClusterKey{Region: testRegion}
	connection := p.connections[key][0]
	exists := func() bool {
		t.Helper()
		_, err := r.client.CoreV1().Services(connection.Namespace).Get(t.Context(), connection.Name, metav1.GetOptions{})
		if apierrors.IsNotFound(err) {
			return false
		}
		require.NoError(t, err)
		return true
	}
	serve := func(id string, certified bool, present ...*ctrlv1.PrivateNetworkConnection) {
		t.Helper()
		version := fmt.Sprintf("v-%s-%d", id, len(present))
		chunks := []*ctrlv1.PrivateNetworkStateChunk{}
		if len(present) > 0 {
			chunks = append(chunks, &ctrlv1.PrivateNetworkStateChunk{Version: version, SnapshotId: id, Connections: present})
		}
		chunks = append(chunks, &ctrlv1.PrivateNetworkStateChunk{Version: version, SnapshotId: id, Complete: true, Total: uint64(len(present)), Certified: certified})
		r.cluster = &testutil.MockClusterClient{StreamPrivateNetworkStateFunc: chunkStream(t, chunks...)}
		require.NoError(t, converge(t, r))
	}

	for _, id := range []string{"one", "two", "three"} {
		serve(id, false)
	}
	require.True(t, exists(), "uncertified snapshots with different IDs")

	serve("four", true)
	serve("five", false, spec)
	serve("six", true)
	require.True(t, exists(), "a grant present in an uncertified snapshot needs two new certified absences")

	serve("seven", false)
	require.True(t, exists())
	serve("eight", true)
	require.False(t, exists(), "a certified absence survives the uncertified snapshots between")
}

func cleanupFixture(t *testing.T) (*Reconciler, string, *discoveryPass, *ctrlv1.PrivateNetworkConnection) {
	t.Helper()
	spec := newTestConnection()
	r := newTestReconciler(t, Config{})
	service, err := r.ensureService(t.Context(), spec, nil)
	require.NoError(t, err)
	require.NoError(t, ensureTestPolicy(t, r, t.Context(), spec, policyName(spec), nil))
	connection, err := r.ensureConnection(t.Context(), spec, aliasName(spec), "", service, nil, "")
	require.NoError(t, err)
	syncCaches(t, r)
	p, err := r.load()
	require.NoError(t, err)
	key := grantKey(connection)
	require.Contains(t, p.policies, key)
	require.Len(t, p.connections[key], 1)
	return r, key, p, spec
}

func cleanupWith(t *testing.T, r *Reconciler, p *discoveryPass, snapshotID string, wanted []*ctrlv1.PrivateNetworkConnection, confirm bool) error {
	t.Helper()
	return r.cleanupGrants(t.Context(), p, &ctrlv1.PrivateNetworkStateChunk{SnapshotId: snapshotID, Certified: true}, wanted, confirm)
}

func TestCleanupDNSDeleteErrorRevokesGrantFirstAndRetries(t *testing.T) {
	r, key, p, spec := cleanupFixture(t)
	failing := true
	r.client.(*fake.Clientset).PrependReactor("delete", "services", func(clienttesting.Action) (bool, runtime.Object, error) {
		if failing {
			return true, nil, apierrors.NewServiceUnavailable("injected")
		}
		return false, nil, nil
	})
	require.NoError(t, cleanupWith(t, r, p, "one", nil, true))
	require.Error(t, cleanupWith(t, r, p, "two", nil, true))
	require.NotContains(t, p.policies, key, "the grant is revoked before DNS")
	require.Empty(t, effectiveFlows(t, r.dynamic.(*fakedynamic.FakeDynamicClient), spec.GetK8SNamespace()))
	require.Contains(t, p.connections, key)
	require.Equal(t, spec.TargetDeploymentId, resolvedDeployment(t, r.client, spec), "failed DNS deletion leaves the alias for retry")

	failing = false
	require.NoError(t, cleanupWith(t, r, p, "three", nil, true))
	require.NotContains(t, p.connections, key)
	require.Nil(t, aliasService(t, r.client, spec))
}

func TestNewLeadershipTermDoesNotInheritAbsences(t *testing.T) {
	spec := newTestConnection()
	client := fake.NewClientset(endpointPod(spec, "a", "10.72.0.11"))
	snapshot := []*ctrlv1.PrivateNetworkConnection{spec}
	var mu sync.Mutex
	var termStarted bool
	var termCalls int
	secondCall := make(chan struct{})
	control := &testutil.MockClusterClient{StreamPrivateNetworkStateFunc: snapshotFunc(t, func(ctx context.Context) ([]*ctrlv1.PrivateNetworkConnection, error) {
		mu.Lock()
		if !termStarted {
			defer mu.Unlock()
			return snapshot, nil
		}
		termCalls++
		calls := termCalls
		mu.Unlock()
		if calls == 1 {
			return nil, nil
		}
		if calls == 2 {
			close(secondCall)
		}
		<-ctx.Done()
		return nil, ctx.Err()
	})}
	clk := clock.NewTestClock()
	r := newTestReconciler(t, Config{Client: client, Cluster: control, Clock: clk})
	require.NoError(t, converge(t, r))
	mu.Lock()
	snapshot = nil
	mu.Unlock()
	require.NoError(t, converge(t, r))
	require.NotNil(t, aliasService(t, client, spec), "the previous term saw one absence")

	mu.Lock()
	termStarted = true
	mu.Unlock()
	startLeading()
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan struct{})
	go func() {
		defer close(done)
		r.Run(ctx)
	}()
	require.Eventually(t, func() bool {
		clk.Tick(intervalPoll)
		select {
		case <-secondCall:
			return true
		default:
			return false
		}
	}, 10*time.Second, 10*time.Millisecond, "the new term completes its first discovery pass")
	cancel()
	<-done

	require.NotNil(t, aliasService(t, client, spec), "the first snapshot of a new term cannot confirm an absence")
	_, err := r.dynamic.Resource(cilium.NetworkPolicyResource).Namespace(spec.GetK8SNamespace()).Get(t.Context(), policyName(spec), metav1.GetOptions{})
	require.NoError(t, err)
}
