package handler_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/apps/agent/pkg/util"
	handler "github.com/unkeyed/unkey/go/cmd/api/routes/v2_ratelimit_set_override"
	"github.com/unkeyed/unkey/go/pkg/entities"
	"github.com/unkeyed/unkey/go/pkg/testutil"
	"github.com/unkeyed/unkey/go/pkg/uid"
)

func TestCreateNewOverride(t *testing.T) {
	ctx := context.Background()
	h := testutil.NewHarness(t)

	h.DB.InsertRatelimitOverride(ctx, entities.RatelimitOverride{
		ID:          uid.Test(),
		WorkspaceID: uid.Test(),
		NamespaceID: uid.Test(),
		Identifier:  "test",
		Limit:       10,
		Duration:    0,
		Async:       false,
		CreatedAt:   time.Time{},
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
		NamespaceId:   util.Pointer(""),
		NamespaceName: nil,
		Identifier:    "",
		Limit:         10,
		Duration:      1000,
	}

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)

	require.Equal(t, 200, res.Status)
	require.NotEqual(t, "", res.Body.OverrideId)
}
