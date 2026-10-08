package resourcecleanup

import (
	"context"
	"fmt"

	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

type repairPage struct {
	AfterPK        uint64
	Scanned        int
	EnvironmentIDs []string
}

func (h *Handler) repair(ctx restate.ObjectContext, name string, scan func(context.Context, uint64) (repairPage, error)) error {
	if err := reportScan(ctx, name, false); err != nil {
		return err
	}
	cursor, err := restate.Get[uint64](ctx, name)
	if err != nil {
		return err
	}
	for range batchesPerRun {
		page, err := restate.Run(ctx, func(rc restate.RunContext) (repairPage, error) {
			queryCtx, cancel := context.WithTimeout(rc, queryTimeout)
			defer cancel()
			return scan(queryCtx, cursor)
		}, restate.WithName("find orphaned "+name), restate.WithMaxRetryAttempts(3))
		if err != nil {
			return fmt.Errorf("scan %s: %w", name, err)
		}
		for _, id := range page.EnvironmentIDs {
			hydrav1.NewEnvironmentServiceClient(ctx, id).Delete().Send(&hydrav1.DeleteEnvironmentRequest{},
				restate.WithIdempotencyKey("orphan-environment-"+id))
		}
		cursor = page.AfterPK
		if page.Scanned < batchSize {
			cursor = 0
		}
		restate.Set(ctx, name, cursor)
		if cursor == 0 {
			return reportScan(ctx, name, true)
		}
	}
	return nil
}

func (h *Handler) scanEnvironments(ctx context.Context, cursor uint64) (repairPage, error) {
	rows, err := h.db.ScanEnvironmentsForCleanup(ctx, db.ScanEnvironmentsForCleanupParams{AfterPk: cursor, Limit: batchSize})
	if err != nil {
		return repairPage{}, err
	}
	page := repairPage{AfterPK: cursor, Scanned: len(rows), EnvironmentIDs: nil}
	for _, row := range rows {
		page.AfterPK = row.Pk
		if row.Orphaned != 0 {
			page.EnvironmentIDs = append(page.EnvironmentIDs, row.ID)
		}
	}
	return page, nil
}

func (h *Handler) scanDeployments(ctx context.Context, cursor uint64) (repairPage, error) {
	rows, err := h.db.ScanDeploymentsForCleanup(ctx, db.ScanDeploymentsForCleanupParams{AfterPk: cursor, Limit: batchSize})
	if err != nil {
		return repairPage{}, err
	}
	page := repairPage{AfterPK: cursor, Scanned: len(rows), EnvironmentIDs: nil}
	for _, row := range rows {
		page.AfterPK = row.Pk
		if row.Orphaned != 0 {
			page.EnvironmentIDs = append(page.EnvironmentIDs, row.EnvironmentID)
		}
	}
	return page, nil
}
