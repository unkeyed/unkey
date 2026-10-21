package privatenetwork

import (
	"context"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/deploy/privatenetwork"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/krane/internal/testutil"
	"github.com/unkeyed/unkey/svc/krane/pkg/labels"
	"google.golang.org/protobuf/proto"
	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
	clienttesting "k8s.io/client-go/testing"
	"k8s.io/utils/ptr"
)

func TestPublicationRetainsConnectionUntilRemoteDiscoveryIsReady(t *testing.T) {
	ctx := t.Context()
	selected := newTestConnection()
	client := fake.NewClientset(endpointPod(selected, "a", "10.72.0.11"))
	other := newTestConnection()
	other.WorkspaceId, other.ProjectId = selected.WorkspaceId, selected.ProjectId
	other.TargetAppSlug, other.ConnectionName = "metrics", "metrics-api"
	control := &testutil.MockClusterClient{StreamPrivateNetworkStateFunc: snapshotFunc(t, func(context.Context) ([]*ctrlv1.PrivateNetworkConnection, error) {
		return []*ctrlv1.PrivateNetworkConnection{selected, other}, nil
	})}
	dynamic := testDynamicClient()
	r := &Reconciler{clock: clock.NewTestClock(), client: client, dynamic: dynamic, cluster: control}
	require.NoError(t, r.reconcile(ctx))
	original := aliasService(t, client, selected)
	require.NotNil(t, original)
	previousDeployment, otherPrevious := selected.TargetDeploymentId, other.TargetDeploymentId
	require.Equal(t, []string{"10.72.0.11"}, resolvedAddresses(t, client, selected))

	selected.TargetDeploymentId = uid.New(uid.DeploymentPrefix)
	other.TargetDeploymentId = uid.New(uid.DeploymentPrefix)
	_, err := client.CoreV1().Pods("customer-1").Create(ctx, endpointPod(other, "other-b", "10.72.0.33"), metav1.CreateOptions{})
	require.NoError(t, err)
	require.NoError(t, r.reconcile(ctx))
	require.Equal(t, original, aliasService(t, client, selected), "B's empty local slice must not replace the durable A alias")
	require.ElementsMatch(t, []flow{
		{selected.CallerDeploymentId, previousDeployment}, {selected.CallerDeploymentId, selected.TargetDeploymentId},
		{other.CallerDeploymentId, otherPrevious}, {other.CallerDeploymentId, other.TargetDeploymentId},
	}, effectiveFlows(t, dynamic, "customer-1"), "the new target is granted before DNS and the published one stays granted")
	require.Equal(t, other.TargetDeploymentId, resolvedDeployment(t, client, other), "an independent connection with ready endpoints moves")
	require.Equal(t, []string{"10.72.0.33"}, resolvedAddresses(t, client, other))

	r = &Reconciler{client: client, dynamic: dynamic, cluster: control, clock: clock.NewTestClock(time.Now().Add(time.Hour))}
	require.NoError(t, r.reconcile(ctx))
	a, err := client.CoreV1().Services("customer-1").Get(ctx, discoveryName(previousDeployment, selected.GetTargetPort()), metav1.GetOptions{})
	require.NoError(t, err)
	require.NotContains(t, a.Annotations, privatenetwork.RetireAfterAnnotation, "the Service DNS still resolves must not retire")
	require.Contains(t, effectiveFlows(t, dynamic, "customer-1"), flow{selected.CallerDeploymentId, previousDeployment}, "a published target keeps its grant past the overlap")

	b, err := client.CoreV1().Services("customer-1").Get(ctx, discoveryName(selected.TargetDeploymentId, selected.GetTargetPort()), metav1.GetOptions{})
	require.NoError(t, err)
	ready := false
	remote := &discoveryv1.EndpointSlice{
		ObjectMeta: metav1.ObjectMeta{Name: "remote-b", Namespace: b.Namespace,
			Labels:          map[string]string{discoveryv1.LabelServiceName: b.Name, discoveryv1.LabelManagedBy: meshSliceManager},
			OwnerReferences: []metav1.OwnerReference{*metav1.NewControllerRef(b, corev1.SchemeGroupVersion.WithKind("Service"))},
		},
		AddressType: discoveryv1.AddressTypeIPv4,
		Endpoints:   []discoveryv1.Endpoint{{Addresses: []string{"10.73.0.22"}, Conditions: discoveryv1.EndpointConditions{Ready: &ready}}},
	}
	remote, err = client.DiscoveryV1().EndpointSlices(b.Namespace).Create(ctx, remote, metav1.CreateOptions{})
	require.NoError(t, err)
	require.NoError(t, r.reconcile(ctx))
	require.Equal(t, original, aliasService(t, client, selected))

	ready = true
	remote.Endpoints[0].Conditions.Ready = &ready
	for _, tc := range []struct {
		name   string
		change func(*discoveryv1.EndpointSlice)
	}{
		{"wrong-owner", func(s *discoveryv1.EndpointSlice) { s.OwnerReferences[0].UID = types.UID(uid.New(uid.TestPrefix)) }},
		{"terminating", func(s *discoveryv1.EndpointSlice) { s.Endpoints[0].Conditions.Terminating = &ready }},
		{"public-address", func(s *discoveryv1.EndpointSlice) { s.Endpoints[0].Addresses = []string{"192.0.2.22"} }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			invalid := remote.DeepCopy()
			tc.change(invalid)
			_, err := client.DiscoveryV1().EndpointSlices(b.Namespace).Update(ctx, invalid, metav1.UpdateOptions{})
			require.NoError(t, err)
			require.NoError(t, r.reconcile(ctx))
			require.Equal(t, previousDeployment, resolvedDeployment(t, client, selected))
		})
	}

	_, err = client.DiscoveryV1().EndpointSlices(b.Namespace).Update(ctx, remote, metav1.UpdateOptions{})
	require.NoError(t, err)
	require.NoError(t, r.reconcile(ctx))
	require.Equal(t, selected.TargetDeploymentId, resolvedDeployment(t, client, selected))
	require.Equal(t, []string{"10.73.0.22"}, resolvedAddresses(t, client, selected))

	require.NoError(t, r.reconcile(ctx))
	a, err = client.CoreV1().Services("customer-1").Get(ctx, a.Name, metav1.GetOptions{})
	require.NoError(t, err)
	require.Contains(t, a.Annotations, privatenetwork.RetireAfterAnnotation)
	require.Contains(t, effectiveFlows(t, dynamic, "customer-1"), flow{selected.CallerDeploymentId, previousDeployment}, "the replaced target keeps its grant during the overlap")
}

