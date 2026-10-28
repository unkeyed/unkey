package rootkeys

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/oapi-codegen/nullable"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/cmd/api/internal/testutil"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

func TestCreateKey(t *testing.T) {
	permissions := []string{"unkey:v1:" + uid.New(uid.WorkspacePrefix) + ":rootKeys/*#read"}
	for _, test := range []struct {
		name string
		args string
		want openapi.V2RootKeysCreateKeyRequestBody
	}{
		{"minimal", "--permissions=" + strings.Join(permissions, ","), openapi.V2RootKeysCreateKeyRequestBody{Permissions: permissions}},
		{"named with expiration", "--permissions=" + strings.Join(permissions, ",") + " --name=automation --expires=1900000000000", openapi.V2RootKeysCreateKeyRequestBody{Permissions: permissions, Name: new("automation"), Expires: nullable.NewNullableWithValue(int64(1900000000000))}},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := testutil.CaptureRequest[openapi.V2RootKeysCreateKeyRequestBody](t, Cmd(), "root-keys create-key "+test.args)
			require.Equal(t, test.want, got)
		})
	}

	want := openapi.V2RootKeysCreateKeyRequestBody{Permissions: []string{}, Expires: nullable.NewNullNullable[int64]()}
	body, err := json.Marshal(want)
	require.NoError(t, err)
	got := testutil.CaptureRequest[openapi.V2RootKeysCreateKeyRequestBody](t, Cmd(), "root-keys create-key --body='"+string(body)+"'")
	require.Equal(t, want, got)
}
