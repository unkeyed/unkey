package privatenetwork

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/logger/loggertest"
	"github.com/unkeyed/unkey/pkg/prometheus/lazy"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/krane/internal/testutil"
	"google.golang.org/protobuf/proto"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes/fake"
	clienttesting "k8s.io/client-go/testing"
)

var metricsRegistry = prometheus.NewRegistry()

func TestMain(m *testing.M) {
	lazy.SetRegistry(metricsRegistry)
	os.Exit(m.Run())
}

func TestReconcileReportsEntryStatesAndTargetSwitches(t *testing.T) {
	ctx := t.Context()
	capture := loggertest.Install(t)
	bound := newTestConnection()
	replica := newTestConnection()
	replica.WorkspaceId, replica.ProjectId = bound.WorkspaceId, bound.ProjectId
	replica.TargetAppSlug, replica.ConnectionName = "caller", "caller"
	replica.TargetDeploymentId = replica.CallerDeploymentId
	replica.ConnectionId = "self-" + replica.CallerDeploymentId
	unresolved := newTestConnection()
	unresolved.WorkspaceId, unresolved.ProjectId = bound.WorkspaceId, bound.ProjectId
	unresolved.TargetDeploymentId = ""
	unresolved.TargetPort = 0
	unresolved.ConnectionName = "ledger"
	snapshot := []*ctrlv1.PrivateNetworkConnection{bound, replica, unresolved}

	client := fake.NewClientset(endpointPod(bound, "a", "10.72.0.11"), endpointPod(replica, "self", "10.72.0.12"))
	control := &testutil.MockClusterClient{StreamPrivateNetworkStateFunc: snapshotFunc(t, func(context.Context) ([]*ctrlv1.PrivateNetworkConnection, error) {
		return snapshot, nil
	})}
	r := &Reconciler{clock: clock.NewTestClock(), client: client, dynamic: testDynamicClient(), cluster: control, clusterKey: &ctrlv1.ClusterKey{}}

	started := time.Now().Unix()
	require.NoError(t, r.reconcile(ctx))
	require.Equal(t, map[string]float64{
		"kind=connection,state=current": 1, "kind=connection,state=unresolved": 1, "kind=replica,state=current": 1,
	}, nonZero(gatherValues(t, "unkey_krane_private_network_entries")))
	require.GreaterOrEqual(t, gatherValues(t, "unkey_krane_private_network_last_completed_pass_unix_seconds")["loop=discovery"], float64(started))

	switched := proto.Clone(bound).(*ctrlv1.PrivateNetworkConnection)
	switched.TargetDeploymentId = uid.New(uid.DeploymentPrefix)
	snapshot = []*ctrlv1.PrivateNetworkConnection{switched, replica, unresolved}
	since := capture.Snapshot()
	require.NoError(t, r.reconcile(ctx))
	require.NoError(t, r.reconcile(ctx))
	require.Equal(t, 1.0, gatherValues(t, "unkey_krane_private_network_entries")["kind=connection,state=waiting_for_endpoints"])
	waiting := recordsWithMessage(capture.Since(since), "private network connection keeps its previous target until the new target has ready endpoints")
	require.Len(t, waiting, 1, "a held switch logs once, not every pass")
	attrs := loggertest.FlatAttrs(waiting[0])
	require.Equal(t, bound.TargetDeploymentId, attrs["published_deployment_id"])
	require.Equal(t, switched.TargetDeploymentId, attrs["target_deployment_id"])
	require.Equal(t, bound.ConnectionId, attrs["connection_id"])

	services, err := client.CoreV1().Services(switched.GetK8SNamespace()).List(ctx, metav1.ListOptions{})
	require.NoError(t, err)
	require.Len(t, services.Items, 3)
	var switchedService *corev1.Service
	for i := range services.Items {
		if services.Items[i].Labels["unkey.com/deployment.id"] == switched.TargetDeploymentId {
			switchedService = &services.Items[i]
		}
	}
	require.NotNil(t, switchedService)
	_, err = client.DiscoveryV1().EndpointSlices(switched.GetK8SNamespace()).Create(ctx, readySlice(switchedService, "dep-b-native", meshSliceManager), metav1.CreateOptions{})
	require.NoError(t, err)
	since = capture.Snapshot()
	require.NoError(t, r.reconcile(ctx))
	require.Equal(t, 0.0, gatherValues(t, "unkey_krane_private_network_entries")["kind=connection,state=waiting_for_endpoints"])
	published := recordsWithMessage(capture.Since(since), "private network connection published")
	require.Len(t, published, 1)
	attrs = loggertest.FlatAttrs(published[0])
	require.Equal(t, bound.TargetDeploymentId, attrs["previous_deployment_id"])
	require.Equal(t, switched.TargetDeploymentId, attrs["deployment_id"])
	require.Equal(t, "2", attrs["revision"])
}

