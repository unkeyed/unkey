package projects

import (
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/svc/api/openapi"
	"testing"
)

func TestDeleteProject(t *testing.T) {
	tests := []struct {
		name, args string
		want       openapi.V2ProjectsDeleteProjectRequestBody
	}{{"by slug", "projects delete-project --project=payments", openapi.V2ProjectsDeleteProjectRequestBody{Project: "payments"}}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, captureAcceptedRequest[openapi.V2ProjectsDeleteProjectRequestBody](t, Cmd(), tt.args))
		})
	}
}
