package engine_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	ch "github.com/ClickHouse/clickhouse-go/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	logdrainv1 "github.com/unkeyed/unkey/gen/proto/logdrain/v1"
	vaultv1 "github.com/unkeyed/unkey/gen/proto/vault/v1"
	"github.com/unkeyed/unkey/pkg/clickhouse"
	"github.com/unkeyed/unkey/pkg/clickhouse/schema"
	"github.com/unkeyed/unkey/pkg/mysql/sqlcomment"
	"github.com/unkeyed/unkey/pkg/testutil/containers"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/logdrain/internal/db"
	"github.com/unkeyed/unkey/svc/logdrain/internal/engine"
	"github.com/unkeyed/unkey/svc/logdrain/internal/lease"
	"github.com/unkeyed/unkey/svc/logdrain/internal/source"
	"google.golang.org/protobuf/proto"
)

type stubVault struct{}

func (stubVault) Liveness(context.Context, *vaultv1.LivenessRequest) (*vaultv1.LivenessResponse, error) {
	return nil, errors.New("not implemented")
}
func (stubVault) Encrypt(context.Context, *vaultv1.EncryptRequest) (*vaultv1.EncryptResponse, error) {
	return nil, errors.New("not implemented")
}
func (stubVault) Decrypt(_ context.Context, req *vaultv1.DecryptRequest) (*vaultv1.DecryptResponse, error) {
	return &vaultv1.DecryptResponse{Plaintext: req.GetEncrypted()}, nil
}
func (stubVault) EncryptBulk(context.Context, *vaultv1.EncryptBulkRequest) (*vaultv1.EncryptBulkResponse, error) {
	return nil, errors.New("not implemented")
}
func (stubVault) DecryptBulk(context.Context, *vaultv1.DecryptBulkRequest) (*vaultv1.DecryptBulkResponse, error) {
	return nil, errors.New("not implemented")
}
func (stubVault) ReEncrypt(context.Context, *vaultv1.ReEncryptRequest) (*vaultv1.ReEncryptResponse, error) {
	return nil, errors.New("not implemented")
}

type collector struct {
	mu         sync.Mutex
	deliveries []schema.LogdrainDeliveryV1
}

func (c *collector) Buffer(delivery schema.LogdrainDeliveryV1) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.deliveries = append(c.deliveries, delivery)
}

func (c *collector) snapshot() []schema.LogdrainDeliveryV1 {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]schema.LogdrainDeliveryV1(nil), c.deliveries...)
}

type capturedRequest struct {
	method         string
	path           string
	header         http.Header
	body           []byte
	responseStatus int
}

type sink struct {
	server       *httptest.Server
	status       atomic.Int64
	mu           sync.Mutex
	delay        time.Duration
	responseBody string
	requests     []capturedRequest
	inflight     atomic.Int64
	maxInflight  atomic.Int64
}

// maxOverlap reports the highest number of requests this sink served at the
// same time, which proves whether deliveries overlapped.
func (s *sink) maxOverlap() int64 {
	return s.maxInflight.Load()
}

func newSink(t *testing.T, status int) *sink {
	t.Helper()
	return newSlowSink(t, status, 0)
}

func newSlowSink(t *testing.T, status int, delay time.Duration) *sink {
	t.Helper()
	s := &sink{delay: delay}
	s.status.Store(int64(status))
	s.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		current := s.inflight.Add(1)
		defer s.inflight.Add(-1)
		for {
			seen := s.maxInflight.Load()
			if current <= seen || s.maxInflight.CompareAndSwap(seen, current) {
				break
			}
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		time.Sleep(s.delay)
		responseStatus := int(s.status.Load())
		s.mu.Lock()
		s.requests = append(s.requests, capturedRequest{method: r.Method, path: r.URL.Path, header: r.Header.Clone(), body: body, responseStatus: responseStatus})
		responseBody := s.responseBody
		s.mu.Unlock()
		w.WriteHeader(responseStatus)
		if responseBody != "" {
			_, _ = io.WriteString(w, responseBody)
		}
	}))
	t.Cleanup(s.server.Close)
	return s
}

func (s *sink) snapshot() []capturedRequest {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]capturedRequest(nil), s.requests...)
}

