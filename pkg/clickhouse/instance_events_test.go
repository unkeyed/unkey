package clickhouse_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	ch "github.com/ClickHouse/clickhouse-go/v2"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/clickhouse/schema"
	"github.com/unkeyed/unkey/pkg/testutil/containers"
)

// TestInstanceEventsInsertRoundTrip verifies that an InstanceEventV1 row
// constructed by ctrl can be written to default.instance_events_raw_v1 via
// PrepareBatch + AppendStruct (the same code path NewBuffer uses) and read
// back without losing fields. The JSON `attributes` column is the focus —
// earlier iterations of this feature flushed silently when AppendStruct
// couldn't map a Go map[string]string into a JSON column. The struct now
// stores a pre-serialized JSON string; this test pins that contract.
func TestInstanceEventsInsertRoundTrip(t *testing.T) {
	t.Parallel()

	cfg := containers.ClickHouse(t)
	opts, err := ch.ParseDSN(cfg.DSN)
	require.NoError(t, err)

	conn, err := ch.Open(opts)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, conn.Close()) })

	ctx := context.Background()
	require.NoError(t, conn.Ping(ctx))

	now := time.Now().UnixMilli()

	// Two rows: one with a populated attributes map (the OOMKilled case the
	// dashboard renders alongside the exit reason), and one with an empty
	// "{}" attributes string (the path that failed silently before because
	// "" is not valid JSON for the column).
	attrs := map[string]string{
		"image":                  "ghcr.io/test/app:v1",
		"image_id":               "ghcr.io/test/app@sha256:abc123",
		"build_id":               "bld_42",
		"memory_limit_mib":       "128",
		"cpu_limit_millicores":   "500",
		"memory_request_mib":     "64",
		"cpu_request_millicores": "100",
	}
	encodedAttrs, err := json.Marshal(attrs)
	require.NoError(t, err)

	rows := []schema.InstanceEventV1{
		{
			Time:             now,
			WorkspaceID:      "ws_test",
			ProjectID:        "proj_test",
			AppID:            "app_test",
			EnvironmentID:    "env_test",
			DeploymentID:     "dep_test_1",
			PodUID:           "pod-uid-1",
			PodName:          "pod-name-1",
			NodeName:         "node-1",
			ContainerName:    "app",
			ContainerID:      "containerd://abc",
			RestartCount:     3,
			EventKind:        "terminated",
			ExitCode:         137,
			Signal:           9,
			Reason:           "OOMKilled",
			Message:          "killed by kernel",
			Region:           "local",
			Platform:         "local",
			EventFingerprint: "deadbeef",
			Attributes:       string(encodedAttrs),
		},
		{
			Time:             now,
			WorkspaceID:      "ws_test",
			ProjectID:        "proj_test",
			AppID:            "app_test",
			EnvironmentID:    "env_test",
			DeploymentID:     "dep_test_2",
			PodUID:           "pod-uid-2",
			PodName:          "pod-name-2",
			NodeName:         "node-1",
			ContainerName:    "app",
			RestartCount:     0,
			EventKind:        "waiting",
			Reason:           "CrashLoopBackOff",
			Region:           "local",
			Platform:         "local",
			EventFingerprint: "feedface",
			// Empty map → "{}", the contract for the JSON column. An
			// empty Go string "" would be rejected by ClickHouse as
			// invalid JSON; "{}" is what marshalAttributes returns.
			Attributes: "{}",
		},
		{
			// 'running' rows carry no exit metadata — only identity +
			// time + attributes. Pinned here so the lifecycle-divider
			// path the dashboard depends on stays insertable end-to-end.
			Time:             now,
			WorkspaceID:      "ws_test",
			ProjectID:        "proj_test",
			AppID:            "app_test",
			EnvironmentID:    "env_test",
			DeploymentID:     "dep_test_3",
			PodUID:           "pod-uid-3",
			PodName:          "pod-name-3",
			NodeName:         "node-1",
			ContainerName:    "app",
			ContainerID:      "containerd://def",
			RestartCount:     0,
			EventKind:        "running",
			Region:           "local",
			Platform:         "local",
			EventFingerprint: "cafebabe",
			Attributes:       "{}",
		},
	}

	batch, err := conn.PrepareBatch(ctx, "INSERT INTO default.instance_events_raw_v1")
	require.NoError(t, err, "PrepareBatch must succeed against a JSON-column table")
	for i := range rows {
		require.NoError(t, batch.AppendStruct(&rows[i]),
			"AppendStruct must accept the row containing JSON-as-string attributes")
	}
	require.NoError(t, batch.Send(), "Send must succeed; this is the call that silently dropped events before")

	// Read back and assert every column round-tripped. The MV merging is
	// async but a fresh row should be queryable immediately on the source
	// table.
	t.Run("OOMKilled row carries the populated attributes map", func(t *testing.T) {
		t.Parallel()
		// JSON columns use the JSONExtractString function for keyed access
		// from a SQL row. Direct dot-notation (attributes.image) works in
		// CH SELECT but lands as a Dynamic value the Go driver doesn't
		// scan cleanly into string vars; routing through
		// JSONExtractString(toJSONString(...)) keeps the receiver a plain
		// String which scan handles.
		var (
			gotKind, gotReason, gotImage, gotImageID, gotBuildID, gotMemLimit string
			gotExit                                                           int32
		)
		err := conn.QueryRow(ctx, `
			SELECT
				event_kind,
				reason,
				exit_code,
				JSONExtractString(toJSONString(attributes), 'image') AS image,
				JSONExtractString(toJSONString(attributes), 'image_id') AS image_id,
				JSONExtractString(toJSONString(attributes), 'build_id') AS build_id,
				JSONExtractString(toJSONString(attributes), 'memory_limit_mib') AS memory_limit_mib
			FROM default.instance_events_raw_v1
			WHERE deployment_id = ?`, "dep_test_1").
			Scan(&gotKind, &gotReason, &gotExit, &gotImage, &gotImageID, &gotBuildID, &gotMemLimit)
		require.NoError(t, err)
		require.Equal(t, "terminated", gotKind)
		require.Equal(t, "OOMKilled", gotReason)
		require.Equal(t, int32(137), gotExit)
		require.Equal(t, "ghcr.io/test/app:v1", gotImage)
		require.Equal(t, "ghcr.io/test/app@sha256:abc123", gotImageID)
		require.Equal(t, "bld_42", gotBuildID)
		require.Equal(t, "128", gotMemLimit)
	})

	t.Run("waiting row carries kubelet reason in the reason column", func(t *testing.T) {
		t.Parallel()
		var gotKind, gotReason string
		err := conn.QueryRow(ctx, `
			SELECT event_kind, reason
			FROM default.instance_events_raw_v1
			WHERE deployment_id = ?`, "dep_test_2").Scan(&gotKind, &gotReason)
		require.NoError(t, err)
		require.Equal(t, "waiting", gotKind)
		require.Equal(t, "CrashLoopBackOff", gotReason)
	})

	t.Run("running row carries no exit metadata", func(t *testing.T) {
		t.Parallel()
		// Running events land with empty/zero exit fields; the dashboard
		// relies on event_kind alone to render the divider.
		var (
			gotKind, gotReason string
			gotExit            int32
		)
		err := conn.QueryRow(ctx, `
			SELECT event_kind, reason, exit_code
			FROM default.instance_events_raw_v1
			WHERE deployment_id = ?`, "dep_test_3").Scan(&gotKind, &gotReason, &gotExit)
		require.NoError(t, err)
		require.Equal(t, "running", gotKind)
		require.Equal(t, "", gotReason)
		require.Equal(t, int32(0), gotExit)
	})

	t.Run("attributes_text materialized column reflects the JSON", func(t *testing.T) {
		t.Parallel()
		// attributes_text is built via toJSONString(attributes); confirms
		// the JSON column actually parsed the input rather than silently
		// dropping it. Round-trip through json.Unmarshal so the assertion
		// doesn't depend on ClickHouse's preferred slash-escaping in the
		// stringified form (it emits ghcr.io\/test, json.Marshal-style).
		var got string
		err := conn.QueryRow(ctx, `
			SELECT attributes_text FROM default.instance_events_raw_v1
			WHERE deployment_id = ?`, "dep_test_1").Scan(&got)
		require.NoError(t, err)

		var parsed map[string]string
		require.NoError(t, json.Unmarshal([]byte(got), &parsed),
			"materialized attributes_text must be valid JSON: %s", got)
		require.Equal(t, "ghcr.io/test/app:v1", parsed["image"])
		require.Equal(t, "bld_42", parsed["build_id"])
	})
}
