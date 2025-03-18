package testutil

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/go/internal/services/keys"
	"github.com/unkeyed/unkey/go/internal/services/permissions"
	"github.com/unkeyed/unkey/go/internal/services/ratelimit"
	"github.com/unkeyed/unkey/go/pkg/clock"
	"github.com/unkeyed/unkey/go/pkg/cluster"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/hash"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"github.com/unkeyed/unkey/go/pkg/testutil/containers"
	"github.com/unkeyed/unkey/go/pkg/uid"
	"github.com/unkeyed/unkey/go/pkg/zen"
	"github.com/unkeyed/unkey/go/pkg/zen/validation"
)

type Resources struct {
	RootWorkspace db.Workspace
	RootKeyring   db.KeyAuth
	UserWorkspace db.Workspace
}

type Harness struct {
	t *testing.T

	Clock *clock.TestClock

	srv        *zen.Server
	containers *containers.Containers
	validator  *validation.Validator

	middleware []zen.Middleware

	DB          db.Database
	Logger      logging.Logger
	Keys        keys.KeyService
	Permissions permissions.PermissionService
	Ratelimit   ratelimit.Service
	Resources   Resources
}

func NewHarness(t *testing.T) *Harness {
	clk := clock.NewTestClock()

	logger := logging.New()

	cont := containers.New(t)

	dsn := cont.RunMySQL()

	db, err := db.New(db.Config{
		Logger:      logger,
		PrimaryDSN:  dsn,
		ReadOnlyDSN: "",
	})
	require.NoError(t, err)

	srv, err := zen.New(zen.Config{
		NodeID: "test",
		Logger: logger,
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

	h := Harness{
		t:           t,
		Logger:      logger,
		srv:         srv,
		containers:  cont,
		validator:   validator,
		Keys:        keyService,
		Permissions: permissionService,
		Ratelimit:   ratelimitService,
		DB:          db,
		// resources are seeded later
		// nolint:exhaustruct
		Resources: Resources{},
		Clock:     clk,

		middleware: []zen.Middleware{
			zen.WithTracing(),
			//	zen.WithMetrics(svc.EventBuffer)
			zen.WithLogging(logger),
			zen.WithErrorHandling(logger),
			zen.WithValidation(validator),
		},
	}

	h.seed()
	return &h
}

// Register registers a route with the harness.
// You can override the middleware by passing a list of middleware.
func (h *Harness) Register(route zen.Route, middleware ...zen.Middleware) {

	if len(middleware) == 0 {
		middleware = h.middleware
	}

	h.srv.RegisterRoute(
		middleware,
		route,
	)

}

func (h *Harness) seed() {

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

func (h *Harness) CreateRootKey(workspaceID string, permissions ...string) string {

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

// Post is a helper function to make a POST request to the API.
// It will hanndle serializing the request and response objects to and from JSON.
func UnmarshalBody[Body any](t *testing.T, r *httptest.ResponseRecorder, body *Body) {

	err := json.Unmarshal(r.Body.Bytes(), &body)
	require.NoError(t, err)

}

type TestResponse[TBody any] struct {
	Status  int
	Headers http.Header
	Body    *TBody
	RawBody string
}

func CallRoute[Req any, Res any](h *Harness, route zen.Route, headers http.Header, req Req) TestResponse[Res] {
	h.t.Helper()

	rr := httptest.NewRecorder()

	body := new(bytes.Buffer)
	err := json.NewEncoder(body).Encode(req)
	require.NoError(h.t, err)

	httpReq := httptest.NewRequest(route.Method(), route.Path(), body)
	httpReq.Header = headers
	if httpReq.Header == nil {
		httpReq.Header = http.Header{}
	}

	h.srv.Mux().ServeHTTP(rr, httpReq)
	require.NoError(h.t, err)

	rawBody := rr.Body.Bytes()

	res := TestResponse[Res]{
		Status:  rr.Code,
		Headers: rr.Header(),
		RawBody: string(rawBody),
		Body:    nil,
	}

	var responseBody Res
	err = json.Unmarshal(rawBody, &responseBody)
	require.NoError(h.t, err)

	res.Body = &responseBody

	return res
}