type auditEvent struct {
	id            string
	insertedAt    int64
	actorMeta     string
	targetTypes   []string
	targetIDs     []string
	targetNames   []string
	targetMetas   []string
	correlationID string
}

type drainState struct {
	status                    string
	committedOffsetInsertedAt int64
	consecutiveFailures       int
}

func TestEngine_Integration(t *testing.T) {
	mysqlCfg := containers.MySQL(t)
	clickhouseCfg := containers.ClickHouse(t)
	mysqlDB, err := sql.Open("mysql", mysqlCfg.DSN)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, mysqlDB.Close()) })

	opts, err := ch.ParseDSN(clickhouseCfg.DSN)
	require.NoError(t, err)
	chConn, err := ch.Open(opts)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, chConn.Close()) })

	t.Run("happy path delivers audit logs with credentials", func(t *testing.T) {
		workspaceID, drainID := uniqueIDs()
		httpSink := newSink(t, http.StatusOK)
		start := time.Now().Add(-time.Second).UnixMilli()
		events := []auditEvent{
			{id: drainID + "_event_1", insertedAt: start, actorMeta: `{"role":"admin"}`, targetTypes: []string{"api"}, targetIDs: []string{"api_123"}, targetNames: []string{"My API"}, targetMetas: []string{`{"region":"us"}`}},
			{id: drainID + "_event_2", insertedAt: start + 1, actorMeta: `{}`},
			{id: drainID + "_event_3", insertedAt: start + 2, actorMeta: `{}`, correlationID: drainID + "_correlation"},
		}
		insertAuditEvents(t, chConn, workspaceID, events)
		seedDrain(t, mysqlDB, workspaceID, drainID, httpSink.server.URL+"/ingest", start-1, "Bearer it-test-token")
		cleanupDrain(t, mysqlDB, drainID)
		deliveries := startEngine(t, mysqlCfg.DSN, clickhouseCfg.HTTPDSN)

		require.EventuallyWithT(t, func(c *assert.CollectT) {
			requests := httpSink.snapshot()
			require.NotEmpty(c, requests)
			ids, eventPayloads, parseErr := parseSuccessfulRequests(requests)
			require.NoError(c, parseErr)
			for _, event := range events {
				require.Contains(c, ids, event.id)
			}
			require.GreaterOrEqual(c, readDrainStateCollect(c, mysqlDB, drainID).committedOffsetInsertedAt, start+2)
			require.Equal(c, "application/json", requests[0].header.Get("Content-Type"))
			require.Equal(c, "Bearer it-test-token", requests[0].header.Get("Authorization"))
			require.Equal(c, "v1", requests[0].header.Get("X-Unkey-Schema-Version"))
			require.Equal(c, drainID, requests[0].header.Get("X-Unkey-Drain-Id"))
			require.Equal(c, workspaceID, requests[0].header.Get("X-Unkey-Workspace-Id"))
			require.Equal(c, http.MethodPost, requests[0].method)
			event := eventPayloads[events[0].id]
			require.Equal(c, events[0].id, event["id"])
			require.NotEmpty(c, event["action"])
			require.NotNil(c, event["occurred_at"])
			actor := event["actor"].(map[string]any)
			require.Equal(c, "user", actor["type"])
			require.Equal(c, "actor_1", actor["id"])
			require.Equal(c, "Integration Tester", actor["name"])
			require.Equal(c, "admin", actor["metadata"].(map[string]any)["role"])
			targets := event["targets"].([]any)
			require.Equal(c, map[string]any{"id": "api_123", "type": "api", "name": "My API", "metadata": map[string]any{"region": "us"}}, targets[0])
			require.Equal(c, events[2].correlationID, eventPayloads[events[2].id]["correlation_id"])
		}, 30*time.Second, 250*time.Millisecond)

		state := readDrainState(t, mysqlDB, drainID)
		require.Zero(t, state.consecutiveFailures)
		require.EventuallyWithT(t, func(c *assert.CollectT) {
			require.True(c, hasDelivery(deliveries.snapshot(), drainID, "success", 3))
		}, 5*time.Second, 100*time.Millisecond)
	})

	t.Run("failed response retries without advancing offset", func(t *testing.T) {
		workspaceID, drainID := uniqueIDs()
		httpSink := newSink(t, http.StatusInternalServerError)
		start := time.Now().Add(-time.Second).UnixMilli()
		events := []auditEvent{{id: drainID + "_event_1", insertedAt: start, actorMeta: `{}`}, {id: drainID + "_event_2", insertedAt: start + 1, actorMeta: `{}`}}
		insertAuditEvents(t, chConn, workspaceID, events)
		seedDrain(t, mysqlDB, workspaceID, drainID, httpSink.server.URL+"/ingest", start-1)
		cleanupDrain(t, mysqlDB, drainID)
		deliveries := startEngine(t, mysqlCfg.DSN, clickhouseCfg.HTTPDSN)

		require.EventuallyWithT(t, func(c *assert.CollectT) {
			state := readDrainStateCollect(c, mysqlDB, drainID)
			require.GreaterOrEqual(c, state.consecutiveFailures, 1)
			require.Equal(c, start-1, state.committedOffsetInsertedAt)
		}, 30*time.Second, 250*time.Millisecond)

		httpSink.status.Store(http.StatusOK)
		// Production retry backoff starts at 1 minute and is intentionally not configurable.
		_, err = mysqlDB.Exec("UPDATE logdrain_state SET next_attempt_at = 0 WHERE logdrain_id = ?", drainID)
		require.NoError(t, err)
		require.EventuallyWithT(t, func(c *assert.CollectT) {
			state := readDrainStateCollect(c, mysqlDB, drainID)
			require.GreaterOrEqual(c, state.committedOffsetInsertedAt, start+1)
			require.Zero(c, state.consecutiveFailures)
			ids, _, parseErr := parseSuccessfulRequests(httpSink.snapshot())
			require.NoError(c, parseErr)
			for _, event := range events {
				require.Contains(c, ids, event.id)
			}
		}, 30*time.Second, 250*time.Millisecond)
		require.True(t, hasDelivery(deliveries.snapshot(), drainID, "error", 2))
		require.True(t, hasDelivery(deliveries.snapshot(), drainID, "success", 2))
	})

	t.Run("client error pauses drain at failure threshold", func(t *testing.T) {
		workspaceID, drainID := uniqueIDs()
		httpSink := newSink(t, http.StatusBadRequest)
		httpSink.responseBody = `{"message":"invalid payload"}`
		start := time.Now().Add(-time.Second).UnixMilli()
		insertAuditEvents(t, chConn, workspaceID, []auditEvent{{id: drainID + "_event_1", insertedAt: start, actorMeta: `{}`}})
		seedDrain(t, mysqlDB, workspaceID, drainID, httpSink.server.URL+"/ingest", start-1)
		cleanupDrain(t, mysqlDB, drainID)
		_, err := mysqlDB.Exec("UPDATE logdrain_state SET consecutive_failures = 4 WHERE logdrain_id = ?", drainID)
		require.NoError(t, err)
		deliveries := startEngine(t, mysqlCfg.DSN, clickhouseCfg.HTTPDSN)

		require.EventuallyWithT(t, func(c *assert.CollectT) {
			state := readDrainStateCollect(c, mysqlDB, drainID)
			require.Equal(c, "paused_by_failure", state.status)
			require.Equal(c, start-1, state.committedOffsetInsertedAt)
			found := false
			for _, delivery := range deliveries.snapshot() {
				if delivery.DrainID == drainID && delivery.Outcome == "error" && delivery.Events == 1 {
					found = true
					require.Equal(c, int32(http.StatusBadRequest), delivery.ResponseStatus)
					require.Equal(c, httpSink.responseBody, delivery.ResponseBody)
					require.Positive(c, delivery.RequestBodyBytes)
				}
			}
			require.True(c, found)
		}, 30*time.Second, 250*time.Millisecond)
	})

	t.Run("concurrent engines do not duplicate a batch while the lease is valid", func(t *testing.T) {
		// One valid lease must prevent concurrent happy-path delivery. Delivery
		// remains at-least-once if the lease expires during an external request.
		workspaceID, drainID := uniqueIDs()
		httpSink := newSlowSink(t, http.StatusOK, time.Second)
		start := time.Now().Add(-time.Second).UnixMilli()
		events := []auditEvent{
			{id: drainID + "_event_1", insertedAt: start, actorMeta: `{}`},
			{id: drainID + "_event_2", insertedAt: start + 1, actorMeta: `{}`},
			{id: drainID + "_event_3", insertedAt: start + 2, actorMeta: `{}`},
		}
		insertAuditEvents(t, chConn, workspaceID, events)
		seedDrain(t, mysqlDB, workspaceID, drainID, httpSink.server.URL+"/ingest", start-1, "Bearer it-test-token")
		cleanupDrain(t, mysqlDB, drainID)
		startEngine(t, mysqlCfg.DSN, clickhouseCfg.HTTPDSN)
		startEngine(t, mysqlCfg.DSN, clickhouseCfg.HTTPDSN)

		require.EventuallyWithT(t, func(c *assert.CollectT) {
			ids, _, parseErr := parseSuccessfulRequests(httpSink.snapshot())
			require.NoError(c, parseErr)
			for _, event := range events {
				require.Contains(c, ids, event.id)
			}
			require.GreaterOrEqual(c, readDrainStateCollect(c, mysqlDB, drainID).committedOffsetInsertedAt, start+2)
		}, 30*time.Second, 250*time.Millisecond)

		require.Never(t, func() bool {
			counts, err := successfulRequestIDCounts(httpSink.snapshot())
			require.NoError(t, err)
			for _, count := range counts {
				if count > 1 {
					return true
				}
			}
			return false
		}, 3*time.Second, 250*time.Millisecond)
	})

	t.Run("one replica processes drains in parallel up to the limit", func(t *testing.T) {
		// MaxConcurrentDrains bounds in-replica parallelism. Two due drains
		// pointed at one slow sink must overlap when the limit is 2. The in-flight
		// set serializes work per drain, so overlap can only come from different drains.
		workspaceA, drainA := uniqueIDs()
		workspaceB, drainB := uniqueIDs()
		require.NotEqual(t, drainA, drainB)
		httpSink := newSlowSink(t, http.StatusOK, time.Second)
		start := time.Now().Add(-time.Second).UnixMilli()
		insertAuditEvents(t, chConn, workspaceA, []auditEvent{{id: drainA + "_event_1", insertedAt: start, actorMeta: `{}`}})
		insertAuditEvents(t, chConn, workspaceB, []auditEvent{{id: drainB + "_event_1", insertedAt: start, actorMeta: `{}`}})
		seedDrain(t, mysqlDB, workspaceA, drainA, httpSink.server.URL+"/ingest", start-1)
		seedDrain(t, mysqlDB, workspaceB, drainB, httpSink.server.URL+"/ingest", start-1)
		cleanupDrain(t, mysqlDB, drainA)
		cleanupDrain(t, mysqlDB, drainB)
		startEngineConcurrent(t, mysqlCfg.DSN, clickhouseCfg.HTTPDSN, 2)

		require.EventuallyWithT(t, func(c *assert.CollectT) {
			ids, _, parseErr := parseSuccessfulRequests(httpSink.snapshot())
			require.NoError(c, parseErr)
			require.Contains(c, ids, drainA+"_event_1")
			require.Contains(c, ids, drainB+"_event_1")
			require.Equal(c, int64(2), httpSink.maxOverlap())
		}, 30*time.Second, 250*time.Millisecond)
	})

	t.Run("composite cursor delivers every event sharing one millisecond", func(t *testing.T) {
		workspaceID, drainID := uniqueIDs()
		httpSink := newSink(t, http.StatusOK)
		insertedAt := time.Now().Add(-time.Second).UnixMilli()
		events := make([]auditEvent, 1000)
		for i := range events {
			events[i] = auditEvent{id: fmt.Sprintf("%s_event_%04d", drainID, i), insertedAt: insertedAt, actorMeta: `{}`}
		}
		insertAuditEvents(t, chConn, workspaceID, events)
		seedDrain(t, mysqlDB, workspaceID, drainID, httpSink.server.URL+"/ingest", insertedAt-1)
		cleanupDrain(t, mysqlDB, drainID)
		startEngine(t, mysqlCfg.DSN, clickhouseCfg.HTTPDSN)

		require.EventuallyWithT(t, func(c *assert.CollectT) {
			counts, parseErr := successfulRequestIDCounts(httpSink.snapshot())
			require.NoError(c, parseErr)
			require.GreaterOrEqual(c, len(counts), len(events))
			for _, event := range events {
				require.Contains(c, counts, event.id)
				require.Equal(c, 1, counts[event.id])
			}
			require.Greater(c, readDrainStateCollect(c, mysqlDB, drainID).committedOffsetInsertedAt, insertedAt)
		}, 60*time.Second, 250*time.Millisecond)
	})

	t.Run("mixed-case cursor advances bytewise across batches", func(t *testing.T) {
		workspaceID, drainID := uniqueIDs()
		httpSink := newSink(t, http.StatusOK)
		insertedAt := time.Now().Add(-time.Second).UnixMilli()
		events := []auditEvent{
			{id: drainID + "_B", insertedAt: insertedAt, actorMeta: `{}`},
			{id: drainID + "_C", insertedAt: insertedAt, actorMeta: `{}`},
			{id: drainID + "_a", insertedAt: insertedAt, actorMeta: `{}`},
			{id: drainID + "_b", insertedAt: insertedAt, actorMeta: `{}`},
		}
		insertAuditEvents(t, chConn, workspaceID, events)
		seedDrain(t, mysqlDB, workspaceID, drainID, httpSink.server.URL+"/ingest", insertedAt-1)
		cleanupDrain(t, mysqlDB, drainID)
		startEngineWithBatchSize(t, mysqlCfg.DSN, clickhouseCfg.HTTPDSN, 1)

		require.EventuallyWithT(t, func(c *assert.CollectT) {
			counts, parseErr := successfulRequestIDCounts(httpSink.snapshot())
			require.NoError(c, parseErr)
			for _, event := range events {
				require.Equal(c, 1, counts[event.id])
			}
		}, 30*time.Second, 250*time.Millisecond)
		require.Never(t, func() bool {
			counts, err := successfulRequestIDCounts(httpSink.snapshot())
			require.NoError(t, err)
			for _, event := range events {
				if counts[event.id] != 1 {
					return true
				}
			}
			return false
		}, time.Second, 100*time.Millisecond)
	})

	t.Run("blocking queue eventually processes every due drain", func(t *testing.T) {
		httpSink := newSink(t, http.StatusOK)
		insertedAt := time.Now().Add(-time.Second).UnixMilli()
		eventIDs := make([]string, 10)
		for i := range eventIDs {
			workspaceID, drainID := uniqueIDs()
			eventIDs[i] = drainID + "_event"
			insertAuditEvents(t, chConn, workspaceID, []auditEvent{{id: eventIDs[i], insertedAt: insertedAt, actorMeta: `{}`}})
			seedDrain(t, mysqlDB, workspaceID, drainID, httpSink.server.URL+"/ingest", insertedAt-1)
			cleanupDrain(t, mysqlDB, drainID)
		}
		startEngineConfigured(t, mysqlCfg.DSN, clickhouseCfg.HTTPDSN, 2, 2)

		require.EventuallyWithT(t, func(c *assert.CollectT) {
			ids, _, parseErr := parseSuccessfulRequests(httpSink.snapshot())
			require.NoError(c, parseErr)
			for _, eventID := range eventIDs {
				require.Contains(c, ids, eventID)
			}
		}, 30*time.Second, 250*time.Millisecond)
	})
}

