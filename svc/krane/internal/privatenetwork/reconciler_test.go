package privatenetwork

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/deploy/privatenetwork"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/krane/internal/cilium"
	"github.com/unkeyed/unkey/svc/krane/internal/testutil"
	"github.com/unkeyed/unkey/svc/krane/pkg/labels"
	"google.golang.org/protobuf/proto"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	fakedynamic "k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/kubernetes/fake"
	clienttesting "k8s.io/client-go/testing"
)

func testDynamicClient(objects ...runtime.Object) *fakedynamic.FakeDynamicClient {
	scheme := runtime.NewScheme()
	return fakedynamic.NewSimpleDynamicClientWithCustomListKinds(scheme, map[schema.GroupVersionResource]string{
		cilium.NetworkPolicyResource: "CiliumNetworkPolicyList",
	}, objects...)
}

func TestSteadyStatePolicyAPICostAt5000Connections(t *testing.T) {
	const count = 5000
	connections := make([]*ctrlv1.PrivateNetworkConnection, count)
	target := newTestConnection()
	for i := range connections {
		connections[i] = proto.Clone(target).(*ctrlv1.PrivateNetworkConnection)
		connections[i].ConnectionId = uid.New(uid.ConnectionPrefix)
		connections[i].CallerDeploymentId = uid.New(uid.DeploymentPrefix)
	}
	// NewClientset's field-managed tracker builds a REST mapper on every create,
	// which pushes the 5000-entry setup pass past the reconcile deadline under
	// -race. The reconciler uses no server-side apply, so the plain tracker fits.
	client, policies := fake.NewSimpleClientset(), testDynamicClient()
	control := &testutil.MockClusterClient{StreamPrivateNetworkStateFunc: chunkStream(t,
		&ctrlv1.PrivateNetworkStateChunk{Version: "v1", SnapshotId: "one", Connections: connections},
		&ctrlv1.PrivateNetworkStateChunk{Version: "v1", SnapshotId: "one", Complete: true, Total: count},
	)}
	r := &Reconciler{clock: clock.NewTestClock(), client: client, dynamic: policies, cluster: control, clusterKey: &ctrlv1.ClusterKey{}}
	require.NoError(t, r.reconcile(t.Context()))
	policies.ClearActions()
	client.ClearActions()
	control.StreamPrivateNetworkStateFunc = chunkStream(t, &ctrlv1.PrivateNetworkStateChunk{Version: "v1", SnapshotId: "one", Complete: true, Unchanged: true, Total: count})
	started := time.Now()
	require.NoError(t, r.reconcile(t.Context()))
	t.Logf("fake-client steady-state pass: %d connections in %s", count, time.Since(started))
	actions := policies.Actions()
	require.Len(t, actions, 1, "unchanged policies need one list and no per-entry GET or write")
	require.True(t, actions[0].Matches("list", "ciliumnetworkpolicies"))
	for _, action := range client.Actions() {
		require.Equal(t, "list", action.GetVerb(), "unchanged discovery objects need no writes or per-entry GETs")
	}
}

func TestReconcileRPCErrorPreservesPublishedObjects(t *testing.T) {
	ctx := t.Context()
	client := fake.NewClientset()
	control := &testutil.MockClusterClient{}
	connectionSpec := newTestConnection()
	control.StreamPrivateNetworkStateFunc = snapshotFunc(t, func(context.Context) ([]*ctrlv1.PrivateNetworkConnection, error) {
		return []*ctrlv1.PrivateNetworkConnection{connectionSpec}, nil
	})
	r := &Reconciler{clock: clock.NewTestClock(), client: client, dynamic: testDynamicClient(), cluster: control, clusterKey: &ctrlv1.ClusterKey{}}
	require.NoError(t, r.reconcile(ctx))

	control.StreamPrivateNetworkStateFunc = snapshotFunc(t, func(context.Context) ([]*ctrlv1.PrivateNetworkConnection, error) {
		return nil, fmt.Errorf("unavailable")
	})
	require.Error(t, r.reconcile(ctx))
	connections, err := client.CoreV1().ConfigMaps("customer-1").List(ctx, metav1.ListOptions{})
	require.NoError(t, err)
	require.Len(t, connections.Items, 1)
	services, err := client.CoreV1().Services("customer-1").List(ctx, metav1.ListOptions{})
	require.NoError(t, err)
	require.Len(t, services.Items, 1)
}

