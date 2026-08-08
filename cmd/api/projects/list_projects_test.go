package projects

import (
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/cmd/api/util"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/svc/api/openapi"
	"testing"
)

func TestListProjects(t *testing.T) {
	tests := []struct {
		name, args string
		want       openapi.V2ProjectsListProjectsRequestBody
	}{{"defaults", "projects list-projects", openapi.V2ProjectsListProjectsRequestBody{Limit: ptr.P(100)}}, {"all options", "projects list-projects --limit=10 --cursor=proj_1 --search=billing", openapi.V2ProjectsListProjectsRequestBody{Limit: ptr.P(10), Cursor: ptr.P("proj_1"), Search: ptr.P("billing")}}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, util.CaptureRequestWithResponse[openapi.V2ProjectsListProjectsRequestBody](t, Cmd(), tt.args, `{"meta":{"requestId":"test"},"data":[],"pagination":{"hasMore":false}}`))
		})
	}
}
