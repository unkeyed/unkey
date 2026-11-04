package privatenetwork

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync/atomic"
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
	discoveryv1 "k8s.io/api/discovery/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	fakedynamic "k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/kubernetes/fake"
	corelisters "k8s.io/client-go/listers/core/v1"
	discoverylisters "k8s.io/client-go/listers/discovery/v1"
	clienttesting "k8s.io/client-go/testing"
	"k8s.io/client-go/tools/cache"
)

func testDynamicClient(objects ...runtime.Object) *fakedynamic.FakeDynamicClient {
	scheme := runtime.NewScheme()
	return fakedynamic.NewSimpleDynamicClientWithCustomListKinds(scheme, map[schema.GroupVersionResource]string{
		cilium.NetworkPolicyResource: "CiliumNetworkPolicyList",
	}, objects...)
}

func newTestReconciler(t *testing.T, cfg Config) *Reconciler {
	t.Helper()
	if cfg.Client == nil {
		cfg.Client = fake.NewClientset()
	}
	if cfg.Dynamic == nil {
		cfg.Dynamic = testDynamicClient()
	}
	if cfg.Cluster == nil {
		cfg.Cluster = &testutil.MockClusterClient{}
	}
	if cfg.ClusterKey == nil {
		cfg.ClusterKey = &ctrlv1.ClusterKey{Region: testRegion}
	}
	if cfg.Clock == nil {
		cfg.Clock = clock.NewTestClock()
	}
	r, err := New(cfg)
	require.NoError(t, err)
	return r
}

func syncCaches(t *testing.T, r *Reconciler) {
	t.Helper()
	syncKubernetesCaches(t, r)
	policies, err := r.dynamic.Resource(cilium.NetworkPolicyResource).List(t.Context(), metav1.ListOptions{LabelSelector: discoverySelector.String()})
	require.NoError(t, err)
	objects := make([]runtime.Object, 0, len(policies.Items))
	for i := range policies.Items {
		objects = append(objects, &policies.Items[i])
	}
	r.policies = cache.NewGenericLister(newTestIndexer(t, cache.Indexers{}, objects...), cilium.NetworkPolicyResource.GroupResource())
}

// syncKubernetesCaches skips the dynamic client so it can run inside a
// dynamic client reactor, which holds the fake's lock.
func syncKubernetesCaches(t *testing.T, r *Reconciler) {
	t.Helper()
	ctx := t.Context()
	services, err := r.client.CoreV1().Services("").List(ctx, metav1.ListOptions{LabelSelector: discoverySelector.String()})
	require.NoError(t, err)
	endpointSlices, err := r.client.DiscoveryV1().EndpointSlices("").List(ctx, metav1.ListOptions{LabelSelector: discoverySelector.String()})
	require.NoError(t, err)
	pods, err := r.client.CoreV1().Pods("").List(ctx, metav1.ListOptions{LabelSelector: labels.New().ManagedByKrane().ComponentDeployment().ToString()})
	require.NoError(t, err)

	var objects []runtime.Object
	for i := range services.Items {
		objects = append(objects, &services.Items[i])
	}
	r.services = corelisters.NewServiceLister(newTestIndexer(t, cache.Indexers{}, objects...))
	objects = nil
	for i := range endpointSlices.Items {
		objects = append(objects, &endpointSlices.Items[i])
	}
	r.endpointSlices = discoverylisters.NewEndpointSliceLister(newTestIndexer(t, cache.Indexers{}, objects...))
	objects = nil
	for i := range pods.Items {
		objects = append(objects, &pods.Items[i])
	}
	r.pods = newTestIndexer(t, cache.Indexers{indexDeployment: deploymentIndex}, objects...)
}

func newTestIndexer(t *testing.T, indexers cache.Indexers, objects ...runtime.Object) cache.Indexer {
	t.Helper()
	indexers[cache.NamespaceIndex] = cache.MetaNamespaceIndexFunc
	indexer := cache.NewIndexer(cache.MetaNamespaceKeyFunc, indexers)
	for _, object := range objects {
		require.NoError(t, indexer.Add(object))
	}
	return indexer
}

func converge(t *testing.T, r *Reconciler) error {
	t.Helper()
	syncCaches(t, r)
	err := r.reconcile(t.Context())
	return errors.Join(err, publishEndpoints(t, r))
}

func publishEndpoints(t *testing.T, r *Reconciler) error {
	t.Helper()
	syncCaches(t, r)
	return r.reconcileEndpoints(t.Context())
}

func writeActions(actions []clienttesting.Action) []clienttesting.Action {
	return slices.DeleteFunc(actions, func(action clienttesting.Action) bool {
		return action.GetVerb() == "list" || action.GetVerb() == "get" || action.GetVerb() == "watch"
	})
}

