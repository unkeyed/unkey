package rootkeys

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/cmd/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

func TestListKeys(t *testing.T) {
	for _, test := range []struct {
		name string
		args string
		want openapi.V2RootKeysListKeysRequestBody
	}{
		{"first page", "", openapi.V2RootKeysListKeysRequestBody{Limit: new(100)}},
		{"next page", " --limit=7 --cursor=next-page", openapi.V2RootKeysListKeysRequestBody{Limit: new(7), Cursor: new("next-page")}},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := testutil.CaptureRequestWithData[openapi.V2RootKeysListKeysRequestBody](t, Cmd(), "root-keys list-keys"+test.args, []openapi.V2RootKeysListKeysResponseData{})
			require.Equal(t, test.want, got)
		})
	}
}
