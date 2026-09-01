package clickhouseuserreconcile_test

import (
	"context"
	"sync"
	"testing"
	"time"

	restate "github.com/restatedev/sdk-go"
	"github.com/stretchr/testify/require"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/testutil/containers"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
	"github.com/unkeyed/unkey/svc/ctrl/worker/cron/clickhouseuserreconcile"
)

func TestHandlerReconcilesOncePerFingerprint(t *testing.T) {
	workspaceIDs := []string{
		"ws_01", "ws_02", "ws_03", "ws_04", "ws_05", "ws_06",
		"ws_07", "ws_08", "ws_09", "ws_10", "ws_11", "ws_12",
	}
	handler, err := clickhouseuserreconcile.New(clickhouseuserreconcile.Config{
		DB: &workspaceListDB{workspaceIDs: workspaceIDs},
	})
	require.NoError(t, err)

	users := &recordingClickhouseUserService{}
	restateConfig := containers.Restate(t,
		hydrav1.NewCronServiceServer(&reconcileCronService{handler: handler}),
		hydrav1.NewClickhouseUserServiceServer(users),
	)

	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()
	client := hydrav1.NewCronServiceIngressClient(restateConfig.IngressClient, "clickhouse-user-reconcile")
	first, err := client.RunClickhouseUserReconcile().Request(ctx, &hydrav1.RunClickhouseUserReconcileRequest{})
	require.NoError(t, err)
	require.Equal(t, int32(len(workspaceIDs)), first.GetUsersReconfigured())
	require.ElementsMatch(t, workspaceIDs, users.workspaceIDs())

	second, err := client.RunClickhouseUserReconcile().Request(ctx, &hydrav1.RunClickhouseUserReconcileRequest{})
	require.NoError(t, err)
	require.Zero(t, second.GetUsersReconfigured())
	require.ElementsMatch(t, workspaceIDs, users.workspaceIDs(),
		"an unchanged fingerprint must not reconcile users again")
}

type workspaceListDB struct {
	db.Database
	workspaceIDs []string
}

func (d *workspaceListDB) ListClickhouseWorkspaceIDs(context.Context) ([]string, error) {
	return d.workspaceIDs, nil
}

type reconcileCronService struct {
	hydrav1.UnimplementedCronServiceServer
	handler *clickhouseuserreconcile.Handler
}

func (s *reconcileCronService) RunClickhouseUserReconcile(
	ctx restate.ObjectContext,
	req *hydrav1.RunClickhouseUserReconcileRequest,
) (*hydrav1.RunClickhouseUserReconcileResponse, error) {
	return s.handler.Handle(ctx, req)
}

type recordingClickhouseUserService struct {
	hydrav1.UnimplementedClickhouseUserServiceServer
	mu         sync.Mutex
	reconciled []string
}

func (s *recordingClickhouseUserService) ReconcileUser(
	ctx restate.ObjectContext,
	_ *hydrav1.ReconcileUserRequest,
) (*hydrav1.ReconcileUserResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.reconciled = append(s.reconciled, restate.Key(ctx))
	return &hydrav1.ReconcileUserResponse{}, nil
}

func (s *recordingClickhouseUserService) workspaceIDs() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string(nil), s.reconciled...)
}
