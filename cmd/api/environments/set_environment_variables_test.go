package environments

import (
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/cmd/api/internal/testutil"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/svc/api/openapi"
	"testing"
)

func TestSetEnvironmentVariables(t *testing.T) {
	kind := openapi.EnvironmentVariableKind("writeonly")
	tests := []struct {
		name, args string
		want       openapi.V2EnvironmentsSetEnvironmentVariablesRequestBody
	}{{"variables", `environments set-environment-variables --project=p --app=a --environment=e --variables='[{"key":"TOKEN","value":"secret"}]'`, openapi.V2EnvironmentsSetEnvironmentVariablesRequestBody{Project: "p", App: "a", Environment: "e", Variables: []openapi.EnvironmentVariableInput{{Key: "TOKEN", Value: "secret", Kind: &kind}}, Prune: ptr.P(false)}}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := testutil.CaptureRequest[openapi.V2EnvironmentsSetEnvironmentVariablesRequestBody](t, Cmd(), tt.args)
			require.Equal(t, tt.want, got)
		})
	}
}
