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
	"github.com/unkeyed/unkey/pkg/deploy/appconnection"
	"github.com/unkeyed/unkey/svc/krane/internal/testutil"
	"github.com/unkeyed/unkey/svc/krane/pkg/labels"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	fakedynamic "k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/kubernetes/fake"
	clienttesting "k8s.io/client-go/testing"
)

func testConnection(deployment string) *ctrlv1.PrivateNetworkConnection {
	return &ctrlv1.PrivateNetworkConnection{
		WorkspaceId: "ws_1", ProjectId: "proj_1", TargetAppId: "app_1", TargetAppSlug: "payments",
		K8SNamespace: "customer-1", TargetDeploymentId: deployment, TargetPort: 8080, TargetEnvironmentId: "env_1",
		CallerDeploymentId: "caller_1", ConnectionId: "connection_1", ConnectionName: "payments-api",
	}
}

func testDynamicClient(objects ...runtime.Object) *fakedynamic.FakeDynamicClient {
	scheme := runtime.NewScheme()
	return fakedynamic.NewSimpleDynamicClientWithCustomListKinds(scheme, map[schema.GroupVersionResource]string{
		policyResource: "CiliumNetworkPolicyList",
	}, objects...)
}

func TestReconcileRPCErrorPreservesPublishedObjects(t *testing.T) {
	ctx := t.Context()
	client := fake.NewClientset()
	control := &testutil.MockClusterClient{}
	control.StreamPrivateNetworkStateFunc = snapshotFunc(t, func(context.Context) ([]*ctrlv1.PrivateNetworkConnection, error) {
		return []*ctrlv1.PrivateNetworkConnection{testConnection("dep_a")}, nil
	})
	r := &Reconciler{client: client, dynamic: testDynamicClient(), cluster: control, clusterKey: &ctrlv1.ClusterKey{}}
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
			initial := testConnection("dep_a")
			desired := testConnection("dep_b")
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
			r := &Reconciler{client: client, dynamic: dynamic, cluster: control, clusterKey: &ctrlv1.ClusterKey{}}
			if verb == "update" {
				require.NoError(t, r.reconcile(ctx))
				current = desired
			}

			failing := true
			dynamic.PrependReactor(verb, policyResource.Resource, func(clienttesting.Action) (bool, runtime.Object, error) {
				if !failing {
					return false, nil, nil
				}
				return true, nil, apierrors.NewForbidden(schema.GroupResource{Resource: policyResource.Resource}, policyName(desired), fmt.Errorf("injected policy failure"))
			})
			require.True(t, apierrors.IsForbidden(r.reconcile(ctx)))
			connection, err := client.CoreV1().ConfigMaps(desired.GetK8SNamespace()).Get(ctx, connectionResourceName(desired), metav1.GetOptions{})
			if verb == "create" {
				require.True(t, apierrors.IsNotFound(err), "DNS ConfigMap was published without its policy")
				require.Empty(t, effectiveFlows(t, dynamic, desired.GetK8SNamespace()))
			} else {
				require.NoError(t, err)
				require.Equal(t, "dep_a", connection.Data["deploymentId"])
				require.Equal(t, []flow{{"caller_1", "dep_a"}}, effectiveFlows(t, dynamic, desired.GetK8SNamespace()))
			}

			failing = false
			require.NoError(t, r.reconcile(ctx))
			connection, err = client.CoreV1().ConfigMaps(desired.GetK8SNamespace()).Get(ctx, connectionResourceName(desired), metav1.GetOptions{})
			require.NoError(t, err)
			require.Equal(t, "dep_b", connection.Data["deploymentId"])
			require.Contains(t, effectiveFlows(t, dynamic, desired.GetK8SNamespace()), flow{"caller_1", "dep_b"})
		})
	}
}

