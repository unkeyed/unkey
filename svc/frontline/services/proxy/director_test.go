package proxy

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/zen"
)

func newTestService(t *testing.T) *service {
	t.Helper()
	svc, err := New(Config{
		InstanceID: "test-instance",
		Platform:   "dev",
		Region:     "local",
		ApexDomain: "unkey.cloud",
		Clock:      clock.New(),
		MaxHops:    3,
	})
	require.NoError(t, err)
	return svc
}

func newTestSession(t *testing.T, method, path, host string) *zen.Session {
	t.Helper()
	req := httptest.NewRequest(method, path, nil)
	req.Host = host
	w := httptest.NewRecorder()
	sess := &zen.Session{}
	err := sess.Init(w, req, 0)
	require.NoError(t, err)
	return sess
}

func TestMakePortalDirector_StripPathPrefix(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		pathPrefix string
		reqPath    string
		wantPath   string
	}{
		{
			name:       "strips /portal prefix from /portal/keys",
			pathPrefix: "/portal",
			reqPath:    "/portal/keys",
			wantPath:   "/keys",
		},
		{
			name:       "strips /portal prefix from /portal root",
			pathPrefix: "/portal",
			reqPath:    "/portal",
			wantPath:   "/",
		},
		{
			name:       "strips /portal prefix from /portal/ with trailing slash",
			pathPrefix: "/portal",
			reqPath:    "/portal/",
			wantPath:   "/",
		},
		{
			name:       "preserves path when no prefix match",
			pathPrefix: "/portal",
			reqPath:    "/other/path",
			wantPath:   "/other/path",
		},
		{
			name:       "no stripping when prefix is empty",
			pathPrefix: "",
			reqPath:    "/portal/keys",
			wantPath:   "/portal/keys",
		},
		{
			name:       "deep nested path after prefix",
			pathPrefix: "/portal",
			reqPath:    "/portal/keys/create",
			wantPath:   "/keys/create",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			svc := newTestService(t)
			sess := newTestSession(t, http.MethodGet, tt.reqPath, "acme.unkey.com")

			director := svc.makePortalDirector(sess, time.Now(), tt.pathPrefix)

			req := httptest.NewRequest(http.MethodGet, tt.reqPath, nil)
			req.Host = "acme.unkey.com"
			director(req)

			require.Equal(t, tt.wantPath, req.URL.Path)
			require.Equal(t, "acme.unkey.com", req.Host, "portal director should preserve original Host")
			require.Equal(t, "https", req.Header.Get(HeaderForwardedProto))
			require.NotEmpty(t, req.Header.Get(HeaderFrontlineID))
			require.NotEmpty(t, req.Header.Get(HeaderRequestID))
		})
	}
}