func TestSteadyStateAt5000ConnectionsWritesNothing(t *testing.T) {
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
	r := newTestReconciler(t, Config{Client: client, Dynamic: policies, Cluster: control})
	require.NoError(t, converge(t, r))
	policies.ClearActions()
	client.ClearActions()
	control.StreamPrivateNetworkStateFunc = chunkStream(t, &ctrlv1.PrivateNetworkStateChunk{Version: "v1", SnapshotId: "one", Complete: true, Unchanged: true, Total: count})
	require.NoError(t, converge(t, r))
	require.Empty(t, writeActions(policies.Actions()), "unchanged policies need no writes")
	require.Empty(t, writeActions(client.Actions()), "unchanged discovery objects need no writes")
}

func TestReconcileRPCErrorPreservesPublishedObjects(t *testing.T) {
	ctx := t.Context()
	client := fake.NewClientset()
	control := &testutil.MockClusterClient{}
	connectionSpec := newTestConnection()
	control.StreamPrivateNetworkStateFunc = snapshotFunc(t, func(context.Context) ([]*ctrlv1.PrivateNetworkConnection, error) {
		return []*ctrlv1.PrivateNetworkConnection{connectionSpec}, nil
	})
	r := newTestReconciler(t, Config{Client: client, Dynamic: testDynamicClient(), Cluster: control})
	require.NoError(t, converge(t, r))

	control.StreamPrivateNetworkStateFunc = snapshotFunc(t, func(context.Context) ([]*ctrlv1.PrivateNetworkConnection, error) {
		return nil, fmt.Errorf("unavailable")
	})
	require.Error(t, converge(t, r))
	require.Equal(t, connectionSpec.TargetDeploymentId, resolvedDeployment(t, client, connectionSpec))
	services, err := client.CoreV1().Services("customer-1").List(ctx, metav1.ListOptions{})
	require.NoError(t, err)
	require.Len(t, services.Items, 2, "one shared Service and one alias")
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
			r := newTestReconciler(t, Config{Client: client, Dynamic: dynamic, Cluster: control})
			if verb == "update" {
				require.NoError(t, converge(t, r))
				current = desired
			}

			failing := true
			dynamic.PrependReactor(verb, cilium.NetworkPolicyResource.Resource, func(clienttesting.Action) (bool, runtime.Object, error) {
				if !failing {
					return false, nil, nil
				}
				return true, nil, apierrors.NewForbidden(schema.GroupResource{Resource: cilium.NetworkPolicyResource.Resource}, policyName(desired), fmt.Errorf("injected policy failure"))
			})
			require.True(t, apierrors.IsForbidden(converge(t, r)))
			if verb == "create" {
				require.Nil(t, aliasService(t, client, desired), "DNS was published without its policy")
				require.Empty(t, effectiveFlows(t, dynamic, desired.GetK8SNamespace()))
			} else {
				require.Equal(t, initial.TargetDeploymentId, resolvedDeployment(t, client, desired))
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
			require.NoError(t, converge(t, r))
			require.Equal(t, desired.TargetDeploymentId, resolvedDeployment(t, client, desired))
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
	r := newTestReconciler(t, Config{Client: client, Dynamic: dynamic, Cluster: control})
	require.NoError(t, converge(t, r))

	failing := true
	dynamic.PrependReactor("delete", cilium.NetworkPolicyResource.Resource, func(clienttesting.Action) (bool, runtime.Object, error) {
		if !failing {
			return false, nil, nil
		}
		return true, nil, apierrors.NewForbidden(schema.GroupResource{Resource: cilium.NetworkPolicyResource.Resource}, policyName(connectionSpec), fmt.Errorf("injected policy failure"))
	})
	snapshot = nil
	require.NoError(t, converge(t, r))
	require.True(t, apierrors.IsForbidden(converge(t, r)))
	require.Equal(t, connectionSpec.TargetDeploymentId, resolvedDeployment(t, client, connectionSpec), "DNS must remain while policy revocation fails")
	require.Equal(t, []flow{{connectionSpec.CallerDeploymentId, connectionSpec.TargetDeploymentId}}, effectiveFlows(t, dynamic, connectionSpec.GetK8SNamespace()))

	failing = false
	require.NoError(t, converge(t, r))
	require.Nil(t, aliasService(t, client, connectionSpec))
	_, err := dynamic.Resource(cilium.NetworkPolicyResource).Namespace(connectionSpec.GetK8SNamespace()).Get(ctx, policyName(connectionSpec), metav1.GetOptions{})
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
	r := newTestReconciler(t, Config{Client: client, Dynamic: testDynamicClient(), Cluster: control, Clock: clk})

	for i, deployment := range []string{initial.TargetDeploymentId, replacement.TargetDeploymentId, initial.TargetDeploymentId} {
		connectionSpec := proto.Clone(initial).(*ctrlv1.PrivateNetworkConnection)
		connectionSpec.TargetDeploymentId = deployment
		control.StreamPrivateNetworkStateFunc = snapshotFunc(t, func(context.Context) ([]*ctrlv1.PrivateNetworkConnection, error) {
			return []*ctrlv1.PrivateNetworkConnection{connectionSpec}, nil
		})
		serviceName := privatenetwork.ServiceName(deployment, connectionSpec.GetTargetPort())
		require.NoError(t, converge(t, r))
		service, err := client.CoreV1().Services(connectionSpec.GetK8SNamespace()).Get(ctx, serviceName, metav1.GetOptions{})
		require.NoError(t, err)
		_, err = client.DiscoveryV1().EndpointSlices(connectionSpec.GetK8SNamespace()).Create(ctx, readySlice(service, fmt.Sprintf("native-%d", i), meshSliceManager), metav1.CreateOptions{})
		require.NoError(t, err)
		require.NoError(t, converge(t, r))

		require.Equal(t, deployment, resolvedDeployment(t, client, connectionSpec))
		require.Len(t, connectionServices(t, client, connectionSpec.GetK8SNamespace()), 1)

		services, err := client.CoreV1().Services(connectionSpec.GetK8SNamespace()).List(ctx, metav1.ListOptions{})
		require.NoError(t, err)
		require.LessOrEqual(t, len(services.Items), 3, "at most two shared Services and one alias")

		service, err = client.CoreV1().Services(connectionSpec.GetK8SNamespace()).Get(ctx, serviceName, metav1.GetOptions{})
		require.NoError(t, err)
		require.Equal(t, corev1.ClusterIPNone, service.Spec.ClusterIP)
		require.False(t, service.Spec.PublishNotReadyAddresses)
		require.Equal(t, "true", service.Annotations[annotationCiliumGlobal])
		require.Equal(t, "true", service.Annotations[annotationCiliumGlobalSlices])
		require.NotContains(t, service.Annotations, privatenetwork.RetireAfterAnnotation)
		require.Empty(t, service.Spec.Selector)
		require.Equal(t, deployment, service.Labels[labels.LabelKeyDeploymentID])
	}

	control.StreamPrivateNetworkStateFunc = snapshotFunc(t, func(context.Context) ([]*ctrlv1.PrivateNetworkConnection, error) {
		return nil, nil
	})
	require.NoError(t, converge(t, r))
	require.Len(t, connectionServices(t, client, "customer-1"), 1)
	require.NoError(t, converge(t, r))
	require.Empty(t, connectionServices(t, client, "customer-1"))
	services, err := client.CoreV1().Services("customer-1").List(ctx, metav1.ListOptions{})
	require.NoError(t, err)
	require.NotEmpty(t, services.Items)
	for _, service := range services.Items {
		deadline, ok := retirementDeadline(&service)
		require.True(t, ok)
		require.Equal(t, now.Add(privatenetwork.ReplacementOverlap), deadline)
	}

	clk.Tick(privatenetwork.ReplacementOverlap - time.Nanosecond)
	require.NoError(t, converge(t, r))
	services, err = client.CoreV1().Services("customer-1").List(ctx, metav1.ListOptions{})
	require.NoError(t, err)
	require.NotEmpty(t, services.Items)
	clk.Tick(time.Nanosecond)
	require.NoError(t, converge(t, r))
	services, err = client.CoreV1().Services("customer-1").List(ctx, metav1.ListOptions{})
	require.NoError(t, err)
	require.Empty(t, services.Items)

	require.Empty(t, mustListReplicaSets(t, client, "customer-1"))
}

func TestReconcileRefusesOtherAppOwnership(t *testing.T) {
	ctx := t.Context()
	client := fake.NewClientset()
	r := newTestReconciler(t, Config{Client: client})
	connectionSpec := newTestConnection()
	serviceName := privatenetwork.ServiceName(connectionSpec.GetTargetDeploymentId(), connectionSpec.GetTargetPort())
	service, err := r.ensureService(ctx, connectionSpec, nil)
	require.NoError(t, err)
	connection, err := r.ensureConnection(ctx, connectionSpec, aliasName(connectionSpec), "", service, nil, "")
	require.NoError(t, err)

	other := proto.Clone(connectionSpec).(*ctrlv1.PrivateNetworkConnection)
	other.TargetDeploymentId, other.TargetAppId = uid.New(uid.DeploymentPrefix), uid.New(uid.AppPrefix)
	_, err = r.ensureService(ctx, other, service)
	require.ErrorContains(t, err, "foreign Service")
	other.ConnectionId = uid.New(uid.ConnectionPrefix)
	_, err = r.ensureConnection(ctx, other, aliasName(other), "", service, connection, connectionSpec.TargetDeploymentId)
	require.ErrorContains(t, err, "foreign Service")
	_, err = r.ensureService(ctx, connectionSpec, connection)
	require.ErrorContains(t, err, "foreign Service", "an alias is never adopted as a shared Service")

	require.Equal(t, connectionSpec.TargetDeploymentId, resolvedDeployment(t, client, connectionSpec))
	service, err = client.CoreV1().Services("customer-1").Get(ctx, serviceName, metav1.GetOptions{})
	require.NoError(t, err)
	require.Equal(t, connectionSpec.TargetDeploymentId, service.Labels[labels.LabelKeyDeploymentID])
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
	r := newTestReconciler(t, Config{Client: client, Dynamic: testDynamicClient(), Cluster: control})
	require.NoError(t, converge(t, r))
	aliasName := aliasService(t, client, initial).Name
	service, err := r.ensureService(ctx, replacement, nil)
	require.NoError(t, err)
	service.UID = types.UID(uid.New(uid.TestPrefix))
	service, err = client.CoreV1().Services(service.Namespace).Update(ctx, service, metav1.UpdateOptions{})
	require.NoError(t, err)
	_, err = client.DiscoveryV1().EndpointSlices(service.Namespace).Create(ctx, readySlice(service, "replacement-ready", meshSliceManager), metav1.CreateOptions{})
	require.NoError(t, err)

	control.StreamPrivateNetworkStateFunc = snapshotFunc(t, func(context.Context) ([]*ctrlv1.PrivateNetworkConnection, error) {
		connection, err := client.CoreV1().Services("customer-1").Get(ctx, aliasName, metav1.GetOptions{})
		require.NoError(t, err)
		connection.ResourceVersion = "2"
		connection.Annotations[privatenetwork.TargetDeploymentAnnotation] = concurrentDeploymentID
		connection.Spec.ExternalName = privatenetwork.ServiceHost(privatenetwork.ServiceName(concurrentDeploymentID, initial.GetTargetPort()), "customer-1")
		_, err = client.CoreV1().Services("customer-1").Update(ctx, connection, metav1.UpdateOptions{})
		require.NoError(t, err)
		client.PrependReactor("update", "services", func(action clienttesting.Action) (bool, runtime.Object, error) {
			update := action.(clienttesting.UpdateAction).GetObject().(*corev1.Service)
			if update.Name == aliasName && update.ResourceVersion != "2" {
				return true, nil, apierrors.NewConflict(schema.GroupResource{Resource: "services"}, update.Name, fmt.Errorf("resource version changed"))
			}
			return false, nil, nil
		})
		return []*ctrlv1.PrivateNetworkConnection{replacement}, nil
	})

	require.True(t, apierrors.IsConflict(converge(t, r)))
	connection, err := client.CoreV1().Services("customer-1").Get(ctx, aliasName, metav1.GetOptions{})
	require.NoError(t, err)
	require.Equal(t, concurrentDeploymentID, connection.Annotations[privatenetwork.TargetDeploymentAnnotation])
	require.Equal(t, privatenetwork.ServiceHost(privatenetwork.ServiceName(concurrentDeploymentID, initial.GetTargetPort()), "customer-1"), connection.Spec.ExternalName)
}

func TestCleanupPreservesForeignResources(t *testing.T) {
	ctx := t.Context()
	spec := newTestConnection()
	external := corev1.ServiceSpec{Type: corev1.ServiceTypeExternalName, ExternalName: privatenetwork.ServiceHost(privatenetwork.ServiceName(spec.TargetDeploymentId, spec.TargetPort), "customer-1")}
	foreign := &corev1.Service{ObjectMeta: metav1.ObjectMeta{
		Name: "foreign", Namespace: "customer-1", Labels: map[string]string{
			labels.LabelKeyManagedBy: "someone-else", labels.LabelKeyComponent: component,
			labels.LabelKeyConnectionID: spec.ConnectionId, labels.LabelKeyCallerDeploymentID: spec.CallerDeploymentId,
		},
	}, Spec: external}
	managed := &corev1.Service{ObjectMeta: metav1.ObjectMeta{
		Name: "managed", Namespace: "customer-1", Labels: connectionLabels(spec),
		Annotations: map[string]string{privatenetwork.TargetDeploymentAnnotation: spec.TargetDeploymentId},
	}, Spec: external}
	client := fake.NewClientset(foreign, managed)
	control := &testutil.MockClusterClient{}
	control.StreamPrivateNetworkStateFunc = snapshotFunc(t, func(context.Context) ([]*ctrlv1.PrivateNetworkConnection, error) {
		return nil, nil
	})
	r := newTestReconciler(t, Config{Client: client, Dynamic: testDynamicClient(), Cluster: control})

	require.NoError(t, converge(t, r))
	_, err := client.CoreV1().Services("customer-1").Get(ctx, "managed", metav1.GetOptions{})
	require.NoError(t, err)
	require.NoError(t, converge(t, r))
	_, err = client.CoreV1().Services("customer-1").Get(ctx, "foreign", metav1.GetOptions{})
	require.NoError(t, err)
	_, err = client.CoreV1().Services("customer-1").Get(ctx, "managed", metav1.GetOptions{})
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
	require.Equal(t, publishedConnectionKey(invalid), rejected[0].connectionKeyRetained)
	require.ErrorContains(t, rejected[0].err, "resolved target port must be positive")
	require.Equal(t, publishedConnectionKey(duplicate), rejected[1].connectionKeyRetained)
	require.Equal(t, publishedConnectionKey(duplicateCopy), rejected[2].connectionKeyRetained)
	require.ErrorContains(t, rejected[1].err, "duplicate connection identity")
	require.Empty(t, rejected[3].connectionKeyRetained, "a nil row identifies no published connection")
}

func TestReconcileRejectsInvalidConnectionAliases(t *testing.T) {
	for _, slug := range []string{"UPPER", "with_underscore", strings.Repeat("x", 64)} {
		t.Run(slug, func(t *testing.T) {
			client := fake.NewClientset()
			control := &testutil.MockClusterClient{}
			valid := newTestConnection()
			incompatible := newTestConnection()
			incompatible.WorkspaceId, incompatible.ProjectId = valid.WorkspaceId, valid.ProjectId
			incompatible.ConnectionName = slug
			control.StreamPrivateNetworkStateFunc = snapshotFunc(t, func(context.Context) ([]*ctrlv1.PrivateNetworkConnection, error) {
				return []*ctrlv1.PrivateNetworkConnection{incompatible, valid}, nil
			})
			r := newTestReconciler(t, Config{Client: client, Dynamic: testDynamicClient(), Cluster: control})

			require.ErrorContains(t, converge(t, r), "invalid connection name")

			connections := connectionServices(t, client, "customer-1")
			require.Len(t, connections, 1, "the valid connection still publishes")
			require.Equal(t, valid.ConnectionId, connections[0].Labels[labels.LabelKeyConnectionID])
		})
	}
}

func TestReconcileFailedConnectionKeepsItsObjectsWithoutBlockingOthers(t *testing.T) {
	for _, tc := range []struct {
		stage, verb, resource string
	}{
		{"service", "create", "services"},
		{"connection", "update", "services"},
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
				if !failing || action.GetNamespace() != rejected.GetK8SNamespace() {
					return false, nil, nil
				}
				return true, nil, apierrors.NewForbidden(schema.GroupResource{Resource: tc.resource}, "", fmt.Errorf("namespace is being terminated"))
			})

			snapshot := []*ctrlv1.PrivateNetworkConnection{rejected, obsolete}
			control := &testutil.MockClusterClient{}
			control.StreamPrivateNetworkStateFunc = snapshotFunc(t, func(context.Context) ([]*ctrlv1.PrivateNetworkConnection, error) {
				return snapshot, nil
			})
			r := newTestReconciler(t, Config{Client: client, Dynamic: testDynamicClient(), Cluster: control})

			require.NoError(t, converge(t, r))
			if tc.stage == "connection" {
				service, ensureErr := r.ensureService(ctx, replacement, nil)
				require.NoError(t, ensureErr)
				_, ensureErr = client.DiscoveryV1().EndpointSlices(replacement.GetK8SNamespace()).Create(ctx, readySlice(service, "failing-native", meshSliceManager), metav1.CreateOptions{})
				require.NoError(t, ensureErr)
			}

			failing = true
			snapshot = []*ctrlv1.PrivateNetworkConnection{replacement, healthy}
			reconcileErr := converge(t, r)
			require.Equal(t, healthy.TargetDeploymentId, resolvedDeployment(t, client, healthy), "connection after the rejected connection")
			require.Equal(t, rejected.TargetDeploymentId, resolvedDeployment(t, client, rejected))
			require.Error(t, converge(t, r))
			require.Nil(t, aliasService(t, client, obsolete), "a connection missing from a complete snapshot is revoked even when another connection fails")
			require.Equal(t, []string{obsolete.TargetDeploymentId}, retiringDeployments(t, client), "the failed connection keeps the Service it still uses")
			require.True(t, apierrors.IsForbidden(reconcileErr), "reconcile() error = %v, want Forbidden", reconcileErr)
			require.ErrorContains(t, reconcileErr, rejected.GetK8SNamespace())

			failing = false
			if tc.stage != "connection" {
				replacementService, err := r.ensureService(ctx, replacement, nil)
				require.NoError(t, err)
				_, err = client.DiscoveryV1().EndpointSlices(replacement.GetK8SNamespace()).Create(ctx, readySlice(replacementService, "replacement-native", meshSliceManager), metav1.CreateOptions{})
				require.NoError(t, err)
			}
			require.NoError(t, converge(t, r))
			require.Equal(t, replacement.TargetDeploymentId, resolvedDeployment(t, client, rejected))
			require.ElementsMatch(t, []string{obsolete.TargetDeploymentId}, retiringDeployments(t, client))
		})
	}
}

