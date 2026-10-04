package privatenetwork

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/svc/krane/internal/cilium"
	"github.com/unkeyed/unkey/svc/krane/internal/testutil"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes/fake"
	clienttesting "k8s.io/client-go/testing"
)

func TestCleanupRequiresTwoIndependentSnapshotIDs(t *testing.T) {
	r, key, connections, policies, _ := cleanupFixture(t)

	require.NoError(t, r.cleanupGrants(t.Context(), "one", nil, connections, policies, true))
	require.Contains(t, connections, key)
	require.NoError(t, r.cleanupGrants(t.Context(), "one", nil, connections, policies, true))
	require.Contains(t, connections, key, "a repeated cached snapshot ID cannot confirm absence")
	require.NoError(t, r.cleanupGrants(t.Context(), "two", nil, connections, policies, true))
	require.NotContains(t, connections, key)
	require.NotContains(t, policies, key)
}

func TestCleanupAbsenceConfirmationResets(t *testing.T) {
	r, key, connections, policies, _ := cleanupFixture(t)
	require.NoError(t, r.cleanupGrants(t.Context(), "one", nil, connections, policies, true))
	require.NoError(t, r.cleanupGrants(t.Context(), "two", map[string]struct{}{key: {}}, connections, policies, true))
	require.Empty(t, r.absences, "returning to wanted resets confirmation")
	require.NoError(t, r.cleanupGrants(t.Context(), "three", nil, connections, policies, false))
	require.Empty(t, r.absences, "an invalid snapshot cannot preserve confirmation")
	require.NoError(t, r.cleanupGrants(t.Context(), "four", nil, connections, policies, true))
	require.Contains(t, connections, key)
}

func TestCleanupRemovesOrphanPolicy(t *testing.T) {
	r, key, _, policies, _ := cleanupFixture(t)
	require.NoError(t, r.cleanupGrants(t.Context(), "one", nil, nil, policies, true))
	require.NoError(t, r.cleanupGrants(t.Context(), "two", nil, nil, policies, true))
	require.NotContains(t, policies, key)
}

func TestConfirmedAbsenceCleanupPrecedesEndpointFailure(t *testing.T) {
	r, key, connections, _, _ := cleanupFixture(t)
	r.clusterKey = &ctrlv1.ClusterKey{}
	r.cluster = &testutil.MockClusterClient{StreamPrivateNetworkStateFunc: chunkStream(t,
		&ctrlv1.PrivateNetworkStateChunk{Version: "v1", SnapshotId: "one", Complete: true, Certified: true},
	)}
	failing := true
	r.client.(*fake.Clientset).PrependReactor("list", "endpointslices", func(clienttesting.Action) (bool, runtime.Object, error) {
		if failing {
			return true, nil, fmt.Errorf("endpoint API unavailable")
		}
		return false, nil, nil
	})
	require.ErrorContains(t, r.reconcile(t.Context()), "endpoint API unavailable")
	connection := connections[key]
	_, err := r.client.CoreV1().ConfigMaps(connection.Namespace).Get(t.Context(), connection.Name, metav1.GetOptions{})
	require.NoError(t, err, "one snapshot cannot confirm absence")
	r.cluster = &testutil.MockClusterClient{StreamPrivateNetworkStateFunc: chunkStream(t,
		&ctrlv1.PrivateNetworkStateChunk{Version: "v1", SnapshotId: "two", Complete: true, Certified: true},
	)}
	require.ErrorContains(t, r.reconcile(t.Context()), "endpoint API unavailable")
	_, err = r.client.CoreV1().ConfigMaps(connection.Namespace).Get(t.Context(), connection.Name, metav1.GetOptions{})
	require.True(t, apierrors.IsNotFound(err))
	_, err = r.dynamic.Resource(cilium.NetworkPolicyResource).Namespace(connection.Namespace).Get(t.Context(), connection.Name, metav1.GetOptions{})
	require.True(t, apierrors.IsNotFound(err))
	failing = false
	require.NoError(t, r.reconcile(t.Context()))
	_, err = r.client.CoreV1().ConfigMaps(connection.Namespace).Get(t.Context(), connection.Name, metav1.GetOptions{})
	require.True(t, apierrors.IsNotFound(err))
}

func TestCleanupDeleteErrorRetainsDNSAndRetries(t *testing.T) {
	r, key, connections, policies, _ := cleanupFixture(t)
	failing := true
	r.dynamic.(interface {
		PrependReactor(string, string, clienttesting.ReactionFunc)
	}).PrependReactor("delete", cilium.NetworkPolicyResource.Resource, func(clienttesting.Action) (bool, runtime.Object, error) {
		if failing {
			return true, nil, apierrors.NewForbidden(schema.GroupResource{Resource: cilium.NetworkPolicyResource.Resource}, "", fmt.Errorf("denied"))
		}
		return false, nil, nil
	})
	require.NoError(t, r.cleanupGrants(t.Context(), "one", nil, connections, policies, true))
	require.Error(t, r.cleanupGrants(t.Context(), "two", nil, connections, policies, true))
	require.Contains(t, connections, key)
	require.Contains(t, policies, key)
	failing = false
	require.NoError(t, r.cleanupGrants(t.Context(), "two", nil, connections, policies, true))
	require.NotContains(t, connections, key)
}

func TestUncertifiedSnapshotsNeverConfirmAbsence(t *testing.T) {
	r, key, connections, _, spec := cleanupFixture(t)
	r.clusterKey = &ctrlv1.ClusterKey{}
	connection := connections[key]
	exists := func() bool {
		t.Helper()
		_, err := r.client.CoreV1().ConfigMaps(connection.Namespace).Get(t.Context(), connection.Name, metav1.GetOptions{})
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
		require.NoError(t, r.reconcile(t.Context()))
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

func cleanupFixture(t *testing.T) (*Reconciler, string, map[string]*corev1.ConfigMap, map[string]*unstructured.Unstructured, *ctrlv1.PrivateNetworkConnection) {
	t.Helper()
	spec := newTestConnection()
	r := &Reconciler{clock: clock.NewTestClock(), client: fake.NewClientset(), dynamic: testDynamicClient()}
	service, err := r.ensureService(t.Context(), spec, "target-service", nil)
	require.NoError(t, err)
	connection, err := r.ensureConnection(t.Context(), spec, connectionResourceName(spec), service, nil, nil)
	require.NoError(t, err)
	require.NoError(t, ensureTestPolicy(t, r, t.Context(), spec, connection.Name, connection))
	policy, err := r.dynamic.Resource(cilium.NetworkPolicyResource).Namespace(connection.Namespace).Get(t.Context(), connection.Name, metav1.GetOptions{})
	require.NoError(t, err)
	key := connection.Namespace + "/" + connection.Name
	return r, key, map[string]*corev1.ConfigMap{key: connection}, map[string]*unstructured.Unstructured{key: policy}, spec
}