func TestReconcilePolicyDeleteFailureRetainsDNSUntilRetry(t *testing.T) {
	ctx := t.Context()
	connectionSpec := testConnection("dep_a")
	client := fake.NewClientset(endpointPod(connectionSpec, "a", "10.72.0.11"))
	dynamic := testDynamicClient()
	control := &testutil.MockClusterClient{}
	snapshot := []*ctrlv1.PrivateNetworkConnection{connectionSpec}
	control.StreamPrivateNetworkStateFunc = snapshotFunc(t, func(context.Context) ([]*ctrlv1.PrivateNetworkConnection, error) { return snapshot, nil })
	r := &Reconciler{client: client, dynamic: dynamic, cluster: control, clusterKey: &ctrlv1.ClusterKey{}}
	require.NoError(t, r.reconcile(ctx))

	failing := true
	dynamic.PrependReactor("delete", policyResource.Resource, func(clienttesting.Action) (bool, runtime.Object, error) {
		if !failing {
			return false, nil, nil
		}
		return true, nil, apierrors.NewForbidden(schema.GroupResource{Resource: policyResource.Resource}, policyName(connectionSpec), fmt.Errorf("injected policy failure"))
	})
	snapshot = nil
	require.True(t, apierrors.IsForbidden(r.reconcile(ctx)))
	_, err := client.CoreV1().ConfigMaps(connectionSpec.GetK8SNamespace()).Get(ctx, connectionResourceName(connectionSpec), metav1.GetOptions{})
	require.NoError(t, err, "DNS must remain while policy revocation fails")
	require.Equal(t, []flow{{"caller_1", "dep_a"}}, effectiveFlows(t, dynamic, connectionSpec.GetK8SNamespace()))

	failing = false
	require.NoError(t, r.reconcile(ctx))
	_, err = client.CoreV1().ConfigMaps(connectionSpec.GetK8SNamespace()).Get(ctx, connectionResourceName(connectionSpec), metav1.GetOptions{})
	require.True(t, apierrors.IsNotFound(err))
	_, err = dynamic.Resource(policyResource).Namespace(connectionSpec.GetK8SNamespace()).Get(ctx, policyName(connectionSpec), metav1.GetOptions{})
	require.True(t, apierrors.IsNotFound(err))
	require.Empty(t, effectiveFlows(t, dynamic, connectionSpec.GetK8SNamespace()), "revocation must not receive replacement overlap")
	services, err := client.CoreV1().Services(connectionSpec.GetK8SNamespace()).List(ctx, metav1.ListOptions{})
	require.NoError(t, err)
	require.Len(t, services.Items, 1, "Service retention must not retain policy access")
}

func TestReconcileConnectionPromotionRollback(t *testing.T) {
	ctx := t.Context()
	client := fake.NewClientset(endpointPod(testConnection("dep_a"), "a", "10.72.0.11"), endpointPod(testConnection("dep_b"), "b", "10.72.0.22"))
	control := &testutil.MockClusterClient{}
	now := time.Date(2026, 9, 21, 12, 0, 0, 0, time.UTC)
	r := &Reconciler{client: client, dynamic: testDynamicClient(), cluster: control, clusterKey: &ctrlv1.ClusterKey{}, now: func() time.Time { return now }}

	for i, deployment := range []string{"dep_a", "dep_b", "dep_a"} {
		connectionSpec := testConnection(deployment)
		control.StreamPrivateNetworkStateFunc = snapshotFunc(t, func(context.Context) ([]*ctrlv1.PrivateNetworkConnection, error) {
			return []*ctrlv1.PrivateNetworkConnection{connectionSpec}, nil
		})
		serviceName := discoveryName(deployment, connectionSpec.GetTargetPort())
		connectionName := resourceName("unkey-pn-connection", "connection_1/caller_1")
		require.NoError(t, r.reconcile(ctx))
		require.NoError(t, r.reconcile(ctx))

		connection, err := client.CoreV1().ConfigMaps(connectionSpec.GetK8SNamespace()).Get(ctx, connectionName, metav1.GetOptions{})
		require.NoError(t, err)
		require.Equal(t, deployment, connection.Data["deploymentId"])
		require.Equal(t, strconv.Itoa(i+1), connection.Data["revision"])

		services, err := client.CoreV1().Services(connectionSpec.GetK8SNamespace()).List(ctx, metav1.ListOptions{})
		require.NoError(t, err)
		require.LessOrEqual(t, len(services.Items), 2)

		service, err := client.CoreV1().Services(connectionSpec.GetK8SNamespace()).Get(ctx, serviceName, metav1.GetOptions{})
		require.NoError(t, err)
		require.Equal(t, corev1.ClusterIPNone, service.Spec.ClusterIP)
		require.False(t, service.Spec.PublishNotReadyAddresses)
		require.Equal(t, "true", service.Annotations[ciliumGlobal])
		require.Equal(t, "true", service.Annotations[ciliumGlobalSlices])
		require.NotContains(t, service.Annotations, appconnection.RetireAfterAnnotation)
		require.Empty(t, service.Spec.Selector)
		require.Equal(t, deployment, service.Labels[labels.LabelKeyDeploymentID])
	}

	control.StreamPrivateNetworkStateFunc = snapshotFunc(t, func(context.Context) ([]*ctrlv1.PrivateNetworkConnection, error) {
		return nil, nil
	})
	require.NoError(t, r.reconcile(ctx))
	services, err := client.CoreV1().Services("customer-1").List(ctx, metav1.ListOptions{})
	require.NoError(t, err)
	require.NotEmpty(t, services.Items)
	for _, service := range services.Items {
		deadline, ok := retirementDeadline(&service)
		require.True(t, ok)
		require.Equal(t, now.Add(appconnection.ReplacementOverlap), deadline)
	}
	connections, err := client.CoreV1().ConfigMaps("customer-1").List(ctx, metav1.ListOptions{})
	require.NoError(t, err)
	require.Empty(t, connections.Items)

	now = now.Add(appconnection.ReplacementOverlap - time.Nanosecond)
	require.NoError(t, r.reconcile(ctx))
	services, err = client.CoreV1().Services("customer-1").List(ctx, metav1.ListOptions{})
	require.NoError(t, err)
	require.NotEmpty(t, services.Items)
	now = now.Add(time.Nanosecond)
	require.NoError(t, r.reconcile(ctx))
	services, err = client.CoreV1().Services("customer-1").List(ctx, metav1.ListOptions{})
	require.NoError(t, err)
	require.Empty(t, services.Items)

	require.Empty(t, mustListReplicaSets(t, client, "customer-1"))
}

