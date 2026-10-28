package rootkeys

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/cmd/api/internal/testutil"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

func TestDeleteKey(t *testing.T) {
	keyID := uid.New(uid.KeyPrefix)
	got := testutil.CaptureRequest[openapi.V2RootKeysDeleteKeyRequestBody](t, Cmd(), "root-keys delete-key --key-id="+keyID+" --yes")
	require.Equal(t, openapi.V2RootKeysDeleteKeyRequestBody{KeyId: keyID}, got)
}
