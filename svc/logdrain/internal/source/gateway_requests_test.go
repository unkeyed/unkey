package source_test

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"
	"testing"
	"time"

	ch "github.com/ClickHouse/clickhouse-go/v2"
	"github.com/stretchr/testify/require"
	logdrainv1 "github.com/unkeyed/unkey/gen/proto/logdrain/v1"
	"github.com/unkeyed/unkey/pkg/clickhouse"
	"github.com/unkeyed/unkey/pkg/clickhouse/schema"
	"github.com/unkeyed/unkey/pkg/testutil/containers"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/logdrain/internal/source"
	"github.com/unkeyed/unkey/svc/logdrain/sink"
	"github.com/unkeyed/unkey/svc/logdrain/sink/axiom"
	"github.com/unkeyed/unkey/svc/logdrain/sink/httpdrain"
)

func TestGatewayRequestsRead_ByteBoundedPrefix(t *testing.T) {
	cfg := containers.ClickHouse(t)
	client, err := clickhouse.New(clickhouse.Config{URL: cfg.HTTPDSN})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, client.Close()) })
	workspace := uid.New(uid.WorkspacePrefix)
	now := time.Now().UnixMilli()
	body := strings.Repeat("<", 1<<20)
	for _, row := range []struct{ id, request, response string }{
		{"a", body, body}, {"b", body, ""}, {"c", "tail", ""},
	} {
		require.NoError(t, client.Conn().Exec(t.Context(), `INSERT INTO frontline_requests_raw_v1
			(workspace_id, request_id, inserted_at, time, request_body, response_body)
			VALUES (?, ?, ?, ?, ?, ?)`, workspace, row.id, now, now, row.request, row.response))
	}
	reader := source.NewGatewayRequests(client)
	queryID := uid.New("query")
	ctx := ch.Context(t.Context(), ch.WithQueryID(queryID), ch.WithSettings(ch.Settings{"log_queries": 1}))
	events, next, err := reader.Read(ctx, workspace, source.Cursor{Time: now}, now+1, 10_000, nil)
	require.NoError(t, err)
	require.Len(t, events, 1)
	require.Equal(t, source.Cursor{Time: now, EventID: "a"}, next)
	payload, ok := events[0].Payload.(sink.GatewayRequestPayload)
	require.True(t, ok)
	require.Equal(t, body, payload.Request.Body)
	require.Equal(t, body, payload.Response.Body)
	httpSink, err := httpdrain.New(httpdrain.Config{
		Endpoint: "https://example.com/logs",
		Format:   logdrainv1.HttpBodyFormat_HTTP_BODY_FORMAT_JSON,
	})
	require.NoError(t, err)
	axiomSink, err := axiom.New(axiom.Config{
		Dataset: "test",
		Token:   "test",
	})
	require.NoError(t, err)
	deliveryCtx, cancel := context.WithCancel(t.Context())
	cancel()
	for _, destination := range []sink.Sink{httpSink, axiomSink} {
		result, err := destination.Deliver(deliveryCtx, sink.Batch{
			Events: events,
		})
		require.ErrorIs(t, err, context.Canceled)
		require.Positive(t, result.RequestBodyBytes)
		require.Less(t, result.RequestBodyBytes, int64(16<<20))
	}

	// Server result counters distinguish a bounded fetch from truncation after Select.
	require.NoError(t, client.Conn().Exec(t.Context(), "SYSTEM FLUSH LOGS"))
	var resultRows, resultBytes uint64
	require.NoError(t, client.Conn().QueryRow(t.Context(), `SELECT result_rows, result_bytes
		FROM system.query_log WHERE query_id = ? AND type = 'QueryFinish'`, queryID).Scan(&resultRows, &resultBytes))
	require.Equal(t, uint64(1), resultRows)
	require.Less(t, resultBytes, uint64(16<<20))

	events, next, err = reader.Read(t.Context(), workspace, next, now+1, 10_000, nil)
	require.NoError(t, err)
	require.Len(t, events, 2)
	require.Equal(t, "b", events[0].EventID)
	require.Equal(t, "c", events[1].EventID)
	require.Equal(t, source.Cursor{Time: now, EventID: "c"}, next)
	events, final, err := reader.Read(t.Context(), workspace, next, now+1, 10_000, nil)
	require.NoError(t, err)
	require.Empty(t, events)
	require.Equal(t, next, final)
}

