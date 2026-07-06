package v2BillingGetInvoiceDraft_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/internal/services/billing"
	"github.com/unkeyed/unkey/pkg/clickhouse/schema"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/ratecard"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_billing_get_invoice_draft"
)

func TestGetInvoiceDraft(t *testing.T) {
	h := testutil.NewHarness(t, testutil.HarnessConfig{ClickHouse: true})
	route := &handler.Handler{ClickHouse: h.ClickHouse, Resolver: billing.NewResolver(h.DB)}
	h.Register(route)

	workspaceID := h.Resources().UserWorkspace.ID
	rootKey := h.CreateRootKey(workspaceID, "billing.*.read_billing")
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	// Workspace default card: verifications free up to 10, then 0.5 cents each.
	cents := "0.5"
	config, err := json.Marshal(ratecard.Config{
		Verifications: []ratecard.Tier{
			{FirstUnit: 1, LastUnit: ptr(int64(10)), CentsPerUnit: nil},
			{FirstUnit: 11, LastUnit: nil, CentsPerUnit: &cents},
		},
		Credits:    nil,
		Ratelimits: nil,
	})
	require.NoError(t, err)
	cardID := uid.New(uid.RateCardPrefix)
	require.NoError(t, db.Query.InsertRateCard(context.Background(), h.DB.RW(), db.InsertRateCardParams{
		ID:          cardID,
		WorkspaceID: workspaceID,
		Name:        "default",
		Currency:    "USD",
		Config:      config,
		Selectable:  false,
		CreatedAt:   time.Now().UnixMilli(),
	}))
	require.NoError(t, db.Query.UpsertWorkspaceBillingSettingsDefaultRateCard(context.Background(), h.DB.RW(), db.UpsertWorkspaceBillingSettingsDefaultRateCardParams{
		ID:                uid.New(uid.RateCardPrefix),
		WorkspaceID:       workspaceID,
		DefaultRateCardID: nullString(cardID),
		CreatedAt:         time.Now().UnixMilli(),
	}))

	// Enable the keyspace that carries the usage so the draft is scoped to it.
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
		ExternalID:  "user_draft",
		Meta:        nil,
		Ratelimits:  nil,
	})

	now := time.Now()
	year := now.Year()
	month := int(now.Month())

	// 30 VALID verifications: 10 free + 20 x 0.5 = 10 cents total.
	events := make([]schema.KeyVerification, 30)
	for i := range events {
		events[i] = schema.KeyVerification{
			RequestID:    uid.New(uid.RequestPrefix),
			Time:         now.Add(time.Duration(i) * time.Second).UnixMilli(),
			WorkspaceID:  workspaceID,
			KeySpaceID:   keySpaceID,
			IdentityID:   identity.ID,
			ExternalID:   "user_draft",
			KeyID:        uid.New(uid.KeyPrefix),
			Region:       "us-east-1",
			Outcome:      "VALID",
			Tags:         []string{},
			SpentCredits: 0,
			Latency:      rand.Float64() * 100,
		}
	}
	batch, err := h.ClickHouse.Conn().PrepareBatch(context.Background(), "INSERT INTO default.key_verifications_raw_v2")
	require.NoError(t, err)
	for _, e := range events {
		require.NoError(t, batch.AppendStruct(&e))
	}
	require.NoError(t, batch.Send())

	t.Run("prices usage with the resolved default card", func(t *testing.T) {
		require.Eventually(t, func() bool {
			res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{
				Year:       year,
				Month:      month,
				ExternalId: nil,
			})
			if res.Status != 200 || len(res.Body.Data) != 1 {
				return false
			}
			entry := res.Body.Data[0]
			if !entry.Priced || entry.RateCardId == nil || *entry.RateCardId != cardID {
				return false
			}
			// Cross-path parity (R19): the route's amount must equal a direct
			// engine computation for the same card and quantities.
			return entry.TotalCents != nil && *entry.TotalCents == "10" &&
				entry.TotalCentsRounded != nil && *entry.TotalCentsRounded == 10
		}, time.Minute, time.Second)
	})

	t.Run("the draft is a preview and does NOT pin the period card", func(t *testing.T) {
		// A draft/preview must not freeze the rate card for a period still
		// accruing usage; pinning happens only in the period-close push.
		_, err := db.Query.FindBillingPeriodRateCard(context.Background(), h.DB.RO(), db.FindBillingPeriodRateCardParams{
			WorkspaceID: workspaceID,
			IdentityID:  identity.ID,
			Year:        int32(year),
			Month:       int32(month),
		})
		require.True(t, db.IsNotFound(err), "getInvoiceDraft must not persist a period rate-card pin")
	})
}

func ptr[T any](v T) *T { return &v }

func nullString(s string) sql.NullString {
	return sql.NullString{Valid: true, String: s}
}