func TestReconcileReplacesLegacyBindingObjectsWithoutChangingDiscovery(t *testing.T) {
	ctx := t.Context()
	connectionSpec := testConnection("dep_a")
	connectionSpec.ConnectionId = "bind_existing"
	client := fake.NewClientset(endpointPod(connectionSpec, "a", "10.72.0.11"))
	control := &testutil.MockClusterClient{}
	control.StreamPrivateNetworkStateFunc = snapshotFunc(t, func(context.Context) ([]*ctrlv1.PrivateNetworkConnection, error) {
		return []*ctrlv1.PrivateNetworkConnection{connectionSpec}, nil
	})
	r := &Reconciler{client: client, dynamic: testDynamicClient(), cluster: control, clusterKey: &ctrlv1.ClusterKey{}}
	legacyName := resourceName("unkey-pn-binding", "bind_existing/caller_1")
	service, err := r.ensureService(ctx, connectionSpec, discoveryName("dep_a", 8080), nil)
	require.NoError(t, err)
	service.UID = "retained-service"
	service, err = client.CoreV1().Services(service.Namespace).Update(ctx, service, metav1.UpdateOptions{})
	require.NoError(t, err)
	legacy, err := r.ensureConnection(ctx, connectionSpec, legacyName, service, nil)
	require.NoError(t, err)
	require.NoError(t, r.ensurePolicy(ctx, connectionSpec, legacyName, legacy))
	legacy.Labels["unkey.com/binding.id"] = legacy.Labels[labels.LabelKeyConnectionID]
	delete(legacy.Labels, labels.LabelKeyConnectionID)
	_, err = client.CoreV1().ConfigMaps(legacy.Namespace).Update(ctx, legacy, metav1.UpdateOptions{})
	require.NoError(t, err)
	policy, err := r.dynamic.Resource(policyResource).Namespace(legacy.Namespace).Get(ctx, legacyName, metav1.GetOptions{})
	require.NoError(t, err)
	policy.SetLabels(legacy.Labels)
	policy.SetAnnotations(map[string]string{"bindings.unkey.com/targets": policy.GetAnnotations()[policyTargetsAnnotation]})
	_, err = r.dynamic.Resource(policyResource).Namespace(legacy.Namespace).Update(ctx, policy, metav1.UpdateOptions{})
	require.NoError(t, err)

	require.NoError(t, r.reconcile(ctx))
	connections, err := client.CoreV1().ConfigMaps(legacy.Namespace).List(ctx, metav1.ListOptions{})
	require.NoError(t, err)
	require.Len(t, connections.Items, 1)
	require.Equal(t, connectionResourceName(connectionSpec), connections.Items[0].Name)
	require.Equal(t, "bind_existing", connections.Items[0].Labels[labels.LabelKeyConnectionID])
	require.NotContains(t, connections.Items[0].Labels, "unkey.com/binding.id")
	require.Equal(t, "payments-api", connections.Items[0].Data["appSlug"])
	require.Equal(t, service.Name, connections.Items[0].Data["serviceName"])
	retained, err := client.CoreV1().Services(service.Namespace).Get(ctx, service.Name, metav1.GetOptions{})
	require.NoError(t, err)
	require.Equal(t, service.UID, retained.UID)
	policies, err := r.dynamic.Resource(policyResource).Namespace(legacy.Namespace).List(ctx, metav1.ListOptions{})
	require.NoError(t, err)
	require.Len(t, policies.Items, 1)
	require.Equal(t, connectionResourceName(connectionSpec), policies.Items[0].GetName())
	require.Equal(t, "bind_existing", policies.Items[0].GetLabels()[labels.LabelKeyConnectionID])
	require.NotContains(t, policies.Items[0].GetAnnotations(), "bindings.unkey.com/targets")
}