func TestReconcileStopsWhenContextEnds(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	client := fake.NewClientset()
	client.PrependReactor("create", "services", func(clienttesting.Action) (bool, runtime.Object, error) {
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
	r := newTestReconciler(t, Config{Client: client, Dynamic: testDynamicClient(), Cluster: control})

	syncCaches(t, r)
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
	self.TargetAppSlug, self.ConnectionName = "caller", privatenetwork.SelfName
	self.TargetDeploymentId = self.CallerDeploymentId
	self.ConnectionId = "self-" + self.CallerDeploymentId
	control := &testutil.MockClusterClient{}
	control.StreamPrivateNetworkStateFunc = snapshotFunc(t, func(context.Context) ([]*ctrlv1.PrivateNetworkConnection, error) {
		return []*ctrlv1.PrivateNetworkConnection{self}, nil
	})
	r := newTestReconciler(t, Config{Client: client, Dynamic: dynamic, Cluster: control})
	require.NoError(t, converge(t, r))

	connections := connectionServices(t, client, "customer-1")
	require.Len(t, connections, 1)
	require.Equal(t, privatenetwork.ConnectionServiceName(self.CallerDeploymentId, privatenetwork.SelfName), connections[0].Name)
	require.Equal(t, self.CallerDeploymentId, connections[0].Labels[labels.LabelKeyCallerDeploymentID])
	service := resolvedTarget(t, client, self)
	require.Equal(t, privatenetwork.ServiceName(self.CallerDeploymentId, self.GetTargetPort()), service.Name, "self resolves the caller's own shared Service")
	require.Equal(t, self.CallerDeploymentId, service.Labels[labels.LabelKeyDeploymentID])
	require.Equal(t, self.TargetAppId, service.Labels[labels.LabelKeyAppID])
	require.Equal(t, "true", service.Annotations[annotationCiliumGlobal])
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
	r := newTestReconciler(t, Config{Client: client, Dynamic: testDynamicClient(), Cluster: control})
	require.NoError(t, converge(t, r))
	client.ClearActions()

	require.NoError(t, converge(t, r))
	require.Empty(t, writeActions(client.Actions()), "a steady state needs no writes")

	require.Len(t, connectionServices(t, client, first.GetK8SNamespace()), 2)
	for _, connectionSpec := range []*ctrlv1.PrivateNetworkConnection{first, second} {
		require.Equal(t, privatenetwork.ServiceName(first.TargetDeploymentId, first.GetTargetPort()), resolvedTarget(t, client, connectionSpec).Name)
	}
	endpointSlices, err := client.DiscoveryV1().EndpointSlices(first.GetK8SNamespace()).List(ctx, metav1.ListOptions{})
	require.NoError(t, err)
	require.Len(t, endpointSlices.Items, 1, "callers share one source EndpointSlice")
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
			r := newTestReconciler(t, Config{Client: client, Dynamic: dynamic, Cluster: control})
			require.NoError(t, converge(t, r))

			if tc.streamErr {
				control.StreamPrivateNetworkStateFunc = failedChunkStream(t, identified("incomplete", tc.chunks...)...)
			} else {
				control.StreamPrivateNetworkStateFunc = chunkStream(t, identified("incomplete", tc.chunks...)...)
			}
			require.ErrorContains(t, converge(t, r), tc.want)
			require.Equal(t, published.TargetDeploymentId, resolvedDeployment(t, client, published), "an incomplete snapshot must not revoke published DNS")
			require.Equal(t, []flow{{published.CallerDeploymentId, published.TargetDeploymentId}}, effectiveFlows(t, dynamic, "customer-1"), "an incomplete snapshot must not change policy")

			control.StreamPrivateNetworkStateFunc = snapshotFunc(t, func(context.Context) ([]*ctrlv1.PrivateNetworkConnection, error) { return nil, nil })
			require.NoError(t, converge(t, r))
			require.Equal(t, []flow{{published.CallerDeploymentId, published.TargetDeploymentId}}, effectiveFlows(t, dynamic, "customer-1"), "one complete snapshot cannot confirm absence")
			require.NoError(t, converge(t, r))
			require.Empty(t, connectionServices(t, client, "customer-1"), "two complete snapshots confirm absence")
			_, err := dynamic.Resource(cilium.NetworkPolicyResource).Namespace(published.GetK8SNamespace()).Get(ctx, policyName(published), metav1.GetOptions{})
			require.True(t, apierrors.IsNotFound(err))
			require.Empty(t, effectiveFlows(t, dynamic, "customer-1"))
		})
	}
}

