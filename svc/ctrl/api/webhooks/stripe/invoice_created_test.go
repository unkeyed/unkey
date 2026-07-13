package stripe

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/mysql/sqlcomment"
	"github.com/unkeyed/unkey/pkg/testutil/containers"
	"github.com/unkeyed/unkey/pkg/webhook"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

func renewalInvoice(customer, subscription string) invoiceCreatedPayload {
	now := time.Now().UTC()
	start := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC).AddDate(0, -1, 0)
	end := start.AddDate(0, 1, 0)
	return invoiceCreatedPayload{
		ID:            "in_test",
		BillingReason: "subscription_cycle",
		Customer:      customer,
		Subscription:  subscription,
		PeriodStart:   start.Unix(),
		PeriodEnd:     end.Unix(),
	}
}

func TestInvoiceCreated_IgnoresNonRenewal(t *testing.T) {
	t.Parallel()

	h := &handler{} //nolint:exhaustruct // early-return paths need no deps
	inv := renewalInvoice("cus_test", "sub_test")
	inv.BillingReason = "subscription_create"
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

	inv.Customer = ""
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

	_, err = database.RW().ExecContext(context.Background(),
		`INSERT INTO workspaces (id, org_id, name, slug, beta_features) VALUES (?, ?, ?, ?, ?)`,
		"ws_deploy", "org_test", "Deploy WS", "deploy-ws", "[]",
	)
	require.NoError(t, err)
	_, err = database.RW().ExecContext(context.Background(),
		`UPDATE workspaces SET deploy_plan = ?, stripe_customer_id = ?, stripe_subscription_id = ? WHERE id = ?`,
		"pro", "cus_deploy", "sub_deploy", "ws_deploy",
	)
	require.NoError(t, err)

	h := &handler{db: database} //nolint:exhaustruct // stripe/restate unused on ignore path
	err = h.invoiceCreated(context.Background(), webhook.Event{}, renewalInvoice("cus_deploy", "sub_other_product"))
	require.ErrorIs(t, err, webhook.ErrIgnore)
}
