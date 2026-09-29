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
	"github.com/unkeyed/unkey/pkg/logger/loggertest"
	"github.com/unkeyed/unkey/pkg/prometheus/lazy"
	"github.com/unkeyed/unkey/svc/krane/internal/testutil"
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
	bound := testApp("dep_a")
	replica := testApp("caller_2")
	replica.AppId, replica.AppSlug, replica.BindingName = "app_caller", "caller", "caller"
	replica.BindingId, replica.CallerDeploymentId = "self-caller_2", "caller_2"
	unresolved := testApp("")
	unresolved.Port = 0
	unresolved.BindingId, unresolved.BindingName, unresolved.CallerDeploymentId = "binding_3", "ledger", "caller_3"
	snapshot := []*ctrlv1.PrivateNetworkApp{bound, replica, unresolved}

	client := fake.NewClientset(endpointPod(bound, "a", "10.72.0.11"), endpointPod(replica, "self", "10.72.0.12"))
	control := &testutil.MockClusterClient{StreamPrivateNetworkStateFunc: snapshotFunc(t, func(context.Context) ([]*ctrlv1.PrivateNetworkApp, error) {
		return snapshot, nil
	})}
	r := &Reconciler{client: client, dynamic: testDynamicClient(), cluster: control, clusterKey: &ctrlv1.ClusterKey{}}

	started := time.Now().Unix()
	require.NoError(t, r.reconcile(ctx))
	require.Equal(t, map[string]float64{
		"kind=binding,state=current": 1, "kind=binding,state=unresolved": 1, "kind=replica,state=current": 1,
	}, nonZero(gatherValues(t, "unkey_krane_private_network_entries")))
	require.GreaterOrEqual(t, gatherValues(t, "unkey_krane_private_network_last_completed_pass_unix_seconds")["loop=discovery"], float64(started))

	switched := testApp("dep_b")
	snapshot = []*ctrlv1.PrivateNetworkApp{switched, replica, unresolved}
	since := capture.Snapshot()
	require.NoError(t, r.reconcile(ctx))
	require.NoError(t, r.reconcile(ctx))
	require.Equal(t, 1.0, gatherValues(t, "unkey_krane_private_network_entries")["kind=binding,state=waiting_for_endpoints"])
	waiting := recordsWithMessage(capture.Since(since), "private network binding keeps its previous target until the new target has ready endpoints")
	require.Len(t, waiting, 1, "a held switch logs once, not every pass")
	attrs := loggertest.FlatAttrs(waiting[0])
	require.Equal(t, "dep_a", attrs["published_deployment_id"])
	require.Equal(t, "dep_b", attrs["target_deployment_id"])
	require.Equal(t, "binding_1", attrs["binding_id"])

	_, err := client.CoreV1().Pods(switched.GetK8SNamespace()).Create(ctx, endpointPod(switched, "b", "10.72.0.21"), metav1.CreateOptions{})
	require.NoError(t, err)
	since = capture.Snapshot()
	require.NoError(t, r.reconcile(ctx))
	require.Equal(t, 0.0, gatherValues(t, "unkey_krane_private_network_entries")["kind=binding,state=waiting_for_endpoints"])
	published := recordsWithMessage(capture.Since(since), "private network binding published")
	require.Len(t, published, 1)
	attrs = loggertest.FlatAttrs(published[0])
	require.Equal(t, "dep_a", attrs["previous_deployment_id"])
	require.Equal(t, "dep_b", attrs["deployment_id"])
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
		{stage: stageEndpointSlice, inject: func(client *fake.Clientset, _ *Reconciler, failing *bool) {
			client.PrependReactor("create", "endpointslices", failWhile(failing, forbidden("endpointslices")))
		}},
		{stage: stagePolicy, inject: func(_ *fake.Clientset, r *Reconciler, failing *bool) {
			r.dynamic.(interface {
				PrependReactor(string, string, clienttesting.ReactionFunc)
			}).PrependReactor("create", "ciliumnetworkpolicies", failWhile(failing, forbidden("ciliumnetworkpolicies")))
		}},
		{stage: stageBinding, inject: func(client *fake.Clientset, _ *Reconciler, failing *bool) {
			client.PrependReactor("create", "configmaps", failWhile(failing, forbidden("configmaps")))
		}},
		{stage: stageInvalidEntry, corrupt: true},
	} {
		t.Run(tc.stage, func(t *testing.T) {
			ctx := t.Context()
			capture := loggertest.Install(t)
			app := testApp("dep_a")
			healthy := testApp("dep_b")
			healthy.AppId, healthy.K8SNamespace = "app_2", "customer-2"
			healthy.BindingId, healthy.BindingName, healthy.CallerDeploymentId = "binding_2", "ledger-api", "caller_2"
			client := fake.NewClientset(endpointPod(app, "a", "10.72.0.11"), endpointPod(healthy, "b", "10.72.0.21"))
			failing := true
			snapshot := func() []*ctrlv1.PrivateNetworkApp {
				if tc.corrupt && failing {
					broken := testApp("dep_a")
					broken.Port = 0
					return []*ctrlv1.PrivateNetworkApp{broken, healthy}
				}
				return []*ctrlv1.PrivateNetworkApp{app, healthy}
			}
			control := &testutil.MockClusterClient{StreamPrivateNetworkStateFunc: snapshotFunc(t, func(context.Context) ([]*ctrlv1.PrivateNetworkApp, error) {
				return snapshot(), nil
			})}
			r := &Reconciler{client: client, dynamic: testDynamicClient(), cluster: control, clusterKey: &ctrlv1.ClusterKey{}}
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
			require.Equal(t, map[string]float64{"kind=binding,state=current": 1, "kind=binding,state=failed": 1},
				nonZero(gatherValues(t, "unkey_krane_private_network_entries")))
			require.GreaterOrEqual(t, gatherValues(t, "unkey_krane_private_network_last_completed_pass_unix_seconds")["loop=discovery"], float64(started),
				"a pass with a failed entry still completes")
			failures := recordsWithMessage(capture.Since(since), "private network entry failed to publish; its previous objects stay published")
			require.Len(t, failures, 1, "a persistent failure logs once")
			require.Equal(t, slog.LevelWarn, failures[0].Level)
			attrs := loggertest.FlatAttrs(failures[0])
			require.Equal(t, tc.stage, attrs["stage"])
			require.Equal(t, "binding_1", attrs["binding_id"])
			require.Equal(t, "caller_1", attrs["caller_deployment_id"])
			require.NotNil(t, attrs["error"])

			failing = false
			since = capture.Snapshot()
			require.NoError(t, r.reconcile(ctx))
			require.Equal(t, map[string]float64{"kind=binding,state=current": 2}, nonZero(gatherValues(t, "unkey_krane_private_network_entries")))
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
			control.StreamPrivateNetworkStateFunc = snapshotFunc(t, func(context.Context) ([]*ctrlv1.PrivateNetworkApp, error) {
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
			control := &testutil.MockClusterClient{StreamPrivateNetworkStateFunc: snapshotFunc(t, func(context.Context) ([]*ctrlv1.PrivateNetworkApp, error) {
				return nil, nil
			})}
			r := &Reconciler{client: client, dynamic: testDynamicClient(), cluster: control, clusterKey: &ctrlv1.ClusterKey{}}
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
	control := &testutil.MockClusterClient{StreamPrivateNetworkStateFunc: snapshotFunc(t, func(context.Context) ([]*ctrlv1.PrivateNetworkApp, error) {
		return nil, fmt.Errorf("ctrl unavailable")
	})}
	r := &Reconciler{client: fake.NewClientset(), dynamic: testDynamicClient(), cluster: control, clusterKey: &ctrlv1.ClusterKey{}}
	require.Error(t, r.reconcile(t.Context()))

	timestamps := gatherValues(t, "unkey_krane_private_network_last_completed_pass_unix_seconds")
	require.Equal(t, 0.0, timestamps["loop=discovery"])
	require.Equal(t, 0.0, timestamps["loop=endpoints"])
}

func TestEndpointRefreshFailuresReportTheirStage(t *testing.T) {
	ctx := t.Context()
	app := testApp("dep_a")
	client := fake.NewClientset(endpointPod(app, "a", "10.72.0.11"))
	control := &testutil.MockClusterClient{StreamPrivateNetworkStateFunc: snapshotFunc(t, func(context.Context) ([]*ctrlv1.PrivateNetworkApp, error) {
		return []*ctrlv1.PrivateNetworkApp{app}, nil
	})}
	r := &Reconciler{client: client, dynamic: testDynamicClient(), cluster: control, clusterKey: &ctrlv1.ClusterKey{}}
	require.NoError(t, r.reconcile(ctx))

	started := time.Now().Unix()
	require.NoError(t, r.reconcileEndpoints(ctx))
	require.GreaterOrEqual(t, gatherValues(t, "unkey_krane_private_network_last_completed_pass_unix_seconds")["loop=endpoints"], float64(started))

	failing := true
	client.PrependReactor("update", "endpointslices", failWhile(&failing, fmt.Errorf("denied")))
	_, err := client.CoreV1().Pods(app.GetK8SNamespace()).Create(ctx, endpointPod(app, "a2", "10.72.0.12"), metav1.CreateOptions{})
	require.NoError(t, err)
	before := gatherValues(t, "unkey_krane_private_network_errors_total")
	require.Error(t, r.reconcileEndpoints(ctx))
	require.Equal(t, map[string]float64{"loop=endpoints,stage=endpoint_slice": 1},
		delta(before, gatherValues(t, "unkey_krane_private_network_errors_total")))
}

func TestLeadershipReportsLeaderGauge(t *testing.T) {
	client := fake.NewClientset()
	control := &testutil.MockClusterClient{StreamPrivateNetworkStateFunc: snapshotFunc(t, func(context.Context) ([]*ctrlv1.PrivateNetworkApp, error) {
		return nil, nil
	})}
	r, err := New(Config{
		Client: client, Dynamic: testDynamicClient(), Cluster: control, ClusterKey: &ctrlv1.ClusterKey{},
		Identity: "krane-0", LeaseNamespace: "unkey",
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
	done := make(chan error, 1)
	go func() { done <- r.Run(ctx) }()
	require.Eventually(t, func() bool {
		return gatherValues(t, "unkey_krane_private_network_leader")[""] == 1
	}, 10*time.Second, 10*time.Millisecond)
	require.Contains(t, gatherValues(t, "unkey_krane_private_network_last_completed_pass_unix_seconds"), "loop=endpoints",
		"the leader exports completion timestamps from the moment it leads")

	cancel()
	require.ErrorIs(t, <-done, context.Canceled)
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