func TestReconcilePolicyWriteFailureDoesNotPublishNewDNSAndRetryConverges(t *testing.T) {
	for _, verb := range []string{"create", "update"} {
		t.Run(verb, func(t *testing.T) {
			ctx := t.Context()
			initial := newTestConnection()
			desired := proto.Clone(initial).(*ctrlv1.PrivateNetworkConnection)
			desired.TargetDeploymentId = uid.New(uid.DeploymentPrefix)
			client := fake.NewClientset(endpointPod(initial, "a", "10.72.0.11"), endpointPod(desired, "b", "10.72.0.22"))
			dynamic := testDynamicClient()
			control := &testutil.MockClusterClient{}
			current := desired
			if verb == "update" {
				current = initial
			}
			control.StreamPrivateNetworkStateFunc = snapshotFunc(t, func(context.Context) ([]*ctrlv1.PrivateNetworkConnection, error) {
				return []*ctrlv1.PrivateNetworkConnection{current}, nil
			})
			r := &Reconciler{clock: clock.NewTestClock(), client: client, dynamic: dynamic, cluster: control, clusterKey: &ctrlv1.ClusterKey{}}
			if verb == "update" {
				require.NoError(t, r.reconcile(ctx))
				current = desired
			}

			failing := true
			dynamic.PrependReactor(verb, cilium.NetworkPolicyResource.Resource, func(clienttesting.Action) (bool, runtime.Object, error) {
				if !failing {
					return false, nil, nil
				}
				return true, nil, apierrors.NewForbidden(schema.GroupResource{Resource: cilium.NetworkPolicyResource.Resource}, policyName(desired), fmt.Errorf("injected policy failure"))
			})
			require.True(t, apierrors.IsForbidden(r.reconcile(ctx)))
			connection, err := client.CoreV1().ConfigMaps(desired.GetK8SNamespace()).Get(ctx, connectionResourceName(desired), metav1.GetOptions{})
			if verb == "create" {
				require.True(t, apierrors.IsNotFound(err), "DNS ConfigMap was published without its policy")
				require.Empty(t, effectiveFlows(t, dynamic, desired.GetK8SNamespace()))
			} else {
				require.NoError(t, err)
				require.Equal(t, initial.TargetDeploymentId, connection.Data["deploymentId"])
				require.Equal(t, []flow{{initial.CallerDeploymentId, initial.TargetDeploymentId}}, effectiveFlows(t, dynamic, desired.GetK8SNamespace()))
			}

			failing = false
			if verb == "update" {
				services, listErr := client.CoreV1().Services(desired.GetK8SNamespace()).List(ctx, metav1.ListOptions{})
				require.NoError(t, listErr)
				for i := range services.Items {
					if services.Items[i].Labels[labels.LabelKeyDeploymentID] == desired.GetTargetDeploymentId() {
						_, createErr := client.DiscoveryV1().EndpointSlices(desired.GetK8SNamespace()).Create(ctx, readySlice(&services.Items[i], "desired-native", meshSliceManager), metav1.CreateOptions{})
						require.NoError(t, createErr)
					}
				}
			}
			require.NoError(t, r.reconcile(ctx))
			connection, err = client.CoreV1().ConfigMaps(desired.GetK8SNamespace()).Get(ctx, connectionResourceName(desired), metav1.GetOptions{})
			require.NoError(t, err)
			require.Equal(t, desired.TargetDeploymentId, connection.Data["deploymentId"])
			require.Contains(t, effectiveFlows(t, dynamic, desired.GetK8SNamespace()), flow{desired.CallerDeploymentId, desired.TargetDeploymentId})
		})
	}
}

func TestReconcilePolicyDeleteFailureRetainsDNSUntilRetry(t *testing.T) {
	ctx := t.Context()
	connectionSpec := newTestConnection()
	client := fake.NewClientset(endpointPod(connectionSpec, "a", "10.72.0.11"))
	dynamic := testDynamicClient()
	control := &testutil.MockClusterClient{}
	snapshot := []*ctrlv1.PrivateNetworkConnection{connectionSpec}
	control.StreamPrivateNetworkStateFunc = snapshotFunc(t, func(context.Context) ([]*ctrlv1.PrivateNetworkConnection, error) { return snapshot, nil })
	r := &Reconciler{clock: clock.NewTestClock(), client: client, dynamic: dynamic, cluster: control, clusterKey: &ctrlv1.ClusterKey{}}
	require.NoError(t, r.reconcile(ctx))

	failing := true
	dynamic.PrependReactor("delete", cilium.NetworkPolicyResource.Resource, func(clienttesting.Action) (bool, runtime.Object, error) {
		if !failing {
			return false, nil, nil
		}
		return true, nil, apierrors.NewForbidden(schema.GroupResource{Resource: cilium.NetworkPolicyResource.Resource}, policyName(connectionSpec), fmt.Errorf("injected policy failure"))
	})
	snapshot = nil
	require.NoError(t, r.reconcile(ctx))
	require.True(t, apierrors.IsForbidden(r.reconcile(ctx)))
	_, err := client.CoreV1().ConfigMaps(connectionSpec.GetK8SNamespace()).Get(ctx, connectionResourceName(connectionSpec), metav1.GetOptions{})
	require.NoError(t, err, "DNS must remain while policy revocation fails")
	require.Equal(t, []flow{{connectionSpec.CallerDeploymentId, connectionSpec.TargetDeploymentId}}, effectiveFlows(t, dynamic, connectionSpec.GetK8SNamespace()))

	failing = false
	require.NoError(t, r.reconcile(ctx))
	_, err = client.CoreV1().ConfigMaps(connectionSpec.GetK8SNamespace()).Get(ctx, connectionResourceName(connectionSpec), metav1.GetOptions{})
	require.True(t, apierrors.IsNotFound(err))
	_, err = dynamic.Resource(cilium.NetworkPolicyResource).Namespace(connectionSpec.GetK8SNamespace()).Get(ctx, policyName(connectionSpec), metav1.GetOptions{})
	require.True(t, apierrors.IsNotFound(err))
	require.Empty(t, effectiveFlows(t, dynamic, connectionSpec.GetK8SNamespace()), "revocation must not receive replacement overlap")
	services, err := client.CoreV1().Services(connectionSpec.GetK8SNamespace()).List(ctx, metav1.ListOptions{})
	require.NoError(t, err)
	require.Len(t, services.Items, 1, "Service retention must not retain policy access")
}

