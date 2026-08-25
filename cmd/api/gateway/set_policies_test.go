package gateway

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/cmd/api/internal/testutil"
	"github.com/unkeyed/unkey/pkg/cli"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

func TestSetPolicies(t *testing.T) {
	tests := []struct {
		name, args string
		count      int
	}{{"minimal empty", "gateway set-policies --project=p --app=a --environment=e --policies=[]", 0}, {"all policy data", `gateway set-policies --project=p --app=a --environment=e --policies=[{"name":"deny","enabled":true,"firewall":{"action":"ACTION_DENY"}}]`, 1}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := testutil.CaptureRequest[openapi.V2GatewaySetPoliciesRequestBody](t, Cmd(), tt.args)
			require.Equal(t, "p", got.Project)
			require.Len(t, got.Policies, tt.count)
		})
	}
}

func TestSetPoliciesRejectsNull(t *testing.T) {
	root := &cli.Command{Name: "unkey", Commands: []*cli.Command{Cmd()}}
	err := root.Run(context.Background(), strings.Fields("unkey gateway set-policies --project=p --app=a --environment=e --policies=null --root-key=test"))
	require.ErrorContains(t, err, "must be a JSON array")
}
