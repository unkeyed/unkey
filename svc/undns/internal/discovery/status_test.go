package discovery

import (
	"context"
	"log/slog"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/deploy/privatenetwork"
	"github.com/unkeyed/unkey/pkg/logger/loggertest"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/krane/pkg/labels"
	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/features"
	featuretesting "k8s.io/client-go/features/testing"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/cache"
)

func TestConnectionStatusesSkipUnchangedDiscovery(t *testing.T) {
	c, _ := startStatusCatalog(t, clock.New())
	store := c.connections.SharedIndexInformer.(*countedStatusInformer).store
	require.Eventually(t, func() bool {
		return connectionCounts(t)[kindConnection][stateActive] == 1
	}, 5*time.Second, 10*time.Millisecond)

	require.Eventually(t, func() bool { return !c.statusDirty.Load() }, 5*time.Second, 10*time.Millisecond)
	passes := store.lists.Load()
	require.Positive(t, passes)
	require.Never(t, func() bool {
		connectionCounts(t)
		return store.lists.Load() != passes
	}, 2200*time.Millisecond, 10*time.Millisecond)
}

func TestConnectionStatusesFollowDiscoveryChanges(t *testing.T) {
	c, client := startStatusCatalog(t, clock.New())
	waitForState := func(state Reason, count int) {
		t.Helper()
		require.Eventually(t, func() bool {
			counts := connectionCounts(t)[kindConnection]
			return len(counts) == 1 && counts[state] == count
		}, 5*time.Second, 10*time.Millisecond)
	}
	waitForState(stateActive, 1)

	slice, err := client.DiscoveryV1().EndpointSlices("default").Get(t.Context(), "imported", metav1.GetOptions{})
	require.NoError(t, err)
	for i := range slice.Endpoints {
		slice.Endpoints[i].Conditions.Ready = new(false)
	}
	_, err = client.DiscoveryV1().EndpointSlices("default").Update(t.Context(), slice, metav1.UpdateOptions{})
	require.NoError(t, err)
	waitForState(ReasonNoReadyEndpoints, 1)

	require.NoError(t, client.DiscoveryV1().EndpointSlices("default").Delete(t.Context(), slice.Name, metav1.DeleteOptions{}))
	for i := range slice.Endpoints {
		slice.Endpoints[i].Conditions.Ready = new(true)
	}
	_, err = client.DiscoveryV1().EndpointSlices("default").Create(t.Context(), slice, metav1.CreateOptions{})
	require.NoError(t, err)
	waitForState(stateActive, 1)

	service := storedService(t, c).DeepCopy()
	require.NoError(t, client.CoreV1().Services("default").Delete(t.Context(), service.Name, metav1.DeleteOptions{}))
	waitForState(ReasonServiceMissing, 1)
	_, err = client.CoreV1().Services("default").Create(t.Context(), service, metav1.CreateOptions{})
	require.NoError(t, err)
	waitForState(stateActive, 1)

	duplicate := storedConnection(t, c).DeepCopy()
	duplicate.Name, duplicate.UID = "duplicate", types.UID(uid.New(uid.TestPrefix))
	_, err = client.CoreV1().ConfigMaps("default").Create(t.Context(), duplicate, metav1.CreateOptions{})
	require.NoError(t, err)
	waitForState(ReasonConnectionAmbiguous, 2)
	require.NoError(t, client.CoreV1().ConfigMaps("default").Delete(t.Context(), "connection", metav1.DeleteOptions{}))
	waitForState(stateActive, 1)

	duplicate.Data[privatenetwork.ConnectionDeploymentKey] = ""
	duplicate.Data[privatenetwork.ConnectionServiceKey] = ""
	_, err = client.CoreV1().ConfigMaps("default").Update(t.Context(), duplicate, metav1.UpdateOptions{})
	require.NoError(t, err)
	waitForState(ReasonConnectionUnresolved, 1)
	require.NoError(t, client.CoreV1().ConfigMaps("default").Delete(t.Context(), duplicate.Name, metav1.DeleteOptions{}))
	require.Eventually(t, func() bool {
		return len(connectionCounts(t)[kindConnection]) == 0
	}, 5*time.Second, 10*time.Millisecond)
}

