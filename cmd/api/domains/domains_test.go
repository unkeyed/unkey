package domains

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/sdks/api/go/v3/models/components"
	"github.com/unkeyed/unkey/cmd/api/internal/testutil"
	"github.com/unkeyed/unkey/pkg/cli"
	"github.com/unkeyed/unkey/pkg/ptr"
)

func TestCreateDomain(t *testing.T) {
	want := components.V2DomainsCreateDomainRequestBody{Project: "payments", App: "api", Environment: "production", Domain: "api.acme.com"}
	require.Equal(t, want, testutil.CaptureRequest[components.V2DomainsCreateDomainRequestBody](t, Cmd(), "domains create-domain --project=payments --app=api --environment=production --domain=api.acme.com"))
}

func TestDeleteDomain(t *testing.T) {
	want := components.V2DomainsDeleteDomainRequestBody{Domain: "dom_1234abcd"}
	require.Equal(t, want, testutil.CaptureRequest[components.V2DomainsDeleteDomainRequestBody](t, Cmd(), "domains delete-domain --domain=dom_1234abcd"))
}

func TestGetDomain(t *testing.T) {
	want := components.V2DomainsGetDomainRequestBody{Domain: "api.acme.com"}
	require.Equal(t, want, testutil.CaptureRequest[components.V2DomainsGetDomainRequestBody](t, Cmd(), "domains get-domain --domain=api.acme.com"))
}

func TestListDomains(t *testing.T) {
	tests := []struct {
		name, args string
		want       components.V2DomainsListDomainsRequestBody
	}{
		{"defaults", "domains list-domains --project=payments --app=api --environment=production", components.V2DomainsListDomainsRequestBody{Project: "payments", App: "api", Environment: "production", Limit: ptr.P(int64(100)), Cursor: nil, Search: nil}},
		{"all options", "domains list-domains --project=payments --app=api --environment=production --limit=25 --cursor=dom_1234abcd --search=acme.com", components.V2DomainsListDomainsRequestBody{Project: "payments", App: "api", Environment: "production", Limit: ptr.P(int64(25)), Cursor: ptr.P("dom_1234abcd"), Search: ptr.P("acme.com")}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, testutil.CaptureRequestWithData[components.V2DomainsListDomainsRequestBody](t, Cmd(), tt.args, []any{}))
		})
	}
}

func TestVerifyDomain(t *testing.T) {
	want := components.V2DomainsVerifyDomainRequestBody{Domain: "api.acme.com"}
	require.Equal(t, want, captureVerifyRequest(t, "domains verify-domain --domain=api.acme.com"))
}

func captureVerifyRequest(t *testing.T, args string) components.V2DomainsVerifyDomainRequestBody {
	t.Helper()
	var body []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var err error
		body, err = io.ReadAll(r.Body)
		require.NoError(t, err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		_, err = w.Write([]byte(`{"meta":{"requestId":"test"},"data":{}}`))
		require.NoError(t, err)
	}))
	t.Cleanup(srv.Close)
	orig := os.Stdout
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stdout = w
	t.Cleanup(func() { os.Stdout = orig; _ = r.Close(); _ = w.Close() })
	root := &cli.Command{Name: "unkey", Commands: []*cli.Command{Cmd()}}
	err = root.Run(context.Background(), strings.Fields(fmt.Sprintf("unkey %s --api-url=%s --root-key=test_key", args, srv.URL)))
	require.NoError(t, err)
	require.NoError(t, w.Close())
	os.Stdout = orig
	_, err = io.Copy(&bytes.Buffer{}, r)
	require.NoError(t, err)
	var req components.V2DomainsVerifyDomainRequestBody
	require.NoError(t, json.Unmarshal(body, &req))
	return req
}
