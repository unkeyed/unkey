package portal

import (
	"context"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/cmd/api/util"
	"github.com/unkeyed/unkey/pkg/cli"
	"github.com/unkeyed/unkey/svc/api/openapi"
	"strings"
	"testing"
)

func TestCreateSession(t *testing.T) {
	tests := []struct {
		name, args string
		preview    bool
		count      int
	}{{"minimal", "portal create-session --slug=my-portal --external-id=u --permissions=keys:read", false, 1}, {"all flags", "portal create-session --slug=my-portal --external-id=u --permissions=keys:read,analytics:read --preview=true", true, 2}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := util.CaptureRequest[openapi.V2PortalCreateSessionRequestBody](t, Cmd(), tt.args)
			require.Equal(t, tt.preview, *got.Preview)
			require.Len(t, got.Permissions, tt.count)
		})
	}
}

func TestCreateSessionPermissionValidation(t *testing.T) {
	root := &cli.Command{Name: "unkey", Commands: []*cli.Command{Cmd()}}
	err := root.Run(context.Background(), strings.Fields("unkey portal create-session --slug=my-portal --external-id=u --permissions=keys:read,keys:delete --root-key=test"))
	require.ErrorContains(t, err, `invalid permission "keys:delete"`)
}