func TestReconcileRetargetsOneCallerWithoutMovingAnotherCallerOfTheSameTarget(t *testing.T) {
	first := newTestConnection()
	second := proto.Clone(first).(*ctrlv1.PrivateNetworkConnection)
	second.CallerDeploymentId = uid.New(uid.DeploymentPrefix)
	second.ConnectionId = uid.New(uid.ConnectionPrefix)
	moved := proto.Clone(first).(*ctrlv1.PrivateNetworkConnection)
	moved.TargetDeploymentId = uid.New(uid.DeploymentPrefix)
	client := fake.NewClientset(endpointPod(first, "a", "10.72.0.11"), endpointPod(moved, "b", "10.72.0.22"))
	dynamic := testDynamicClient()
	snapshot := []*ctrlv1.PrivateNetworkConnection{first, second}
	control := &testutil.MockClusterClient{StreamPrivateNetworkStateFunc: snapshotFunc(t, func(context.Context) ([]*ctrlv1.PrivateNetworkConnection, error) {
		return snapshot, nil
	})}
	r := newTestReconciler(t, Config{Client: client, Dynamic: dynamic, Cluster: control})
	require.NoError(t, converge(t, r))
	require.NotEqual(t, aliasService(t, client, first).Name, aliasService(t, client, second).Name, "callers sharing an alias get separate Services")

	snapshot = []*ctrlv1.PrivateNetworkConnection{moved, second}
	require.NoError(t, converge(t, r))
	require.NoError(t, converge(t, r))
	require.Equal(t, []string{"10.72.0.22"}, resolvedAddresses(t, client, moved))
	require.Equal(t, []string{"10.72.0.11"}, resolvedAddresses(t, client, second))
	require.ElementsMatch(t, []flow{
		{first.CallerDeploymentId, first.TargetDeploymentId}, {first.CallerDeploymentId, moved.TargetDeploymentId},
		{second.CallerDeploymentId, first.TargetDeploymentId},
	}, effectiveFlows(t, dynamic, "customer-1"), "the moved caller keeps its old grant during overlap; the other caller gains nothing")
	require.Empty(t, retiringDeployments(t, client), "a target another caller still resolves stays published")
}

