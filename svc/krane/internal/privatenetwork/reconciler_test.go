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
	privatecontract "github.com/unkeyed/unkey/pkg/privatenetwork"
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

func testApp(deployment string) *ctrlv1.PrivateNetworkApp {
	return &ctrlv1.PrivateNetworkApp{
		WorkspaceId: "ws_1", ProjectId: "proj_1", AppId: "app_1", AppSlug: "payments",
		K8SNamespace: "customer-1", DeploymentId: deployment, Port: 8080, EnvironmentId: "env_1",
		CallerDeploymentId: "caller_1", BindingId: "binding_1", BindingName: "payments-api",
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
	control.GetPrivateNetworkStateFunc = func(context.Context, *ctrlv1.GetPrivateNetworkStateRequest) (*ctrlv1.GetPrivateNetworkStateResponse, error) {
		return &ctrlv1.GetPrivateNetworkStateResponse{Apps: []*ctrlv1.PrivateNetworkApp{testApp("dep_a")}}, nil
	}
	r := &Reconciler{client: client, dynamic: testDynamicClient(), cluster: control, clusterKey: &ctrlv1.ClusterKey{}}
	require.NoError(t, r.reconcile(ctx))

	control.GetPrivateNetworkStateFunc = func(context.Context, *ctrlv1.GetPrivateNetworkStateRequest) (*ctrlv1.GetPrivateNetworkStateResponse, error) {
		return nil, fmt.Errorf("unavailable")
	}
	require.Error(t, r.reconcile(ctx))
	bindings, err := client.CoreV1().ConfigMaps("customer-1").List(ctx, metav1.ListOptions{})
	require.NoError(t, err)
	require.Len(t, bindings.Items, 1)
	services, err := client.CoreV1().Services("customer-1").List(ctx, metav1.ListOptions{})
	require.NoError(t, err)
	require.Len(t, services.Items, 1)
}

func TestReconcileBindingPromotionRollback(t *testing.T) {
	ctx := t.Context()
	client := fake.NewClientset(endpointPod(testApp("dep_a"), "a", "10.72.0.11"), endpointPod(testApp("dep_b"), "b", "10.72.0.22"))
	control := &testutil.MockClusterClient{}
	now := time.Date(2026, 9, 21, 12, 0, 0, 0, time.UTC)
	r := &Reconciler{client: client, dynamic: testDynamicClient(), cluster: control, clusterKey: &ctrlv1.ClusterKey{}, now: func() time.Time { return now }}

	for i, deployment := range []string{"dep_a", "dep_b", "dep_a"} {
		app := testApp(deployment)
		control.GetPrivateNetworkStateFunc = func(context.Context, *ctrlv1.GetPrivateNetworkStateRequest) (*ctrlv1.GetPrivateNetworkStateResponse, error) {
			return &ctrlv1.GetPrivateNetworkStateResponse{Apps: []*ctrlv1.PrivateNetworkApp{app}}, nil
		}
		serviceName := discoveryName(deployment, app.GetPort())
		bindingName := resourceName("unkey-pn-binding", "binding_1/caller_1")
		require.NoError(t, r.reconcile(ctx))
		require.NoError(t, r.reconcile(ctx))

		binding, err := client.CoreV1().ConfigMaps(app.GetK8SNamespace()).Get(ctx, bindingName, metav1.GetOptions{})
		require.NoError(t, err)
		require.Equal(t, deployment, binding.Data["deploymentId"])
		require.Equal(t, strconv.Itoa(i+1), binding.Data["revision"])

		services, err := client.CoreV1().Services(app.GetK8SNamespace()).List(ctx, metav1.ListOptions{})
		require.NoError(t, err)
		require.LessOrEqual(t, len(services.Items), 2)

		service, err := client.CoreV1().Services(app.GetK8SNamespace()).Get(ctx, serviceName, metav1.GetOptions{})
		require.NoError(t, err)
		require.Equal(t, corev1.ClusterIPNone, service.Spec.ClusterIP)
		require.False(t, service.Spec.PublishNotReadyAddresses)
		require.Equal(t, "true", service.Annotations[ciliumGlobal])
		require.Equal(t, "true", service.Annotations[ciliumGlobalSlices])
		require.NotContains(t, service.Annotations, privatecontract.RetireAfterAnnotation)
		require.Empty(t, service.Spec.Selector)
		require.Equal(t, deployment, service.Labels[labels.LabelKeyDeploymentID])
	}

	control.GetPrivateNetworkStateFunc = func(context.Context, *ctrlv1.GetPrivateNetworkStateRequest) (*ctrlv1.GetPrivateNetworkStateResponse, error) {
		return &ctrlv1.GetPrivateNetworkStateResponse{}, nil
	}
	require.NoError(t, r.reconcile(ctx))
	services, err := client.CoreV1().Services("customer-1").List(ctx, metav1.ListOptions{})
	require.NoError(t, err)
	require.NotEmpty(t, services.Items)
	for _, service := range services.Items {
		deadline, ok := retirementDeadline(&service)
		require.True(t, ok)
		require.Equal(t, now.Add(privatecontract.ReplacementOverlap), deadline)
	}
	bindings, err := client.CoreV1().ConfigMaps("customer-1").List(ctx, metav1.ListOptions{})
	require.NoError(t, err)
	require.Empty(t, bindings.Items)

	now = now.Add(privatecontract.ReplacementOverlap - time.Nanosecond)
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

func TestServiceRetirementDeadlineSurvivesReconcilerRestart(t *testing.T) {
	ctx := t.Context()
	app := testApp("dep_a")
	pod := endpointPod(app, "a", "10.72.0.84")
	client := fake.NewClientset(pod)
	control := &testutil.MockClusterClient{}
	control.GetPrivateNetworkStateFunc = func(context.Context, *ctrlv1.GetPrivateNetworkStateRequest) (*ctrlv1.GetPrivateNetworkStateResponse, error) {
		return &ctrlv1.GetPrivateNetworkStateResponse{Apps: []*ctrlv1.PrivateNetworkApp{app}}, nil
	}
	started := time.Date(2026, 9, 21, 12, 0, 0, 0, time.UTC)
	dynamic := testDynamicClient()
	r := &Reconciler{client: client, dynamic: dynamic, cluster: control, clusterKey: &ctrlv1.ClusterKey{}, now: func() time.Time { return started }}
	require.NoError(t, r.reconcile(ctx))
	control.GetPrivateNetworkStateFunc = func(context.Context, *ctrlv1.GetPrivateNetworkStateRequest) (*ctrlv1.GetPrivateNetworkStateResponse, error) {
		return &ctrlv1.GetPrivateNetworkStateResponse{}, nil
	}
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
	app := testApp("dep_a")
	service, err := r.ensureService(ctx, app, "service", nil)
	require.NoError(t, err)
	_, err = r.ensureBinding(ctx, app, "binding", service, nil)
	require.NoError(t, err)
	service, err = client.CoreV1().Services("customer-1").Get(ctx, "service", metav1.GetOptions{})
	require.NoError(t, err)
	binding, err := client.CoreV1().ConfigMaps("customer-1").Get(ctx, "binding", metav1.GetOptions{})
	require.NoError(t, err)

	other := testApp("dep_b")
	other.AppId = "app_2"
	_, err = r.ensureService(ctx, other, "service", service)
	require.ErrorContains(t, err, "foreign Service")
	other.BindingId = "binding_2"
	_, err = r.ensureBinding(ctx, other, "binding", service, binding)
	require.ErrorContains(t, err, "foreign ConfigMap")

	service, err = client.CoreV1().Services("customer-1").Get(ctx, "service", metav1.GetOptions{})
	require.NoError(t, err)
	require.Equal(t, "dep_a", service.Labels[labels.LabelKeyDeploymentID])
	binding, err = client.CoreV1().ConfigMaps("customer-1").Get(ctx, "binding", metav1.GetOptions{})
	require.NoError(t, err)
	require.Equal(t, "dep_a", binding.Data["deploymentId"])
}

func TestReconcileRejectsBindingChangedDuringSnapshotFetch(t *testing.T) {
	ctx := t.Context()
	client := fake.NewClientset(endpointPod(testApp("dep_b"), "b", "10.72.0.22"))
	control := &testutil.MockClusterClient{}
	control.GetPrivateNetworkStateFunc = func(context.Context, *ctrlv1.GetPrivateNetworkStateRequest) (*ctrlv1.GetPrivateNetworkStateResponse, error) {
		return &ctrlv1.GetPrivateNetworkStateResponse{Apps: []*ctrlv1.PrivateNetworkApp{testApp("dep_a")}}, nil
	}
	r := &Reconciler{client: client, dynamic: testDynamicClient(), cluster: control, clusterKey: &ctrlv1.ClusterKey{}}
	require.NoError(t, r.reconcile(ctx))
	bindingName := resourceName("unkey-pn-binding", "binding_1/caller_1")

	control.GetPrivateNetworkStateFunc = func(context.Context, *ctrlv1.GetPrivateNetworkStateRequest) (*ctrlv1.GetPrivateNetworkStateResponse, error) {
		binding, err := client.CoreV1().ConfigMaps("customer-1").Get(ctx, bindingName, metav1.GetOptions{})
		require.NoError(t, err)
		binding.ResourceVersion = "2"
		binding.Data["deploymentId"] = "dep_c"
		binding.Data["revision"] = "2"
		_, err = client.CoreV1().ConfigMaps("customer-1").Update(ctx, binding, metav1.UpdateOptions{})
		require.NoError(t, err)
		client.PrependReactor("update", "configmaps", func(action clienttesting.Action) (bool, runtime.Object, error) {
			update := action.(clienttesting.UpdateAction).GetObject().(*corev1.ConfigMap)
			if update.ResourceVersion != "2" {
				return true, nil, apierrors.NewConflict(schema.GroupResource{Resource: "configmaps"}, update.Name, fmt.Errorf("resource version changed"))
			}
			return false, nil, nil
		})
		return &ctrlv1.GetPrivateNetworkStateResponse{Apps: []*ctrlv1.PrivateNetworkApp{testApp("dep_b")}}, nil
	}

	require.True(t, apierrors.IsConflict(r.reconcile(ctx)))
	binding, err := client.CoreV1().ConfigMaps("customer-1").Get(ctx, bindingName, metav1.GetOptions{})
	require.NoError(t, err)
	require.Equal(t, "dep_c", binding.Data["deploymentId"])
	require.Equal(t, "2", binding.Data["revision"])
}

func TestCleanupPreservesForeignResources(t *testing.T) {
	ctx := t.Context()
	foreign := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{
		Name: "foreign", Namespace: "customer-1", Labels: map[string]string{
			labels.LabelKeyManagedBy: "someone-else", labels.LabelKeyComponent: component,
		},
	}}
	managed := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{
		Name: "managed", Namespace: "customer-1", Labels: bindingLabels(testApp("dep_a")),
	}, Data: map[string]string{"revision": "1"}}
	client := fake.NewClientset(foreign, managed)
	control := &testutil.MockClusterClient{}
	control.GetPrivateNetworkStateFunc = func(context.Context, *ctrlv1.GetPrivateNetworkStateRequest) (*ctrlv1.GetPrivateNetworkStateResponse, error) {
		return &ctrlv1.GetPrivateNetworkStateResponse{}, nil
	}
	r := &Reconciler{client: client, dynamic: testDynamicClient(), cluster: control, clusterKey: &ctrlv1.ClusterKey{}}

	require.NoError(t, r.reconcile(ctx))
	_, err := client.CoreV1().ConfigMaps("customer-1").Get(ctx, "foreign", metav1.GetOptions{})
	require.NoError(t, err)
	_, err = client.CoreV1().ConfigMaps("customer-1").Get(ctx, "managed", metav1.GetOptions{})
	require.Error(t, err)
}