func TestReconcileEntryFailuresReportTheirStage(t *testing.T) {
	forbidden := func(resource string) error {
		return apierrors.NewForbidden(schema.GroupResource{Resource: resource}, "", fmt.Errorf("denied"))
	}
	for _, tc := range []struct {
		stage   string
		inject  func(client *fake.Clientset, r *Reconciler, failing *bool)
		corrupt bool
	}{
		{stage: stageNamespace, inject: func(client *fake.Clientset, _ *Reconciler, failing *bool) {
			client.PrependReactor("create", "namespaces", failWhile(failing, forbidden("namespaces")))
		}},
		{stage: stageService, inject: func(client *fake.Clientset, _ *Reconciler, failing *bool) {
			client.PrependReactor("create", "services", failWhile(failing, forbidden("services")))
		}},
		{stage: stagePolicy, inject: func(_ *fake.Clientset, r *Reconciler, failing *bool) {
			r.dynamic.(interface {
				PrependReactor(string, string, clienttesting.ReactionFunc)
			}).PrependReactor("create", "ciliumnetworkpolicies", failWhile(failing, forbidden("ciliumnetworkpolicies")))
		}},
		{stage: stageConnection, inject: func(client *fake.Clientset, _ *Reconciler, failing *bool) {
			client.PrependReactor("create", "configmaps", failWhile(failing, forbidden("configmaps")))
		}},
		{stage: stageInvalidEntry, corrupt: true},
	} {
		t.Run(tc.stage, func(t *testing.T) {
			ctx := t.Context()
			capture := loggertest.Install(t)
			connectionSpec := newTestConnection()
			healthy := newTestConnection()
			healthy.WorkspaceId, healthy.ProjectId = connectionSpec.WorkspaceId, connectionSpec.ProjectId
			healthy.K8SNamespace, healthy.ConnectionName = "customer-2", "ledger-api"
			client := fake.NewClientset(endpointPod(connectionSpec, "a", "10.72.0.11"), endpointPod(healthy, "b", "10.72.0.21"))
			failing := true
			snapshot := func() []*ctrlv1.PrivateNetworkConnection {
				if tc.corrupt && failing {
					broken := proto.Clone(connectionSpec).(*ctrlv1.PrivateNetworkConnection)
					broken.TargetPort = 0
					return []*ctrlv1.PrivateNetworkConnection{broken, healthy}
				}
				return []*ctrlv1.PrivateNetworkConnection{connectionSpec, healthy}
			}
			control := &testutil.MockClusterClient{StreamPrivateNetworkStateFunc: snapshotFunc(t, func(context.Context) ([]*ctrlv1.PrivateNetworkConnection, error) {
				return snapshot(), nil
			})}
			r := &Reconciler{clock: clock.NewTestClock(), client: client, dynamic: testDynamicClient(), cluster: control, clusterKey: &ctrlv1.ClusterKey{}}
			if tc.inject != nil {
				tc.inject(client, r, &failing)
			}

			before := gatherValues(t, "unkey_krane_private_network_errors_total")
			since := capture.Snapshot()
			started := time.Now().Unix()
			require.Error(t, r.reconcile(ctx))
			require.Error(t, r.reconcile(ctx))

			after := gatherValues(t, "unkey_krane_private_network_errors_total")
			require.Equal(t, map[string]float64{"loop=discovery,stage=" + tc.stage: 2}, delta(before, after), "error counts after two failing passes")
			require.Equal(t, map[string]float64{"kind=connection,state=current": 1, "kind=connection,state=failed": 1},
				nonZero(gatherValues(t, "unkey_krane_private_network_entries")))
			require.GreaterOrEqual(t, gatherValues(t, "unkey_krane_private_network_last_completed_pass_unix_seconds")["loop=discovery"], float64(started),
				"a pass with a failed entry still completes")
			failures := recordsWithMessage(capture.Since(since), "private network entry failed to publish; its previous objects stay published")
			require.Len(t, failures, 1, "a persistent failure logs once")
			require.Equal(t, slog.LevelWarn, failures[0].Level)
			attrs := loggertest.FlatAttrs(failures[0])
			require.Equal(t, tc.stage, attrs["stage"])
			require.Equal(t, connectionSpec.ConnectionId, attrs["connection_id"])
			require.Equal(t, connectionSpec.CallerDeploymentId, attrs["caller_deployment_id"])
			require.NotNil(t, attrs["error"])

			failing = false
			since = capture.Snapshot()
			require.NoError(t, r.reconcile(ctx))
			require.Equal(t, map[string]float64{"kind=connection,state=current": 2}, nonZero(gatherValues(t, "unkey_krane_private_network_entries")))
			require.Len(t, recordsWithMessage(capture.Since(since), "private network entry recovered"), 1)
		})
	}
}

