package apps

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/cmd/api/internal/testutil"
	"github.com/unkeyed/unkey/pkg/cli"
)

func TestCreateApp(t *testing.T) {
	tests := []struct {
		name, args string
		want       map[string]any
	}{
		{"with git", `apps create-app --project=payments --name=Payments --slug=payments-api --git='{"repository":"unkeyed/api","defaultBranch":"main"}'`, map[string]any{"project": "payments", "name": "Payments", "slug": "payments-api", "git": map[string]any{"repository": "unkeyed/api", "defaultBranch": "main"}}},
		{"with OCI", `apps create-app --project=payments --name=Payments --slug=payments-api --oci='{"image":"ghcr.io/acme/payments:v1.2.3"}'`, map[string]any{"project": "payments", "name": "Payments", "slug": "payments-api", "oci": map[string]any{"image": "ghcr.io/acme/payments:v1.2.3"}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, testutil.CaptureRequest[map[string]any](t, Cmd(), tt.args))
		})
	}
}

func TestCreateAppRejectsMultipleSources(t *testing.T) {
	root := &cli.Command{Name: "unkey", Commands: []*cli.Command{Cmd()}}
	err := root.Run(context.Background(), strings.Fields(`unkey apps create-app --project=payments --name=Payments --slug=payments-api --git={} --oci={}`))
	require.ErrorContains(t, err, "exactly one of --git or --oci is required")
}
