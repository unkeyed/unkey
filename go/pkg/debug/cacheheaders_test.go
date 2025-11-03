package debug

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/go/pkg/zen"
)

func TestRecordCacheHitWithoutSession(t *testing.T) {
	ctx := context.Background()

	// Should not panic when no session in context
	require.NotPanics(t, func() {
		RecordCacheHit(ctx, "ApiByID", "FRESH", 100*time.Microsecond)
	})
}

func TestRecordCacheHitWithSession(t *testing.T) {
	// Enable cache headers for this test
	EnableCacheHeaders()
	defer DisableCacheHeaders()

	tests := []struct {
		name      string
		cacheName string
		status    string
		latency   time.Duration
		expected  string
	}{
		{
			name:      "cache miss with millisecond latency",
			cacheName: "ApiByID",
			status:    "MISS",
			latency:   2*time.Millisecond + 500*time.Microsecond,
			expected:  "ApiByID:2.50ms:MISS",
		},
		{
			name:      "cache hit with microsecond latency",
			cacheName: "RootKeyByHash",
			status:    "FRESH",
			latency:   150 * time.Microsecond,
			expected:  "RootKeyByHash:150us:FRESH",
		},
		{
			name:      "stale cache with exact millisecond",
			cacheName: "PermissionsByApiId",
			status:    "STALE",
			latency:   1 * time.Millisecond,
			expected:  "PermissionsByApiId:1.00ms:STALE",
		},
		{
			name:      "very fast cache hit",
			cacheName: "WorkspaceById",
			status:    "FRESH",
			latency:   10 * time.Microsecond,
			expected:  "WorkspaceById:10us:FRESH",
		},
		{
			name:      "cache error",
			cacheName: "KeyVerification",
			status:    "ERROR",
			latency:   500 * time.Microsecond,
			expected:  "KeyVerification:500us:ERROR",
		},
		{
			name:      "very slow operation",
			cacheName: "SlowCache",
			status:    "MISS",
			latency:   100 * time.Millisecond,
			expected:  "SlowCache:100.00ms:MISS",
		},
		{
			name:      "sub-microsecond latency rounds to zero",
			cacheName: "FastCache",
			status:    "FRESH",
			latency:   100 * time.Nanosecond,
			expected:  "FastCache:0us:FRESH",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a real zen session
			req := httptest.NewRequest(http.MethodPost, "/test", nil)
			recorder := httptest.NewRecorder()
			session := &zen.Session{}
			err := session.Init(recorder, req, 0)
			require.NoError(t, err)

			ctx := zen.WithSession(context.Background(), session)

			// Record the cache hit
			RecordCacheHit(ctx, tt.cacheName, tt.status, tt.latency)

			// Check that the header was added
			headers := recorder.Header().Values("X-Unkey-Debug-Cache")
			require.Len(t, headers, 1, "Expected exactly one cache debug header")
			require.Equal(t, tt.expected, headers[0])
		})
	}
}

func TestRecordCacheHitMultipleOperations(t *testing.T) {
	// Enable cache headers for this test
	EnableCacheHeaders()
	defer DisableCacheHeaders()

	// Test that multiple cache operations result in multiple headers
	req := httptest.NewRequest(http.MethodPost, "/test", nil)
	recorder := httptest.NewRecorder()
	session := &zen.Session{}
	err := session.Init(recorder, req, 0)
	require.NoError(t, err)

	ctx := zen.WithSession(context.Background(), session)

	// Record multiple cache operations
	RecordCacheHit(ctx, "ApiByID", "MISS", 2*time.Millisecond)
	RecordCacheHit(ctx, "RootKeyByHash", "FRESH", 150*time.Microsecond)
	RecordCacheHit(ctx, "PermissionsByApiId", "STALE", 750*time.Microsecond)

	// Check that all headers were added
	headers := recorder.Header().Values("X-Unkey-Debug-Cache")
	require.Len(t, headers, 3, "Expected three cache debug headers")

	expected := []string{
		"ApiByID:2.00ms:MISS",
		"RootKeyByHash:150us:FRESH",
		"PermissionsByApiId:750us:STALE",
	}

	require.Equal(t, expected, headers)
}

