package workspace

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/cmd/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

func TestGetUsage(t *testing.T) {
	period := openapi.V2WorkspaceGetUsageRequestPeriod{Year: 2026, Month: 9}
	encoded, err := json.Marshal(period)
	require.NoError(t, err)
	for _, test := range []struct {
		name string
		args string
		want openapi.V2WorkspaceGetUsageRequestBody
	}{
		{"current month", "", openapi.V2WorkspaceGetUsageRequestBody{}},
		{"past month", " --period='" + string(encoded) + "'", openapi.V2WorkspaceGetUsageRequestBody{Period: &period}},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := testutil.CaptureRequest[openapi.V2WorkspaceGetUsageRequestBody](t, Cmd(), "workspace get-usage"+test.args)
			require.Equal(t, test.want, got)
		})
	}
}