func TestReconcileConnectionPromotionRollback(t *testing.T) {
	ctx := t.Context()
	initial := newTestConnection()
	replacement := proto.Clone(initial).(*ctrlv1.PrivateNetworkConnection)
	replacement.TargetDeploymentId = uid.New(uid.DeploymentPrefix)
	client := fake.NewClientset(endpointPod(initial, "a", "10.72.0.11"), endpointPod(replacement, "b", "10.72.0.22"))
	control := &testutil.MockClusterClient{}
	now := time.Date(2026, 9, 21, 12, 0, 0, 0, time.UTC)
	clk := clock.NewTestClock(now)
	r := &Reconciler{client: client, dynamic: testDynamicClient(), cluster: control, clusterKey: &ctrlv1.ClusterKey{}, clock: clk}

	for i, deployment := range []string{initial.TargetDeploymentId, replacement.TargetDeploymentId, initial.TargetDeploymentId} {
		connectionSpec := proto.Clone(initial).(*ctrlv1.PrivateNetworkConnection)
		connectionSpec.TargetDeploymentId = deployment
		control.StreamPrivateNetworkStateFunc = snapshotFunc(t, func(context.Context) ([]*ctrlv1.PrivateNetworkConnection, error) {
			return []*ctrlv1.PrivateNetworkConnection{connectionSpec}, nil
		})
		serviceName := discoveryName(deployment, connectionSpec.GetTargetPort())
		connectionName := connectionResourceName(connectionSpec)
		require.NoError(t, r.reconcile(ctx))
		service, err := client.CoreV1().Services(connectionSpec.GetK8SNamespace()).Get(ctx, serviceName, metav1.GetOptions{})
		require.NoError(t, err)
		_, err = client.DiscoveryV1().EndpointSlices(connectionSpec.GetK8SNamespace()).Create(ctx, readySlice(service, fmt.Sprintf("native-%d", i), meshSliceManager), metav1.CreateOptions{})
		require.NoError(t, err)
		require.NoError(t, r.reconcile(ctx))

		connection, err := client.CoreV1().ConfigMaps(connectionSpec.GetK8SNamespace()).Get(ctx, connectionName, metav1.GetOptions{})
		require.NoError(t, err)
		require.Equal(t, deployment, connection.Data["deploymentId"])
		require.Equal(t, strconv.Itoa(i+1), connection.Data["revision"])

		services, err := client.CoreV1().Services(connectionSpec.GetK8SNamespace()).List(ctx, metav1.ListOptions{})
		require.NoError(t, err)
		require.LessOrEqual(t, len(services.Items), 2)

		service, err = client.CoreV1().Services(connectionSpec.GetK8SNamespace()).Get(ctx, serviceName, metav1.GetOptions{})
		require.NoError(t, err)
		require.Equal(t, corev1.ClusterIPNone, service.Spec.ClusterIP)
		require.False(t, service.Spec.PublishNotReadyAddresses)
		require.Equal(t, "true", service.Annotations[ciliumGlobal])
		require.Equal(t, "true", service.Annotations[ciliumGlobalSlices])
		require.NotContains(t, service.Annotations, privatenetwork.RetireAfterAnnotation)
		require.Empty(t, service.Spec.Selector)
		require.Equal(t, deployment, service.Labels[labels.LabelKeyDeploymentID])
	}

	control.StreamPrivateNetworkStateFunc = snapshotFunc(t, func(context.Context) ([]*ctrlv1.PrivateNetworkConnection, error) {
		return nil, nil
	})
	require.NoError(t, r.reconcile(ctx))
	connections, err := client.CoreV1().ConfigMaps("customer-1").List(ctx, metav1.ListOptions{})
	require.NoError(t, err)
	require.Len(t, connections.Items, 1)
	require.NoError(t, r.reconcile(ctx))
	services, err := client.CoreV1().Services("customer-1").List(ctx, metav1.ListOptions{})
	require.NoError(t, err)
	require.NotEmpty(t, services.Items)
	for _, service := range services.Items {
		deadline, ok := retirementDeadline(&service)
		require.True(t, ok)
		require.Equal(t, now.Add(privatenetwork.ReplacementOverlap), deadline)
	}
	connections, err = client.CoreV1().ConfigMaps("customer-1").List(ctx, metav1.ListOptions{})
	require.NoError(t, err)
	require.Empty(t, connections.Items)

	clk.Tick(privatenetwork.ReplacementOverlap - time.Nanosecond)
	require.NoError(t, r.reconcile(ctx))
	services, err = client.CoreV1().Services("customer-1").List(ctx, metav1.ListOptions{})
	require.NoError(t, err)
	require.NotEmpty(t, services.Items)
	clk.Tick(time.Nanosecond)
	require.NoError(t, r.reconcile(ctx))
	services, err = client.CoreV1().Services("customer-1").List(ctx, metav1.ListOptions{})
	require.NoError(t, err)
	require.Empty(t, services.Items)

	require.Empty(t, mustListReplicaSets(t, client, "customer-1"))
}

