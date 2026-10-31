package db_test

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"
	frontlinev1 "github.com/unkeyed/unkey/gen/proto/frontline/v1"
	pkgdb "github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/testutil/containers"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/frontline/internal/db"
	"google.golang.org/protobuf/encoding/protojson"
)

func TestFindFrontlineRouteByFQDN_OwnershipWithoutInstances(t *testing.T) {
	mysql := containers.MySQL(t)
	connection, err := sql.Open("mysql", mysql.DSN)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, connection.Close()) })
	tx, err := connection.BeginTx(t.Context(), nil)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, tx.Rollback()) })
	queries := (&db.Queries{}).WithTx(tx)
	config, err := protojson.Marshal(&frontlinev1.Config{})
	require.NoError(t, err)
	expected := make(map[string]pkgdb.InsertDeploymentParams)
	for range 2 {
		deployment := pkgdb.InsertDeploymentParams{
			ID: uid.New(uid.DeploymentPrefix), K8sName: uid.DNS1035(),
			WorkspaceID: uid.New(uid.WorkspacePrefix), ProjectID: uid.New(uid.ProjectPrefix),
			AppID: uid.New(uid.AppPrefix), EnvironmentID: uid.New(uid.EnvironmentPrefix),
			Source: "unknown", Status: "ready", ShutdownSignal: "SIGTERM", UpstreamProtocol: "http1", DeploymentTrigger: "unknown",
			SentinelConfig: config, EncryptedEnvironmentVariables: []byte{},
		}
		require.NoError(t, pkgdb.Query.InsertDeployment(t.Context(), tx, deployment))
		hostname := uid.New(uid.TestPrefix) + ".example.test"
		require.NoError(t, pkgdb.Query.InsertFrontlineRoute(t.Context(), tx, pkgdb.InsertFrontlineRouteParams{
			ID: uid.New(uid.FrontlineRoutePrefix), ProjectID: deployment.ProjectID, AppID: deployment.AppID,
			DeploymentID: deployment.ID, EnvironmentID: deployment.EnvironmentID,
			FullyQualifiedDomainName: hostname, Sticky: "deployment",
		}))
		expected[hostname] = deployment
	}
	for hostname, deployment := range expected {
		route, err := queries.FindFrontlineRouteByFQDN(t.Context(), hostname)
		require.NoError(t, err)
		require.Equal(t, deployment.WorkspaceID, route.WorkspaceID)
		require.Equal(t, deployment.ProjectID, route.ProjectID)
		require.Equal(t, deployment.AppID, route.AppID)
		require.Equal(t, deployment.EnvironmentID, route.EnvironmentID)
		require.Equal(t, deployment.ID, route.DeploymentID)
		instances, err := queries.FindInstancesByDeploymentID(t.Context(), deployment.ID)
		require.NoError(t, err)
		require.Empty(t, instances)
	}
	_, err = queries.FindFrontlineRouteByFQDN(t.Context(), uid.New(uid.TestPrefix)+".example.test")
	require.ErrorIs(t, err, sql.ErrNoRows)
}
