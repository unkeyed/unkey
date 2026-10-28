package gdprerasure

import (
	"fmt"

	"github.com/ClickHouse/clickhouse-go/v2/lib/chcol"
	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
)

// batchRowsMax bounds each MySQL statement. PlanetScale rejects a statement
// that touches more than 100,000 rows; staying well under keeps row locks short
// on shared tables such as keys.
const batchRowsMax = 10_000

func (d mysqlDelete) execute(ctx restate.Context, w *Workflow, s subject) (int64, error) {
	where, args, err := s.bind(d.where)
	if err != nil {
		return 0, restate.ToTerminalError(err)
	}
	return w.drainMySQL(ctx, d.name(), "DELETE FROM `"+d.table+"` WHERE "+where+" LIMIT ?", args)
}

func (u mysqlUpdate) execute(ctx restate.Context, w *Workflow, s subject) (int64, error) {
	clause, args, err := s.bind("SET " + u.set + " WHERE " + u.where)
	if err != nil {
		return 0, restate.ToTerminalError(err)
	}
	return w.drainMySQL(ctx, u.name(), "UPDATE `"+u.table+"` "+clause+" LIMIT ?", args)
}

// drainMySQL runs a batched statement until a batch affects fewer rows than
// the limit. There is no batch cap: an erasure must remove everything, and each
// batch is its own journaled step, so a crash resumes at the next batch.
func (w *Workflow) drainMySQL(ctx restate.Context, name, query string, args []any) (int64, error) {
	args = append(args, batchRowsMax)
	var total int64
	for batch := 0; ; batch++ {
		rows, err := restate.Run(ctx, func(rc restate.RunContext) (int64, error) {
			res, err := w.db.RW().ExecContext(rc, query, args...)
			if err != nil {
				return 0, err
			}
			return res.RowsAffected()
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

func (d clickhouseDelete) execute(ctx restate.Context, w *Workflow, s subject) (int64, error) {
	where, args, err := s.bind(d.where)
	if err != nil {
		return 0, restate.ToTerminalError(err)
	}
	statement := "DELETE FROM " + d.table + " WHERE " + where
	if d.mode == deleteMutation {
		statement = "ALTER TABLE " + d.table + " DELETE WHERE " + where + " SETTINGS mutations_sync = 2"
	}
	return w.mutateClickHouse(ctx, d.name(), d.table, where, args, statement, args)
}

func (u clickhouseUpdate) execute(ctx restate.Context, w *Workflow, s subject) (int64, error) {
	where, whereArgs, err := s.bind(u.where)
	if err != nil {
		return 0, restate.ToTerminalError(err)
	}
	set, setArgs, err := s.bind(u.set)
	if err != nil {
		return 0, restate.ToTerminalError(err)
	}
	statement := "ALTER TABLE " + u.table + " UPDATE " + set + " WHERE " + where + " SETTINGS mutations_sync = 2"
	return w.mutateClickHouse(ctx, u.name(), u.table, where, whereArgs, statement, append(setArgs, whereArgs...))
}

// mutateClickHouse counts the rows matching where and, when there are any,
// runs statement. Skipping the statement when nothing matches keeps a repeated
// erasure from queueing mutations that rewrite nothing.
func (w *Workflow) mutateClickHouse(
	ctx restate.Context,
	name, table, where string,
	whereArgs []any,
	statement string,
	statementArgs []any,
) (int64, error) {
	return restate.Run(ctx, func(rc restate.RunContext) (int64, error) {
		result, err := w.clickhouse.QueryToMaps(rc, "SELECT count() AS matched FROM "+table+" WHERE "+where, whereArgs...)
		if err != nil {
			return 0, err
		}
		matched, err := countOf(result)
		if err != nil || matched == 0 {
			return 0, err
		}
		if err := w.clickhouse.Exec(rc, statement, statementArgs...); err != nil {
			return 0, err
		}
		return matched, nil
	}, restate.WithName(name))
}

// countOf reads a single count() row. The noop client returns no rows, which
// counts as nothing to erase.
func countOf(result []map[string]any) (int64, error) {
	if len(result) == 0 {
		return 0, nil
	}
	variant, ok := result[0]["matched"].(chcol.Variant)
	if !ok {
		return 0, fmt.Errorf("count returned %T, not a variant", result[0]["matched"])
	}
	matched, ok := variant.Any().(uint64)
	if !ok {
		return 0, fmt.Errorf("count returned %T, not uint64", variant.Any())
	}
	return int64(matched), nil
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
