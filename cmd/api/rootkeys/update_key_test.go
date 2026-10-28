package rootkeys

import (
	"encoding/json"
	"testing"

	"github.com/oapi-codegen/nullable"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/cmd/api/internal/testutil"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

func TestUpdateKey(t *testing.T) {
	keyID := uid.New(uid.KeyPrefix)
	permission := "unkey:v1:" + uid.New(uid.WorkspacePrefix) + ":rootKeys/*#read"
	for _, test := range []struct {
		name string
		args string
		want openapi.V2RootKeysUpdateKeyRequestBody
	}{
		{"omitted updates", "", openapi.V2RootKeysUpdateKeyRequestBody{KeyId: keyID}},
		{"disable", " --enabled=false", openapi.V2RootKeysUpdateKeyRequestBody{KeyId: keyID, Enabled: new(false)}},
		{"clear permissions", " --permissions=", openapi.V2RootKeysUpdateKeyRequestBody{KeyId: keyID, Permissions: new([]string{})}},
		{"replace permissions and name", " --name=automation --permissions=" + permission, openapi.V2RootKeysUpdateKeyRequestBody{KeyId: keyID, Name: nullable.NewNullableWithValue("automation"), Permissions: new([]string{permission})}},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := testutil.CaptureRequest[openapi.V2RootKeysUpdateKeyRequestBody](t, Cmd(), "root-keys update-key --key-id="+keyID+test.args)
			require.Equal(t, test.want, got)
		})
	}

	want := openapi.V2RootKeysUpdateKeyRequestBody{KeyId: keyID, Name: nullable.NewNullNullable[string](), Enabled: new(false), Permissions: new([]string{})}
	body, err := json.Marshal(want)
	require.NoError(t, err)
	got := testutil.CaptureRequest[openapi.V2RootKeysUpdateKeyRequestBody](t, Cmd(), "root-keys update-key --body='"+string(body)+"'")
	require.Equal(t, want, got)
}