func TestPublicationIgnoresReadinessOfReplacedTargetServiceIncarnation(t *testing.T) {
	ctx := t.Context()
	connectionSpec := newTestConnection()
	client := fake.NewClientset(endpointPod(connectionSpec, "a", "10.72.0.11"))
	assignServiceUIDs(client)
	dynamic := testDynamicClient()
	snapshot := connectionSpec
	control := &testutil.MockClusterClient{StreamPrivateNetworkStateFunc: snapshotFunc(t, func(context.Context) ([]*ctrlv1.PrivateNetworkConnection, error) {
		return []*ctrlv1.PrivateNetworkConnection{snapshot}, nil
	})}
	r := &Reconciler{clock: clock.NewTestClock(), client: client, dynamic: dynamic, cluster: control}
	require.NoError(t, r.reconcile(ctx))

	replacement := proto.Clone(connectionSpec).(*ctrlv1.PrivateNetworkConnection)
	replacement.TargetDeploymentId = uid.New(uid.DeploymentPrefix)
	snapshot = replacement
	require.NoError(t, r.reconcile(ctx))
	stale, err := client.CoreV1().Services("customer-1").Get(ctx, discoveryName(replacement.TargetDeploymentId, replacement.GetTargetPort()), metav1.GetOptions{})
	require.NoError(t, err)
	_, err = client.DiscoveryV1().EndpointSlices("customer-1").Create(ctx, readySlice(stale, "imported-old-incarnation", meshSliceManager), metav1.CreateOptions{})
	require.NoError(t, err)
	require.NoError(t, client.CoreV1().Services("customer-1").Delete(ctx, stale.Name, metav1.DeleteOptions{}))

	require.NoError(t, r.reconcile(ctx))
	current, err := client.CoreV1().Services("customer-1").Get(ctx, stale.Name, metav1.GetOptions{})
	require.NoError(t, err)
	require.NotEqual(t, stale.UID, current.UID)
	require.Equal(t, connectionSpec.TargetDeploymentId, resolvedDeployment(t, client, connectionSpec), "a ready slice of the old incarnation must not move DNS")
	require.ElementsMatch(t, []flow{{connectionSpec.CallerDeploymentId, connectionSpec.TargetDeploymentId}, {connectionSpec.CallerDeploymentId, replacement.TargetDeploymentId}},
		effectiveFlows(t, dynamic, "customer-1"))

	_, err = client.CoreV1().Pods("customer-1").Create(ctx, endpointPod(replacement, "b", "10.72.0.22"), metav1.CreateOptions{})
	require.NoError(t, err)
	require.NoError(t, r.reconcileEndpoints(ctx))
	require.NoError(t, r.reconcile(ctx))
	require.Equal(t, replacement.TargetDeploymentId, resolvedDeployment(t, client, connectionSpec))
	require.Equal(t, []string{"10.72.0.22"}, resolvedAddresses(t, client, connectionSpec))
}