func TestReconcileRefusesOtherAppOwnership(t *testing.T) {
	ctx := t.Context()
	client := fake.NewClientset()
	r := &Reconciler{clock: clock.NewTestClock(), client: client}
	connectionSpec := newTestConnection()
	service, err := r.ensureService(ctx, connectionSpec, "service", nil)
	require.NoError(t, err)
	_, err = r.ensureConnection(ctx, connectionSpec, "connection", service, nil, nil)
	require.NoError(t, err)
	service, err = client.CoreV1().Services("customer-1").Get(ctx, "service", metav1.GetOptions{})
	require.NoError(t, err)
	connection, err := client.CoreV1().ConfigMaps("customer-1").Get(ctx, "connection", metav1.GetOptions{})
	require.NoError(t, err)

	other := proto.Clone(connectionSpec).(*ctrlv1.PrivateNetworkConnection)
	other.TargetDeploymentId, other.TargetAppId = uid.New(uid.DeploymentPrefix), uid.New(uid.AppPrefix)
	_, err = r.ensureService(ctx, other, "service", service)
	require.ErrorContains(t, err, "foreign Service")
	other.ConnectionId = uid.New(uid.ConnectionPrefix)
	_, err = r.ensureConnection(ctx, other, "connection", service, connection, nil)
	require.ErrorContains(t, err, "foreign ConfigMap")

	service, err = client.CoreV1().Services("customer-1").Get(ctx, "service", metav1.GetOptions{})
	require.NoError(t, err)
	require.Equal(t, connectionSpec.TargetDeploymentId, service.Labels[labels.LabelKeyDeploymentID])
	connection, err = client.CoreV1().ConfigMaps("customer-1").Get(ctx, "connection", metav1.GetOptions{})
	require.NoError(t, err)
	require.Equal(t, connectionSpec.TargetDeploymentId, connection.Data["deploymentId"])
}

func TestReconcileRejectsConnectionChangedDuringSnapshotFetch(t *testing.T) {
	ctx := t.Context()
	initial := newTestConnection()
	replacement := proto.Clone(initial).(*ctrlv1.PrivateNetworkConnection)
	replacement.TargetDeploymentId = uid.New(uid.DeploymentPrefix)
	concurrentDeploymentID := uid.New(uid.DeploymentPrefix)
	client := fake.NewClientset(endpointPod(replacement, "b", "10.72.0.22"))
	control := &testutil.MockClusterClient{}
	control.StreamPrivateNetworkStateFunc = snapshotFunc(t, func(context.Context) ([]*ctrlv1.PrivateNetworkConnection, error) {
		return []*ctrlv1.PrivateNetworkConnection{initial}, nil
	})
	r := &Reconciler{clock: clock.NewTestClock(), client: client, dynamic: testDynamicClient(), cluster: control, clusterKey: &ctrlv1.ClusterKey{}}
	require.NoError(t, r.reconcile(ctx))
	connectionName := connectionResourceName(initial)
	service, err := r.ensureService(ctx, replacement, discoveryName(replacement.GetTargetDeploymentId(), replacement.GetTargetPort()), nil)
	require.NoError(t, err)
	service.UID = types.UID(uid.New(uid.TestPrefix))
	service, err = client.CoreV1().Services(service.Namespace).Update(ctx, service, metav1.UpdateOptions{})
	require.NoError(t, err)
	_, err = client.DiscoveryV1().EndpointSlices(service.Namespace).Create(ctx, readySlice(service, "replacement-ready", meshSliceManager), metav1.CreateOptions{})
	require.NoError(t, err)

	control.StreamPrivateNetworkStateFunc = snapshotFunc(t, func(context.Context) ([]*ctrlv1.PrivateNetworkConnection, error) {
		connection, err := client.CoreV1().ConfigMaps("customer-1").Get(ctx, connectionName, metav1.GetOptions{})
		require.NoError(t, err)
		connection.ResourceVersion = "2"
		connection.Data["deploymentId"] = concurrentDeploymentID
		connection.Data["revision"] = "2"
		_, err = client.CoreV1().ConfigMaps("customer-1").Update(ctx, connection, metav1.UpdateOptions{})
		require.NoError(t, err)
		client.PrependReactor("update", "configmaps", func(action clienttesting.Action) (bool, runtime.Object, error) {
			update := action.(clienttesting.UpdateAction).GetObject().(*corev1.ConfigMap)
			if update.ResourceVersion != "2" {
				return true, nil, apierrors.NewConflict(schema.GroupResource{Resource: "configmaps"}, update.Name, fmt.Errorf("resource version changed"))
			}
			return false, nil, nil
		})
		return []*ctrlv1.PrivateNetworkConnection{replacement}, nil
	})

	require.True(t, apierrors.IsConflict(r.reconcile(ctx)))
	connection, err := client.CoreV1().ConfigMaps("customer-1").Get(ctx, connectionName, metav1.GetOptions{})
	require.NoError(t, err)
	require.Equal(t, concurrentDeploymentID, connection.Data["deploymentId"])
	require.Equal(t, "2", connection.Data["revision"])
}