func TestServiceRetirementDeadlineSurvivesReconcilerRestart(t *testing.T) {
	ctx := t.Context()
	connectionSpec := testConnection("dep_a")
	pod := endpointPod(connectionSpec, "a", "10.72.0.84")
	client := fake.NewClientset(pod)
	control := &testutil.MockClusterClient{}
	control.StreamPrivateNetworkStateFunc = snapshotFunc(t, func(context.Context) ([]*ctrlv1.PrivateNetworkConnection, error) {
		return []*ctrlv1.PrivateNetworkConnection{connectionSpec}, nil
	})
	started := time.Date(2026, 9, 21, 12, 0, 0, 0, time.UTC)
	dynamic := testDynamicClient()
	r := &Reconciler{client: client, dynamic: dynamic, cluster: control, clusterKey: &ctrlv1.ClusterKey{}, now: func() time.Time { return started }}
	require.NoError(t, r.reconcile(ctx))
	control.StreamPrivateNetworkStateFunc = snapshotFunc(t, func(context.Context) ([]*ctrlv1.PrivateNetworkConnection, error) {
		return nil, nil
	})
	require.NoError(t, r.reconcile(ctx))

	restartedAt := started.Add(10 * time.Minute)
	r = &Reconciler{client: client, dynamic: dynamic, cluster: control, clusterKey: &ctrlv1.ClusterKey{}, now: func() time.Time { return restartedAt }}
	require.NoError(t, r.reconcile(ctx))

	services, err := client.CoreV1().Services("customer-1").List(ctx, metav1.ListOptions{})
	require.NoError(t, err)
	require.Len(t, services.Items, 1)
	deadline, ok := retirementDeadline(&services.Items[0])
	require.True(t, ok)
	require.Equal(t, started.Add(30*time.Minute), deadline)
	require.Equal(t, []string{"10.72.0.84"}, sourceAddresses(t, client, &services.Items[0]))

	pod.Status.Conditions[0].Status = corev1.ConditionFalse
	_, err = client.CoreV1().Pods(pod.Namespace).UpdateStatus(ctx, pod, metav1.UpdateOptions{})
	require.NoError(t, err)
	require.NoError(t, r.reconcileEndpoints(ctx))
	require.Empty(t, sourceAddresses(t, client, &services.Items[0]))

	pod.Status.Conditions[0].Status = corev1.ConditionTrue
	_, err = client.CoreV1().Pods(pod.Namespace).UpdateStatus(ctx, pod, metav1.UpdateOptions{})
	require.NoError(t, err)
	require.NoError(t, r.reconcileEndpoints(ctx))
	require.Equal(t, []string{"10.72.0.84"}, sourceAddresses(t, client, &services.Items[0]))
}

