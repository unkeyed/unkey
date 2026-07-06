package v2BillingGetUsage_test

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/internal/services/billing"
	"github.com/unkeyed/unkey/pkg/clickhouse/schema"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_billing_get_usage"
)

func TestGetUsage(t *testing.T) {
	h := testutil.NewHarness(t, testutil.HarnessConfig{ClickHouse: true})
	route := &handler.Handler{ClickHouse: h.ClickHouse, Resolver: billing.NewResolver(h.DB)}
	h.Register(route)

	workspaceID := h.Resources().UserWorkspace.ID
	rootKey := h.CreateRootKey(workspaceID, "billing.*.read_billing")
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	// Enable the keyspace the usage is recorded under so it is in scope.
	keySpaceID := uid.New(uid.KeySpacePrefix)
	require.NoError(t, db.Query.UpsertBillingBillableResource(context.Background(), h.DB.RW(), db.UpsertBillingBillableResourceParams{
		ID:           uid.New(uid.RateCardPrefix),
		WorkspaceID:  workspaceID,
		ResourceType: db.BillingBillableResourcesResourceTypeKeyspace,
		ResourceID:   keySpaceID,
		CreatedAt:    time.Now().UnixMilli(),
	}))

	identity := h.CreateIdentity(seed.CreateIdentityRequest{
		WorkspaceID: workspaceID,
		ExternalID:  "user_usage",
		Meta:        nil,
		Ratelimits:  nil,
	})

	now := time.Now()
	year := now.Year()
	month := int(now.Month())

	// Seed 25 VALID verifications spending 2 credits each, straight into the
	// raw table; the MV cascade rolls them up per identity.
	events := make([]schema.KeyVerification, 25)
	for i := range events {
		events[i] = schema.KeyVerification{
			RequestID:    uid.New(uid.RequestPrefix),
			Time:         now.Add(time.Duration(i) * time.Second).UnixMilli(),
			WorkspaceID:  workspaceID,
			KeySpaceID:   keySpaceID,
			IdentityID:   identity.ID,
			ExternalID:   "user_usage",
			KeyID:        uid.New(uid.KeyPrefix),
			Region:       "us-east-1",
			Outcome:      "VALID",
			Tags:         []string{},
			SpentCredits: 2,
			Latency:      rand.Float64() * 100,
		}
	}
	batch, err := h.ClickHouse.Conn().PrepareBatch(context.Background(), "INSERT INTO default.key_verifications_raw_v2")
	require.NoError(t, err)
	for _, e := range events {
		require.NoError(t, batch.AppendStruct(&e))
	}
	require.NoError(t, batch.Send())

	t.Run("returns per-identity quantities", func(t *testing.T) {
		require.Eventually(t, func() bool {
			res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{
				Year:       year,
				Month:      month,
				ExternalId: nil,
			})
			if res.Status != 200 || len(res.Body.Data) != 1 {
				return false
			}
			row := res.Body.Data[0]
			return row.ExternalId == "user_usage" && row.Verifications == 25 && row.SpentCredits == 50
		}, time.Minute, time.Second)
	})

	t.Run("missing permission is rejected", func(t *testing.T) {
		weakKey := h.CreateRootKey(workspaceID, "api.*.read_api")
		weakHeaders := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", weakKey)},
		}
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, weakHeaders, handler.Request{
			Year:       year,
			Month:      month,
			ExternalId: nil,
		})
		require.Equal(t, 403, res.Status, "%s", res.RawBody)
	})

	t.Run("invalid month fails validation", func(t *testing.T) {
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{
			Year:       year,
			Month:      13,
			ExternalId: nil,
		})
		require.Equal(t, 400, res.Status, "%s", res.RawBody)
	})
}