func TestCleanupPreservesForeignResources(t *testing.T) {
	ctx := t.Context()
	foreign := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{
		Name: "foreign", Namespace: "customer-1", Labels: map[string]string{
			labels.LabelKeyManagedBy: "someone-else", labels.LabelKeyComponent: component,
		},
	}}
	managed := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{
		Name: "managed", Namespace: "customer-1", Labels: connectionLabels(newTestConnection()),
	}, Data: map[string]string{"revision": "1"}}
	client := fake.NewClientset(foreign, managed)
	control := &testutil.MockClusterClient{}
	control.StreamPrivateNetworkStateFunc = snapshotFunc(t, func(context.Context) ([]*ctrlv1.PrivateNetworkConnection, error) {
		return nil, nil
	})
	r := &Reconciler{clock: clock.NewTestClock(), client: client, dynamic: testDynamicClient(), cluster: control, clusterKey: &ctrlv1.ClusterKey{}}

	require.NoError(t, r.reconcile(ctx))
	_, err := client.CoreV1().ConfigMaps("customer-1").Get(ctx, "managed", metav1.GetOptions{})
	require.NoError(t, err)
	require.NoError(t, r.reconcile(ctx))
	_, err = client.CoreV1().ConfigMaps("customer-1").Get(ctx, "foreign", metav1.GetOptions{})
	require.NoError(t, err)
	_, err = client.CoreV1().ConfigMaps("customer-1").Get(ctx, "managed", metav1.GetOptions{})
	require.True(t, apierrors.IsNotFound(err))
}

func TestValidateSnapshotRejectsOnlyInvalidConnections(t *testing.T) {
	valid := newTestConnection()
	invalid := newTestConnection()
	invalid.WorkspaceId, invalid.ProjectId, invalid.TargetAppId = valid.WorkspaceId, valid.ProjectId, valid.TargetAppId
	invalid.TargetPort = 0
	duplicate := newTestConnection()
	duplicate.WorkspaceId, duplicate.ProjectId, duplicate.TargetAppId = valid.WorkspaceId, valid.ProjectId, valid.TargetAppId
	duplicateCopy := proto.Clone(duplicate).(*ctrlv1.PrivateNetworkConnection)
	duplicateCopy.ConnectionId = uid.New(uid.ConnectionPrefix)
	duplicate.ConnectionName, duplicateCopy.ConnectionName = "shared", "shared"

	snapshotConnections, rejected := validateSnapshot([]*ctrlv1.PrivateNetworkConnection{valid, invalid, duplicate, duplicateCopy, nil})
	require.Equal(t, []*ctrlv1.PrivateNetworkConnection{valid}, snapshotConnections)
	require.Len(t, rejected, 4)
	require.Equal(t, publishedConnectionKey(invalid), rejected[0].retainedConnectionKey)
	require.ErrorContains(t, rejected[0].err, "resolved target port must be positive")
	require.Equal(t, publishedConnectionKey(duplicate), rejected[1].retainedConnectionKey)
	require.Equal(t, publishedConnectionKey(duplicateCopy), rejected[2].retainedConnectionKey)
	require.ErrorContains(t, rejected[1].err, "duplicate connection identity")
	require.Empty(t, rejected[3].retainedConnectionKey, "a nil row identifies no published connection")
}

func TestReconcileRejectsInvalidConnectionAliases(t *testing.T) {
	for _, slug := range []string{"UPPER", "with_underscore", strings.Repeat("x", 64)} {
		t.Run(slug, func(t *testing.T) {
			ctx := t.Context()
			client := fake.NewClientset()
			control := &testutil.MockClusterClient{}
			valid := newTestConnection()
			incompatible := newTestConnection()
			incompatible.WorkspaceId, incompatible.ProjectId = valid.WorkspaceId, valid.ProjectId
			incompatible.ConnectionName = slug
			control.StreamPrivateNetworkStateFunc = snapshotFunc(t, func(context.Context) ([]*ctrlv1.PrivateNetworkConnection, error) {
				return []*ctrlv1.PrivateNetworkConnection{incompatible, valid}, nil
			})
			r := &Reconciler{clock: clock.NewTestClock(), client: client, dynamic: testDynamicClient(), cluster: control, clusterKey: &ctrlv1.ClusterKey{}}

			require.ErrorContains(t, r.reconcile(ctx), "invalid connection name")

			connections, err := client.CoreV1().ConfigMaps("customer-1").List(ctx, metav1.ListOptions{})
			require.NoError(t, err)
			require.Len(t, connections.Items, 1, "the valid connection still publishes")
			require.Equal(t, valid.ConnectionId, connections.Items[0].Labels[labels.LabelKeyConnectionID])
		})
	}
}

