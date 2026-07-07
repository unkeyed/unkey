package handler_test

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_apis_list_keys"
)

// TestUnauthorizedDoesNotLeakExistence guards against a resource-existence
// oracle: a caller who lacks read permission on an API must not be able to tell
// whether that API exists. If an existing-but-forbidden API returns 403 while a
// non-existent API returns 404, the status code itself discloses existence, so
// an attacker with a low-privilege root key can enumerate API IDs in the
// workspace. Both cases must return the same masked 404.
func TestUnauthorizedDoesNotLeakExistence(t *testing.T) {
	ctx := context.Background()
	h := testutil.NewHarness(t)

	route := &handler.Handler{
		DB:       h.DB,
		Vault:    h.Vault,
		ApiCache: h.Caches.LiveApiByID,
	}

	h.Register(route)

	workspace := h.Resources().UserWorkspace

	// Create a real keyspace + API that the caller is NOT allowed to read.
	keySpaceID := uid.New(uid.KeySpacePrefix)
	err := db.Query.InsertKeySpace(ctx, h.DB.RW(), db.InsertKeySpaceParams{
		ID:            keySpaceID,
		WorkspaceID:   workspace.ID,
		CreatedAtM:    time.Now().UnixMilli(),
		DefaultPrefix: sql.NullString{Valid: false},
		DefaultBytes:  sql.NullInt32{Valid: false},
	})
	require.NoError(t, err)

	existingApiID := uid.New("api")
	err = db.Query.InsertApi(ctx, h.DB.RW(), db.InsertApiParams{
		ID:          existingApiID,
		Name:        "Secret API",
		WorkspaceID: workspace.ID,
		AuthType:    db.NullApisAuthType{Valid: true, ApisAuthType: db.ApisAuthTypeKey},
		KeyAuthID:   sql.NullString{Valid: true, String: keySpaceID},
		CreatedAtM:  time.Now().UnixMilli(),
	})
	require.NoError(t, err)

	// A root key in the same workspace that has no read permission on this API.
	rootKey := h.CreateRootKey(workspace.ID, "workspace.read")

	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	// Baseline: a non-existent API returns 404 for this caller.
	nonExistent := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](
		h,
		route,
		headers,
		handler.Request{ApiId: uid.New("api")},
	)
	require.Equal(t, 404, nonExistent.Status)

	// An existing-but-forbidden API must return the same status, otherwise the
	// caller learns the API exists.
	existing := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](
		h,
		route,
		headers,
		handler.Request{ApiId: existingApiID},
	)

	require.Equal(t, nonExistent.Status, existing.Status,
		"unauthorized caller must not distinguish an existing API (%d) from a non-existent one (%d)",
		existing.Status, nonExistent.Status)
	require.Equal(t, 404, existing.Status,
		"existing API the caller cannot read must be masked as 404, not 403")
}
