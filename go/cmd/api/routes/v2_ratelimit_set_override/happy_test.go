package handler_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/apps/agent/pkg/util"
	handler "github.com/unkeyed/unkey/go/cmd/api/routes/v2_ratelimit_set_override"
	"github.com/unkeyed/unkey/go/pkg/database"
	"github.com/unkeyed/unkey/go/pkg/entities"
	"github.com/unkeyed/unkey/go/pkg/logging"
	"github.com/unkeyed/unkey/go/pkg/testutil"
	"github.com/unkeyed/unkey/go/pkg/uid"
)

func TestCreateNewOverride(t *testing.T) {
	ctx := context.Background()
	h := testutil.NewHarness(t)

	c := testutil.NewContainers(t)

	mysqlAddr := c.RunMySQL()

	db, err := database.New(database.Config{
		PrimaryDSN:  mysqlAddr,
		ReadOnlyDSN: "",
		Logger:      logging.NewNoop(),
	})
	require.NoError(t, err)

	db.InsertRatelimitOverride(ctx, entities.RatelimitOverride{
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
		DB:   db,
		Keys: nil,
	})

	h.Register(route)

	req := handler.Request{
		NamespaceId:   util.Pointer(""),
		NamespaceName: nil,
		Identifier:    "",
		Limit:         10,
		Duration:      1000,
	}
	res := testutil.CallRoute[handler.Request, handler.Response](h, route, nil, req)

	require.Equal(t, 200, res.Status)
	require.NotEqual(t, "", res.Body.OverrideId)
}
