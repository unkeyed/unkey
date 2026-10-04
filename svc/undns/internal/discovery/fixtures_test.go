package discovery

import (
	"context"
	"fmt"
	"maps"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	promclient "github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/deploy/privatenetwork"
	"github.com/unkeyed/unkey/pkg/prometheus"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/krane/pkg/labels"
	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/features"
	featuretesting "k8s.io/client-go/features/testing"
	"k8s.io/client-go/kubernetes/fake"
)

var testRegistry *promclient.Registry

func TestMain(m *testing.M) {
	testRegistry = prometheus.NewServiceRegistry()
	os.Exit(m.Run())
}

func requireReason(t *testing.T, err error, want Reason, msgAndArgs ...any) {
	t.Helper()
	require.Error(t, err, msgAndArgs...)
	require.Equal(t, want, ReasonOf(err), msgAndArgs...)
}

func gaugeValues(t testing.TB, name string) map[string]float64 {
	t.Helper()
	families, err := testRegistry.Gather()
	if err != nil {
		t.Errorf("gather metrics: %v", err)
		return nil
	}
	values := map[string]float64{}
	for _, family := range families {
		if family.GetName() != name {
			continue
		}
		for _, metric := range family.GetMetric() {
			pairs := make([]string, 0, len(metric.GetLabel()))
			for _, label := range metric.GetLabel() {
				pairs = append(pairs, label.GetName()+"="+label.GetValue())
			}
			values[strings.Join(pairs, ",")] = metric.GetGauge().GetValue()
		}
	}
	return values
}

func connectionCounts(t testing.TB) map[string]map[Reason]int {
	t.Helper()
	counts := map[string]map[Reason]int{kindConnection: {}, kindReplica: {}}
	for _, kind := range []string{kindConnection, kindReplica} {
		for _, state := range connectionStates {
			if value := gaugeValues(t, "unkey_dns_connections")["kind="+kind+",state="+string(state)]; value != 0 {
				counts[kind][state] = int(value)
			}
		}
	}
	return counts
}

func startCatalog(t *testing.T) *Catalog {
	t.Helper()
	featuretesting.SetFeatureDuringTest(t, features.WatchListClient, false)
	c, err := New(fake.NewSimpleClientset(discoveryObjects(t)...), 30*time.Second, clock.New())
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(t.Context())
	var running sync.WaitGroup
	running.Go(func() { require.NoError(t, c.Run(ctx)) })
	t.Cleanup(func() {
		cancel()
		running.Wait()
	})
	require.Eventually(t, c.Ready, 5*time.Second, 10*time.Millisecond)
	return c
}

func seededCatalog(t *testing.T) *Catalog {
	t.Helper()
	c := catalogForTest()
	for _, object := range discoveryObjects(t) {
		switch typed := object.(type) {
		case *corev1.Pod:
			require.NoError(t, c.pods.GetStore().Add(typed))
		case *corev1.ConfigMap:
			require.NoError(t, c.connections.GetStore().Add(typed))
		case *corev1.Service:
			require.NoError(t, c.services.GetStore().Add(typed))
		case *discoveryv1.EndpointSlice:
			require.NoError(t, c.slices.GetStore().Add(typed))
		default:
			t.Fatalf("unexpected discovery object %T", object)
		}
	}
	return c
}

func storedConnection(t *testing.T, c *Catalog) *corev1.ConfigMap {
	t.Helper()
	object, exists, err := c.connections.GetStore().GetByKey("default/connection")
	require.NoError(t, err)
	require.True(t, exists)
	return object.(*corev1.ConfigMap)
}

func updateStoredConnection(t *testing.T, c *Catalog, data map[string]string) {
	t.Helper()
	connection := storedConnection(t, c).DeepCopy()
	for key, value := range data {
		connection.Data[key] = value
	}
	require.NoError(t, c.connections.GetStore().Update(connection))
}

func storedService(t *testing.T, c *Catalog) *corev1.Service {
	t.Helper()
	object, exists, err := c.services.GetStore().GetByKey("default/service-a")
	require.NoError(t, err)
	require.True(t, exists)
	return object.(*corev1.Service)
}

