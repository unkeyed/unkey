package portal

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/sdks/api/go/v3/models/components"
	"github.com/unkeyed/unkey/cmd/api/internal/testutil"
)

func TestDeletePortal(t *testing.T) {
	want := components.V2PortalDeletePortalRequestBody{Portal: "acme-portal"}
	require.Equal(t, want, testutil.CaptureRequest[components.V2PortalDeletePortalRequestBody](t, Cmd(), "portal delete-portal --portal=acme-portal"))
}