func TestPublicationRejectsAnnotationThatDisagreesWithTargetService(t *testing.T) {
	for name, tamper := range map[string]func(alias *corev1.Service, other *corev1.Service){
		"annotation": func(alias *corev1.Service, _ *corev1.Service) {
			alias.Annotations[privatenetwork.TargetDeploymentAnnotation] = uid.New(uid.DeploymentPrefix)
		},
		"external-name": func(alias *corev1.Service, other *corev1.Service) {
			alias.Spec.ExternalName = privatenetwork.ServiceHost(other.Name, other.Namespace)
		},
		"connection-service-target": func(alias *corev1.Service, _ *corev1.Service) {
			alias.Spec.ExternalName = privatenetwork.ServiceHost(alias.Name, alias.Namespace)
		},
	} {
		t.Run(name, func(t *testing.T) {
			ctx := t.Context()
			connectionSpec := newTestConnection()
			otherApp := newTestConnection()
			otherApp.WorkspaceId, otherApp.ProjectId = connectionSpec.WorkspaceId, connectionSpec.ProjectId
			otherApp.ConnectionName = "other"
			client := fake.NewClientset(endpointPod(connectionSpec, "a", "10.72.0.11"))
			dynamic := testDynamicClient()
			snapshot := []*ctrlv1.PrivateNetworkConnection{connectionSpec, otherApp}
			control := &testutil.MockClusterClient{StreamPrivateNetworkStateFunc: snapshotFunc(t, func(context.Context) ([]*ctrlv1.PrivateNetworkConnection, error) {
				return snapshot, nil
			})}
			r := &Reconciler{clock: clock.NewTestClock(), client: client, dynamic: dynamic, cluster: control}
			require.NoError(t, r.reconcile(ctx))

			alias := aliasService(t, client, connectionSpec)
			otherTarget, err := client.CoreV1().Services("customer-1").Get(ctx, discoveryName(otherApp.TargetDeploymentId, otherApp.GetTargetPort()), metav1.GetOptions{})
			require.NoError(t, err)
			tamper(alias, otherTarget)
			tampered, err := client.CoreV1().Services("customer-1").Update(ctx, alias, metav1.UpdateOptions{})
			require.NoError(t, err)

			replacement := proto.Clone(connectionSpec).(*ctrlv1.PrivateNetworkConnection)
			replacement.TargetDeploymentId = uid.New(uid.DeploymentPrefix)
			_, err = client.CoreV1().Pods("customer-1").Create(ctx, endpointPod(replacement, "b", "10.72.0.22"), metav1.CreateOptions{})
			require.NoError(t, err)
			snapshot = []*ctrlv1.PrivateNetworkConnection{replacement, otherApp}
			require.ErrorContains(t, r.reconcile(ctx), "connection Service")
			require.Equal(t, tampered, aliasService(t, client, connectionSpec), "a rejected alias stays as published")
			require.NotContains(t, effectiveFlows(t, dynamic, "customer-1"), flow{connectionSpec.CallerDeploymentId, replacement.TargetDeploymentId}, "a rejected alias does not change its grant")
			require.Contains(t, effectiveFlows(t, dynamic, "customer-1"), flow{connectionSpec.CallerDeploymentId, connectionSpec.TargetDeploymentId})
		})
	}
}