func uniqueIDs() (string, string) {
	suffix := time.Now().UnixNano()
	return fmt.Sprintf("ws_it_%d", suffix), fmt.Sprintf("ld_it_%d", suffix)
}

func insertAuditEvents(t *testing.T, conn ch.Conn, workspaceID string, events []auditEvent) {
	t.Helper()
	ctx := context.Background()
	for _, event := range events {
		err := conn.Exec(ctx, "INSERT INTO audit_logs_raw_v1 (workspace_id, bucket, event_id, event, time, inserted_at, source, description, actor_type, actor_id, actor_name, actor_meta, remote_ip, user_agent, meta, `targets.type`, `targets.id`, `targets.name`, `targets.meta`, correlation_id) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
			workspaceID, "integration", event.id, "integration.test", event.insertedAt, event.insertedAt, "platform", "integration test event", "user", "actor_1", "Integration Tester", event.actorMeta, "127.0.0.1", "integration-test", `{}`, event.targetTypes, event.targetIDs, event.targetNames, event.targetMetas, event.correlationID)
		require.NoError(t, err)
	}
}

func seedDrain(t *testing.T, database *sql.DB, workspaceID, drainID, url string, offset int64, encrypted ...string) {
	t.Helper()
	var encryptedSecret string
	if len(encrypted) > 0 {
		encryptedSecret = encrypted[0]
	}
	var headers []*logdrainv1.HttpHeader
	if encryptedSecret != "" {
		headers = append(headers, &logdrainv1.HttpHeader{
			Name:           "Authorization",
			EncryptedValue: encryptedSecret,
		})
	}
	config := &logdrainv1.Config{Destination: &logdrainv1.Config_Http{Http: &logdrainv1.HttpConfig{
		Url:     url,
		Format:  logdrainv1.HttpBodyFormat_HTTP_BODY_FORMAT_JSON,
		Headers: headers,
	}}}
	encoded, err := proto.Marshal(config)
	require.NoError(t, err)
	createdAt := time.Now().UnixMilli()
	_, err = database.Exec("INSERT INTO logdrains (id, workspace_id, name, stream, config, enabled, created_at) VALUES (?, ?, ?, 'audit_logs', ?, true, ?)", drainID, workspaceID, "integration test", encoded, createdAt)
	require.NoError(t, err)
	_, err = database.Exec("INSERT INTO logdrain_state (logdrain_id, committed_offset_inserted_at, lease_id, fencing_token, lease_expires_at) VALUES (?, ?, '', '', 0)", drainID, offset)
	require.NoError(t, err)
}

