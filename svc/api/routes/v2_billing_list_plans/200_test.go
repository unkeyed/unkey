package v2BillingListPlans_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/internal/services/billing"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/ratecard"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_billing_list_plans"
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

func TestListPlans(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{DB: h.DB, Resolver: billing.NewResolver(h.DB)}
	h.Register(route)

	workspaceID := h.Resources().UserWorkspace.ID
	rootKey := h.CreateRootKey(workspaceID, "billing.*.read_billing")
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	pro := insertCard(t, h.DB, workspaceID, "pro", true)
	basic := insertCard(t, h.DB, workspaceID, "basic", true)
	private := insertCard(t, h.DB, workspaceID, "private", false)
	_ = basic

	identity := h.CreateIdentity(seed.CreateIdentityRequest{
		WorkspaceID: workspaceID,
		ExternalID:  "user_plans",
		Meta:        nil,
		Ratelimits:  nil,
	})

	t.Run("lists only the selectable set", func(t *testing.T) {
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{
			ExternalId: "user_plans",
		})
		require.Equal(t, 200, res.Status, "%s", res.RawBody)
		require.Len(t, res.Body.Data.Plans, 2)
		for _, p := range res.Body.Data.Plans {
			require.NotEqual(t, private, p.Id, "non-selectable cards must not be listed")
		}
	})

	t.Run("selection is reflected as selected and effective", func(t *testing.T) {
		require.NoError(t, db.Query.UpdateIdentitySelectedRateCard(context.Background(), h.DB.RW(), db.UpdateIdentitySelectedRateCardParams{
			SelectedRateCardID: sql.NullString{Valid: true, String: pro},
			WorkspaceID:        workspaceID,
			IdentityID:         identity.ID,
		}))
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{
			ExternalId: "user_plans",
		})
		require.Equal(t, 200, res.Status, "%s", res.RawBody)
		require.NotNil(t, res.Body.Data.SelectedRateCardId)
		require.Equal(t, pro, *res.Body.Data.SelectedRateCardId)
		require.NotNil(t, res.Body.Data.EffectiveRateCardId)
		require.Equal(t, pro, *res.Body.Data.EffectiveRateCardId)
	})

	t.Run("unknown identity is a 404", func(t *testing.T) {
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{
			ExternalId: "user_missing",
		})
		require.Equal(t, 404, res.Status, "%s", res.RawBody)
	})
}
