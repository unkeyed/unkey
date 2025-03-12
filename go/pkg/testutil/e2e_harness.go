package testutil

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/go/apps/api"
	"github.com/unkeyed/unkey/go/internal/services/keys"
	"github.com/unkeyed/unkey/go/internal/services/permissions"
	"github.com/unkeyed/unkey/go/internal/services/ratelimit"
	"github.com/unkeyed/unkey/go/pkg/clock"
	"github.com/unkeyed/unkey/go/pkg/cluster"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/hash"
	"github.com/unkeyed/unkey/go/pkg/logging"
	"github.com/unkeyed/unkey/go/pkg/port"
	"github.com/unkeyed/unkey/go/pkg/uid"
	"github.com/unkeyed/unkey/go/pkg/zen"
	"github.com/unkeyed/unkey/go/pkg/zen/validation"
)

type IntegrationHarness struct {
	t *testing.T

	Clock *clock.TestClock

	containers *Containers
	validator  *validation.Validator
	ports      *port.FreePort

	nodes      []api.Config
	middleware []zen.Middleware

	dbDsn       string
	DB          db.Database
	Logger      logging.Logger
	Keys        keys.KeyService
	Permissions permissions.PermissionService
	Ratelimit   ratelimit.Service
	Resources   Resources
}

func NewIntegrationHarness(t *testing.T, nodes int) *IntegrationHarness {
	clk := clock.NewTestClock()

	logger := logging.New(logging.Config{Development: true, NoColor: false})

	containers := NewContainers(t)

	dsn := containers.RunMySQL()

	db, err := db.New(db.Config{
		Logger:      logger,
		PrimaryDSN:  dsn,
		ReadOnlyDSN: "",
	})
	require.NoError(t, err)

	keyService, err := keys.New(keys.Config{
		Logger: logger,
		DB:     db,
	})
	require.NoError(t, err)

	validator, err := validation.New()
	require.NoError(t, err)

	permissionService := permissions.New(permissions.Config{
		DB:     db,
		Logger: logger,
	})

	ratelimitService, err := ratelimit.New(ratelimit.Config{
		Logger:  logger,
		Cluster: cluster.NewNoop("test", "localhost"),
		Clock:   clk,
	})
	require.NoError(t, err)

	h := IntegrationHarness{
		t:           t,
		dbDsn:       dsn,
		Logger:      logger,
		ports:       port.New(),
		containers:  containers,
		validator:   validator,
		Keys:        keyService,
		Permissions: permissionService,
		Ratelimit:   ratelimitService,
		DB:          db,
		// resources are seeded later
		// nolint:exhaustruct
		Resources: Resources{},
		Clock:     clk,
		nodes:     []api.Config{},
	}
	h.seed()
	for range nodes {
		h.startApi()
	}

	return &h
}

func (h *IntegrationHarness) startApi() {

	addrs := []string{}
	for _, peer := range h.nodes {
		addrs = append(addrs, fmt.Sprintf("%s:%d", peer.ClusterAdvertiseAddrStatic, peer.ClusterGossipPort))
	}

	config := api.Config{
		Platform:                    "test",
		Image:                       "test",
		HttpPort:                    h.ports.Get(),
		Region:                      "test",
		Clock:                       h.Clock,
		ClusterEnabled:              true,
		ClusterNodeID:               uid.New("test"),
		ClusterAdvertiseAddrStatic:  "localhost",
		ClusterRpcPort:              h.ports.Get(),
		ClusterGossipPort:           h.ports.Get(),
		ClusterDiscoveryStaticAddrs: addrs,
		LogsColor:                   true,
		ClickhouseURL:               "",
		DatabasePrimary:             h.dbDsn,
		DatabaseReadonlyReplica:     "",
		OtelOtlpEndpoint:            "",
	}
	h.nodes = append(h.nodes, config)

	go func() {

		err := api.Run(context.Background(), config)
		require.NoError(h.t, err)
	}()

}