func TestPublicationRenamedAliasMovesDNSAfterNewNameIsPublished(t *testing.T) {
	ctx := t.Context()
	connectionSpec := newTestConnection()
	client := fake.NewClientset(endpointPod(connectionSpec, "a", "10.72.0.11"))
	dynamic := testDynamicClient()
	snapshot := connectionSpec
	control := &testutil.MockClusterClient{StreamPrivateNetworkStateFunc: snapshotFunc(t, func(context.Context) ([]*ctrlv1.PrivateNetworkConnection, error) {
		return []*ctrlv1.PrivateNetworkConnection{snapshot}, nil
	})}
	r := &Reconciler{clock: clock.NewTestClock(), client: client, dynamic: dynamic, cluster: control}
	require.NoError(t, r.reconcile(ctx))

	renamed := proto.Clone(connectionSpec).(*ctrlv1.PrivateNetworkConnection)
	renamed.ConnectionName = "billing"
	snapshot = renamed
	failing := true
	client.PrependReactor("create", "services", func(clienttesting.Action) (bool, runtime.Object, error) {
		if failing {
			return true, nil, apierrors.NewServiceUnavailable("injected")
		}
		return false, nil, nil
	})
	require.Error(t, r.reconcile(ctx))
	require.Nil(t, aliasService(t, client, renamed))
	require.Equal(t, connectionSpec.TargetDeploymentId, resolvedDeployment(t, client, connectionSpec), "the old name stays until the new name is published")

	failing = false
	require.NoError(t, r.reconcile(ctx))
	require.Nil(t, aliasService(t, client, connectionSpec))
	require.Equal(t, connectionSpec.TargetDeploymentId, resolvedDeployment(t, client, renamed))
	require.Equal(t, []flow{{connectionSpec.CallerDeploymentId, connectionSpec.TargetDeploymentId}}, effectiveFlows(t, dynamic, "customer-1"))
}

func TestPublicationUnresolvedTargetWithdrawsDNSAndGrant(t *testing.T) {
	ctx := t.Context()
	connectionSpec := newTestConnection()
	client := fake.NewClientset(endpointPod(connectionSpec, "a", "10.72.0.11"))
	dynamic := testDynamicClient()
	snapshot := connectionSpec
	control := &testutil.MockClusterClient{StreamPrivateNetworkStateFunc: snapshotFunc(t, func(context.Context) ([]*ctrlv1.PrivateNetworkConnection, error) {
		return []*ctrlv1.PrivateNetworkConnection{snapshot}, nil
	})}
	r := &Reconciler{clock: clock.NewTestClock(), client: client, dynamic: dynamic, cluster: control}
	require.NoError(t, r.reconcile(ctx))

	unresolved := proto.Clone(connectionSpec).(*ctrlv1.PrivateNetworkConnection)
	unresolved.TargetDeploymentId, unresolved.TargetPort = "", 0
	snapshot = unresolved
	require.NoError(t, r.reconcile(ctx))
	require.Nil(t, aliasService(t, client, connectionSpec))
	require.Empty(t, effectiveFlows(t, dynamic, "customer-1"))
	require.NoError(t, r.reconcile(ctx))
	require.Equal(t, []string{connectionSpec.TargetDeploymentId}, retiringDeployments(t, client), "the shared Service retires once no alias resolves it")
}

