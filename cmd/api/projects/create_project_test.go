package projects

import (
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/cmd/api/util"
	"github.com/unkeyed/unkey/svc/api/openapi"
	"testing"
)

func TestCreateProject(t *testing.T) {
	tests := []struct {
		name, args string
		want       openapi.V2ProjectsCreateProjectRequestBody
	}{{"all fields", "projects create-project --name=Payments --slug=payments", openapi.V2ProjectsCreateProjectRequestBody{Name: "Payments", Slug: "payments"}}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, util.CaptureRequest[openapi.V2ProjectsCreateProjectRequestBody](t, Cmd(), tt.args))
		})
	}
}
