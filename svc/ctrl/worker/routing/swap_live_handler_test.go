package routing_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/mysql/sqlcomment"
	restateadmin "github.com/unkeyed/unkey/pkg/restate/admin"
	"github.com/unkeyed/unkey/pkg/testutil/containers"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
	"github.com/unkeyed/unkey/svc/ctrl/worker/routing"
)

// TestSwapMissingDeploymentDoesNotRetry proves an automatic swap whose target
// deployment row is already gone finishes. An automatic promotion reads the
// deployment and its app inside restate.Run, and both lookups return a normal
// error, so a deleted row leaves the swap retrying.
func TestSwapMissingDeploymentDoesNotRetry(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Minute)
	t.Cleanup(cancel)

	database, err := db.New(containers.MySQL(t).DSN, sqlcomment.Disabled())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, database.Close()) })

	svc := routing.New(routing.Config{DB: database, DefaultDomain: "kebap.test"})
	restateCfg := containers.Restate(t, hydrav1.NewRoutingServiceServer(svc))
	admin := restateadmin.New(restateadmin.Config{BaseURL: restateCfg.AdminURL, APIKey: ""})

	environmentID := uid.New(uid.EnvironmentPrefix)
	sent, err := hydrav1.NewRoutingServiceIngressClient(restateCfg.IngressClient, environmentID).
		SwapLiveDeployment().
		Send(ctx, &hydrav1.SwapLiveDeploymentRequest{
			DeploymentId:       uid.New(uid.DeploymentPrefix),
			AutomaticPromotion: true,
		})
	require.NoError(t, err)

	require.Eventually(t, func() bool {
		live, liveErr := admin.FindLiveInvocations(ctx, []string{sent.Id()})
		require.NoError(t, liveErr)
		return !live[sent.Id()]
	}, 20*time.Second, 250*time.Millisecond,
		"invocation %s is still retrying; a missing deployment must be final", sent.Id())
}
