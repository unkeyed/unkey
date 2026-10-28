package rootkeys

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/oapi-codegen/nullable"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/cmd/api/internal/testutil"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

func TestRerollKey(t *testing.T) {
	keyID := uid.New(uid.KeyPrefix)
	for _, expiration := range []int64{0, 86400000} {
		t.Run(fmt.Sprint(expiration), func(t *testing.T) {
			got := testutil.CaptureRequest[openapi.V2RootKeysRerollKeyRequestBody](t, Cmd(), fmt.Sprintf("root-keys reroll-key --key-id=%s --expiration=%d", keyID, expiration))
			require.Equal(t, openapi.V2RootKeysRerollKeyRequestBody{KeyId: keyID, Expiration: nullable.NewNullableWithValue(expiration)}, got)
		})
	}
	want := openapi.V2RootKeysRerollKeyRequestBody{KeyId: keyID, Expiration: nullable.NewNullNullable[int64]()}
	body, err := json.Marshal(want)
	require.NoError(t, err)
	got := testutil.CaptureRequest[openapi.V2RootKeysRerollKeyRequestBody](t, Cmd(), "root-keys reroll-key --body='"+string(body)+"'")
	require.Equal(t, want, got)
}
