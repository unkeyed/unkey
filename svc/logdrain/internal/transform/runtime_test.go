package transform

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/clickhouse/schema"
	"github.com/unkeyed/unkey/svc/logdrain/internal/sinks"
)

func TestRuntime_BasicMapping(t *testing.T) {
	t.Parallel()

	row := schema.RuntimeLog{
		Time:          1700000000000,
		InsertedAt:    1700000000050,
		Severity:      "info",
		Message:       "ready to serve",
		WorkspaceID:   "ws_1",
		ProjectID:     "proj_2",
		EnvironmentID: "env_3",
		AppID:         "app_4",
		DeploymentID:  "dep_5",
		K8sPodName:    "deploy-pod-1",
		Region:        "us-east-1",
		Platform:      "aws",
		Attributes:    `{"http_status":200,"path":"/healthz"}`,
	}
	rec, ok := Runtime(row, RuntimeFilter{})
	require.True(t, ok)
	require.Equal(t, sinks.RecordRuntime, rec.Kind)
	require.Equal(t, "ready to serve", rec.Body)
	require.Equal(t, "info", rec.SeverityText)
	require.Equal(t, "ws_1", rec.WorkspaceID)
	require.Equal(t, "deploy-pod-1", rec.K8sPodName)
	require.Equal(t, "/healthz", rec.Attributes["path"])
}

func TestRuntime_EmptyAttributes_LeavesMapNil(t *testing.T) {
	t.Parallel()

	// Plain text logs come in with Attributes == "". The transform must not
	// allocate an empty map — sinks rely on the nil/empty distinction to
	// suppress attribute keys from the wire payload.
	row := schema.RuntimeLog{Severity: "info", Message: "hello"}
	rec, ok := Runtime(row, RuntimeFilter{})
	require.True(t, ok)
	require.Nil(t, rec.Attributes)
}

func TestRuntime_MalformedJSONAttributes_AreDropped(t *testing.T) {
	t.Parallel()

	// Unparseable attributes are dropped silently rather than failing the
	// whole record. The data plane already handled fallback parsing
	// upstream; anything reaching us is well-formed or empty, so a corrupt
	// row is exceptional and shouldn't stall the pipeline.
	row := schema.RuntimeLog{Severity: "info", Message: "hello", Attributes: "not json"}
	rec, ok := Runtime(row, RuntimeFilter{})
	require.True(t, ok)
	require.Nil(t, rec.Attributes)
}

func TestRuntime_SeverityFilter(t *testing.T) {
	t.Parallel()

	cases := []struct {
		min      string
		level    string
		wantPass bool
	}{
		{"", "info", true},
		{"", "debug", true},
		{"warn", "info", false},
		{"warn", "warn", true},
		{"warn", "error", true},
		{"error", "warn", false},
		{"error", "error", true},
	}
	for _, c := range cases {
		row := schema.RuntimeLog{Severity: c.level, Message: "x"}
		_, ok := Runtime(row, RuntimeFilter{MinSeverity: c.min})
		require.Equal(t, c.wantPass, ok, "min=%q level=%q", c.min, c.level)
	}
}