func TestValidateSnapshotRejectsEntireInvalidSnapshot(t *testing.T) {
	valid := testApp("dep_a")
	invalid := testApp("dep_b")
	invalid.Port = 0

	apps, err := validateSnapshot([]*ctrlv1.PrivateNetworkApp{valid, invalid})
	require.Error(t, err)
	require.Nil(t, apps)
}

func TestReconcileRejectsInvalidBindingAliases(t *testing.T) {
	for _, slug := range []string{"UPPER", "with_underscore", strings.Repeat("x", 64)} {
		t.Run(slug, func(t *testing.T) {
			ctx := t.Context()
			client := fake.NewClientset()
			control := &testutil.MockClusterClient{}
			incompatible := testApp("dep_b")
			incompatible.AppId = "app_2"
			incompatible.BindingName = slug
			incompatible.BindingId, incompatible.CallerDeploymentId = "binding_2", "caller_2"
			control.GetPrivateNetworkStateFunc = func(context.Context, *ctrlv1.GetPrivateNetworkStateRequest) (*ctrlv1.GetPrivateNetworkStateResponse, error) {
				return &ctrlv1.GetPrivateNetworkStateResponse{Apps: []*ctrlv1.PrivateNetworkApp{incompatible, testApp("dep_a")}}, nil
			}
			r := &Reconciler{client: client, dynamic: testDynamicClient(), cluster: control, clusterKey: &ctrlv1.ClusterKey{}}
			require.ErrorContains(t, r.reconcile(ctx), "invalid binding name")
			bindings, err := client.CoreV1().ConfigMaps("customer-1").List(ctx, metav1.ListOptions{})
			require.NoError(t, err)
			require.Empty(t, bindings.Items)
		})
	}
}

