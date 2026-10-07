package projects

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/cmd/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

func TestListProjects(t *testing.T) {
	tests := []struct {
		name, args string
		want       openapi.V2ProjectsListProjectsRequestBody
	}{{"defaults", "projects list-projects", openapi.V2ProjectsListProjectsRequestBody{Limit: new(100)}}, {"all options", "projects list-projects --limit=10 --cursor=proj_1 --search=billing", openapi.V2ProjectsListProjectsRequestBody{Limit: new(10), Cursor: new("proj_1"), Search: new("billing")}}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, testutil.CaptureRequestWithData[openapi.V2ProjectsListProjectsRequestBody](t, Cmd(), tt.args, []any{}))
		})
	}
}