func TestRecordCacheHitSessionRetrieval(t *testing.T) {
	// Enable cache headers for this test
	EnableCacheHeaders()
	defer DisableCacheHeaders()

	// Test that the context session retrieval works correctly
	req := httptest.NewRequest(http.MethodPost, "/test", nil)
	recorder := httptest.NewRecorder()
	session := &zen.Session{}
	err := session.Init(recorder, req, 0)
	require.NoError(t, err)

	t.Run("with session in context", func(t *testing.T) {
		ctx := zen.WithSession(context.Background(), session)

		// Verify we can retrieve the session
		retrievedSession, ok := zen.SessionFromContext(ctx)
		require.True(t, ok, "Should be able to retrieve session from context")
		require.Equal(t, session, retrievedSession, "Retrieved session should match original")

		// Record cache hit should work
		RecordCacheHit(ctx, "TestCache", "HIT", 50*time.Microsecond)

		headers := recorder.Header().Values("X-Unkey-Debug-Cache")
		require.Len(t, headers, 1)
		require.Equal(t, "TestCache:50us:HIT", headers[0])
	})

	t.Run("without session in context", func(t *testing.T) {
		emptyCtx := context.Background()

		// Verify we cannot retrieve a session
		_, ok := zen.SessionFromContext(emptyCtx)
		require.False(t, ok, "Should not be able to retrieve session from empty context")

		// Record cache hit should be no-op
		initialHeaders := len(recorder.Header().Values("X-Unkey-Debug-Cache"))
		RecordCacheHit(emptyCtx, "TestCache", "HIT", 50*time.Microsecond)
		finalHeaders := len(recorder.Header().Values("X-Unkey-Debug-Cache"))

		require.Equal(t, initialHeaders, finalHeaders, "No headers should be added without session")
	})
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		expected string
	}{
		{
			name:     "sub-microsecond rounds to zero",
			duration: 100 * time.Nanosecond,
			expected: "0us",
		},
		{
			name:     "microseconds with decimal",
			duration: 1500 * time.Nanosecond,
			expected: "2us", // 1.5 rounds to 2
		},
		{
			name:     "exact microseconds",
			duration: 150 * time.Microsecond,
			expected: "150us",
		},
		{
			name:     "microseconds just under 1ms",
			duration: 999 * time.Microsecond,
			expected: "999us",
		},
		{
			name:     "exactly 1 millisecond",
			duration: 1 * time.Millisecond,
			expected: "1.00ms",
		},
		{
			name:     "milliseconds with fraction",
			duration: 1*time.Millisecond + 500*time.Microsecond,
			expected: "1.50ms",
		},
		{
			name:     "large millisecond value",
			duration: 123*time.Millisecond + 456*time.Microsecond,
			expected: "123.46ms",
		},
		{
			name:     "seconds converted to milliseconds",
			duration: 2*time.Second + 500*time.Millisecond,
			expected: "2500.00ms",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatDuration(tt.duration)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestCacheHeaderName(t *testing.T) {
	// Enable cache headers for this test
	EnableCacheHeaders()
	defer DisableCacheHeaders()

	// Test that the correct header name is used
	req := httptest.NewRequest(http.MethodPost, "/test", nil)
	recorder := httptest.NewRecorder()
	session := &zen.Session{}
	err := session.Init(recorder, req, 0)
	require.NoError(t, err)

	ctx := zen.WithSession(context.Background(), session)

	RecordCacheHit(ctx, "TestCache", "FRESH", 100*time.Microsecond)

	// Verify the header name is correct
	require.Len(t, recorder.Header().Values("X-Unkey-Debug-Cache"), 1, "Should have X-Unkey-Debug-Cache header")
	require.Len(t, recorder.Header().Values("X-Cache-Debug"), 0, "Should not have old X-Cache-Debug header")
}