func TestReconcileFailedConnectionKeepsItsObjectsWithoutBlockingOthers(t *testing.T) {
	for _, tc := range []struct {
		stage, verb, resource string
	}{
		{"namespace", "create", "namespaces"},
		{"service", "create", "services"},
		{"connection", "update", "configmaps"},
	} {
		t.Run(tc.stage, func(t *testing.T) {
			ctx := t.Context()
			rejected := newTestConnection()
			replacement := proto.Clone(rejected).(*ctrlv1.PrivateNetworkConnection)
			replacement.TargetDeploymentId = uid.New(uid.DeploymentPrefix)
			healthy := newTestConnection()
			healthy.WorkspaceId, healthy.ProjectId = rejected.WorkspaceId, rejected.ProjectId
			healthy.TargetAppSlug, healthy.K8SNamespace, healthy.ConnectionName = "ledger", "customer-2", "ledger-api"
			obsolete := newTestConnection()
			obsolete.WorkspaceId, obsolete.ProjectId = rejected.WorkspaceId, rejected.ProjectId
			obsolete.TargetAppSlug, obsolete.K8SNamespace, obsolete.ConnectionName = "audit", "customer-2", "audit-api"

			client := fake.NewClientset(endpointPod(replacement, "a2", "10.72.0.12"))
			failing := false
			client.PrependReactor(tc.verb, tc.resource, func(action clienttesting.Action) (bool, runtime.Object, error) {
				namespace := action.GetNamespace()
				if tc.resource == "namespaces" {
					namespace = action.(clienttesting.CreateAction).GetObject().(*corev1.Namespace).Name
				}
				if !failing || namespace != rejected.GetK8SNamespace() {
					return false, nil, nil
				}
				return true, nil, apierrors.NewForbidden(schema.GroupResource{Resource: tc.resource}, "", fmt.Errorf("namespace is being terminated"))
			})

			snapshot := []*ctrlv1.PrivateNetworkConnection{rejected, obsolete}
			control := &testutil.MockClusterClient{}
			control.StreamPrivateNetworkStateFunc = snapshotFunc(t, func(context.Context) ([]*ctrlv1.PrivateNetworkConnection, error) {
				return snapshot, nil
			})
			r := &Reconciler{clock: clock.NewTestClock(), client: client, dynamic: testDynamicClient(), cluster: control, clusterKey: &ctrlv1.ClusterKey{}}

			connection := func(connectionSpec *ctrlv1.PrivateNetworkConnection) (*corev1.ConfigMap, error) {
				return client.CoreV1().ConfigMaps(connectionSpec.GetK8SNamespace()).Get(ctx, resourceName("unkey-pn-connection", connectionSpec.GetConnectionId()+"/"+connectionSpec.GetCallerDeploymentId()), metav1.GetOptions{})
			}

			require.NoError(t, r.reconcile(ctx))
			if tc.resource == "configmaps" {
				service, ensureErr := r.ensureService(ctx, replacement, discoveryName(replacement.GetTargetDeploymentId(), replacement.GetTargetPort()), nil)
				require.NoError(t, ensureErr)
				_, ensureErr = client.DiscoveryV1().EndpointSlices(replacement.GetK8SNamespace()).Create(ctx, readySlice(service, "failing-native", meshSliceManager), metav1.CreateOptions{})
				require.NoError(t, ensureErr)
			}
			if tc.resource == "namespaces" {
				require.NoError(t, client.CoreV1().Namespaces().Delete(ctx, rejected.GetK8SNamespace(), metav1.DeleteOptions{}))
			}

			failing = true
			snapshot = []*ctrlv1.PrivateNetworkConnection{replacement, healthy}
			reconcileErr := r.reconcile(ctx)
			published, err := connection(healthy)
			require.NoError(t, err, "connection after the rejected connection")
			require.Equal(t, healthy.TargetDeploymentId, published.Data["deploymentId"])
			kept, err := connection(rejected)
			require.NoError(t, err)
			require.Equal(t, rejected.TargetDeploymentId, kept.Data["deploymentId"])
			require.Equal(t, "1", kept.Data["revision"])
			require.Error(t, r.reconcile(ctx))
			_, err = connection(obsolete)
			require.True(t, apierrors.IsNotFound(err), "a connection missing from a complete snapshot is revoked even when another connection fails; connection(obsolete) error = %v", err)
			require.Equal(t, []string{obsolete.TargetDeploymentId}, retiringDeployments(t, client), "the failed connection keeps the Service it still uses")
			require.True(t, apierrors.IsForbidden(reconcileErr), "reconcile() error = %v, want Forbidden", reconcileErr)
			require.ErrorContains(t, reconcileErr, rejected.GetK8SNamespace())

			failing = false
			if tc.resource != "configmaps" {
				replacementService, err := r.ensureService(ctx, replacement, discoveryName(replacement.GetTargetDeploymentId(), replacement.GetTargetPort()), nil)
				require.NoError(t, err)
				_, err = client.DiscoveryV1().EndpointSlices(replacement.GetK8SNamespace()).Create(ctx, readySlice(replacementService, "replacement-native", meshSliceManager), metav1.CreateOptions{})
				require.NoError(t, err)
			}
			require.NoError(t, r.reconcile(ctx))
			switched, err := connection(rejected)
			require.NoError(t, err)
			require.Equal(t, replacement.TargetDeploymentId, switched.Data["deploymentId"])
			require.Equal(t, "2", switched.Data["revision"])
			require.ElementsMatch(t, []string{obsolete.TargetDeploymentId}, retiringDeployments(t, client))
		})
	}
}

