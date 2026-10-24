package keyrefill_test

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	restate "github.com/restatedev/sdk-go"
	"github.com/stretchr/testify/require"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/healthcheck"
	"github.com/unkeyed/unkey/pkg/testutil/containers"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
	"github.com/unkeyed/unkey/svc/ctrl/worker/cron/keyrefill"
)

func TestFailedRefillReleasesDateForCatchUp(t *testing.T) {
	database := &refillDB{}
	database.failing.Store(true)
	handler, err := keyrefill.New(keyrefill.Config{
		DB:        database,
		Heartbeat: healthcheck.NewNoop(),
	})
	require.NoError(t, err)
	cfg := containers.Restate(t, hydrav1.NewCronServiceServer(&refillService{handler: handler}).
		ConfigureHandler("RunKeyRefill", keyrefill.RetryPolicy()))
	client := hydrav1.NewCronServiceIngressClient(cfg.IngressClient, "2026-09-22")

	ctx, cancel := context.WithTimeout(t.Context(), 20*time.Second)
	t.Cleanup(cancel)
	_, err = client.RunKeyRefill().Request(ctx, &hydrav1.RunKeyRefillRequest{})
	require.ErrorContains(t, err, "database unavailable")
	require.NoError(t, ctx.Err(), "a persistent failure must terminate before the request deadline")

	database.failing.Store(false)
	catchUpCtx, cancelCatchUp := context.WithTimeout(t.Context(), 20*time.Second)
	t.Cleanup(cancelCatchUp)
	response, err := client.RunKeyRefill().Request(catchUpCtx, &hydrav1.RunKeyRefillRequest{})
	require.NoError(t, err, "a fresh invocation must be able to retry the failed date")
	require.Zero(t, response.GetKeysRefilled())
}

type refillDB struct {
	db.Database
	failing atomic.Bool
}

func (d *refillDB) ListKeysForRefill(context.Context, db.ListKeysForRefillParams) ([]db.ListKeysForRefillRow, error) {
	if d.failing.Load() {
		return nil, errors.New("database unavailable")
	}
	return nil, nil
}

type refillService struct {
	hydrav1.UnimplementedCronServiceServer
	handler *keyrefill.Handler
}

func (s *refillService) RunKeyRefill(ctx restate.ObjectContext, req *hydrav1.RunKeyRefillRequest) (*hydrav1.RunKeyRefillResponse, error) {
	return s.handler.Handle(ctx, req)
}
