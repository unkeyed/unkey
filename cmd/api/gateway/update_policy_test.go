package gateway

import (
	"context"
	"strings"
	"testing"

	"github.com/oapi-codegen/nullable"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/cmd/api/util"
	"github.com/unkeyed/unkey/pkg/cli"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

func TestUpdatePolicy(t *testing.T) {
	base := "gateway update-policy --project=p --app=a --environment=e --policy-id=pol_1"
	tests := []struct {
		name, args string
		check      func(*testing.T, openapi.V2GatewayUpdatePolicyRequestBody)
	}{
		{"optional", base + ` --policy={"enabled":false}`, func(t *testing.T, got openapi.V2GatewayUpdatePolicyRequestBody) {
			require.NotNil(t, got.Enabled)
			require.False(t, *got.Enabled)
		}},
		{"name and keyauth", base + ` --policy={"name":"n","keyauth":{"keyspaces":["ks_1"]}}`, func(t *testing.T, got openapi.V2GatewayUpdatePolicyRequestBody) {
			require.NotNil(t, got.Name)
			require.Equal(t, "n", *got.Name)
			require.NotNil(t, got.Keyauth)
		}},
		{"clear match", base + ` --policy={"match":null}`, func(t *testing.T, got openapi.V2GatewayUpdatePolicyRequestBody) {
			require.Equal(t, nullable.NewNullableWithValue([]openapi.MatchExpr{}), got.Match)
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := util.CaptureRequest[openapi.V2GatewayUpdatePolicyRequestBody](t, Cmd(), tt.args)
			require.Equal(t, "pol_1", got.PolicyId)
			tt.check(t, got)
		})
	}
}

func TestUpdatePolicyRejectsInvalidUpdates(t *testing.T) {
	base := "unkey gateway update-policy --project=p --app=a --environment=e --policy-id=pol_1 --root-key=test"
	tests := []struct {
		name string
		args string
		want string
	}{
		{name: "no updates", args: base + " --policy={}", want: "at least one update field"},
		{name: "multiple rules", args: base + ` --policy={"keyauth":{"keyspaces":["ks_1"]},"firewall":{"action":"ACTION_DENY"}}`, want: "at most one of"},
		{name: "unknown field", args: base + ` --policy={"unknown":true}`, want: `unknown field "unknown"`},
		{name: "null name", args: base + ` --policy={"name":null,"enabled":true}`, want: "name in --policy must be a string, not null"},
		{name: "null enabled", args: base + ` --policy={"enabled":null,"name":"updated"}`, want: "enabled in --policy must be a boolean, not null"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := &cli.Command{Name: "unkey", Commands: []*cli.Command{Cmd()}}
			err := root.Run(context.Background(), strings.Fields(tt.args))
			require.ErrorContains(t, err, tt.want)
		})
	}
}