func TestGatewayRequestsRead_OversizedEventBlocksCursor(t *testing.T) {
	cfg := containers.ClickHouse(t)
	client, err := clickhouse.New(clickhouse.Config{URL: cfg.HTTPDSN})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, client.Close()) })
	workspace := uid.New(uid.WorkspacePrefix)
	now := time.Now().UnixMilli()
	for _, row := range []struct {
		id    string
		bytes int
	}{{"a", 5}, {"b", 3 << 20}, {"c", 7}} {
		require.NoError(t, client.Conn().Exec(t.Context(), `INSERT INTO frontline_requests_raw_v1
			(workspace_id, request_id, inserted_at, time, query_params)
			VALUES (?, ?, ?, ?, map('captured', [?]))`, workspace, row.id, now, now, strings.Repeat("x", row.bytes)))
	}
	reader := source.NewGatewayRequests(client)
	events, next, err := reader.Read(t.Context(), workspace, source.Cursor{Time: now}, now+1, 10_000, nil)
	require.NoError(t, err)
	require.Equal(t, 1, len(events))
	require.Equal(t, source.Cursor{Time: now, EventID: "a"}, next)
	for range 2 {
		queryID := uid.New("query")
		ctx := ch.Context(t.Context(), ch.WithQueryID(queryID), ch.WithSettings(ch.Settings{"log_queries": 1}))
		events, cursor, err := reader.Read(ctx, workspace, next, now+1, 10_000, nil)
		require.ErrorContains(t, err, "gateway cursor group exceeds batch limits")
		require.Nil(t, events)
		require.Equal(t, next, cursor)
		require.NoError(t, client.Conn().Exec(t.Context(), "SYSTEM FLUSH LOGS"))
		var resultBytes uint64
		require.NoError(t, client.Conn().QueryRow(t.Context(), `SELECT sum(result_bytes)
			FROM system.query_log WHERE query_id = ? AND type = 'QueryFinish'`, queryID).Scan(&resultBytes))
		require.Less(t, resultBytes, uint64(64<<10))
	}
}

func TestGatewayRequestsRead_KeepsCursorGroupsTogether(t *testing.T) {
	for _, tc := range []struct {
		name  string
		limit int
		bytes int
	}{
		{"row boundary", 2, 1},
		{"byte boundary", 10_000, 1 << 20},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg := containers.ClickHouse(t)
			client, err := clickhouse.New(clickhouse.Config{URL: cfg.HTTPDSN})
			require.NoError(t, err)
			t.Cleanup(func() { require.NoError(t, client.Close()) })
			workspace := uid.New(uid.WorkspacePrefix)
			now := time.Now().UnixMilli()
			for i, id := range []string{"a", "b", "b", "c"} {
				require.NoError(t, client.Conn().Exec(t.Context(), `INSERT INTO frontline_requests_raw_v1
					(workspace_id, request_id, inserted_at, time, request_body, path)
					VALUES (?, ?, ?, ?, ?, ?)`, workspace, id, now, now, strings.Repeat("x", tc.bytes), strconv.Itoa(i)))
			}
			reader := source.NewGatewayRequests(client)
			events, next, err := reader.Read(t.Context(), workspace, source.Cursor{Time: now}, now+1, tc.limit, nil)
			require.NoError(t, err)
			require.Equal(t, 1, len(events))
			require.Equal(t, source.Cursor{Time: now, EventID: "a"}, next)
			events, next, err = reader.Read(t.Context(), workspace, next, now+1, tc.limit, nil)
			require.NoError(t, err)
			require.Equal(t, 2, len(events))
			require.Equal(t, "b", events[0].EventID)
			require.Equal(t, "b", events[1].EventID)
			require.Equal(t, source.Cursor{Time: now, EventID: "b"}, next)
			events, _, err = reader.Read(t.Context(), workspace, next, now+1, tc.limit, nil)
			require.NoError(t, err)
			require.Equal(t, 1, len(events))
			require.Equal(t, "c", events[0].EventID)
			events, unchanged, err := reader.Read(t.Context(), workspace, source.Cursor{Time: now, EventID: "a"}, now+1, 1, nil)
			require.ErrorContains(t, err, "gateway cursor group exceeds batch limits")
			require.Nil(t, events)
			require.Equal(t, source.Cursor{Time: now, EventID: "a"}, unchanged)
		})
	}
}