func TestReconcilePublishesOnlyConnectionsThatRunHere(t *testing.T) {
	ctx := t.Context()
	callerHere, targetHere, elsewhere := newTestConnection(), newTestConnection(), newTestConnection()
	callerHere.TargetRegions = []string{"eu-central-1"}
	targetHere.CallerRegions = []string{"eu-central-1"}
	targetHere.K8SNamespace = "customer-2"
	elsewhere.CallerRegions, elsewhere.TargetRegions = []string{"eu-central-1"}, []string{"ap-south-1"}
	elsewhere.K8SNamespace = "customer-3"
	snapshot := []*ctrlv1.PrivateNetworkConnection{callerHere, targetHere, elsewhere}
	client := fake.NewClientset(endpointPod(targetHere, "a", "10.72.0.11"))
	dynamic := testDynamicClient()
	control := &testutil.MockClusterClient{StreamPrivateNetworkStateFunc: snapshotFunc(t, func(context.Context) ([]*ctrlv1.PrivateNetworkConnection, error) {
		return snapshot, nil
	})}
	r := newTestReconciler(t, Config{Client: client, Dynamic: dynamic, Cluster: control})
	require.NoError(t, converge(t, r))

	policyExists := func(spec *ctrlv1.PrivateNetworkConnection) bool {
		_, err := dynamic.Resource(cilium.NetworkPolicyResource).Namespace(spec.GetK8SNamespace()).Get(ctx, policyName(spec), metav1.GetOptions{})
		if apierrors.IsNotFound(err) {
			return false
		}
		require.NoError(t, err)
		return true
	}
	sharedExists := func(spec *ctrlv1.PrivateNetworkConnection) bool {
		_, err := client.CoreV1().Services(spec.GetK8SNamespace()).Get(ctx, privatenetwork.ServiceName(spec.GetTargetDeploymentId(), spec.GetTargetPort()), metav1.GetOptions{})
		if apierrors.IsNotFound(err) {
			return false
		}
		require.NoError(t, err)
		return true
	}

	require.NotNil(t, aliasService(t, client, callerHere))
	require.True(t, policyExists(callerHere))
	require.True(t, sharedExists(callerHere))

	require.Nil(t, aliasService(t, client, targetHere), "a caller elsewhere resolves its alias in its own cluster")
	require.True(t, policyExists(targetHere), "the target's cluster must admit the remote caller")
	require.True(t, sharedExists(targetHere), "the target's cluster exports its endpoints")

	require.Nil(t, aliasService(t, client, elsewhere))
	require.False(t, policyExists(elsewhere))
	require.False(t, sharedExists(elsewhere))
	_, err := client.CoreV1().Namespaces().Get(ctx, elsewhere.GetK8SNamespace(), metav1.GetOptions{})
	require.True(t, apierrors.IsNotFound(err), "no namespace for a workspace that runs nothing here")

	callerHere.CallerRegions = []string{"eu-central-1"}
	callerHere.TargetRegions = []string{testRegion}
	require.NoError(t, converge(t, r))
	require.Nil(t, aliasService(t, client, callerHere), "a caller that moved away withdraws its alias here")
	require.True(t, policyExists(callerHere))
}

