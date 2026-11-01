package gdprerasure

import (
	"context"

	restate "github.com/restatedev/sdk-go"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

// step is one erasure action. A subject's steps run in registry order, and
// every step converges: run again, it finds nothing left and affects zero rows.
type step interface {
	name() string
	execute(ctx restate.Context, w *Workflow, s subject) (int64, error)
}

// mysqlQuery runs one batch of a generated erasure query for the subject and
// returns the rows it affected.
type mysqlQuery func(ctx context.Context, q db.Querier, s subject) (int64, error)

// mysqlDelete removes the rows of table that query matches, in bounded
// batches. by tells apart two deletes from the same table in one registry.
type mysqlDelete struct {
	table string
	by    string
	query mysqlQuery
}

func (d mysqlDelete) name() string { return qualify("delete mysql "+d.table, d.by) }

// mysqlUpdate rewrites the rows of table that query matches, in bounded
// batches. query must stop matching a row once it has rewritten it, so the
// batches drain.
type mysqlUpdate struct {
	table string
	by    string
	query mysqlQuery
}

func (u mysqlUpdate) name() string { return qualify("update mysql "+u.table, u.by) }

// clickhouseWorkspaceDelete removes the workspace's rows from table, one of
// clickhouse.WorkspaceErasureTables.
type clickhouseWorkspaceDelete struct {
	table string
}

func (d clickhouseWorkspaceDelete) name() string { return "delete clickhouse " + d.table }

// clickhouseEndUserDelete removes the end user's rows from table, one of
// clickhouse.EndUserErasureTables.
type clickhouseEndUserDelete struct {
	table string
}

func (d clickhouseEndUserDelete) name() string { return "delete clickhouse " + d.table }

// pseudonymizeAuditLogs blanks the user's personal data in the audit log rows
// they wrote.
type pseudonymizeAuditLogs struct{}

func (pseudonymizeAuditLogs) name() string { return "update clickhouse audit_logs_raw_v1" }

// teardownCompute stops the workspace's running Deploy compute and waits for it
// to drain, so no row is deleted from under a running pod.
type teardownCompute struct{}

func (teardownCompute) name() string { return "teardown compute" }

// deleteStripeCustomer deletes the workspace's Stripe customer.
type deleteStripeCustomer struct{}

func (deleteStripeCustomer) name() string { return "delete stripe customer" }

// deleteDepotProjects deletes the Depot projects of the workspace's projects.
type deleteDepotProjects struct{}

func (deleteDepotProjects) name() string { return "delete depot projects" }

// deleteWorkOSOrganization deletes the workspace's WorkOS organization.
type deleteWorkOSOrganization struct{}

func (deleteWorkOSOrganization) name() string { return "delete workos organization" }

// deleteResendContact deletes the dashboard user's Resend contact.
type deleteResendContact struct{}

func (deleteResendContact) name() string { return "delete resend contact" }

// deletePlainCustomer deletes the dashboard user's Plain customer.
type deletePlainCustomer struct{}

func (deletePlainCustomer) name() string { return "delete plain customer" }

// deleteWorkOSUser deletes the dashboard user's WorkOS account.
type deleteWorkOSUser struct{}

func (deleteWorkOSUser) name() string { return "delete workos user" }

func qualify(name, by string) string {
	if by == "" {
		return name
	}
	return name + " by " + by
}