func TestGatewayRequestsRead_Payload(t *testing.T) {
	cfg := containers.ClickHouse(t)
	client, err := clickhouse.New(clickhouse.Config{URL: cfg.HTTPDSN})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, client.Close()) })
	workspaceID := uid.New(uid.WorkspacePrefix)
	requestID := uid.New(uid.RequestPrefix)
	projectID := uid.New(uid.ProjectPrefix)
	appID := uid.New(uid.AppPrefix)
	environmentID := uid.New(uid.EnvironmentPrefix)
	deploymentID := uid.New(uid.DeploymentPrefix)
	requestBody, err := json.Marshal(map[string]string{"input": "[REDACTED]"})
	require.NoError(t, err)
	responseBody, err := json.Marshal(map[string]bool{"ok": true})
	require.NoError(t, err)
	now := time.Now().UnixMilli()
	batch, err := client.Conn().PrepareBatch(t.Context(), clickhouse.InsertQuery[schema.FrontlineRequest]())
	require.NoError(t, err)
	require.NoError(t, batch.AppendStruct(&schema.FrontlineRequest{
		RequestID:       requestID,
		Time:            now - 3600000,
		WorkspaceID:     workspaceID,
		ProjectID:       projectID,
		AppID:           appID,
		EnvironmentID:   environmentID,
		DeploymentID:    deploymentID,
		InstanceAddress: "10.0.0.1",
		Region:          "eu-west-1",
		Method:          "POST",
		Host:            "api.example.com",
		Path:            "/orders",
		QueryString:     "tag=a&tag=b",
		QueryParams:     map[string][]string{"tag": {"a", "b"}},
		RequestHeaders:  []string{"Authorization: [REDACTED]", "X-Custom: value"},
		RequestBody:     string(requestBody),
		ResponseStatus:  201,
		ResponseHeaders: []string{"Content-Type: application/json"},
		ResponseBody:    string(responseBody),
		UserAgent:       "test-agent",
		IPAddress:       "192.0.2.1",
		TotalLatency:    53,
		InstanceLatency: 41,
		GatewayLatency:  12,
	}))
	require.NoError(t, batch.Send())
	events, cursor, err := source.NewGatewayRequests(client).Read(t.Context(), workspaceID, source.Cursor{Time: now - 1}, time.Now().UnixMilli()+1000, 10, nil)
	require.NoError(t, err)
	require.Len(t, events, 1)
	require.Equal(t, "gateway_requests", events[0].Stream)
	require.Equal(t, now-3600000, events[0].Time)
	require.GreaterOrEqual(t, cursor.Time, now)
	require.Equal(t, requestID, cursor.EventID)
	require.Equal(t, sink.GatewayRequestPayload{
		RequestID:     requestID,
		ProjectID:     projectID,
		AppID:         appID,
		EnvironmentID: environmentID,
		DeploymentID:  deploymentID,
		Region:        "eu-west-1",
		Request: sink.GatewayRequest{
			Method:      "POST",
			Host:        "api.example.com",
			Path:        "/orders",
			QueryString: "tag=a&tag=b",
			QueryParams: map[string][]string{
				"tag": {"a", "b"},
			},
			Headers:   []string{"Authorization: [REDACTED]", "X-Custom: value"},
			Body:      string(requestBody),
			UserAgent: "test-agent",
			IPAddress: "192.0.2.1",
		},
		Response: sink.GatewayResponse{
			Status:  201,
			Headers: []string{"Content-Type: application/json"},
			Body:    string(responseBody),
		},
		Latency: sink.GatewayRequestLatency{
			Total:    53,
			Instance: 41,
			Gateway:  12,
		},
	}, events[0].Payload)
}

func TestGatewayRequestsRead_FilteredCursorBounds(t *testing.T) {
	cfg := containers.ClickHouse(t)
	client, err := clickhouse.New(clickhouse.Config{URL: cfg.HTTPDSN})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, client.Close()) })
	workspace := uid.New(uid.WorkspacePrefix)
	now := time.Now().UnixMilli()
	for _, row := range []struct {
		workspace, id string
		insertedAt    int64
		status        int32
	}{
		{workspace, "z", now - 1, 400},
		{workspace, "a", now, 400},
		{workspace, "b", now, 200},
		{workspace, "c", now, 401},
		{workspace, "d", now, 503},
		{workspace, "e", now, 302},
		{workspace, "f", now, 499},
		{workspace, "a", now + 1, 404},
		{workspace, "g", now + 2, 403},
		{uid.New(uid.WorkspacePrefix), "h", now, 400},
	} {
		require.NoError(t, client.Conn().Exec(t.Context(), `INSERT INTO frontline_requests_raw_v1
			(workspace_id, request_id, inserted_at, time, response_status) VALUES (?, ?, ?, ?, ?)`, row.workspace, row.id, row.insertedAt, now-60000, row.status))
	}
	reader := source.NewGatewayRequests(client)
	from := source.Cursor{Time: now, EventID: "a"}
	filter := &logdrainv1.Config{Stream: &logdrainv1.Config_GatewayRequests{GatewayRequests: &logdrainv1.GatewayRequestStreamConfig{StatusClasses: []logdrainv1.HttpStatusClass{logdrainv1.HttpStatusClass_HTTP_STATUS_CLASS_4XX, logdrainv1.HttpStatusClass_HTTP_STATUS_CLASS_5XX}}}}
	page, next, err := reader.Read(t.Context(), workspace, from, now+2, 2, filter)
	require.NoError(t, err)
	require.Len(t, page, 2)
	require.Equal(t, "c", page[0].EventID)
	require.Equal(t, "d", page[1].EventID)
	require.Equal(t, source.Cursor{Time: now, EventID: "d"}, next)
	page, next, err = reader.Read(t.Context(), workspace, next, now+2, 2, filter)
	require.NoError(t, err)
	require.Len(t, page, 2)
	require.Equal(t, "f", page[0].EventID)
	require.Equal(t, "a", page[1].EventID)
	require.Equal(t, source.Cursor{Time: now + 1, EventID: "a"}, next)
	page, final, err := reader.Read(t.Context(), workspace, next, now+2, 2, nil)
	require.NoError(t, err)
	require.Empty(t, page)
	require.Equal(t, next, final)
	page, _, err = reader.Read(t.Context(), workspace, from, now+2, 2, nil)
	require.NoError(t, err)
	require.Len(t, page, 2)
	require.Equal(t, "b", page[0].EventID)
}