func cleanupDrain(t *testing.T, database *sql.DB, drainID string) {
	t.Helper()
	t.Cleanup(func() {
		_, err := database.Exec("DELETE FROM logdrain_state WHERE logdrain_id = ?", drainID)
		require.NoError(t, err)
		_, err = database.Exec("DELETE FROM logdrains WHERE id = ?", drainID)
		require.NoError(t, err)
	})
}

// startEngine wires a full engine against the test containers. The ClickHouse
// DSN must use the HTTP transport (ClickHouseConfig.HTTPDSN): the source query
// binds Int64 server-side parameters, which ClickHouse 25.6 rejects over the
// native protocol ("Cannot parse quoted string") but accepts over HTTP. The
// deployed service configures an http:// URL as well.
func startEngine(t *testing.T, mysqlDSN, clickhouseDSN string) *collector {
	t.Helper()
	return startEngineConcurrent(t, mysqlDSN, clickhouseDSN, 1)
}

func startEngineWithBatchSize(t *testing.T, mysqlDSN, clickhouseDSN string, batchSize int) *collector {
	t.Helper()
	return startEngineConfiguredWithBatchSize(t, mysqlDSN, clickhouseDSN, 1, 0, batchSize)
}

// startEngineConcurrent starts an engine whose poll cycle may process up to
// maxConcurrentDrains drains in parallel.
func startEngineConcurrent(t *testing.T, mysqlDSN, clickhouseDSN string, maxConcurrentDrains int) *collector {
	t.Helper()
	return startEngineConfigured(t, mysqlDSN, clickhouseDSN, maxConcurrentDrains, 0)
}