func aliasName(connectionSpec *ctrlv1.PrivateNetworkConnection) string {
	return privatenetwork.ConnectionServiceName(connectionSpec.GetCallerDeploymentId(), connectionSpec.GetConnectionName())
}

func aliasService(t *testing.T, client kubernetes.Interface, connectionSpec *ctrlv1.PrivateNetworkConnection) *corev1.Service {
	t.Helper()
	service, err := client.CoreV1().Services(connectionSpec.GetK8SNamespace()).Get(t.Context(), aliasName(connectionSpec), metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		return nil
	}
	require.NoError(t, err)
	return service
}

func resolvedTarget(t *testing.T, client kubernetes.Interface, connectionSpec *ctrlv1.PrivateNetworkConnection) *corev1.Service {
	t.Helper()
	alias := aliasService(t, client, connectionSpec)
	if alias == nil {
		return nil
	}
	require.Equal(t, corev1.ServiceTypeExternalName, alias.Spec.Type)
	require.Empty(t, alias.Spec.Ports)
	require.Empty(t, alias.Spec.Selector)
	name, _, _ := strings.Cut(alias.Spec.ExternalName, ".")
	require.Equal(t, privatenetwork.ServiceHost(name, alias.Namespace), alias.Spec.ExternalName)
	target, err := client.CoreV1().Services(alias.Namespace).Get(t.Context(), name, metav1.GetOptions{})
	require.NoError(t, err)
	require.Equal(t, corev1.ClusterIPNone, target.Spec.ClusterIP)
	require.Equal(t, "true", target.Annotations[ciliumGlobal])
	require.Equal(t, target.Labels[labels.LabelKeyDeploymentID], alias.Annotations[privatenetwork.TargetDeploymentAnnotation])
	return target
}

func resolvedDeployment(t *testing.T, client kubernetes.Interface, connectionSpec *ctrlv1.PrivateNetworkConnection) string {
	t.Helper()
	target := resolvedTarget(t, client, connectionSpec)
	if target == nil {
		return ""
	}
	return target.Labels[labels.LabelKeyDeploymentID]
}

func resolvedAddresses(t *testing.T, client kubernetes.Interface, connectionSpec *ctrlv1.PrivateNetworkConnection) []string {
	t.Helper()
	target := resolvedTarget(t, client, connectionSpec)
	if target == nil {
		return nil
	}
	slices, err := client.DiscoveryV1().EndpointSlices(target.Namespace).List(t.Context(), metav1.ListOptions{LabelSelector: discoveryv1.LabelServiceName + "=" + target.Name})
	require.NoError(t, err)
	var addresses []string
	for i := range slices.Items {
		for _, address := range privatenetwork.AppendReadyAddresses(nil, target, &slices.Items[i]) {
			addresses = append(addresses, address.String())
		}
	}
	return addresses
}

func connectionServices(t *testing.T, client kubernetes.Interface, namespace string) []corev1.Service {
	t.Helper()
	list, err := client.CoreV1().Services(namespace).List(t.Context(), metav1.ListOptions{})
	require.NoError(t, err)
	var connections []corev1.Service
	for _, service := range list.Items {
		if service.Spec.Type == corev1.ServiceTypeExternalName {
			connections = append(connections, service)
		}
	}
	return connections
}

