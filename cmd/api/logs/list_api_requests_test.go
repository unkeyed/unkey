package logs

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/cmd/api/internal/testutil"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

func TestListAPIRequests(t *testing.T) {
	filters := openapi.V2LogsListApiRequestsFilters{
		Time:      &openapi.TimeFilter{Gte: new(int64(1700000000000)), Lt: new(int64(1700003600000))},
		Path:      &openapi.TextFilter{StartsWith: new([]string{"/v2/keys."})},
		RequestId: &openapi.ExactFilter{Is: new([]string{uid.New(uid.RequestPrefix)})},
		Status:    &openapi.IntFilter{Gte: new(int64(400)), Lt: new(int64(600))},
	}
	encoded, err := json.Marshal(filters)
	require.NoError(t, err)
	for _, test := range []struct {
		name string
		args string
		want openapi.V2LogsListApiRequestsRequestBody
	}{
		{"defaults", "", openapi.V2LogsListApiRequestsRequestBody{Limit: new(50)}},
		{"filtered page", " --limit=7 --cursor=next-page --filters='" + string(encoded) + "'", openapi.V2LogsListApiRequestsRequestBody{Limit: new(7), Cursor: new("next-page"), Filters: &filters}},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := testutil.CaptureRequestWithData[openapi.V2LogsListApiRequestsRequestBody](t, Cmd(), "logs list-api-requests"+test.args, []openapi.ApiRequestLog{})
			require.Equal(t, test.want, got)
		})
	}
}
