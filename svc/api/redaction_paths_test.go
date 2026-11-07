package api

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/redaction"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

// Startup refuses to serve when the bundled spec yields no paths, so a spec edit
// that breaks path extraction must fail here instead of in production.
func TestBundledSpecYieldsRedactionPaths(t *testing.T) {
	t.Parallel()

	paths, err := redaction.PathsFromSpec(openapi.Spec)
	require.NoError(t, err)
	require.NotEmpty(t, paths)
	require.Contains(t, paths, "key")
}
