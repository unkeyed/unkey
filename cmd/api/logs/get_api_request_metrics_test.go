package logs

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/cmd/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

func TestGetAPIRequestMetrics(t *testing.T) {
	filters := openapi.V2LogsGetApiRequestMetricsFilters{
		Time:   &openapi.TimeFilter{Gte: new(int64(1700000000000)), Lt: new(int64(1700003600000))},
		Path:   &openapi.TextFilter{Is: new([]string{"/v2/keys.verifyKey"})},
		Status: &openapi.IntFilter{Gte: new(int64(400)), Lt: new(int64(600))},
	}
	encoded, err := json.Marshal(filters)
	require.NoError(t, err)
	for _, test := range []struct {
		name string
		args string
		want openapi.V2LogsGetApiRequestMetricsRequestBody
	}{
		{"no filter", "", openapi.V2LogsGetApiRequestMetricsRequestBody{}},
		{"filtered", " --filters='" + string(encoded) + "'", openapi.V2LogsGetApiRequestMetricsRequestBody{Filters: &filters}},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := testutil.CaptureRequestWithData[openapi.V2LogsGetApiRequestMetricsRequestBody](t, Cmd(), "logs get-api-request-metrics"+test.args, []openapi.ApiRequestMetricsPoint{})
			require.Equal(t, test.want, got)
		})
	}
}
