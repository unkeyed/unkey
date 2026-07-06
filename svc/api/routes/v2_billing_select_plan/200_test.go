package v2BillingSelectPlan_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/ratecard"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_billing_select_plan"
)

func insertCard(t *testing.T, database db.Database, workspaceID, name string, selectable bool) string {
	t.Helper()
	id := uid.New(uid.RateCardPrefix)
	cents := "0.1"
	config, err := json.Marshal(ratecard.Config{
		Verifications: []ratecard.Tier{{FirstUnit: 1, LastUnit: nil, CentsPerUnit: &cents}},
		Credits:       nil,
		Ratelimits:    nil,
	})
	require.NoError(t, err)
	require.NoError(t, db.Query.InsertRateCard(context.Background(), database.RW(), db.InsertRateCardParams{
		ID:          id,
		WorkspaceID: workspaceID,
		Name:        name,
		Currency:    "USD",
		Config:      config,
		Selectable:  selectable,
		CreatedAt:   time.Now().UnixMilli(),
	}))
	return id
}

func TestSelectPlan(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{DB: h.DB}
	h.Register(route)

	workspaceID := h.Resources().UserWorkspace.ID
	rootKey := h.CreateRootKey(workspaceID, "billing.*.update_billing")
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	selectable := insertCard(t, h.DB, workspaceID, "pro", true)
	private := insertCard(t, h.DB, workspaceID, "private", false)

	identity := h.CreateIdentity(seed.CreateIdentityRequest{
		WorkspaceID: workspaceID,
		ExternalID:  "user_select",
		Meta:        nil,
		Ratelimits:  nil,
	})

	t.Run("records a selectable card", func(t *testing.T) {
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{
			ExternalId: "user_select",
			RateCardId: selectable,
		})
		require.Equal(t, 200, res.Status, "%s", res.RawBody)
		require.Equal(t, selectable, res.Body.Data.RateCardId)

		row, err := db.Query.FindIdentityByID(context.Background(), h.DB.RO(), db.FindIdentityByIDParams{
			WorkspaceID: workspaceID,
			IdentityID:  identity.ID,
			Deleted:     false,
		})
		require.NoError(t, err)
		require.True(t, row.SelectedRateCardID.Valid)
		require.Equal(t, selectable, row.SelectedRateCardID.String)
	})

	t.Run("rejects a non-selectable card", func(t *testing.T) {
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{
			ExternalId: "user_select",
			RateCardId: private,
		})
		require.Equal(t, 400, res.Status, "%s", res.RawBody)
	})

	t.Run("unknown identity is a 404", func(t *testing.T) {
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{
			ExternalId: "user_missing",
			RateCardId: selectable,
		})
		require.Equal(t, 404, res.Status, "%s", res.RawBody)
	})

	t.Run("unknown rate card is a 404", func(t *testing.T) {
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{
			ExternalId: "user_select",
			RateCardId: "rc_does_not_exist",
		})
		require.Equal(t, 404, res.Status, "%s", res.RawBody)
	})

	t.Run("missing permission is rejected", func(t *testing.T) {
		weakKey := h.CreateRootKey(workspaceID, "identity.*.read_identity")
		weakHeaders := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", weakKey)},
		}
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, weakHeaders, handler.Request{
			ExternalId: "user_select",
			RateCardId: selectable,
		})
		require.Equal(t, 403, res.Status, "%s", res.RawBody)
	})
}
