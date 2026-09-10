package source_test

import (
	"encoding/json"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	logdrainv1 "github.com/unkeyed/unkey/gen/proto/logdrain/v1"
	"github.com/unkeyed/unkey/pkg/clickhouse"
	"github.com/unkeyed/unkey/pkg/testutil/containers"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/logdrain/internal/source"
)

func TestGatewayRequestsRead_Payload(t *testing.T) {
	cfg := containers.ClickHouse(t)
	client, err := clickhouse.New(clickhouse.Config{URL: cfg.HTTPDSN})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, client.Close()) })
	workspaceID := uid.New("workspace")
	now := time.Now().UnixMilli()
	require.NoError(t, client.Conn().Exec(t.Context(), `INSERT INTO frontline_requests_raw_v1
		(workspace_id, request_id, time, project_id, app_id, environment_id, deployment_id,
		region, method, host, path, response_status, total_latency, instance_latency, gateway_latency,
		instance_address, request_headers, request_body, query_string, ip_address)
		VALUES (?, 'req_1', ?, 'project_1', 'app_1', 'env_1', 'deployment_1',
		'eu-west-1', 'POST', 'api.example.com', '/orders', 201, 53, 41, 12,
		'10.0.0.1', ['Cookie: secret'], 'secret', 'token=secret', '192.0.2.1')`, workspaceID, now-3600000))
	events, cursor, err := source.NewGatewayRequests(client).Read(t.Context(), workspaceID, source.Cursor{Time: now - 1}, time.Now().UnixMilli()+1000, 10, nil)
	require.NoError(t, err)
	require.Len(t, events, 1)
	require.Equal(t, "gateway_requests", events[0].Stream)
	require.Equal(t, now-3600000, events[0].Time)
	require.GreaterOrEqual(t, cursor.Time, now)
	require.Equal(t, "req_1", cursor.EventID)
	encoded, err := json.Marshal(events[0].Payload)
	require.NoError(t, err)
	require.JSONEq(t, `{"request_id":"req_1","project_id":"project_1","app_id":"app_1","environment_id":"env_1","deployment_id":"deployment_1","region":"eu-west-1","method":"POST","host":"api.example.com","path":"/orders","response_status":201,"total_latency":53,"instance_latency":41,"gateway_latency":12}`, string(encoded))
}

func TestGatewayRequestsRead_FilteredCursorBounds(t *testing.T) {
	cfg := containers.ClickHouse(t)
	client, err := clickhouse.New(clickhouse.Config{URL: cfg.HTTPDSN})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, client.Close()) })
	workspace := uid.New("workspace")
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
		{uid.New("workspace"), "h", now, 400},
	} {
		require.NoError(t, client.Conn().Exec(t.Context(), `INSERT INTO frontline_requests_raw_v1
			(workspace_id, request_id, inserted_at, time, response_status) VALUES (?, ?, ?, ?, ?)`, row.workspace, row.id, row.insertedAt, now-60000, row.status))
	}
	reader := source.NewGatewayRequests(client)
	from := source.Cursor{Time: now, EventID: "a"}
	filter := &logdrainv1.Config{Stream: &logdrainv1.Config_GatewayRequests{GatewayRequests: &logdrainv1.GatewayRequestStreamConfig{StatusClasses: []int32{4, 5}}}}
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
	workspace := uid.New("workspace")
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
		StatusClasses: []int32{5}, ProjectIds: []string{"project"}, AppIds: []string{"app", "app2"}, EnvironmentIds: []string{"env", "env2"},
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
	for _, status := range []int32{-1, 0, 1, 6, 200, 503} {
		t.Run(strconv.Itoa(int(status)), func(t *testing.T) {
			from := source.Cursor{Time: 123, EventID: "retained"}
			filter := &logdrainv1.Config{Stream: &logdrainv1.Config_GatewayRequests{GatewayRequests: &logdrainv1.GatewayRequestStreamConfig{StatusClasses: []int32{status}}}}
			events, next, err := source.NewGatewayRequests(nil).Read(t.Context(), "workspace", from, 456, 10, filter)
			require.ErrorContains(t, err, "status class must be between 2 and 5")
			require.Nil(t, events)
			require.Equal(t, from, next)
		})
	}
}