func TestReconcileRefusesOtherAppOwnership(t *testing.T) {
	ctx := t.Context()
	client := fake.NewClientset()
	r := &Reconciler{client: client}
	connectionSpec := testConnection("dep_a")
	service, err := r.ensureService(ctx, connectionSpec, "service", nil)
	require.NoError(t, err)
	_, err = r.ensureConnection(ctx, connectionSpec, "connection", service, nil)
	require.NoError(t, err)
	service, err = client.CoreV1().Services("customer-1").Get(ctx, "service", metav1.GetOptions{})
	require.NoError(t, err)
	connection, err := client.CoreV1().ConfigMaps("customer-1").Get(ctx, "connection", metav1.GetOptions{})
	require.NoError(t, err)

	other := testConnection("dep_b")
	other.TargetAppId = "app_2"
	_, err = r.ensureService(ctx, other, "service", service)
	require.ErrorContains(t, err, "foreign Service")
	other.ConnectionId = "connection_2"
	_, err = r.ensureConnection(ctx, other, "connection", service, connection)
	require.ErrorContains(t, err, "foreign ConfigMap")

	service, err = client.CoreV1().Services("customer-1").Get(ctx, "service", metav1.GetOptions{})
	require.NoError(t, err)
	require.Equal(t, "dep_a", service.Labels[labels.LabelKeyDeploymentID])
	connection, err = client.CoreV1().ConfigMaps("customer-1").Get(ctx, "connection", metav1.GetOptions{})
	require.NoError(t, err)
	require.Equal(t, "dep_a", connection.Data["deploymentId"])
}

func TestReconcileRejectsConnectionChangedDuringSnapshotFetch(t *testing.T) {
	ctx := t.Context()
	client := fake.NewClientset(endpointPod(testConnection("dep_b"), "b", "10.72.0.22"))
	control := &testutil.MockClusterClient{}
	control.StreamPrivateNetworkStateFunc = snapshotFunc(t, func(context.Context) ([]*ctrlv1.PrivateNetworkConnection, error) {
		return []*ctrlv1.PrivateNetworkConnection{testConnection("dep_a")}, nil
	})
	r := &Reconciler{client: client, dynamic: testDynamicClient(), cluster: control, clusterKey: &ctrlv1.ClusterKey{}}
	require.NoError(t, r.reconcile(ctx))
	connectionName := resourceName("unkey-pn-connection", "connection_1/caller_1")

	control.StreamPrivateNetworkStateFunc = snapshotFunc(t, func(context.Context) ([]*ctrlv1.PrivateNetworkConnection, error) {
		connection, err := client.CoreV1().ConfigMaps("customer-1").Get(ctx, connectionName, metav1.GetOptions{})
		require.NoError(t, err)
		connection.ResourceVersion = "2"
		connection.Data["deploymentId"] = "dep_c"
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
		return []*ctrlv1.PrivateNetworkConnection{testConnection("dep_b")}, nil
	})

	require.True(t, apierrors.IsConflict(r.reconcile(ctx)))
	connection, err := client.CoreV1().ConfigMaps("customer-1").Get(ctx, connectionName, metav1.GetOptions{})
	require.NoError(t, err)
	require.Equal(t, "dep_c", connection.Data["deploymentId"])
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
		Name: "managed", Namespace: "customer-1", Labels: connectionLabels(testConnection("dep_a")),
	}, Data: map[string]string{"revision": "1"}}
	client := fake.NewClientset(foreign, managed)
	control := &testutil.MockClusterClient{}
	control.StreamPrivateNetworkStateFunc = snapshotFunc(t, func(context.Context) ([]*ctrlv1.PrivateNetworkConnection, error) {
		return nil, nil
	})
	r := &Reconciler{client: client, dynamic: testDynamicClient(), cluster: control, clusterKey: &ctrlv1.ClusterKey{}}

	require.NoError(t, r.reconcile(ctx))
	_, err := client.CoreV1().ConfigMaps("customer-1").Get(ctx, "foreign", metav1.GetOptions{})
	require.NoError(t, err)
	_, err = client.CoreV1().ConfigMaps("customer-1").Get(ctx, "managed", metav1.GetOptions{})
	require.Error(t, err)
}

