package apps

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/cmd/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

func TestListApps(t *testing.T) {
	tests := []struct {
		name, args string
		want       openapi.V2AppsListAppsRequestBody
	}{{"defaults", "apps list-apps --project=payments", openapi.V2AppsListAppsRequestBody{Project: "payments", Limit: new(100)}}, {"all options", "apps list-apps --project=payments --limit=25 --cursor=app_1 --search=checkout", openapi.V2AppsListAppsRequestBody{Project: "payments", Limit: new(25), Cursor: new("app_1"), Search: new("checkout")}}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, testutil.CaptureRequestWithData[openapi.V2AppsListAppsRequestBody](t, Cmd(), tt.args, []any{}))
		})
	}
}
