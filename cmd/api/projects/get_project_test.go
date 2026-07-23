package projects

import (
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/cmd/api/util"
	"github.com/unkeyed/unkey/svc/api/openapi"
	"testing"
)

func TestGetProject(t *testing.T) {
	tests := []struct {
		name, args string
		want       openapi.V2ProjectsGetProjectRequestBody
	}{{"by id", "projects get-project --project=proj_123", openapi.V2ProjectsGetProjectRequestBody{Project: "proj_123"}}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, util.CaptureRequest[openapi.V2ProjectsGetProjectRequestBody](t, Cmd(), tt.args))
		})
	}
}