func TestValidateSnapshotRejectsOnlyInvalidConnections(t *testing.T) {
	valid := testConnection("dep_a")
	invalid := testConnection("dep_b")
	invalid.TargetPort = 0
	invalid.ConnectionId, invalid.CallerDeploymentId = "connection_2", "caller_2"
	duplicate := testConnection("dep_c")
	duplicate.ConnectionId = "connection_3"
	duplicateCopy := testConnection("dep_c")
	duplicateCopy.ConnectionId = "connection_4"
	duplicate.CallerDeploymentId, duplicateCopy.CallerDeploymentId = "caller_3", "caller_3"
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
			incompatible := testConnection("dep_b")
			incompatible.TargetAppId = "app_2"
			incompatible.ConnectionName = slug
			incompatible.ConnectionId, incompatible.CallerDeploymentId = "connection_2", "caller_2"
			control.StreamPrivateNetworkStateFunc = snapshotFunc(t, func(context.Context) ([]*ctrlv1.PrivateNetworkConnection, error) {
				return []*ctrlv1.PrivateNetworkConnection{incompatible, testConnection("dep_a")}, nil
			})
			r := &Reconciler{client: client, dynamic: testDynamicClient(), cluster: control, clusterKey: &ctrlv1.ClusterKey{}}

			require.ErrorContains(t, r.reconcile(ctx), "invalid connection name")

			connections, err := client.CoreV1().ConfigMaps("customer-1").List(ctx, metav1.ListOptions{})
			require.NoError(t, err)
			require.Len(t, connections.Items, 1, "the valid connection still publishes")
			require.Equal(t, "connection_1", connections.Items[0].Labels[labels.LabelKeyConnectionID])
		})
	}
}