func TestGatewayRequestsRead_ResourceFiltersBeforeLimit(t *testing.T) {
	cfg := containers.ClickHouse(t)
	client, err := clickhouse.New(clickhouse.Config{URL: cfg.HTTPDSN})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, client.Close()) })
	workspace := uid.New(uid.WorkspacePrefix)
	now := time.Now().UnixMilli()
	for _, row := range []struct {
		id, project, app, environment string
		status                        int32
	}{
		{"a", "other", "app", "env", 503},
		{"b", "project", "other", "env", 503},
		{"c", "project", "app", "other", 503},
		{"d", "project", "app", "env", 201},
		{"e", "project", "app", "env", 503},
		{"f", "project", "app2", "env2", 502},
	} {
		require.NoError(t, client.Conn().Exec(t.Context(), `INSERT INTO frontline_requests_raw_v1
			(workspace_id, request_id, inserted_at, time, project_id, app_id, environment_id, response_status)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)`, workspace, row.id, now, now, row.project, row.app, row.environment, row.status))
	}
	filter := &logdrainv1.Config{Stream: &logdrainv1.Config_GatewayRequests{GatewayRequests: &logdrainv1.GatewayRequestStreamConfig{
		StatusClasses: []logdrainv1.HttpStatusClass{logdrainv1.HttpStatusClass_HTTP_STATUS_CLASS_5XX}, ProjectIds: []string{"project"}, AppIds: []string{"app", "app2"}, EnvironmentIds: []string{"env", "env2"},
	}}}
	reader := source.NewGatewayRequests(client)
	page, next, err := reader.Read(t.Context(), workspace, source.Cursor{Time: now}, now+1, 1, filter)
	require.NoError(t, err)
	require.Len(t, page, 1)
	require.Equal(t, "e", page[0].EventID)
	page, _, err = reader.Read(t.Context(), workspace, next, now+1, 1, filter)
	require.NoError(t, err)
	require.Len(t, page, 1)
	require.Equal(t, "f", page[0].EventID)
	filter.GetGatewayRequests().ProjectIds = nil
	filter.GetGatewayRequests().AppIds = nil
	filter.GetGatewayRequests().EnvironmentIds = nil
	page, _, err = reader.Read(t.Context(), workspace, source.Cursor{Time: now}, now+1, 1, filter)
	require.NoError(t, err)
	require.Len(t, page, 1)
	require.Equal(t, "a", page[0].EventID)
}

func TestGatewayRequestsRead_RejectsInvalidStatuses(t *testing.T) {
	for _, status := range []logdrainv1.HttpStatusClass{-1, logdrainv1.HttpStatusClass_HTTP_STATUS_CLASS_UNSPECIFIED, 1, 6, 200, 503} {
		t.Run(strconv.Itoa(int(status)), func(t *testing.T) {
			from := source.Cursor{Time: 123, EventID: "retained"}
			filter := &logdrainv1.Config{Stream: &logdrainv1.Config_GatewayRequests{GatewayRequests: &logdrainv1.GatewayRequestStreamConfig{StatusClasses: []logdrainv1.HttpStatusClass{status}}}}
			events, next, err := source.NewGatewayRequests(nil).Read(t.Context(), "workspace", from, 456, 10, filter)
			require.ErrorContains(t, err, "status class must be between 2 and 5")
			require.Nil(t, events)
			require.Equal(t, from, next)
		})
	}
}