func TestConnectionStatusesExpireWithoutDiscoveryChanges(t *testing.T) {
	start := time.Date(2026, 10, 7, 12, 0, 0, 0, time.UTC)
	testClock := clock.NewTestClock(start)
	c, client := startStatusCatalog(t, testClock)
	require.Eventually(t, func() bool {
		return connectionCounts(t)[kindConnection][stateActive] == 1
	}, 5*time.Second, 10*time.Millisecond)

	service := storedService(t, c).DeepCopy()
	service.Annotations = map[string]string{privatenetwork.RetireAfterAnnotation: start.Add(time.Hour).Format(time.RFC3339Nano)}
	_, err := client.CoreV1().Services("default").Update(t.Context(), service, metav1.UpdateOptions{})
	require.NoError(t, err)
	store := c.connections.SharedIndexInformer.(*countedStatusInformer).store
	passes := store.lists.Load()
	require.Eventually(t, func() bool { return store.lists.Load() > passes }, 5*time.Second, 10*time.Millisecond)

	testClock.Set(start.Add(time.Hour - time.Nanosecond))
	require.Never(t, func() bool {
		return connectionCounts(t)[kindConnection][ReasonServiceRetired] != 0
	}, 1100*time.Millisecond, 10*time.Millisecond)
	testClock.Set(start.Add(time.Hour))
	require.Eventually(t, func() bool {
		return connectionCounts(t)[kindConnection][ReasonServiceRetired] == 1
	}, 5*time.Second, 10*time.Millisecond)
}

func startStatusCatalog(t *testing.T, clk clock.Clock) (*Catalog, *fake.Clientset) {
	t.Helper()
	featuretesting.SetFeatureDuringTest(t, features.WatchListClient, false)
	client := fake.NewSimpleClientset(discoveryObjects(t)...)
	c, err := New(client, 30*time.Second, clk)
	require.NoError(t, err)
	c.connections.SharedIndexInformer = &countedStatusInformer{
		SharedIndexInformer: c.connections.SharedIndexInformer,
		store:               &countedStatusStore{Store: c.connections.GetStore()},
	}
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)
	go func() { done <- c.Run(ctx) }()
	t.Cleanup(func() {
		cancel()
		require.NoError(t, <-done)
	})
	waitForFakeDiscoveryWatches(t, client)
	return c, client
}

type countedStatusInformer struct {
	cache.SharedIndexInformer
	store *countedStatusStore
}

func (i *countedStatusInformer) GetStore() cache.Store { return i.store }

type countedStatusStore struct {
	cache.Store
	lists atomic.Int64
}

func (s *countedStatusStore) List() []any {
	s.lists.Add(1)
	return s.Store.List()
}