func TestReconcileAbortReportsStageWithoutCompletingPass(t *testing.T) {
	for _, tc := range []struct {
		stage  string
		inject func(client *fake.Clientset, control *testutil.MockClusterClient)
	}{
		{stage: stageSnapshot, inject: func(_ *fake.Clientset, control *testutil.MockClusterClient) {
			control.StreamPrivateNetworkStateFunc = snapshotFunc(t, func(context.Context) ([]*ctrlv1.PrivateNetworkConnection, error) {
				return nil, fmt.Errorf("ctrl unavailable")
			})
		}},
		{stage: stageList, inject: func(client *fake.Clientset, _ *testutil.MockClusterClient) {
			client.PrependReactor("list", "configmaps", func(clienttesting.Action) (bool, runtime.Object, error) {
				return true, nil, fmt.Errorf("API server unavailable")
			})
		}},
	} {
		t.Run(tc.stage, func(t *testing.T) {
			client := fake.NewClientset()
			control := &testutil.MockClusterClient{StreamPrivateNetworkStateFunc: snapshotFunc(t, func(context.Context) ([]*ctrlv1.PrivateNetworkConnection, error) {
				return nil, nil
			})}
			r := &Reconciler{clock: clock.NewTestClock(), client: client, dynamic: testDynamicClient(), cluster: control, clusterKey: &ctrlv1.ClusterKey{}}
			require.NoError(t, r.reconcile(t.Context()), "an idle cluster completes passes")
			completed := gatherValues(t, "unkey_krane_private_network_last_completed_pass_unix_seconds")["loop=discovery"]
			require.NotZero(t, completed)

			tc.inject(client, control)
			before := gatherValues(t, "unkey_krane_private_network_errors_total")
			time.Sleep(time.Second)
			require.Error(t, r.reconcile(t.Context()))
			require.Equal(t, map[string]float64{"loop=discovery,stage=" + tc.stage: 1},
				delta(before, gatherValues(t, "unkey_krane_private_network_errors_total")))
			require.Equal(t, completed, gatherValues(t, "unkey_krane_private_network_last_completed_pass_unix_seconds")["loop=discovery"],
				"an aborted pass must not advance the completion timestamp")
		})
	}
}