func TestReconcileRejectedAppDoesNotBlockOtherAppsOrPruneState(t *testing.T) {
	for _, tc := range []struct {
		stage, verb, resource string
	}{
		{"namespace", "create", "namespaces"},
		{"service", "create", "services"},
		{"endpoints", "create", "endpointslices"},
		{"binding", "update", "configmaps"},
	} {
		t.Run(tc.stage, func(t *testing.T) {
			ctx := t.Context()
			rejected := testApp("dep_a1")
			replacement := testApp("dep_a2")
			healthy := testApp("dep_b")
			healthy.AppId, healthy.AppSlug, healthy.K8SNamespace = "app_2", "ledger", "customer-2"
			healthy.BindingId, healthy.BindingName, healthy.CallerDeploymentId = "binding_2", "ledger-api", "caller_2"
			obsolete := testApp("dep_c")
			obsolete.AppId, obsolete.AppSlug, obsolete.K8SNamespace = "app_3", "audit", "customer-2"
			obsolete.BindingId, obsolete.BindingName, obsolete.CallerDeploymentId = "binding_3", "audit-api", "caller_3"
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
			snapshot := []*ctrlv1.PrivateNetworkApp{rejected, obsolete}
			control := &testutil.MockClusterClient{}
			control.GetPrivateNetworkStateFunc = func(context.Context, *ctrlv1.GetPrivateNetworkStateRequest) (*ctrlv1.GetPrivateNetworkStateResponse, error) {
				return &ctrlv1.GetPrivateNetworkStateResponse{Apps: snapshot}, nil
			}
			r := &Reconciler{client: client, dynamic: testDynamicClient(), cluster: control, clusterKey: &ctrlv1.ClusterKey{}}
			binding := func(app *ctrlv1.PrivateNetworkApp) (*corev1.ConfigMap, error) {
				return client.CoreV1().ConfigMaps(app.GetK8SNamespace()).Get(ctx, resourceName("unkey-pn-binding", app.GetBindingId()+"/"+app.GetCallerDeploymentId()), metav1.GetOptions{})
			}
			require.NoError(t, r.reconcile(ctx))

			failing = true
			snapshot = []*ctrlv1.PrivateNetworkApp{replacement, healthy}
			reconcileErr := r.reconcile(ctx)
			published, err := binding(healthy)
			require.NoError(t, err, "binding for the app after the rejected app")
			require.Equal(t, "dep_b", published.Data["deploymentId"])
			kept, err := binding(rejected)
			require.NoError(t, err)
			require.Equal(t, "dep_a1", kept.Data["deploymentId"])
			require.Equal(t, "1", kept.Data["revision"])
			_, err = binding(obsolete)
			require.NoError(t, err, "cleanup deleted an omitted app's binding during a partial pass")
			require.Empty(t, retiringDeployments(t, client))
			require.True(t, apierrors.IsForbidden(reconcileErr), "reconcile() error = %v, want Forbidden", reconcileErr)
			require.ErrorContains(t, reconcileErr, rejected.GetK8SNamespace())

			failing = false
			require.NoError(t, r.reconcile(ctx))
			switched, err := binding(rejected)
			require.NoError(t, err)
			require.Equal(t, "dep_a2", switched.Data["deploymentId"])
			require.Equal(t, "2", switched.Data["revision"])
			_, err = binding(obsolete)
			require.True(t, apierrors.IsNotFound(err), "binding(obsolete) error = %v, want NotFound", err)
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
	later := testApp("dep_b")
	later.AppId, later.AppSlug, later.K8SNamespace = "app_2", "ledger", "customer-2"
	later.BindingId, later.BindingName, later.CallerDeploymentId = "binding_2", "ledger-api", "caller_2"
	control := &testutil.MockClusterClient{}
	control.GetPrivateNetworkStateFunc = func(context.Context, *ctrlv1.GetPrivateNetworkStateRequest) (*ctrlv1.GetPrivateNetworkStateResponse, error) {
		return &ctrlv1.GetPrivateNetworkStateResponse{Apps: []*ctrlv1.PrivateNetworkApp{testApp("dep_a"), later}}, nil
	}
	r := &Reconciler{client: client, dynamic: testDynamicClient(), cluster: control, clusterKey: &ctrlv1.ClusterKey{}}

	require.ErrorIs(t, r.reconcile(ctx), context.Canceled)
	services, err := client.CoreV1().Services(later.GetK8SNamespace()).List(t.Context(), metav1.ListOptions{})
	require.NoError(t, err)
	require.Empty(t, services.Items, "reconcile continued with later apps after its context ended")
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
	unresolved := testApp("")
	unresolved.Port = 0
	legacyUnresolved := testApp("")
	legacyUnresolved.BindingId, legacyUnresolved.BindingName = "binding_legacy", "legacy"
	apps, err := validateSnapshot([]*ctrlv1.PrivateNetworkApp{unresolved, legacyUnresolved})
	require.NoError(t, err)
	require.Len(t, apps, 2)

	resolvedWithoutPort := testApp("dep_a")
	resolvedWithoutPort.Port = 0
	_, err = validateSnapshot([]*ctrlv1.PrivateNetworkApp{resolvedWithoutPort})
	require.ErrorContains(t, err, "resolved target port must be positive")
}

func TestReconcileReplicaDiscoveryPublishesCallerDeploymentForPeers(t *testing.T) {
	ctx := t.Context()
	client := fake.NewClientset()
	dynamic := testDynamicClient()
	self := testApp("caller_1")
	self.AppId, self.AppSlug, self.BindingName = "app_caller", "caller", "unkey-replicas"
	self.BindingId = "self-caller_1"
	control := &testutil.MockClusterClient{}
	control.GetPrivateNetworkStateFunc = func(context.Context, *ctrlv1.GetPrivateNetworkStateRequest) (*ctrlv1.GetPrivateNetworkStateResponse, error) {
		return &ctrlv1.GetPrivateNetworkStateResponse{Apps: []*ctrlv1.PrivateNetworkApp{self}}, nil
	}
	r := &Reconciler{client: client, dynamic: dynamic, cluster: control, clusterKey: &ctrlv1.ClusterKey{}}
	require.NoError(t, r.reconcile(ctx))

	bindings, err := client.CoreV1().ConfigMaps("customer-1").List(ctx, metav1.ListOptions{})
	require.NoError(t, err)
	require.Len(t, bindings.Items, 1)
	require.Equal(t, "unkey-replicas", bindings.Items[0].Data["appSlug"])
	require.Equal(t, "caller_1", bindings.Items[0].Data["deploymentId"])
	require.Equal(t, "caller_1", bindings.Items[0].Labels[labels.LabelKeyCallerDeploymentID])
	service, err := client.CoreV1().Services("customer-1").Get(ctx, bindings.Items[0].Data["serviceName"], metav1.GetOptions{})
	require.NoError(t, err)
	require.Equal(t, "caller_1", service.Labels[labels.LabelKeyDeploymentID])
	require.Equal(t, "app_caller", service.Labels[labels.LabelKeyAppID])
	require.Equal(t, "true", service.Annotations[ciliumGlobal])
	require.Equal(t, []flow{{"caller_1", "caller_1"}}, effectiveFlows(t, dynamic, "customer-1"))
}

// TestReconcileEnsuresSharedTargetOncePerPass guarantees that a reconcile pass
// refreshes each target Service's EndpointSlices once, however many caller
// deployments bind to it. The pass runs every poll interval against the
// Kubernetes API server.
func TestReconcileEnsuresSharedTargetOncePerPass(t *testing.T) {
	ctx := t.Context()
	first := testApp("dep_a")
	second := testApp("dep_a")
	second.CallerDeploymentId = "caller_2"
	second.BindingId = "binding_2"
	client := fake.NewClientset(endpointPod(first, "a", "10.72.0.11"))
	control := &testutil.MockClusterClient{}
	control.GetPrivateNetworkStateFunc = func(context.Context, *ctrlv1.GetPrivateNetworkStateRequest) (*ctrlv1.GetPrivateNetworkStateResponse, error) {
		return &ctrlv1.GetPrivateNetworkStateResponse{Apps: []*ctrlv1.PrivateNetworkApp{first, second}}, nil
	}
	r := &Reconciler{client: client, dynamic: testDynamicClient(), cluster: control, clusterKey: &ctrlv1.ClusterKey{}}
	require.NoError(t, r.reconcile(ctx))
	client.ClearActions()

	require.NoError(t, r.reconcile(ctx))
	sliceLists := 0
	for _, action := range client.Actions() {
		if action.GetVerb() == "list" && action.GetResource().Resource == "endpointslices" {
			sliceLists++
		}
	}
	require.Equal(t, 1, sliceLists, "EndpointSlice lists for one shared target Service")

	bindings, err := client.CoreV1().ConfigMaps(first.GetK8SNamespace()).List(ctx, metav1.ListOptions{})
	require.NoError(t, err)
	require.Len(t, bindings.Items, 2)
	for _, binding := range bindings.Items {
		require.Equal(t, discoveryName("dep_a", first.GetPort()), binding.Data["serviceName"])
	}
}
