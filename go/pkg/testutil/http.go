package testutil

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/go/api"
	"github.com/unkeyed/unkey/go/internal/services/keys"
	"github.com/unkeyed/unkey/go/pkg/clock"
	"github.com/unkeyed/unkey/go/pkg/database"
	"github.com/unkeyed/unkey/go/pkg/entities"
	"github.com/unkeyed/unkey/go/pkg/hash"
	"github.com/unkeyed/unkey/go/pkg/logging"
	"github.com/unkeyed/unkey/go/pkg/uid"
	"github.com/unkeyed/unkey/go/pkg/zen"
	"github.com/unkeyed/unkey/go/pkg/zen/validation"
)

type Resources struct {
	RootWorkspace entities.Workspace
	RootKeyring   entities.Keyring
	UserWorkspace entities.Workspace
}

type Harness struct {
	t *testing.T

	Clock clock.Clock

	srv        *zen.Server
	containers *Containers
	validator  *validation.Validator

	middleware []zen.Middleware

	DB        database.Database
	Logger    logging.Logger
	Keys      keys.KeyService
	Resources Resources
}

func NewHarness(t *testing.T) *Harness {
	clk := clock.NewTestClock()

	logger := logging.New(logging.Config{Development: true, NoColor: false})

	containers := NewContainers(t)

	dsn := containers.RunMySQL()

	db, err := database.New(database.Config{
		Logger:      logger,
		PrimaryDSN:  dsn,
		ReadOnlyDSN: "",
		Clock:       clk,
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

	h := Harness{
		t:          t,
		Logger:     logger,
		srv:        srv,
		containers: containers,
		validator:  validator,
		Keys:       keyService,
		DB:         db,
		// resources are seeded later
		// nolint:exhaustruct
		Resources: Resources{},
		Clock:     clk,

		middleware: []zen.Middleware{
			zen.WithTracing(),
			//	zen.WithMetrics(svc.EventBuffer)
			zen.WithLogging(logger),
			zen.WithErrorHandling(),
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

	rootWorkspace := entities.Workspace{
		ID:                   uid.New("test_ws"),
		TenantID:             "unkey",
		Name:                 "unkey",
		CreatedAt:            time.Now(),
		DeletedAt:            time.Time{},
		Plan:                 entities.WorkspacePlanPro,
		Enabled:              true,
		DeleteProtection:     true,
		BetaFeatures:         make(map[string]interface{}),
		Features:             make(map[string]interface{}),
		StripeCustomerID:     "",
		StripeSubscriptionID: "",
		TrialEnds:            time.Time{},
		PlanLockedUntil:      time.Time{},
	}

	err := h.DB.InsertWorkspace(context.Background(), rootWorkspace)
	require.NoError(h.t, err)

	rootKeyring := entities.Keyring{
		ID:                 uid.New("test_kr"),
		WorkspaceID:        rootWorkspace.ID,
		StoreEncryptedKeys: false,
		DefaultPrefix:      "test",
		DefaultBytes:       16,
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Time{},
		DeletedAt:          time.Time{},
	}

	err = h.DB.InsertKeyring(context.Background(), rootKeyring)
	require.NoError(h.t, err)

	userWorkspace := entities.Workspace{
		ID:                   uid.New("test_ws"),
		TenantID:             "user",
		Name:                 "user",
		CreatedAt:            time.Now(),
		DeletedAt:            time.Time{},
		Plan:                 entities.WorkspacePlanPro,
		Enabled:              true,
		DeleteProtection:     true,
		BetaFeatures:         make(map[string]interface{}),
		Features:             make(map[string]interface{}),
		StripeCustomerID:     "",
		StripeSubscriptionID: "",
		TrialEnds:            time.Time{},
		PlanLockedUntil:      time.Time{},
	}

	err = h.DB.InsertWorkspace(context.Background(), userWorkspace)
	require.NoError(h.t, err)

	h.Resources = Resources{
		RootWorkspace: rootWorkspace,
		RootKeyring:   rootKeyring,
		UserWorkspace: userWorkspace,
	}

}

func (h *Harness) CreateRootKey() string {

	key := uid.New("test_root_key")

	err := h.DB.InsertKey(context.Background(), entities.Key{
		ID:                uid.New("test_root_key"),
		Hash:              hash.Sha256(key),
		WorkspaceID:       h.Resources.RootWorkspace.ID,
		ForWorkspaceID:    h.Resources.UserWorkspace.ID,
		KeyringID:         h.Resources.RootKeyring.ID,
		Start:             key[:4],
		Name:              "test",
		Identity:          nil,
		Meta:              make(map[string]any),
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Time{},
		DeletedAt:         time.Time{},
		Enabled:           true,
		Environment:       "",
		Expires:           time.Time{},
		Permissions:       []string{},
		RemainingRequests: nil,
	})
	require.NoError(h.t, err)

	return key

}

// Post is a helper function to make a POST request to the API.
// It will hanndle serializing the request and response objects to and from JSON.
func UnmarshalBody[Body any](t *testing.T, r *httptest.ResponseRecorder, body *Body) {

	err := json.Unmarshal(r.Body.Bytes(), &body)
	require.NoError(t, err)

}

type TestResponse[TBody any] struct {
	Status    int
	Headers   http.Header
	Body      *TBody
	ErrorBody *api.BaseError
	RawBody   string
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
		Status:    rr.Code,
		Headers:   rr.Header(),
		RawBody:   string(rawBody),
		Body:      nil,
		ErrorBody: nil,
	}

	if rr.Code < 400 {
		var responseBody Res
		err = json.Unmarshal(rawBody, &responseBody)
		require.NoError(h.t, err)
		res.Body = &responseBody
	} else {
		var errorBody api.BaseError
		err = json.Unmarshal(rawBody, &errorBody)
		require.NoError(h.t, err)
		res.ErrorBody = &errorBody
	}

	return res
}
