package handler_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	handler "github.com/unkeyed/unkey/go/cmd/api/routes/v2_ratelimit_set_override"
	"github.com/unkeyed/unkey/go/pkg/entities"
	"github.com/unkeyed/unkey/go/pkg/testutil"
	"github.com/unkeyed/unkey/go/pkg/uid"
)

func TestCreateNewOverrideSuccessfully(t *testing.T) {
	ctx := context.Background()
	h := testutil.NewHarness(t)

	namespaceID := uid.New("test_ns")
	namespaceName := "test_namespace"
	h.DB.InsertRatelimitNamespace(ctx, entities.RatelimitNamespace{
		ID:          namespaceID,
		WorkspaceID: h.Resources.UserWorkspace.ID,
		Name:        namespaceName,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Time{},
		DeletedAt:   time.Time{},
	})

	route := handler.New(handler.Services{
		DB:     h.DB,
		Keys:   h.Keys,
		Logger: h.Logger,
	})

	h.Register(route)

	rootKey := h.CreateRootKey()

	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	req := handler.Request{
		NamespaceId:   nil,
		NamespaceName: &namespaceName,
		Identifier:    "test_identifier",
		Limit:         10,
		Duration:      1000,
	}

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
	require.Equal(t, 200, res.Status, "expected 200, received: %v", res.Body)
	require.NotNil(t, res.Body)
	require.NotEqual(t, "", res.Body.OverrideId, "Override ID should not be empty, got: %+v", res.Body)

	override, err := h.DB.FindRatelimitOverrideByID(ctx, h.Resources.UserWorkspace.ID, res.Body.OverrideId)
	require.NoError(t, err)

	require.Equal(t, namespaceID, override.NamespaceID)
	require.Equal(t, req.Identifier, override.Identifier)
	require.Equal(t, req.Limit, int64(override.Limit))
	require.Equal(t, req.Duration, override.Duration.Milliseconds())
	require.False(t, override.CreatedAt.IsZero())
	require.True(t, override.UpdatedAt.IsZero())
	require.True(t, override.DeletedAt.IsZero())

}