func assignServiceUIDs(client *fake.Clientset) {
	client.PrependReactor("create", "services", func(action clienttesting.Action) (bool, runtime.Object, error) {
		service := action.(clienttesting.CreateAction).GetObject().(*corev1.Service)
		if service.UID == "" {
			service.UID = types.UID(uid.New(uid.TestPrefix))
		}
		return false, nil, nil
	})
}

func TestRegionalAliasesResolveOneRegionAndFollowTheGlobalAlias(t *testing.T) {
	ctx := t.Context()
	connectionSpec := newTestConnection()
	client := fake.NewClientset(endpointPod(connectionSpec, "a", "10.72.0.11"))
	dynamic := testDynamicClient()
	serve := func(id string, certified bool, topology *ctrlv1.PrivateNetworkTopology, spec *ctrlv1.PrivateNetworkConnection) func(context.Context, *ctrlv1.StreamPrivateNetworkStateRequest) (*connect.ServerStreamForClient[ctrlv1.PrivateNetworkStateChunk], error) {
		chunks := identified(id,
			&ctrlv1.PrivateNetworkStateChunk{Connections: []*ctrlv1.PrivateNetworkConnection{spec}},
			&ctrlv1.PrivateNetworkStateChunk{Complete: true, Total: 1, Topology: topology},
		)
		chunks[1].Certified = certified
		return chunkStream(t, chunks...)
	}
	control := &testutil.MockClusterClient{StreamPrivateNetworkStateFunc: serve("one", true, testTopology(), connectionSpec)}
	r := &Reconciler{clock: clock.NewTestClock(), client: client, dynamic: dynamic, cluster: control, clusterKey: &ctrlv1.ClusterKey{Platform: "aws", Region: "us-east-1", CellId: "aws-us-east-1"}}
	require.NoError(t, r.reconcile(ctx))

	shared := resolvedTarget(t, client, connectionSpec)
	remote := readySlice(shared, "imported-eu", meshSliceManager)
	remote.Endpoints[0].Addresses, remote.Endpoints[0].Hostname = []string{"10.80.0.5"}, ptr.To("eu-west-1")
	_, err := client.DiscoveryV1().EndpointSlices("customer-1").Create(ctx, remote, metav1.CreateOptions{})
	require.NoError(t, err)
	require.ElementsMatch(t, []string{"10.72.0.11", "10.80.0.5"}, resolvedAddresses(t, client, connectionSpec))
	require.Equal(t, []string{"10.72.0.11"}, regionalAddresses(t, client, connectionSpec, "us-east-1"), "local source endpoints carry this cluster's region")
	require.Equal(t, []string{"10.80.0.5"}, regionalAddresses(t, client, connectionSpec, "eu-west-1"))

	replacement := proto.Clone(connectionSpec).(*ctrlv1.PrivateNetworkConnection)
	replacement.TargetDeploymentId = uid.New(uid.DeploymentPrefix)
	control.StreamPrivateNetworkStateFunc = serve("two", true, testTopology(), replacement)
	require.NoError(t, r.reconcile(ctx))
	require.Equal(t, []string{"10.80.0.5"}, regionalAddresses(t, client, connectionSpec, "eu-west-1"), "regional aliases wait with the global alias")
	require.ElementsMatch(t, []flow{{connectionSpec.CallerDeploymentId, connectionSpec.TargetDeploymentId}, {connectionSpec.CallerDeploymentId, replacement.TargetDeploymentId}},
		effectiveFlows(t, dynamic, "customer-1"))

	_, err = client.CoreV1().Pods("customer-1").Create(ctx, endpointPod(replacement, "b", "10.72.0.22"), metav1.CreateOptions{})
	require.NoError(t, err)
	require.NoError(t, r.reconcileEndpoints(ctx))
	require.NoError(t, r.reconcile(ctx))
	require.Equal(t, []string{"10.72.0.22"}, resolvedAddresses(t, client, connectionSpec))
	require.Equal(t, []string{"10.72.0.22"}, regionalAddresses(t, client, connectionSpec, "us-east-1"))
	require.Empty(t, regionalAddresses(t, client, connectionSpec, "eu-west-1"), "a region without replicas stays published and answers no addresses")

	local := &ctrlv1.PrivateNetworkTopology{Version: 1, Clusters: []*ctrlv1.ClusterKey{{Platform: "aws", Region: "us-east-1", CellId: "aws-us-east-1"}}}
	control.StreamPrivateNetworkStateFunc = serve("three", false, local, replacement)
	require.NoError(t, r.reconcile(ctx))
	require.NotNil(t, regionalService(t, client, connectionSpec, "eu-west-1"), "an uncertified topology does not remove a region")
	control.StreamPrivateNetworkStateFunc = serve("four", true, local, replacement)
	require.NoError(t, r.reconcile(ctx))
	require.Nil(t, regionalService(t, client, connectionSpec, "eu-west-1"))
	require.NotNil(t, regionalService(t, client, connectionSpec, "us-east-1"))

	unresolved := proto.Clone(replacement).(*ctrlv1.PrivateNetworkConnection)
	unresolved.TargetDeploymentId, unresolved.TargetPort = "", 0
	control.StreamPrivateNetworkStateFunc = serve("five", false, local, unresolved)
	require.NoError(t, r.reconcile(ctx))
	require.Empty(t, connectionServices(t, client, "customer-1"), "an unresolved target withdraws every alias")
}