// TestResolveFailuresReportDistinctReasons guarantees that each way a private
// name can fail has its own reason, both for the query path and for the
// connection state reported in unkey_dns_connections. Operators rely on these
// reasons to tell configuration faults from discovery outages.
func TestResolveFailuresReportDistinctReasons(t *testing.T) {
	for _, tc := range []struct {
		name            string
		source          string
		alias           string
		mutate          func(t *testing.T, c *Catalog)
		identityStale   bool
		discoveryStale  bool
		identifyReason  Reason
		resolveReason   Reason
		unknown         bool
		connectionState Reason
		connections     int
	}{
		{name: "answer", connectionState: stateActive},
		{name: "unknown alias", alias: "ledger", unknown: true, connectionState: stateActive},
		{name: "unknown source", source: "127.0.0.9", identifyReason: ReasonUnknownCaller, connectionState: stateActive},
		{
			name: "reused source IP",
			mutate: func(t *testing.T, c *Catalog) {
				other := callerPod("127.0.0.1", "production", testCaller())
				other.Name = "other-caller"
				require.NoError(t, c.pods.GetStore().Add(other))
			},
			identifyReason: ReasonAmbiguousCaller, connectionState: stateActive,
		},
		{
			name: "caller without deployment",
			mutate: func(t *testing.T, c *Catalog) {
				object, exists, err := c.pods.GetStore().GetByKey("default/caller")
				require.NoError(t, err)
				require.True(t, exists)
				pod := object.(*corev1.Pod).DeepCopy()
				delete(pod.Labels, labels.LabelKeyDeploymentID)
				require.NoError(t, c.pods.GetStore().Update(pod))
			},
			identifyReason: ReasonIneligibleCaller, connectionState: stateActive,
		},
		{
			name:          "pod watch stale",
			mutate:        func(_ *testing.T, c *Catalog) { c.pods.lastContact.Store(0) },
			identityStale: true, discoveryStale: true, connectionState: stateActive,
		},
		{
			name:           "endpoint watch stale",
			mutate:         func(_ *testing.T, c *Catalog) { c.slices.lastContact.Store(0) },
			discoveryStale: true, connectionState: stateActive,
		},
		{
			name: "no selected target",
			mutate: func(t *testing.T, c *Catalog) {
				updateStoredConnection(t, c, map[string]string{"deploymentId": "", "serviceName": "", "revision": "2"})
			},
			resolveReason: ReasonConnectionUnresolved, connectionState: ReasonConnectionUnresolved,
		},
		{
			name:          "corrupt revision",
			mutate:        func(t *testing.T, c *Catalog) { updateStoredConnection(t, c, map[string]string{"revision": "latest"}) },
			resolveReason: ReasonConnectionInvalid, connectionState: ReasonConnectionInvalid,
		},
		{
			name: "duplicate connection",
			mutate: func(t *testing.T, c *Catalog) {
				duplicate := storedConnection(t, c).DeepCopy()
				duplicate.Name, duplicate.UID = "connection-copy", types.UID(uid.New(uid.TestPrefix))
				require.NoError(t, c.connections.GetStore().Add(duplicate))
			},
			resolveReason: ReasonConnectionAmbiguous, connectionState: ReasonConnectionAmbiguous, connections: 2,
		},
		{
			name: "discovery service deleted",
			mutate: func(t *testing.T, c *Catalog) {
				require.NoError(t, c.services.GetStore().Delete(storedService(t, c)))
			},
			resolveReason: ReasonServiceMissing, connectionState: ReasonServiceMissing,
		},
		{
			name: "discovery service publishes unready pods",
			mutate: func(t *testing.T, c *Catalog) {
				service := storedService(t, c).DeepCopy()
				service.Spec.PublishNotReadyAddresses = true
				require.NoError(t, c.services.GetStore().Update(service))
			},
			resolveReason: ReasonServiceRejected, connectionState: ReasonServiceRejected,
		},
		{
			name: "discovery service retired",
			mutate: func(t *testing.T, c *Catalog) {
				service := storedService(t, c).DeepCopy()
				service.Annotations = map[string]string{privatenetwork.RetireAfterAnnotation: time.Now().Add(-time.Minute).Format(time.RFC3339Nano)}
				require.NoError(t, c.services.GetStore().Update(service))
			},
			resolveReason: ReasonServiceRetired, connectionState: ReasonServiceRetired,
		},
		{
			name: "target lost its endpoints",
			mutate: func(t *testing.T, c *Catalog) {
				object, exists, err := c.slices.GetStore().GetByKey("default/imported")
				require.NoError(t, err)
				require.True(t, exists)
				slice := object.(*discoveryv1.EndpointSlice).DeepCopy()
				for i := range slice.Endpoints {
					slice.Endpoints[i].Conditions.Ready = new(false)
				}
				require.NoError(t, c.slices.GetStore().Update(slice))
			},
			resolveReason: ReasonNoReadyEndpoints, connectionState: ReasonNoReadyEndpoints,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := startCatalog(t)
			if tc.mutate != nil {
				tc.mutate(t, c)
			}
			source, alias := tc.source, tc.alias
			if source == "" {
				source = "127.0.0.1"
			}
			if alias == "" {
				alias = "payments"
			}

			require.Equal(t, !tc.identityStale, c.IdentityReady())
			require.Equal(t, !tc.discoveryStale, c.Ready())
			caller, err := c.Identify(netipAddress(source))
			if tc.identifyReason != "" {
				requireReason(t, err, tc.identifyReason)
			} else {
				require.NoError(t, err)
				addresses, exists, err := c.Resolve(caller, alias)
				switch {
				case tc.resolveReason != "":
					requireReason(t, err, tc.resolveReason)
					require.True(t, exists)
				case tc.unknown:
					require.NoError(t, err)
					require.False(t, exists)
				default:
					require.NoError(t, err)
					require.True(t, exists)
					require.Len(t, addresses, 40)
				}
			}

			c.updateStatuses()
			want := map[string]map[Reason]int{kindConnection: {tc.connectionState: max(tc.connections, 1)}, kindReplica: {}}
			require.Equal(t, want, connectionCounts(t), "connection states after %s", tc.name)
		})
	}
}

// TestIdleCatalogReportsZeroConnectionStates guarantees that every kind and
// state series exists even when no connection is published, so alerts on
// failing states see 0 rather than an absent series.
func TestIdleCatalogReportsZeroConnectionStates(t *testing.T) {
	catalogForTest().updateStatuses()
	values := gaugeValues(t, "unkey_dns_connections")
	require.Len(t, values, 2*len(connectionStates))
	for series, value := range values {
		require.Zero(t, value, "idle connection state %s", series)
	}
}