func TestReconcileStopsWhenContextEnds(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	client := fake.NewClientset()
	client.PrependReactor("create", "namespaces", func(clienttesting.Action) (bool, runtime.Object, error) {
		cancel()
		return true, nil, context.Canceled
	})
	first := newTestConnection()
	later := newTestConnection()
	later.WorkspaceId, later.ProjectId = first.WorkspaceId, first.ProjectId
	later.TargetAppSlug, later.K8SNamespace, later.ConnectionName = "ledger", "customer-2", "ledger-api"
	control := &testutil.MockClusterClient{}
	control.StreamPrivateNetworkStateFunc = snapshotFunc(t, func(context.Context) ([]*ctrlv1.PrivateNetworkConnection, error) {
		return []*ctrlv1.PrivateNetworkConnection{first, later}, nil
	})
	r := &Reconciler{clock: clock.NewTestClock(), client: client, dynamic: testDynamicClient(), cluster: control, clusterKey: &ctrlv1.ClusterKey{}}

	require.ErrorIs(t, r.reconcile(ctx), context.Canceled)
	services, err := client.CoreV1().Services(later.GetK8SNamespace()).List(t.Context(), metav1.ListOptions{})
	require.NoError(t, err)
	require.Empty(t, services.Items, "reconcile continued with later connections after its context ended")
}

func retiringDeployments(t *testing.T, client *fake.Clientset) []string {
	t.Helper()
	services, err := client.CoreV1().Services("").List(t.Context(), metav1.ListOptions{})
	require.NoError(t, err)
	var deployments []string
	for _, service := range services.Items {
		if _, ok := retirementDeadline(&service); ok {
			deployments = append(deployments, service.Labels[labels.LabelKeyDeploymentID])
		}
	}
	return deployments
}

func mustListReplicaSets(t *testing.T, client *fake.Clientset, namespace string) []string {
	t.Helper()
	list, err := client.AppsV1().ReplicaSets(namespace).List(t.Context(), metav1.ListOptions{})
	require.NoError(t, err)
	names := make([]string, len(list.Items))
	for i := range list.Items {
		names[i] = list.Items[i].Name
	}
	return names
}

func TestValidateSnapshotRequiresPortOnlyForResolvedTargets(t *testing.T) {
	unresolved := newTestConnection()
	unresolved.TargetDeploymentId = ""
	unresolved.TargetPort = 0
	snapshotConnections, rejected := validateSnapshot([]*ctrlv1.PrivateNetworkConnection{unresolved})
	require.Empty(t, rejected)
	require.Len(t, snapshotConnections, 1)

	resolvedWithoutPort := newTestConnection()
	resolvedWithoutPort.TargetPort = 0
	_, rejected = validateSnapshot([]*ctrlv1.PrivateNetworkConnection{resolvedWithoutPort})
	require.Len(t, rejected, 1)
	require.ErrorContains(t, rejected[0].err, "resolved target port must be positive")
}

func TestReconcileReplicaDiscoveryPublishesCallerDeploymentForPeers(t *testing.T) {
	ctx := t.Context()
	client := fake.NewClientset()
	dynamic := testDynamicClient()
	self := newTestConnection()
	self.TargetAppSlug, self.ConnectionName = "caller", "caller"
	self.TargetDeploymentId = self.CallerDeploymentId
	self.ConnectionId = "self-" + self.CallerDeploymentId
	control := &testutil.MockClusterClient{}
	control.StreamPrivateNetworkStateFunc = snapshotFunc(t, func(context.Context) ([]*ctrlv1.PrivateNetworkConnection, error) {
		return []*ctrlv1.PrivateNetworkConnection{self}, nil
	})
	r := &Reconciler{clock: clock.NewTestClock(), client: client, dynamic: dynamic, cluster: control, clusterKey: &ctrlv1.ClusterKey{}}
	require.NoError(t, r.reconcile(ctx))

	connections, err := client.CoreV1().ConfigMaps("customer-1").List(ctx, metav1.ListOptions{})
	require.NoError(t, err)
	require.Len(t, connections.Items, 1)
	require.Equal(t, "caller", connections.Items[0].Data["appSlug"])
	require.Equal(t, self.CallerDeploymentId, connections.Items[0].Data["deploymentId"])
	require.Equal(t, self.CallerDeploymentId, connections.Items[0].Labels[labels.LabelKeyCallerDeploymentID])
	service, err := client.CoreV1().Services("customer-1").Get(ctx, connections.Items[0].Data["serviceName"], metav1.GetOptions{})
	require.NoError(t, err)
	require.Equal(t, self.CallerDeploymentId, service.Labels[labels.LabelKeyDeploymentID])
	require.Equal(t, self.TargetAppId, service.Labels[labels.LabelKeyAppID])
	require.Equal(t, "true", service.Annotations[ciliumGlobal])
	policies, err := dynamic.Resource(cilium.NetworkPolicyResource).Namespace("customer-1").List(ctx, metav1.ListOptions{})
	require.NoError(t, err)
	require.Empty(t, policies.Items, "the deployment policy grants replica traffic")
}