func TestReconcileRegionMoveKeepsOldTargetGrantUntilOverlapEnds(t *testing.T) {
	setup := func(t *testing.T) (*Reconciler, *clock.TestClock, *ctrlv1.PrivateNetworkConnection, func(...*ctrlv1.PrivateNetworkConnection), func() map[string]time.Time) {
		t.Helper()
		old := newTestConnection()
		old.CallerRegions, old.TargetRegions = []string{"eu-central-1"}, []string{testRegion}
		client := fake.NewClientset(endpointPod(old, "a", "10.72.0.11"))
		dynamic := testDynamicClient()
		var snapshot []*ctrlv1.PrivateNetworkConnection
		control := &testutil.MockClusterClient{StreamPrivateNetworkStateFunc: snapshotFunc(t, func(context.Context) ([]*ctrlv1.PrivateNetworkConnection, error) {
			return snapshot, nil
		})}
		clk := clock.NewTestClock(time.Date(2026, 11, 2, 12, 0, 0, 0, time.UTC))
		r := newTestReconciler(t, Config{Clock: clk, Client: client, Dynamic: dynamic, Cluster: control})
		publish := func(connections ...*ctrlv1.PrivateNetworkConnection) { snapshot = connections }
		grants := func() map[string]time.Time {
			t.Helper()
			policy, err := dynamic.Resource(cilium.NetworkPolicyResource).Namespace(old.GetK8SNamespace()).Get(t.Context(), policyName(old), metav1.GetOptions{})
			if apierrors.IsNotFound(err) {
				return nil
			}
			require.NoError(t, err)
			targets := map[string]time.Time{}
			require.NoError(t, json.Unmarshal([]byte(policy.GetAnnotations()[policyTargetsAnnotation]), &targets))
			require.Len(t, policySpecs(t, policy), 2*len(targets))
			return targets
		}
		publish(old)
		require.NoError(t, converge(t, r))
		require.Equal(t, map[string]time.Time{old.GetTargetDeploymentId(): {}}, grants())
		return r, clk, old, publish, grants
	}
	moved := func(old *ctrlv1.PrivateNetworkConnection) *ctrlv1.PrivateNetworkConnection {
		replacement := proto.Clone(old).(*ctrlv1.PrivateNetworkConnection)
		replacement.TargetDeploymentId, replacement.TargetRegions = uid.New(uid.DeploymentPrefix), []string{"eu-central-1"}
		return replacement
	}

	t.Run("the old target stays admitted for the overlap", func(t *testing.T) {
		r, clk, old, publish, grants := setup(t)
		publish(moved(old))
		deadline := clk.Now().Add(privatenetwork.ReplacementOverlap)
		for range 3 {
			require.NoError(t, converge(t, r))
			require.Equal(t, map[string]time.Time{old.GetTargetDeploymentId(): deadline}, grants(), "certified snapshots without a local caller or target must not revoke the old grant early")
		}
		clk.Tick(privatenetwork.ReplacementOverlap - time.Nanosecond)
		require.NoError(t, converge(t, r))
		require.NotNil(t, grants())
		clk.Tick(time.Nanosecond)
		require.NoError(t, converge(t, r))
		require.Nil(t, grants(), "the grant ends when the overlap ends")
		require.NoError(t, converge(t, r))
		require.Nil(t, grants())
	})

	t.Run("revoking the connection during the overlap removes the grant after two certified snapshots", func(t *testing.T) {
		r, _, old, publish, grants := setup(t)
		publish(moved(old))
		require.NoError(t, converge(t, r))
		require.NotNil(t, grants())
		publish()
		require.NoError(t, converge(t, r))
		require.NotNil(t, grants(), "the first certified absence is not confirmed yet")
		require.NoError(t, converge(t, r))
		require.Nil(t, grants())
	})

	t.Run("revoking the connection removes the grant after two certified snapshots", func(t *testing.T) {
		r, _, _, publish, grants := setup(t)
		publish()
		require.NoError(t, converge(t, r))
		require.NotNil(t, grants())
		require.NoError(t, converge(t, r))
		require.Nil(t, grants())
	})
}