// startEngineConfigured starts an engine with explicit worker and queue bounds.
func startEngineConfigured(t *testing.T, mysqlDSN, clickhouseDSN string, maxConcurrentDrains, workQueueSize int) *collector {
	t.Helper()
	return startEngineConfiguredWithBatchSize(t, mysqlDSN, clickhouseDSN, maxConcurrentDrains, workQueueSize, 100)
}

// startEngineConfiguredWithBatchSize starts the lease and delivery services
// with explicit worker, queue, and batch bounds.
func startEngineConfiguredWithBatchSize(t *testing.T, mysqlDSN, clickhouseDSN string, maxConcurrentDrains, workQueueSize, batchSize int) *collector {
	t.Helper()
	database, err := db.New(mysqlDSN, sqlcomment.ForService("logdrain-integration-test", "test"))
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, database.Close()) })
	leaseID := uid.New("")
	leaseService, err := lease.New(lease.Config{DB: database, LeaseID: leaseID})
	require.NoError(t, err)
	chClient, err := clickhouse.New(clickhouse.Config{URL: clickhouseDSN})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, chClient.Close()) })
	deliveries := &collector{}
	eng, err := engine.New(engine.Config{
		DB:                          database,
		LeaseID:                     leaseID,
		Source:                      source.NewAuditLogs(chClient),
		Vault:                       stubVault{},
		Deliveries:                  deliveries,
		PollInterval:                200 * time.Millisecond,
		WatermarkLag:                0,
		BatchSize:                   batchSize,
		PauseThreshold:              5,
		MaxConcurrentDrains:         maxConcurrentDrains,
		WorkQueueSize:               workQueueSize,
		UnsafeAllowPrivateEndpoints: true,
	})
	require.NoError(t, err)
	ctx, cancel := context.WithCancel(context.Background())
	leaseDone := make(chan error, 1)
	engineDone := make(chan error, 1)
	go func() { leaseDone <- leaseService.Run(ctx) }()
	go func() { engineDone <- eng.Run(ctx) }()
	t.Cleanup(func() {
		cancel()
		require.NoError(t, <-engineDone)
		require.NoError(t, <-leaseDone)
	})
	return deliveries
}

