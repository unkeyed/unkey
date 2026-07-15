package handler_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_ratelimit_get_override"
)

func TestOverrideNotFound(t *testing.T) {
	ctx := context.Background()
	h := testutil.NewHarness(t)

	// Create a namespace but no override
	namespaceID := uid.New("test_ns")
	err := db.Query.InsertRatelimitNamespace(ctx, h.DB.RW(), db.InsertRatelimitNamespaceParams{
		ID:          namespaceID,
		WorkspaceID: h.Resources().UserWorkspace.ID,
		Name:        uid.New("test"),
		CreatedAt:   time.Now().UnixMilli(),
	})
	require.NoError(t, err)

	route := &handler.Handler{
		DB:             h.DB,
		NamespaceCache: h.Caches.RatelimitNamespace,
	}

	h.Register(route)
	rootKey := h.CreateRootKey(h.Resources().UserWorkspace.ID, fmt.Sprintf("ratelimit.%s.read_override", namespaceID))

	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	// Test with non-existent identifier
	t.Run("override not found", func(t *testing.T) {
		req := handler.Request{
			Namespace:  namespaceID,
			Identifier: "non_existent_identifier",
		}

		res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, headers, req)
		require.Equal(t, http.StatusNotFound, res.Status, "got: %s", res.RawBody)
		require.NotNil(t, res.Body)
		require.Equal(t, "https://unkey.com/docs/errors/unkey/data/ratelimit_override_not_found", res.Body.Error.Type)
		require.Equal(t, http.StatusNotFound, res.Body.Error.Status)
	})

	// Test with non-existent namespace
	t.Run("namespace not found", func(t *testing.T) {
		req := handler.Request{
			Namespace:  "ns_nonexistent",
			Identifier: "some_identifier",
		}

		res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, headers, req)
		require.Equal(t, http.StatusNotFound, res.Status)
		require.NotNil(t, res.Body)
		require.Equal(t, "https://unkey.com/docs/errors/unkey/data/ratelimit_namespace_not_found", res.Body.Error.Type)
	})

	// Test with non-existent namespace name
	t.Run("namespace name not found", func(t *testing.T) {
		req := handler.Request{
			Namespace:  "nonexistent_namespace",
			Identifier: "some_identifier",
		}

		res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, headers, req)
		require.Equal(t, http.StatusNotFound, res.Status)
		require.NotNil(t, res.Body)
		require.Equal(t, "https://unkey.com/docs/errors/unkey/data/ratelimit_namespace_not_found", res.Body.Error.Type)
	})

	// Test with non-existent identifier
	t.Run("identifier not found", func(t *testing.T) {

		namespaceID := uid.New(uid.TestPrefix)
		rootKey := h.CreateRootKey(h.Resources().UserWorkspace.ID, fmt.Sprintf("ratelimit.%s.read_override", namespaceID))
		err := db.Query.InsertRatelimitNamespace(ctx, h.DB.RW(), db.InsertRatelimitNamespaceParams{
			ID:          namespaceID,
			WorkspaceID: h.Resources().UserWorkspace.ID,
			Name:        uid.New(uid.TestPrefix),
			CreatedAt:   time.Now().UnixMilli(),
		})
		require.NoError(t, err)

		nonExistentIdentifier := "nonexistent_identifier"
		req := handler.Request{
			Namespace:  namespaceID,
			Identifier: nonExistentIdentifier,
		}

		res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, http.Header{
			"Content-Type":  []string{"application/json"},
			"Authorization": []string{fmt.Sprintf("Bearer %s", rootKey)},
		}, req)
		require.Equal(t, http.StatusNotFound, res.Status)
		require.NotNil(t, res.Body)
		require.Equal(t, "https://unkey.com/docs/errors/unkey/data/ratelimit_override_not_found", res.Body.Error.Type)
	})
}

// TestOverrideResponsesDoNotLeakExistence guarantees a caller without
// read_override permission receives the same 404 whether or not an override
// exists, so the response cannot be used to enumerate which identifiers have
// overrides.
func TestOverrideResponsesDoNotLeakExistence(t *testing.T) {
	ctx := context.Background()
	h := testutil.NewHarness(t)

	namespaceID := uid.New("test_ns")
	err := db.Query.InsertRatelimitNamespace(ctx, h.DB.RW(), db.InsertRatelimitNamespaceParams{
		ID:          namespaceID,
		WorkspaceID: h.Resources().UserWorkspace.ID,
		Name:        uid.New("test"),
		CreatedAt:   time.Now().UnixMilli(),
	})
	require.NoError(t, err)

	existingIdentifier := "existing_identifier"
	err = db.Query.InsertRatelimitOverride(ctx, h.DB.RW(), db.InsertRatelimitOverrideParams{
		ID:          uid.New(uid.RatelimitOverridePrefix),
		WorkspaceID: h.Resources().UserWorkspace.ID,
		NamespaceID: namespaceID,
		Identifier:  existingIdentifier,
		Limit:       10,
		Duration:    1000,
		CreatedAt:   time.Now().UnixMilli(),
	})
	require.NoError(t, err)

	route := &handler.Handler{
		DB:             h.DB,
		NamespaceCache: h.Caches.RatelimitNamespace,
	}
	h.Register(route)

	rootKey := h.CreateRootKey(h.Resources().UserWorkspace.ID)
	headers := http.Header{
		"Content-Type":  []string{"application/json"},
		"Authorization": []string{fmt.Sprintf("Bearer %s", rootKey)},
	}

	probe := func(identifier string) testutil.TestResponse[openapi.NotFoundErrorResponse] {
		return testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, headers, handler.Request{
			Namespace:  namespaceID,
			Identifier: identifier,
		})
	}

	existing := probe(existingIdentifier)
	missing := probe("non_existent_identifier")

	require.Equal(t, http.StatusNotFound, existing.Status, "got: %s", existing.RawBody)
	require.Equal(t, http.StatusNotFound, missing.Status, "got: %s", missing.RawBody)
	require.Equal(t, existing.Body.Error.Type, missing.Body.Error.Type)
	require.Equal(t, existing.Body.Error.Detail, missing.Body.Error.Detail)
	require.Equal(t, existing.Body.Error.Status, missing.Body.Error.Status)
	require.Equal(t, existing.Body.Error.Title, missing.Body.Error.Title)
}
