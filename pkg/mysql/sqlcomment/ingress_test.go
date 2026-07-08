package sqlcomment

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRestateInvokeRoute(t *testing.T) {
	t.Parallel()

	tests := []struct {
		path string
		want string
		ok   bool
	}{
		{path: "/invoke/hydra.v1.DeployService/Deploy", want: "hydra.v1.DeployService/Deploy", ok: true},
		{path: "/invoke/hydra.v1.CronService/RunKeyRefill", want: "hydra.v1.CronService/RunKeyRefill", ok: true},
		{path: "/health", ok: false},
		{path: "/invoke/only-one-part", ok: false},
	}

	for _, tc := range tests {
		got, ok := restateInvokeRoute(tc.path)
		require.Equal(t, tc.ok, ok, tc.path)
		if tc.ok {
			require.Equal(t, tc.want, got)
		}
	}
}

func TestWrapRestateInvokeHandler(t *testing.T) {
	t.Parallel()

	var got Dynamic
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = DynamicFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodPost, "/invoke/hydra.v1.DeployService/Deploy", nil)
	rec := httptest.NewRecorder()
	WrapRestateInvokeHandler(inner).ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, Dynamic{Route: "hydra.v1.DeployService/Deploy", Source: "restate"}, got)
}

func TestWrapRestateInvokeHandler_skipsNonInvokePaths(t *testing.T) {
	t.Parallel()

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, Dynamic{Route: "", Source: ""}, DynamicFromContext(r.Context()))
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	WrapRestateInvokeHandler(inner).ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
}

// Ensure context values propagate the way ctrl-worker DB calls expect.
func TestWrapRestateInvokeHandler_contextRoundTrip(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	ctx = WithDynamic(ctx, Dynamic{Route: "hydra.v1.OpenapiService/ScrapeSpec", Source: "restate"})
	require.Equal(t, "restate", DynamicFromContext(ctx).Source)
}
