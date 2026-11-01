package gdprerasure

import (
	"fmt"

	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/clickhouse"
)

// batchRowsMax is the LIMIT of every gdpr_erasure query. PlanetScale rejects a
// statement that touches more than 100,000 rows; staying well under keeps row
// locks short on shared tables such as keys.
const batchRowsMax = 10_000

func (d mysqlDelete) execute(ctx restate.Context, w *Workflow, s subject) (int64, error) {
	return w.drainMySQL(ctx, d.name(), d.query, s)
}

func (u mysqlUpdate) execute(ctx restate.Context, w *Workflow, s subject) (int64, error) {
	return w.drainMySQL(ctx, u.name(), u.query, s)
}

// drainMySQL runs a batched query until a batch affects fewer rows than the
// limit. There is no batch cap: an erasure must remove everything, and each
// batch is its own journaled step, so a crash resumes at the next batch.
func (w *Workflow) drainMySQL(ctx restate.Context, name string, query mysqlQuery, s subject) (int64, error) {
	var total int64
	for batch := 0; ; batch++ {
		rows, err := restate.Run(ctx, func(rc restate.RunContext) (int64, error) {
			return query(rc, w.db, s)
		}, restate.WithName(fmt.Sprintf("%s batch-%d", name, batch)))
		if err != nil {
			return total, fmt.Errorf("batch %d: %w", batch, err)
		}
		total += rows
		if rows < batchRowsMax {
			return total, nil
		}
	}
}

func (d clickhouseWorkspaceDelete) execute(ctx restate.Context, w *Workflow, s subject) (int64, error) {
	return restate.Run(ctx, func(rc restate.RunContext) (int64, error) {
		return w.clickhouse.EraseWorkspaceRows(rc, d.table, s.workspaceID)
	}, restate.WithName(d.name()))
}

func (d clickhouseEndUserDelete) execute(ctx restate.Context, w *Workflow, s subject) (int64, error) {
	return restate.Run(ctx, func(rc restate.RunContext) (int64, error) {
		return w.clickhouse.EraseEndUserRows(rc, d.table, clickhouse.EndUserErasure{
			WorkspaceID: s.workspaceID,
			ExternalID:  s.externalID,
			IdentityIDs: s.identityIDs,
		})
	}, restate.WithName(d.name()))
}

func (p pseudonymizeAuditLogs) execute(ctx restate.Context, w *Workflow, s subject) (int64, error) {
	return restate.Run(ctx, func(rc restate.RunContext) (int64, error) {
		return w.clickhouse.PseudonymizeAuditActor(rc, s.userID)
	}, restate.WithName(p.name()))
}

func (teardownCompute) execute(ctx restate.Context, _ *Workflow, s subject) (int64, error) {
	res, err := hydrav1.NewDeployTeardownServiceClient(ctx, s.workspaceID).
		Teardown().
		Request(&hydrav1.TeardownRequest{Mode: hydrav1.TeardownMode_TEARDOWN_MODE_ARCHIVE})
	if err != nil {
		return 0, err
	}
	if !res.GetDrained() {
		return 0, restate.TerminalErrorf(
			"workspace %s compute did not drain; nothing was deleted, submit a new erasure once it has drained",
			s.workspaceID,
		)
	}
	return int64(res.GetDeploymentsStopped()), nil
}

func (d deleteStripeCustomer) execute(ctx restate.Context, w *Workflow, s subject) (int64, error) {
	if s.stripeCustomerID == "" {
		return 0, nil
	}
	return deleteExternal(ctx, d.name(), func(rc restate.RunContext) (bool, error) {
		return w.stripe.DeleteCustomer(rc, s.stripeCustomerID)
	})
}

func (d deleteDepotProjects) execute(ctx restate.Context, w *Workflow, s subject) (int64, error) {
	var total int64
	for _, projectID := range s.depotProjectIDs {
		rows, err := deleteExternal(ctx, d.name()+" "+projectID, func(rc restate.RunContext) (bool, error) {
			return w.depot.DeleteProject(rc, projectID)
		})
		if err != nil {
			return total, err
		}
		total += rows
	}
	return total, nil
}

func (d deleteWorkOSOrganization) execute(ctx restate.Context, w *Workflow, s subject) (int64, error) {
	if s.orgID == "" {
		return 0, nil
	}
	return deleteExternal(ctx, d.name(), func(rc restate.RunContext) (bool, error) {
		return w.workos.DeleteOrganization(rc, s.orgID)
	})
}

func (d deleteResendContact) execute(ctx restate.Context, w *Workflow, s subject) (int64, error) {
	if s.email == "" {
		return 0, nil
	}
	return deleteExternal(ctx, d.name(), func(rc restate.RunContext) (bool, error) {
		return w.resend.DeleteContact(rc, s.email)
	})
}

func (d deletePlainCustomer) execute(ctx restate.Context, w *Workflow, s subject) (int64, error) {
	return restate.Run(ctx, func(rc restate.RunContext) (int64, error) {
		return w.plain.DeleteCustomers(rc, s.email, s.userID)
	}, restate.WithName(d.name()))
}

func (d deleteWorkOSUser) execute(ctx restate.Context, w *Workflow, s subject) (int64, error) {
	return deleteExternal(ctx, d.name(), func(rc restate.RunContext) (bool, error) {
		return w.workos.DeleteUser(rc, s.userID)
	})
}

// deleteExternal deletes one record in an external system and counts it as
// one row when it still existed.
func deleteExternal(ctx restate.Context, name string, del func(restate.RunContext) (bool, error)) (int64, error) {
	existed, err := restate.Run(ctx, del, restate.WithName(name))
	if err != nil || !existed {
		return 0, err
	}
	return 1, nil
}
