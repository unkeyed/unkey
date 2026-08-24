package zen

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/timing"
)

func TestWithSessionProvidesTimingResponseWriter(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	response := httptest.NewRecorder()
	session := &Session{}
	require.NoError(t, session.Init(response, request, 0))

	ctx := WithSession(context.Background(), session)
	timing.Record(ctx, timing.Entry{Name: "cache_get", Duration: time.Millisecond})

	require.Equal(t, []string{"cache_get=1ms"}, response.Header().Values(timing.HeaderName))
	storedSession, ok := SessionFromContext(ctx)
	require.True(t, ok)
	require.Same(t, session, storedSession)
}
