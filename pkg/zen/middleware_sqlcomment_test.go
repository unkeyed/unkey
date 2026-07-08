package zen_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/mysql/sqlcomment"
	"github.com/unkeyed/unkey/pkg/zen"
)

func TestWithSQLComment_attachesRoutePattern(t *testing.T) {
	t.Parallel()

	srv, err := zen.New(zen.Config{})
	require.NoError(t, err)

	srv.RegisterRoute(
		[]zen.Middleware{zen.WithSQLComment()},
		zen.NewRoute(http.MethodPost, "/v2/keys.verifyKey", func(ctx context.Context, s *zen.Session) error {
			tags := sqlcomment.DynamicFromContext(ctx)
			require.Equal(t, "POST /v2/keys.verifyKey", tags.Route)
			require.Equal(t, "http", tags.Source)
			return s.JSON(http.StatusOK, map[string]string{"ok": "true"})
		}),
	)

	req := httptest.NewRequest(http.MethodPost, "/v2/keys.verifyKey", nil)
	rec := httptest.NewRecorder()
	srv.Mux().ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
}