func TestReconcileFailedConnectionKeepsItsObjectsWithoutBlockingOthers(t *testing.T) {
	for _, tc := range []struct {
		stage, verb, resource string
	}{
		{"namespace", "create", "namespaces"},
		{"service", "create", "services"},
		{"endpoints", "create", "endpointslices"},
		{"connection", "update", "configmaps"},
	} {
		t.Run(tc.stage, func(t *testing.T) {
			ctx := t.Context()
			rejected := testConnection("dep_a1")
			replacement := testConnection("dep_a2")
			healthy := testConnection("dep_b")
			healthy.TargetAppId, healthy.TargetAppSlug, healthy.K8SNamespace = "app_2", "ledger", "customer-2"
			healthy.ConnectionId, healthy.ConnectionName, healthy.CallerDeploymentId = "connection_2", "ledger-api", "caller_2"
			obsolete := testConnection("dep_c")
			obsolete.TargetAppId, obsolete.TargetAppSlug, obsolete.K8SNamespace = "app_3", "audit", "customer-2"
			obsolete.ConnectionId, obsolete.ConnectionName, obsolete.CallerDeploymentId = "connection_3", "audit-api", "caller_3"

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
			r := &Reconciler{client: client, dynamic: testDynamicClient(), cluster: control, clusterKey: &ctrlv1.ClusterKey{}}

			connection := func(connectionSpec *ctrlv1.PrivateNetworkConnection) (*corev1.ConfigMap, error) {
				return client.CoreV1().ConfigMaps(connectionSpec.GetK8SNamespace()).Get(ctx, resourceName("unkey-pn-connection", connectionSpec.GetConnectionId()+"/"+connectionSpec.GetCallerDeploymentId()), metav1.GetOptions{})
			}

			require.NoError(t, r.reconcile(ctx))
			if tc.resource == "namespaces" {
				require.NoError(t, client.CoreV1().Namespaces().Delete(ctx, rejected.GetK8SNamespace(), metav1.DeleteOptions{}))
			}

			failing = true
			snapshot = []*ctrlv1.PrivateNetworkConnection{replacement, healthy}
			reconcileErr := r.reconcile(ctx)
			published, err := connection(healthy)
			require.NoError(t, err, "connection after the rejected connection")
			require.Equal(t, "dep_b", published.Data["deploymentId"])
			kept, err := connection(rejected)
			require.NoError(t, err)
			require.Equal(t, "dep_a1", kept.Data["deploymentId"])
			require.Equal(t, "1", kept.Data["revision"])
			_, err = connection(obsolete)
			require.True(t, apierrors.IsNotFound(err), "a connection missing from a complete snapshot is revoked even when another connection fails; connection(obsolete) error = %v", err)
			require.Equal(t, []string{"dep_c"}, retiringDeployments(t, client), "the failed connection keeps the Service it still uses")
			require.True(t, apierrors.IsForbidden(reconcileErr), "reconcile() error = %v, want Forbidden", reconcileErr)
			require.ErrorContains(t, reconcileErr, rejected.GetK8SNamespace())

			failing = false
			require.NoError(t, r.reconcile(ctx))
			switched, err := connection(rejected)
			require.NoError(t, err)
			require.Equal(t, "dep_a2", switched.Data["deploymentId"])
			require.Equal(t, "2", switched.Data["revision"])
			require.ElementsMatch(t, []string{"dep_a1", "dep_c"}, retiringDeployments(t, client))
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
	later := testConnection("dep_b")
	later.TargetAppId, later.TargetAppSlug, later.K8SNamespace = "app_2", "ledger", "customer-2"
	later.ConnectionId, later.ConnectionName, later.CallerDeploymentId = "connection_2", "ledger-api", "caller_2"
	control := &testutil.MockClusterClient{}
	control.StreamPrivateNetworkStateFunc = snapshotFunc(t, func(context.Context) ([]*ctrlv1.PrivateNetworkConnection, error) {
		return []*ctrlv1.PrivateNetworkConnection{testConnection("dep_a"), later}, nil
	})
	r := &Reconciler{client: client, dynamic: testDynamicClient(), cluster: control, clusterKey: &ctrlv1.ClusterKey{}}

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
	unresolved := testConnection("")
	unresolved.TargetPort = 0
	legacyUnresolved := testConnection("")
	legacyUnresolved.ConnectionId, legacyUnresolved.ConnectionName = "connection_legacy", "legacy"
	snapshotConnections, rejected := validateSnapshot([]*ctrlv1.PrivateNetworkConnection{unresolved, legacyUnresolved})
	require.Empty(t, rejected)
	require.Len(t, snapshotConnections, 2)

	resolvedWithoutPort := testConnection("dep_a")
	resolvedWithoutPort.TargetPort = 0
	_, rejected = validateSnapshot([]*ctrlv1.PrivateNetworkConnection{resolvedWithoutPort})
	require.Len(t, rejected, 1)
	require.ErrorContains(t, rejected[0].err, "resolved target port must be positive")
}

func TestReconcileReplicaDiscoveryPublishesCallerDeploymentForPeers(t *testing.T) {
	ctx := t.Context()
	client := fake.NewClientset()
	dynamic := testDynamicClient()
	self := testConnection("caller_1")
	self.TargetAppId, self.TargetAppSlug, self.ConnectionName = "app_caller", "caller", "caller"
	self.ConnectionId = "self-caller_1"
	control := &testutil.MockClusterClient{}
	control.StreamPrivateNetworkStateFunc = snapshotFunc(t, func(context.Context) ([]*ctrlv1.PrivateNetworkConnection, error) {
		return []*ctrlv1.PrivateNetworkConnection{self}, nil
	})
	r := &Reconciler{client: client, dynamic: dynamic, cluster: control, clusterKey: &ctrlv1.ClusterKey{}}
	require.NoError(t, r.reconcile(ctx))

	connections, err := client.CoreV1().ConfigMaps("customer-1").List(ctx, metav1.ListOptions{})
	require.NoError(t, err)
	require.Len(t, connections.Items, 1)
	require.Equal(t, "caller", connections.Items[0].Data["appSlug"])
	require.Equal(t, "caller_1", connections.Items[0].Data["deploymentId"])
	require.Equal(t, "caller_1", connections.Items[0].Labels[labels.LabelKeyCallerDeploymentID])
	service, err := client.CoreV1().Services("customer-1").Get(ctx, connections.Items[0].Data["serviceName"], metav1.GetOptions{})
	require.NoError(t, err)
	require.Equal(t, "caller_1", service.Labels[labels.LabelKeyDeploymentID])
	require.Equal(t, "app_caller", service.Labels[labels.LabelKeyAppID])
	require.Equal(t, "true", service.Annotations[ciliumGlobal])
	require.Equal(t, []flow{{"caller_1", "caller_1"}}, effectiveFlows(t, dynamic, "customer-1"))
}

// TestReconcileEnsuresSharedTargetOncePerPass guarantees that a steady-state
// reconcile pass lists EndpointSlices and namespaces once, however many caller
// deployments and targets it covers, and writes no namespaces. The pass runs
// every poll interval against the Kubernetes API server.
func TestReconcileEnsuresSharedTargetOncePerPass(t *testing.T) {
	ctx := t.Context()
	first := testConnection("dep_a")
	second := testConnection("dep_a")
	second.CallerDeploymentId = "caller_2"
	second.ConnectionId = "connection_2"
	client := fake.NewClientset(endpointPod(first, "a", "10.72.0.11"))
	control := &testutil.MockClusterClient{}
	control.StreamPrivateNetworkStateFunc = snapshotFunc(t, func(context.Context) ([]*ctrlv1.PrivateNetworkConnection, error) {
		return []*ctrlv1.PrivateNetworkConnection{first, second}, nil
	})
	r := &Reconciler{client: client, dynamic: testDynamicClient(), cluster: control, clusterKey: &ctrlv1.ClusterKey{}}
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
		require.Equal(t, discoveryName("dep_a", first.GetTargetPort()), connection.Data["serviceName"])
	}
}

func TestReconcileIncompleteSnapshotPreservesPolicyAndDNSUntilCompleteRevocation(t *testing.T) {
	for _, tc := range []struct {
		name      string
		chunks    []*ctrlv1.PrivateNetworkStateChunk
		streamErr bool
		want      string
	}{
		{
			name: "truncated",
			chunks: []*ctrlv1.PrivateNetworkStateChunk{{Connections: []*ctrlv1.PrivateNetworkConnection{
				testConnection("dep_b"),
			}}},
			want: "ended before it was complete",
		},
		{
			name: "count_mismatch",
			chunks: []*ctrlv1.PrivateNetworkStateChunk{
				{Connections: []*ctrlv1.PrivateNetworkConnection{testConnection("dep_b")}},
				{Complete: true, Total: 2},
			},
			want: "complete chunk reports 2",
		},
		{
			name: "failed_mid_stream", streamErr: true,
			chunks: []*ctrlv1.PrivateNetworkStateChunk{{Connections: []*ctrlv1.PrivateNetworkConnection{
				testConnection("dep_b"),
			}}},
			want: "unavailable",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := t.Context()
			published := testConnection("dep_a")
			client := fake.NewClientset(endpointPod(published, "a", "10.72.0.11"))
			dynamic := testDynamicClient()
			control := &testutil.MockClusterClient{}
			control.StreamPrivateNetworkStateFunc = snapshotFunc(t, func(context.Context) ([]*ctrlv1.PrivateNetworkConnection, error) {
				return []*ctrlv1.PrivateNetworkConnection{published}, nil
			})
			r := &Reconciler{client: client, dynamic: dynamic, cluster: control, clusterKey: &ctrlv1.ClusterKey{}}
			require.NoError(t, r.reconcile(ctx))

			if tc.streamErr {
				control.StreamPrivateNetworkStateFunc = failedChunkStream(t, tc.chunks...)
			} else {
				control.StreamPrivateNetworkStateFunc = chunkStream(t, tc.chunks...)
			}
			require.ErrorContains(t, r.reconcile(ctx), tc.want)
			connections, err := client.CoreV1().ConfigMaps("customer-1").List(ctx, metav1.ListOptions{})
			require.NoError(t, err)
			require.Len(t, connections.Items, 1, "an incomplete snapshot must not revoke published DNS")
			require.Equal(t, "dep_a", connections.Items[0].Data["deploymentId"])
			require.Equal(t, []flow{{"caller_1", "dep_a"}}, effectiveFlows(t, dynamic, "customer-1"), "an incomplete snapshot must not change policy")

			control.StreamPrivateNetworkStateFunc = snapshotFunc(t, func(context.Context) ([]*ctrlv1.PrivateNetworkConnection, error) { return nil, nil })
			require.NoError(t, r.reconcile(ctx))
			connections, err = client.CoreV1().ConfigMaps("customer-1").List(ctx, metav1.ListOptions{})
			require.NoError(t, err)
			require.Empty(t, connections.Items, "a later complete snapshot applies revocation")
			_, err = dynamic.Resource(policyResource).Namespace(published.GetK8SNamespace()).Get(ctx, policyName(published), metav1.GetOptions{})
			require.True(t, apierrors.IsNotFound(err))
			require.Empty(t, effectiveFlows(t, dynamic, "customer-1"))
		})
	}
}