func readDrainState(t *testing.T, database *sql.DB, drainID string) drainState {
	t.Helper()
	return readDrainStateCollect(t, database, drainID)
}

func readDrainStateCollect(t require.TestingT, database *sql.DB, drainID string) drainState {
	var state drainState
	err := database.QueryRow("SELECT status, committed_offset_inserted_at, consecutive_failures FROM logdrain_state WHERE logdrain_id = ?", drainID).Scan(&state.status, &state.committedOffsetInsertedAt, &state.consecutiveFailures)
	require.NoError(t, err)
	return state
}

func hasDelivery(deliveries []schema.LogdrainDeliveryV1, drainID, outcome string, events int64) bool {
	for _, delivery := range deliveries {
		if delivery.DrainID == drainID && delivery.Outcome == outcome && delivery.Events == events {
			return true
		}
	}
	return false
}

// decodeDeliveredEvents parses the default HTTP drain body: one JSON array of
// {"event":...,"timestamp":...} objects.
func decodeDeliveredEvents(body []byte) ([]map[string]any, error) {
	var lines []struct {
		Event map[string]any `json:"event"`
	}
	if err := json.Unmarshal(body, &lines); err != nil {
		return nil, fmt.Errorf("decode JSON body: %w", err)
	}
	events := make([]map[string]any, 0, len(lines))
	for _, line := range lines {
		if line.Event == nil {
			return nil, errors.New("delivered event is not an object")
		}
		events = append(events, line.Event)
	}
	return events, nil
}

func parseSuccessfulRequests(requests []capturedRequest) (map[string]struct{}, map[string]map[string]any, error) {
	ids := make(map[string]struct{})
	events := make(map[string]map[string]any)
	for _, request := range requests {
		if request.responseStatus < 200 || request.responseStatus >= 300 {
			continue
		}
		delivered, err := decodeDeliveredEvents(request.body)
		if err != nil {
			return nil, nil, err
		}
		for _, event := range delivered {
			id, ok := event["id"].(string)
			if !ok {
				return nil, nil, errors.New("delivered event id is not a string")
			}
			ids[id] = struct{}{}
			events[id] = event
		}
	}
	return ids, events, nil
}

func successfulRequestIDCounts(requests []capturedRequest) (map[string]int, error) {
	counts := make(map[string]int)
	for _, request := range requests {
		if request.responseStatus < 200 || request.responseStatus >= 300 {
			continue
		}
		delivered, err := decodeDeliveredEvents(request.body)
		if err != nil {
			return nil, err
		}
		for _, event := range delivered {
			id, ok := event["id"].(string)
			if !ok {
				return nil, errors.New("delivered event id is not a string")
			}
			counts[id]++
		}
	}
	return counts, nil
}
