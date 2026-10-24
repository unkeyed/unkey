package keyrefill_test

import (
	"context"
	"strings"
	"testing"
	"time"

	restate "github.com/restatedev/sdk-go"
	"github.com/stretchr/testify/require"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/healthcheck"
	"github.com/unkeyed/unkey/pkg/mysql/sqlcomment"
	"github.com/unkeyed/unkey/pkg/testutil/containers"
	"github.com/unkeyed/unkey/svc/ctrl/integration/seed"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
	"github.com/unkeyed/unkey/svc/ctrl/worker/cron/keyrefill"
)

// TestHandlerRollsBackRefillOnAuditFailure protects the atomicity of a refill
// and its audit event when MySQL rejects the outbox insert.
func TestHandlerRollsBackRefillOnAuditFailure(t *testing.T) {
	database, err := db.New(containers.MySQLIsolated(t).DSN, sqlcomment.Disabled())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, database.Close()) })
	seeder := seed.New(t, database, nil)
	workspace := seeder.CreateWorkspace(t.Context())
	api := seeder.CreateAPI(t.Context(), seed.CreateApiRequest{WorkspaceID: workspace.ID})
	remaining, amount := int64(17), int64(123)
	key := seeder.CreateKey(t.Context(), seed.CreateKeyRequest{
		WorkspaceID: workspace.ID, KeySpaceID: api.KeyAuthID.String,
		Remaining: &remaining, RefillAmount: &amount,
	})
	handler, err := keyrefill.New(keyrefill.Config{
		DB: &auditFailureDB{Database: database, keyID: key.KeyID}, Heartbeat: healthcheck.NewNoop(),
	})
	require.NoError(t, err)
	cfg := containers.Restate(t, hydrav1.NewCronServiceServer(&refillService{handler: handler}).
		ConfigureHandler("RunKeyRefill", restate.WithInvocationRetryPolicy(
			restate.WithMaxRetryAttempts(1), restate.KillOnMaxAttempts(),
		)))
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()
	client := hydrav1.NewCronServiceIngressClient(cfg.IngressClient, time.Now().UTC().Format("2006-01-02"))
	_, err = client.RunKeyRefill().Request(ctx, &hydrav1.RunKeyRefillRequest{})
	require.Error(t, err)
	require.NotErrorIs(t, err, context.DeadlineExceeded)
	stored, err := database.FindKeyByID(t.Context(), key.KeyID)
	require.NoError(t, err)
	require.Equal(t, remaining, stored.RemainingRequests.Int64)
	require.False(t, stored.LastRefillAt.Valid)
	rows, err := database.ListClickhouseOutboxByWorkspace(t.Context(), workspace.ID)
	require.NoError(t, err)
	require.Empty(t, rows)
}

type auditFailureDB struct {
	db.Database
	keyID string
}

func (d *auditFailureDB) ListKeysForRefill(ctx context.Context, params db.ListKeysForRefillParams) ([]db.ListKeysForRefillRow, error) {
	key, err := d.Database.FindKeyByID(ctx, d.keyID)
	if err != nil {
		return nil, err
	}
	if params.AfterPk >= key.Pk {
		return nil, nil
	}
	return []db.ListKeysForRefillRow{{
		Pk: key.Pk, ID: key.ID, WorkspaceID: strings.Repeat("x", 49),
		RefillAmount: key.RefillAmount, RemainingRequests: key.RemainingRequests, Name: key.Name,
	}}, nil
}

type refillService struct {
	hydrav1.UnimplementedCronServiceServer
	handler *keyrefill.Handler
}

func (s *refillService) RunKeyRefill(ctx restate.ObjectContext, req *hydrav1.RunKeyRefillRequest) (*hydrav1.RunKeyRefillResponse, error) {
	return s.handler.Handle(ctx, req)
}