func regionalService(t *testing.T, client kubernetes.Interface, connectionSpec *ctrlv1.PrivateNetworkConnection, region string) *corev1.Service {
	t.Helper()
	service, err := client.CoreV1().Services(connectionSpec.GetK8SNamespace()).Get(t.Context(),
		privatenetwork.RegionalConnectionServiceName(connectionSpec.GetCallerDeploymentId(), connectionSpec.GetConnectionName(), region), metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		return nil
	}
	require.NoError(t, err)
	return service
}

func regionalAddresses(t *testing.T, client kubernetes.Interface, connectionSpec *ctrlv1.PrivateNetworkConnection, region string) []string {
	t.Helper()
	alias := regionalService(t, client, connectionSpec, region)
	require.NotNil(t, alias)
	require.Equal(t, corev1.ServiceTypeExternalName, alias.Spec.Type)
	require.Equal(t, region, alias.Annotations[privatenetwork.RegionAnnotation])
	host, found := strings.CutPrefix(alias.Spec.ExternalName, region+".")
	require.True(t, found)
	name, _, _ := strings.Cut(host, ".")
	require.Equal(t, privatenetwork.ServiceHost(name, alias.Namespace), host)
	target, err := client.CoreV1().Services(alias.Namespace).Get(t.Context(), name, metav1.GetOptions{})
	require.NoError(t, err)
	require.Equal(t, target.Labels[labels.LabelKeyDeploymentID], alias.Annotations[privatenetwork.TargetDeploymentAnnotation])
	require.Equal(t, resolvedTarget(t, client, connectionSpec).Name, target.Name, "a regional alias resolves the same deployment as the global alias")

	slices, err := client.DiscoveryV1().EndpointSlices(target.Namespace).List(t.Context(), metav1.ListOptions{LabelSelector: discoveryv1.LabelServiceName + "=" + target.Name})
	require.NoError(t, err)
	addresses := []string{}
	for i := range slices.Items {
		slice := slices.Items[i].DeepCopy()
		slice.Endpoints = nil
		for _, endpoint := range slices.Items[i].Endpoints {
			if endpoint.Hostname != nil && *endpoint.Hostname == region {
				slice.Endpoints = append(slice.Endpoints, endpoint)
			}
		}
		for _, address := range privatenetwork.AppendReadyAddresses(nil, target, slice) {
			addresses = append(addresses, address.String())
		}
	}
	return addresses
}