func TestEnsureServiceCreatesMissingNamespace(t *testing.T) {
	connectionSpec := newTestConnection()
	client := fake.NewClientset()
	client.PrependReactor("create", "services", func(action clienttesting.Action) (bool, runtime.Object, error) {
		if _, err := client.Tracker().Get(corev1.SchemeGroupVersion.WithResource("namespaces"), "", action.GetNamespace()); err != nil {
			return true, nil, err
		}
		return false, nil, nil
	})
	r := newTestReconciler(t, Config{Client: client})

	_, err := r.ensureService(t.Context(), connectionSpec, nil)
	require.NoError(t, err)
	_, err = client.CoreV1().Namespaces().Get(t.Context(), connectionSpec.GetK8SNamespace(), metav1.GetOptions{})
	require.NoError(t, err)
}

func TestRunRediscoversWhenEndpointSlicesChange(t *testing.T) {
	client := fake.NewClientset()
	var passes atomic.Int32
	control := &testutil.MockClusterClient{StreamPrivateNetworkStateFunc: snapshotFunc(t, func(context.Context) ([]*ctrlv1.PrivateNetworkConnection, error) {
		passes.Add(1)
		return nil, nil
	})}
	r := newTestReconciler(t, Config{Client: client, Cluster: control, Clock: clock.NewTestClock()})

	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan struct{})
	go func() { defer close(done); r.Run(ctx) }()
	t.Cleanup(func() { cancel(); <-done })
	require.Eventually(t, func() bool { return passes.Load() == 1 }, 10*time.Second, 10*time.Millisecond)

	slice := &discoveryv1.EndpointSlice{
		ObjectMeta:  metav1.ObjectMeta{Name: uid.New(uid.TestPrefix), Namespace: "customer", Labels: discoveryLabels},
		AddressType: discoveryv1.AddressTypeIPv4,
	}
	_, err := client.DiscoveryV1().EndpointSlices(slice.Namespace).Create(ctx, slice, metav1.CreateOptions{})
	require.NoError(t, err)
	require.Eventually(t, func() bool { return passes.Load() >= 2 }, 10*time.Second, 10*time.Millisecond, "an EndpointSlice change starts discovery before the poll interval")
}