func discoveryObjects(t testing.TB) []runtime.Object {
	t.Helper()
	identity := testCaller()
	appID, deploymentID := uid.New(uid.AppPrefix), uid.New(uid.DeploymentPrefix)
	pod := callerPod("127.0.0.1", "production", identity)

	connection := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: "connection", Namespace: "default", UID: types.UID(uid.New(uid.TestPrefix)), Labels: map[string]string{
			labels.LabelKeyManagedBy: "krane", labels.LabelKeyComponent: privatenetwork.DiscoveryComponent,
			labels.LabelKeyWorkspaceID: identity.Workspace, labels.LabelKeyProjectID: identity.Project,
			labels.LabelKeyAppID: appID, privatenetwork.EnvironmentKindLabel: "production",
			labels.LabelKeyCallerDeploymentID: identity.Deployment, labels.LabelKeyConnectionID: uid.New(uid.ConnectionPrefix),
		}},
		Data: connectionData(t, "payments", deploymentID, "service-a", "1"),
	}

	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Name: "service-a", Namespace: "default", UID: types.UID(uid.New(uid.TestPrefix)), Labels: map[string]string{
			labels.LabelKeyManagedBy: "krane", labels.LabelKeyComponent: privatenetwork.DiscoveryComponent, labels.LabelKeyWorkspaceID: identity.Workspace,
			labels.LabelKeyProjectID: identity.Project, labels.LabelKeyAppID: appID,
			labels.LabelKeyDeploymentID: deploymentID,
		}},
		Spec: corev1.ServiceSpec{ClusterIP: corev1.ClusterIPNone},
	}

	ready := true
	controller := true
	endpoints := make([]discoveryv1.Endpoint, 0, 40)
	for i := 1; i <= 40; i++ {
		endpoints = append(endpoints, discoveryv1.Endpoint{Addresses: []string{fmt.Sprintf("10.0.0.%d", i)}, Conditions: discoveryv1.EndpointConditions{Ready: &ready}})
	}

	slice := &discoveryv1.EndpointSlice{
		ObjectMeta: metav1.ObjectMeta{Name: "imported", Namespace: "default",
			Labels:          ciliumImportedSliceLabels(service),
			OwnerReferences: []metav1.OwnerReference{{APIVersion: "v1", Kind: "Service", Name: service.Name, UID: service.UID, Controller: &controller}},
		},
		AddressType: discoveryv1.AddressTypeIPv4,
		Endpoints:   endpoints,
	}

	return []runtime.Object{pod, connection, service, slice}
}

func ciliumImportedSliceLabels(service *corev1.Service) map[string]string {
	sliceLabels := maps.Clone(service.Labels)
	sliceLabels[discoveryv1.LabelServiceName] = service.Name
	sliceLabels[discoveryv1.LabelManagedBy] = "endpointslice-mesh-controller.cilium.io"
	sliceLabels["multicluster.kubernetes.io/source-cluster"] = "remote"
	return sliceLabels
}

func callerPod(address, kind string, caller Caller) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "caller", Namespace: caller.Namespace, UID: types.UID(uid.New(uid.TestPrefix)), Labels: map[string]string{
			labels.LabelKeyManagedBy: "krane", labels.LabelKeyComponent: "deployment",
			labels.LabelKeyWorkspaceID: caller.Workspace, labels.LabelKeyProjectID: caller.Project,
			labels.LabelKeyAppID: uid.New(uid.AppPrefix), labels.LabelKeyEnvironmentID: uid.New(uid.EnvironmentPrefix),
			labels.LabelKeyDeploymentID: caller.Deployment, privatenetwork.EnvironmentKindLabel: kind,
		}},
		Status: corev1.PodStatus{Phase: corev1.PodRunning, PodIP: address, PodIPs: []corev1.PodIP{{IP: address}}},
	}
}

// The fake tracker does not replay changes between List and Watch.
func waitForFakeDiscoveryWatches(t *testing.T, client *fake.Clientset) {
	t.Helper()
	require.Eventually(t, func() bool {
		pending := map[string]bool{"pods": true, "configmaps": true, "services": true, "endpointslices": true}
		for _, action := range client.Actions() {
			if action.GetVerb() == "watch" && action.GetNamespace() == "" {
				delete(pending, action.GetResource().Resource)
			}
		}
		return len(pending) == 0
	}, 5*time.Second, time.Millisecond)
}