func TestReconcileEnsuresSharedTargetOncePerPass(t *testing.T) {
	ctx := t.Context()
	first := newTestConnection()
	second := proto.Clone(first).(*ctrlv1.PrivateNetworkConnection)
	second.CallerDeploymentId = uid.New(uid.DeploymentPrefix)
	second.ConnectionId = uid.New(uid.ConnectionPrefix)
	client := fake.NewClientset(endpointPod(first, "a", "10.72.0.11"))
	control := &testutil.MockClusterClient{}
	control.StreamPrivateNetworkStateFunc = snapshotFunc(t, func(context.Context) ([]*ctrlv1.PrivateNetworkConnection, error) {
		return []*ctrlv1.PrivateNetworkConnection{first, second}, nil
	})
	r := &Reconciler{clock: clock.NewTestClock(), client: client, dynamic: testDynamicClient(), cluster: control, clusterKey: &ctrlv1.ClusterKey{}}
	require.NoError(t, r.reconcile(ctx))
	client.ClearActions()

	require.NoError(t, r.reconcile(ctx))
	calls := map[string]int{}
	for _, action := range client.Actions() {
		calls[action.GetVerb()+" "+action.GetResource().Resource]++
	}
	require.Equal(t, 1, calls["list endpointslices"], "EndpointSlice lists for one shared target Service")
	require.Equal(t, 1, calls["list namespaces"], "namespace lists")
	require.Zero(t, calls["create namespaces"], "namespace creates for namespaces that exist")

	connections, err := client.CoreV1().ConfigMaps(first.GetK8SNamespace()).List(ctx, metav1.ListOptions{})
	require.NoError(t, err)
	require.Len(t, connections.Items, 2)
	for _, connection := range connections.Items {
		require.Equal(t, discoveryName(first.TargetDeploymentId, first.GetTargetPort()), connection.Data["serviceName"])
	}
}

func TestReconcileIncompleteSnapshotPreservesPolicyAndDNSUntilConfirmedAbsence(t *testing.T) {
	for _, tc := range []struct {
		name      string
		chunks    []*ctrlv1.PrivateNetworkStateChunk
		streamErr bool
		want      string
	}{
		{
			name: "truncated",
			chunks: []*ctrlv1.PrivateNetworkStateChunk{{Connections: []*ctrlv1.PrivateNetworkConnection{
				newTestConnection(),
			}}},
			want: "ended before it was complete",
		},
		{
			name: "count_mismatch",
			chunks: []*ctrlv1.PrivateNetworkStateChunk{
				{Connections: []*ctrlv1.PrivateNetworkConnection{newTestConnection()}},
				{Complete: true, Total: 2},
			},
			want: "complete chunk reports 2",
		},
		{
			name: "failed_mid_stream", streamErr: true,
			chunks: []*ctrlv1.PrivateNetworkStateChunk{{Connections: []*ctrlv1.PrivateNetworkConnection{
				newTestConnection(),
			}}},
			want: "unavailable",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := t.Context()
			published := newTestConnection()
			client := fake.NewClientset(endpointPod(published, "a", "10.72.0.11"))
			dynamic := testDynamicClient()
			control := &testutil.MockClusterClient{}
			control.StreamPrivateNetworkStateFunc = snapshotFunc(t, func(context.Context) ([]*ctrlv1.PrivateNetworkConnection, error) {
				return []*ctrlv1.PrivateNetworkConnection{published}, nil
			})
			r := &Reconciler{clock: clock.NewTestClock(), client: client, dynamic: dynamic, cluster: control, clusterKey: &ctrlv1.ClusterKey{}}
			require.NoError(t, r.reconcile(ctx))

			if tc.streamErr {
				control.StreamPrivateNetworkStateFunc = failedChunkStream(t, identified("incomplete", tc.chunks...)...)
			} else {
				control.StreamPrivateNetworkStateFunc = chunkStream(t, identified("incomplete", tc.chunks...)...)
			}
			require.ErrorContains(t, r.reconcile(ctx), tc.want)
			connections, err := client.CoreV1().ConfigMaps("customer-1").List(ctx, metav1.ListOptions{})
			require.NoError(t, err)
			require.Len(t, connections.Items, 1, "an incomplete snapshot must not revoke published DNS")
			require.Equal(t, published.TargetDeploymentId, connections.Items[0].Data["deploymentId"])
			require.Equal(t, []flow{{published.CallerDeploymentId, published.TargetDeploymentId}}, effectiveFlows(t, dynamic, "customer-1"), "an incomplete snapshot must not change policy")

			control.StreamPrivateNetworkStateFunc = snapshotFunc(t, func(context.Context) ([]*ctrlv1.PrivateNetworkConnection, error) { return nil, nil })
			require.NoError(t, r.reconcile(ctx))
			require.Equal(t, []flow{{published.CallerDeploymentId, published.TargetDeploymentId}}, effectiveFlows(t, dynamic, "customer-1"), "one complete snapshot cannot confirm absence")
			require.NoError(t, r.reconcile(ctx))
			connections, err = client.CoreV1().ConfigMaps("customer-1").List(ctx, metav1.ListOptions{})
			require.NoError(t, err)
			require.Empty(t, connections.Items, "two complete snapshots confirm absence")
			_, err = dynamic.Resource(cilium.NetworkPolicyResource).Namespace(published.GetK8SNamespace()).Get(ctx, policyName(published), metav1.GetOptions{})
			require.True(t, apierrors.IsNotFound(err))
			require.Empty(t, effectiveFlows(t, dynamic, "customer-1"))
		})
	}
}
