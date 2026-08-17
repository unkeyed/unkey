package stripe

import (
	"context"
	"database/sql"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	stripesdk "github.com/stripe/stripe-go/v86"
	"github.com/unkeyed/unkey/pkg/mysql/sqlcomment"
	"github.com/unkeyed/unkey/pkg/testutil/containers"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/pkg/webhook"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

// renewalInvoice builds a payload in the CURRENT (2025-03-31.basil and later)
// shape, carrying the subscription under parent.subscription_details. Building
// it any other way would not exercise the path production actually receives.
func renewalInvoice(customer, subscription string) stripesdk.Invoice {
	now := time.Now().UTC()
	start := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC).AddDate(0, -1, 0)
	end := start.AddDate(0, 1, 0)
	//nolint:exhaustruct // the handler reads only these fields off the SDK invoice
	return stripesdk.Invoice{
		ID:            "in_test",
		BillingReason: stripesdk.InvoiceBillingReasonSubscriptionCycle,
		Customer:      &stripesdk.Customer{ID: customer}, //nolint:exhaustruct // unexpanded id, as delivered
		Parent: &stripesdk.InvoiceParent{ //nolint:exhaustruct // only the subscription reference is read
			SubscriptionDetails: &stripesdk.InvoiceParentSubscriptionDetails{ //nolint:exhaustruct // ditto
				Subscription: &stripesdk.Subscription{ID: subscription}, //nolint:exhaustruct // unexpanded id, as delivered
			},
		},
		PeriodStart: start.Unix(),
		PeriodEnd:   end.Unix(),
	}
}

// invoiceJSON wraps a payload fragment in the fields every invoice.created
// carries. This is the event's data.object, which is exactly what the verified
// webhook transport passes to invoiceCreated.
func invoiceJSON(fragment string) string {
	return `{"id":"in_1","object":"invoice","billing_reason":"subscription_cycle","customer":"cus_1",` +
		`"period_start":1750000000,"period_end":1752678400` + fragment + `}`
}

// TestInvoiceCreated_DecodesSubscription goes through JSON rather than building
// the struct in Go, because the bug it guards against was a wire-format bug: a
// hand-rolled struct read a top-level "subscription" field that Stripe removed in
// 2025-03-31.basil, so every renewal decoded as a non-renewal. A struct literal
// cannot catch a field moving in the payload, since it asserts only that
// deploySubscriptionID walks pointers the way it was written to. Feeding it bytes
// is the only way to put the decode itself under test.
//
// Caveat worth knowing: these payloads are hand-authored to match Stripe's
// documented shape, not captured from a live webhook. If a field name here is
// wrong in the same way production is wrong, the test passes anyway. Replacing
// them with a recorded event would close that gap; nothing else in the repo
// captures Stripe fixtures yet, so there is no pattern to follow.
func TestInvoiceCreated_DecodesSubscription(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		fragment string
		want     string
	}{
		{
			name:     "basil and later: parent.subscription_details.subscription",
			fragment: `,"parent":{"type":"subscription_details","subscription_details":{"subscription":"sub_current"}}`,
			want:     "sub_current",
		},
		{
			// Deliberately unsupported. The SDK's Invoice has no top-level
			// subscription field, because Stripe removed it, so reading one would
			// mean reintroducing exactly the hand-rolled json tag that caused this
			// bug. An endpoint pinned to a pre-basil API version therefore resolves
			// to "" and is ignored, which the ErrIgnore message names explicitly
			// instead of failing silently. Our endpoint is on 2026-06-24.dahlia.
			name:     "pre-basil top-level subscription is not read",
			fragment: `,"subscription":"sub_legacy"`,
			want:     "",
		},
		{
			name:     "parent present but expanded to a full object",
			fragment: `,"parent":{"type":"subscription_details","subscription_details":{"subscription":{"id":"sub_expanded","object":"subscription"}}}`,
			want:     "sub_expanded",
		},
		{
			name:     "neither shape present",
			fragment: ``,
			want:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var got stripesdk.Invoice
			require.NoError(t, json.Unmarshal([]byte(invoiceJSON(tt.fragment)), &got))
			require.Equal(t, "in_1", got.ID)
			require.Equal(t, "cus_1", customerID(got))
			require.Equal(t, stripesdk.InvoiceBillingReasonSubscriptionCycle, got.BillingReason)
			require.Equal(t, int64(1750000000), got.PeriodStart)
			require.Equal(t, int64(1752678400), got.PeriodEnd)
			require.Equal(t, tt.want, deploySubscriptionID(got))
		})
	}
}

