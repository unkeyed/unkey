package analytics

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/sdks/api/go/v3/models/components"
	"github.com/unkeyed/unkey/cmd/api/internal/testutil"
)

func TestGetRuntimeLogs(t *testing.T) {
	args := `analytics get-runtime-logs --query='SELECT * FROM runtime_logs_v1'`
	want := components.V2AnalyticsGetRuntimeLogsRequestBody{
		Query: "SELECT * FROM runtime_logs_v1",
	}

	got := testutil.CaptureRequestWithData[components.V2AnalyticsGetRuntimeLogsRequestBody](t, Cmd(), args, []any{})
	require.Equal(t, want, got)
}