func (h *IntegrationHarness) seed() {

	ctx := context.Background()

	insertRootWorkspaceParams := db.InsertWorkspaceParams{
		ID:        uid.New("test_ws"),
		TenantID:  "unkey",
		Name:      "unkey",
		CreatedAt: time.Now().UnixMilli(),
	}

	err := db.Query.InsertWorkspace(ctx, h.DB.RW(), insertRootWorkspaceParams)
	require.NoError(h.t, err)

	rootWorkspace, err := db.Query.FindWorkspaceByID(ctx, h.DB.RW(), insertRootWorkspaceParams.ID)
	require.NoError(h.t, err)

	insertRootKeyringParams := db.InsertKeyringParams{
		ID:                 uid.New("test_kr"),
		WorkspaceID:        rootWorkspace.ID,
		StoreEncryptedKeys: false,
		DefaultPrefix:      sql.NullString{String: "test", Valid: true},
		DefaultBytes:       sql.NullInt32{Int32: 8, Valid: true},
		CreatedAtM:         time.Now().UnixMilli(),
	}

	err = db.Query.InsertKeyring(context.Background(), h.DB.RW(), insertRootKeyringParams)
	require.NoError(h.t, err)

	rootKeyring, err := db.Query.FindKeyringByID(ctx, h.DB.RW(), insertRootKeyringParams.ID)
	require.NoError(h.t, err)

	insertUserWorkspaceParams := db.InsertWorkspaceParams{
		ID:        uid.New("test_ws"),
		TenantID:  "user",
		Name:      "user",
		CreatedAt: time.Now().UnixMilli(),
	}

	err = db.Query.InsertWorkspace(ctx, h.DB.RW(), insertUserWorkspaceParams)
	require.NoError(h.t, err)

	userWorkspace, err := db.Query.FindWorkspaceByID(ctx, h.DB.RW(), insertUserWorkspaceParams.ID)
	require.NoError(h.t, err)

	h.Resources = Resources{
		RootWorkspace: rootWorkspace,
		RootKeyring:   rootKeyring,
		UserWorkspace: userWorkspace,
	}

}

func (h *IntegrationHarness) CreateRootKey(workspaceID string, permissions ...string) string {

	key := uid.New("test_root_key")

	insertKeyParams := db.InsertKeyParams{
		ID:                uid.New("test_root_key"),
		Hash:              hash.Sha256(key),
		WorkspaceID:       h.Resources.RootWorkspace.ID,
		ForWorkspaceID:    sql.NullString{String: workspaceID, Valid: true},
		KeyringID:         h.Resources.RootKeyring.ID,
		Start:             key[:4],
		CreatedAtM:        time.Now().UnixMilli(),
		Enabled:           true,
		Name:              sql.NullString{String: "", Valid: false},
		IdentityID:        sql.NullString{String: "", Valid: false},
		Meta:              sql.NullString{String: "", Valid: false},
		Expires:           sql.NullTime{Time: time.Time{}, Valid: false},
		RemainingRequests: sql.NullInt32{Int32: 0, Valid: false},
		RatelimitAsync:    sql.NullBool{Bool: false, Valid: false},
		RatelimitLimit:    sql.NullInt32{Int32: 0, Valid: false},
		RatelimitDuration: sql.NullInt64{Int64: 0, Valid: false},
		Environment:       sql.NullString{String: "", Valid: false},
	}

	err := db.Query.InsertKey(context.Background(), h.DB.RW(), insertKeyParams)
	require.NoError(h.t, err)

	if len(permissions) > 0 {
		for _, permission := range permissions {
			permissionID := uid.New(uid.TestPrefix)
			err = db.Query.InsertPermission(context.Background(), h.DB.RW(), db.InsertPermissionParams{
				ID:          permissionID,
				WorkspaceID: h.Resources.RootWorkspace.ID,
				Name:        permission,
				Description: sql.NullString{String: "", Valid: false},
				CreatedAt:   time.Now().UnixMilli(),
			})
			require.NoError(h.t, err)

			err = db.Query.InsertKeyPermission(context.Background(), h.DB.RW(), db.InsertKeyPermissionParams{
				PermissionID: permissionID,
				KeyID:        insertKeyParams.ID,
				WorkspaceID:  h.Resources.RootWorkspace.ID,
				CreatedAt:    time.Now().UnixMilli(),
			})
			require.NoError(h.t, err)
		}
	}

	return key

}

func CallRouteE2e[Req any, Res any](h *IntegrationHarness, method string, path string, headers http.Header, req Req) TestResponse[Res] {
	h.t.Helper()

	require.NotEmpty(h.t, h.nodes)

	node := h.nodes[rand.IntN(len(h.nodes))]

	body := new(bytes.Buffer)
	err := json.NewEncoder(body).Encode(req)
	require.NoError(h.t, err)

	httpReq, err := http.NewRequest(method, fmt.Sprintf("http://localhost:%d%s", node.HttpPort, path), body)
	require.NoError(h.t, err)

	httpReq.Header = headers
	if httpReq.Header == nil {
		httpReq.Header = http.Header{}
	}

	httpRes, err := http.DefaultClient.Do(httpReq)
	require.NoError(h.t, err)
	defer httpRes.Body.Close()

	rawBody, err := io.ReadAll(httpRes.Body)
	require.NoError(h.t, err)

	res := TestResponse[Res]{
		Status:  httpRes.StatusCode,
		Headers: httpRes.Header.Clone(),
		RawBody: string(rawBody),
		Body:    nil,
	}

	var responseBody Res
	err = json.Unmarshal(rawBody, &responseBody)
	require.NoError(h.t, err)

	res.Body = &responseBody

	return res
}
