package ctxutil_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/ctxutil"
)

func TestRequestIDContextFunctions(t *testing.T) {
	// Test setting and getting request ID
	requestID := "test-request-id-123"
	ctx := context.Background()

	// Test with empty context
	emptyID := ctxutil.GetRequestID(ctx)
	require.Empty(t, emptyID)

	// Test setting and retrieving
	ctx = ctxutil.SetRequestID(ctx, requestID)
	retrievedID := ctxutil.GetRequestID(ctx)
	require.Equal(t, requestID, retrievedID)

	// Test overwriting
	newRequestID := "new-request-id-456"
	ctx = ctxutil.SetRequestID(ctx, newRequestID)
	retrievedID = ctxutil.GetRequestID(ctx)
	require.Equal(t, newRequestID, retrievedID)
}