func TestReplicaConnectionsReportTheirOwnKind(t *testing.T) {
	c := seededCatalog(t)
	replica := storedConnection(t, c).DeepCopy()
	replica.Name, replica.UID = "replica", types.UID(uid.New(uid.TestPrefix))
	replica.Labels[labels.LabelKeyCallerDeploymentID] = replica.Data["deploymentId"]
	replica.Labels[labels.LabelKeyConnectionID] = "self-" + replica.Data["deploymentId"]
	replica.Data["appSlug"] = "caller-app"
	require.NoError(t, c.connections.GetStore().Add(replica))

	c.updateStatuses()
	counts := connectionCounts(t)
	require.Equal(t, 1, counts[kindConnection][stateActive], "directed connections: %v", counts)
	require.Equal(t, 1, counts[kindReplica][stateActive], "replica connections: %v", counts)
}

func TestUnavailablePublishedTargetReportsFailure(t *testing.T) {
	c := seededCatalog(t)
	capture := loggertest.Install(t)
	c.updateStatuses()
	service := storedService(t, c).DeepCopy()
	deploymentID := uid.New(uid.DeploymentPrefix)
	service.Name, service.UID = "service-b", types.UID(uid.New(uid.TestPrefix))
	service.Labels[labels.LabelKeyDeploymentID] = deploymentID
	require.NoError(t, c.services.GetStore().Add(service))
	updateStoredConnection(t, c, map[string]string{"deploymentId": deploymentID, "serviceName": "service-b", "revision": "2"})

	since := capture.Snapshot()
	c.updateStatuses()
	require.Equal(t, 1, connectionCounts(t)[kindConnection][ReasonNoReadyEndpoints])
	record := findRecord(t, capture.Since(since), "private DNS connection cannot be served")
	attrs := loggertest.FlatAttrs(record)
	require.Equal(t, deploymentID, attrs["target_deployment_id"])
}

func TestConnectionStateChangesAreLoggedOnce(t *testing.T) {
	c := seededCatalog(t)
	connection := storedConnection(t, c)
	capture := loggertest.Install(t)
	since := capture.Snapshot()
	c.updateStatuses()
	require.Empty(t, connectionRecords(capture.Since(since)), "healthy connections seen on startup must not be logged")

	service := storedService(t, c)
	require.NoError(t, c.services.GetStore().Delete(service))
	since = capture.Snapshot()
	c.updateStatuses()
	c.updateStatuses()
	records := connectionRecords(capture.Since(since))
	require.Len(t, records, 1, "a failing connection logs once per state change")
	require.Equal(t, slog.LevelWarn, records[0].Level)
	attrs := loggertest.FlatAttrs(records[0])
	require.Equal(t, string(ReasonServiceMissing), attrs["state"])
	require.Equal(t, string(stateActive), attrs["previous_state"])
	require.Equal(t, connection.Labels[labels.LabelKeyConnectionID], attrs["connection_id"])
	require.Equal(t, connection.Labels[labels.LabelKeyCallerDeploymentID], attrs["caller_deployment_id"])
	require.Equal(t, connection.Data["deploymentId"], attrs["target_deployment_id"])
	require.Equal(t, "payments", attrs["alias"])

	require.NoError(t, c.services.GetStore().Add(service))
	since = capture.Snapshot()
	c.updateStatuses()
	c.updateStatuses()
	records = connectionRecords(capture.Since(since))
	require.Len(t, records, 1, "a recovered connection logs once")
	require.Equal(t, slog.LevelInfo, records[0].Level)
	attrs = loggertest.FlatAttrs(records[0])
	require.Equal(t, string(stateActive), attrs["state"])
	require.Equal(t, string(ReasonServiceMissing), attrs["previous_state"])
}

func findRecord(t *testing.T, records []slog.Record, message string) slog.Record {
	t.Helper()
	for _, record := range records {
		if record.Message == message {
			return record
		}
	}
	t.Fatalf("no log record %q in %d records", message, len(records))
	return slog.Record{}
}

func connectionRecords(records []slog.Record) []slog.Record {
	var matched []slog.Record
	for _, record := range records {
		switch record.Message {
		case "private DNS connection cannot be served",
			"private DNS connection serves its target":
			matched = append(matched, record)
		}
	}
	return matched
}