func TestLeaderWithoutCompletedPassReportsStaleTimestamp(t *testing.T) {
	startLeading()
	control := &testutil.MockClusterClient{StreamPrivateNetworkStateFunc: snapshotFunc(t, func(context.Context) ([]*ctrlv1.PrivateNetworkConnection, error) {
		return nil, fmt.Errorf("ctrl unavailable")
	})}
	r := &Reconciler{clock: clock.NewTestClock(), client: fake.NewClientset(), dynamic: testDynamicClient(), cluster: control, clusterKey: &ctrlv1.ClusterKey{}}
	require.Error(t, r.reconcile(t.Context()))

	timestamps := gatherValues(t, "unkey_krane_private_network_last_completed_pass_unix_seconds")
	require.Equal(t, 0.0, timestamps["loop=discovery"])
	require.Equal(t, 0.0, timestamps["loop=endpoints"])
}

func TestLeadershipReportsLeaderGauge(t *testing.T) {
	client := fake.NewClientset()
	control := &testutil.MockClusterClient{StreamPrivateNetworkStateFunc: snapshotFunc(t, func(context.Context) ([]*ctrlv1.PrivateNetworkConnection, error) {
		return nil, nil
	})}
	r, err := New(Config{
		Client: client, Dynamic: testDynamicClient(), Cluster: control, ClusterKey: &ctrlv1.ClusterKey{},
		Clock: clock.NewTestClock(),
	})
	require.NoError(t, err)
	require.Equal(t, 0.0, gatherValues(t, "unkey_krane_private_network_leader")[""])
	for loop, stages := range loopStages {
		for _, stage := range stages {
			require.Contains(t, gatherValues(t, "unkey_krane_private_network_errors_total"), "loop="+loop+",stage="+stage,
				"errors series for an idle cluster must exist at zero")
		}
	}

	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan struct{})
	go func() {
		defer close(done)
		r.Run(ctx)
	}()
	require.Eventually(t, func() bool {
		return gatherValues(t, "unkey_krane_private_network_leader")[""] == 1
	}, 10*time.Second, 10*time.Millisecond)
	require.Contains(t, gatherValues(t, "unkey_krane_private_network_last_completed_pass_unix_seconds"), "loop=discovery",
		"the leader exports completion timestamps from the moment it leads")

	cancel()
	<-done
	require.Equal(t, 0.0, gatherValues(t, "unkey_krane_private_network_leader")[""])
}

func failWhile(failing *bool, err error) clienttesting.ReactionFunc {
	return func(action clienttesting.Action) (bool, runtime.Object, error) {
		if !*failing || action.GetNamespace() == "customer-2" {
			return false, nil, nil
		}
		if create, ok := action.(clienttesting.CreateAction); ok {
			if namespace, ok := create.GetObject().(*corev1.Namespace); ok && namespace.Name == "customer-2" {
				return false, nil, nil
			}
		}
		return true, nil, err
	}
}

func gatherValues(t *testing.T, name string) map[string]float64 {
	t.Helper()
	families, err := metricsRegistry.Gather()
	require.NoError(t, err)
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
			values[strings.Join(pairs, ",")] = metric.GetCounter().GetValue() + metric.GetGauge().GetValue()
		}
	}
	return values
}

func delta(before, after map[string]float64) map[string]float64 {
	changed := map[string]float64{}
	for key, value := range after {
		if value != before[key] {
			changed[key] = value - before[key]
		}
	}
	return changed
}

func nonZero(values map[string]float64) map[string]float64 {
	kept := map[string]float64{}
	for key, value := range values {
		if value != 0 {
			kept[key] = value
		}
	}
	return kept
}

func recordsWithMessage(records []slog.Record, message string) []slog.Record {
	var matched []slog.Record
	for _, record := range records {
		if record.Message == message {
			matched = append(matched, record)
		}
	}
	return matched
}