func TestInvoiceCreated_IgnoresNonRenewal(t *testing.T) {
	t.Parallel()

	h := &handler{} //nolint:exhaustruct // early-return paths need no deps
	inv := renewalInvoice("cus_test", "sub_test")
	inv.BillingReason = stripesdk.InvoiceBillingReasonSubscriptionCreate
	err := h.invoiceCreated(context.Background(), webhook.Event{}, inv)
	require.ErrorIs(t, err, webhook.ErrIgnore)
}

func TestInvoiceCreated_IgnoresUnknownCustomer(t *testing.T) {
	t.Parallel()

	mysqlCfg := containers.MySQL(t)
	database, err := db.New(mysqlCfg.DSN, sqlcomment.Disabled())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, database.Close()) })

	h := &handler{db: database} //nolint:exhaustruct // restate/stripe unused on ignore path
	err = h.invoiceCreated(context.Background(), webhook.Event{}, renewalInvoice("cus_no_deploy_workspace", "sub_test"))
	require.ErrorIs(t, err, webhook.ErrIgnore)
}

func TestInvoiceCreated_IgnoresMissingCustomerOrPeriod(t *testing.T) {
	t.Parallel()

	h := &handler{} //nolint:exhaustruct // early-return paths need no deps
	inv := renewalInvoice("cus_test", "sub_test")

	inv.Customer = nil
	err := h.invoiceCreated(context.Background(), webhook.Event{}, inv)
	require.ErrorIs(t, err, webhook.ErrIgnore)

	inv = renewalInvoice("cus_test", "sub_test")
	inv.PeriodStart = 0
	err = h.invoiceCreated(context.Background(), webhook.Event{}, inv)
	require.ErrorIs(t, err, webhook.ErrIgnore)

	inv = renewalInvoice("cus_test", "")
	err = h.invoiceCreated(context.Background(), webhook.Event{}, inv)
	require.ErrorIs(t, err, webhook.ErrIgnore)

	inv = renewalInvoice("cus_test", "sub_test")
	inv.PeriodEnd = 0
	err = h.invoiceCreated(context.Background(), webhook.Event{}, inv)
	require.ErrorIs(t, err, webhook.ErrIgnore)
}

func TestInvoiceCreated_RejectsEmptyBillingReason(t *testing.T) {
	t.Parallel()

	mysqlCfg := containers.MySQL(t)
	database, err := db.New(mysqlCfg.DSN, sqlcomment.Disabled())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, database.Close()) })

	h := &handler{db: database} //nolint:exhaustruct // stripe/restate unused on ignore path
	inv := renewalInvoice("cus_test", "sub_test")
	inv.BillingReason = ""
	err = h.invoiceCreated(context.Background(), webhook.Event{}, inv)
	require.ErrorIs(t, err, webhook.ErrIgnore)
}

func TestFindDeployWorkspaceByStripeCustomerID_RequiresDeployPlan(t *testing.T) {
	t.Parallel()

	mysqlCfg := containers.MySQL(t)
	database, err := db.New(mysqlCfg.DSN, sqlcomment.Disabled())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, database.Close()) })

	_, err = database.FindDeployWorkspaceByStripeCustomerID(context.Background(), sql.NullString{
		String: "cus_without_workspace",
		Valid:  true,
	})
	require.True(t, db.IsNotFound(err))
}

func TestInvoiceCreated_IgnoresMismatchedSubscription(t *testing.T) {
	t.Parallel()

	mysqlCfg := containers.MySQL(t)
	database, err := db.New(mysqlCfg.DSN, sqlcomment.Disabled())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, database.Close()) })

	// Unique ids per run. The MySQL container is reused across `go test`
	// invocations, so hardcoded ids made this pass once and then fail on every
	// re-run with a duplicate-key error on workspaces.id.
	wsID := uid.New(uid.WorkspacePrefix)
	customerID := uid.New("cus")
	subID := uid.New("sub")

	require.NoError(t, database.InsertWorkspace(context.Background(), db.InsertWorkspaceParams{
		ID:        wsID,
		OrgID:     uid.New(uid.OrgPrefix),
		Name:      "Deploy WS",
		Slug:      wsID,
		CreatedAt: time.Now().UnixMilli(),
	}))
	_, err = database.RW().ExecContext(context.Background(),
		`INSERT INTO workspace_billing (workspace_id, plan, stripe_customer_id) VALUES (?, ?, ?)`,
		wsID, "pro", customerID,
	)
	require.NoError(t, err)
	_, err = database.RW().ExecContext(context.Background(),
		`INSERT INTO billing_subscriptions (workspace_id, product, stripe_subscription_id) VALUES (?, 'compute', ?)`,
		wsID, subID,
	)
	require.NoError(t, err)

	h := &handler{db: database} //nolint:exhaustruct // stripe/restate unused on ignore path
	err = h.invoiceCreated(context.Background(), webhook.Event{}, renewalInvoice(customerID, "sub_other_product"))
	require.ErrorIs(t, err, webhook.ErrIgnore)
}
