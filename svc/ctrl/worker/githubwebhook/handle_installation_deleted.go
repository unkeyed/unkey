package githubwebhook

import (
	"context"

	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

func (s *Service) HandleInstallationDeleted(ctx restate.ObjectContext, req *hydrav1.HandleInstallationDeletedRequest) (*hydrav1.HandleInstallationDeletedResponse, error) {
	err := restate.RunVoid(ctx, func(runCtx restate.RunContext) error {
		return db.TxRetry(runCtx, s.db.RW(), func(txCtx context.Context, tx db.DBTX) error {
			queries := db.NewQueries(tx)
			if err := queries.DeleteGithubRepoConnectionsByInstallationId(txCtx, req.GetInstallationId()); err != nil {
				return err
			}
			return queries.DeleteGithubAppInstallationsByInstallationId(txCtx, req.GetInstallationId())
		})
	}, restate.WithName("delete GitHub installation"))
	if err != nil {
		return nil, err
	}
	return &hydrav1.HandleInstallationDeletedResponse{}, nil
}
