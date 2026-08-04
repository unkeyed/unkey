package handler_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/dns/domaingate"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_domains_create_domain"
)

// TestCreateDomainPermissions is a permission matrix. Rejections are 404, not
// 403: a caller that may not read the environment must not be able to tell a
// real environment from a missing one by the status code alone.
func TestCreateDomainPermissions(t *testing.T) {
	h := testutil.NewHarness(t)

	ctrlClient := &testutil.MockCustomDomainClient{
		AddCustomDomainFunc: func(_ context.Context, _ *ctrlv1.AddCustomDomainRequest) (*ctrlv1.AddCustomDomainResponse, error) {
			return &ctrlv1.AddCustomDomainResponse{
				DomainId:          uid.New(uid.DomainPrefix),
				TargetCname:       "a1b2c3d4e5f6g7h8.cname.unkey.com",
				VerificationToken: "3ZQ8xK1mP7vT5nR2wY6bJ4hL",
			}, nil
		},
	}
	route := &handler.Handler{DB: h.DB, CtrlClient: ctrlClient}
	h.Register(route)

	env := seedEnvironment(t, h)

	testCases := []struct {
		name        string
		permissions []string
		shouldPass  bool
	}{
		{name: "wildcard permission", permissions: []string{"environment.*.create_domain"}, shouldPass: true},
		{name: "specific environment permission", permissions: []string{fmt.Sprintf("environment.%s.create_domain", env.environmentID)}, shouldPass: true},
		{name: "permission alongside unrelated grants", permissions: []string{"api.*.read_api", "environment.*.create_domain"}, shouldPass: true},
		{name: "read action is not enough", permissions: []string{"environment.*.read_environment"}, shouldPass: false},
		{name: "update action is not enough", permissions: []string{"environment.*.update_environment"}, shouldPass: false},
		{name: "adjacent set action is not enough", permissions: []string{"environment.*.set_environment_variables"}, shouldPass: false},
		{name: "create_app is not enough", permissions: []string{"project.*.create_app"}, shouldPass: false},
		{name: "action scoped to the wrong resource type", permissions: []string{"app.*.create_domain"}, shouldPass: false},
		{name: "other environment id does not match", permissions: []string{fmt.Sprintf("environment.%s.create_domain", uid.New(uid.EnvironmentPrefix))}, shouldPass: false},
		{name: "legacy urn is not accepted", permissions: []string{fmt.Sprintf("unkey:v1:%s:environments/*#create_domain", env.workspaceID)}, shouldPass: false},
		{name: "unrelated permission", permissions: []string{"api.*.read_api"}, shouldPass: false},
		{name: "no permissions", permissions: []string{}, shouldPass: false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			rootKey := h.CreateRootKey(env.workspaceID, tc.permissions...)
			headers := authHeaders(rootKey)

			res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, makeRequest(env, randomDomain()))
			if tc.shouldPass {
				require.Equal(t, 200, res.Status, "expected 200 for %v, received: %s", tc.permissions, res.RawBody)
				return
			}
			require.Equal(t, http.StatusNotFound, res.Status, "expected 404 for %v, received: %s", tc.permissions, res.RawBody)
			require.NotContains(t, res.RawBody, env.environmentID, "masked 404 leaked the environment id: %s", res.RawBody)
		})
	}
}

// TestCreateDomainPlanAllowanceExceeded pins that a workspace at its custom
// domain allowance gets a 403 naming the two ways out. 403 rather than 429
// because waiting does not change the answer.
//
// The handler runs the allowance check itself, so ctrl is never reached: assert
// that too, since a round trip to be told no is the thing the local check exists
// to avoid.
func TestCreateDomainPlanAllowanceExceeded(t *testing.T) {
	h := testutil.NewHarness(t)

	ctrlClient := &testutil.MockCustomDomainClient{}
	route := &handler.Handler{DB: h.DB, CtrlClient: ctrlClient}
	h.Register(route)

	env := seedEnvironment(t, h)
	setCustomDomainAllowance(t, h, env.workspaceID, 0)
	rootKey := h.CreateRootKey(env.workspaceID, "environment.*.create_domain")

	res := testutil.CallRoute[handler.Request, openapi.ForbiddenErrorResponse](h, route, authHeaders(rootKey), makeRequest(env, randomDomain()))
	require.Equal(t, http.StatusForbidden, res.Status, "expected 403, received: %s", res.RawBody)
	require.Equal(t, "https://unkey.com/docs/errors/unkey/limits/custom_domain_limit_exceeded", res.Body.Error.Type)
	require.Contains(t, res.Body.Error.Detail, "Upgrade your plan",
		"the detail must name a way out, received: %s", res.RawBody)

	require.Empty(t, ctrlClient.AddCustomDomainCalls, "the allowance check must reject before the RPC")

	// The allowance counts stay internal: they describe billing state the caller
	// cannot act on, and the way out is the same either way.
	require.NotContains(t, res.RawBody, "attached")
}

// TestCreateDomainMissingWorkspaceLimits pins the fail-closed path. A workspace
// whose limits row billing never wrote is refused rather than defaulted, and the
// caller is told to contact support rather than to upgrade a plan they may
// already have.
func TestCreateDomainMissingWorkspaceLimits(t *testing.T) {
	h := testutil.NewHarness(t)

	ctrlClient := &testutil.MockCustomDomainClient{}
	route := &handler.Handler{DB: h.DB, CtrlClient: ctrlClient}
	h.Register(route)

	env := seedEnvironment(t, h)
	_, err := h.DB.RW().ExecContext(context.Background(), "DELETE FROM `limits` WHERE workspace_id = ?", env.workspaceID)
	require.NoError(t, err)
	rootKey := h.CreateRootKey(env.workspaceID, "environment.*.create_domain")

	res := testutil.CallRoute[handler.Request, openapi.InternalServerErrorResponse](h, route, authHeaders(rootKey), makeRequest(env, randomDomain()))
	require.Equal(t, http.StatusInternalServerError, res.Status, "expected 500, received: %s", res.RawBody)
	require.Contains(t, res.Body.Error.Detail, "support@unkey.com",
		"the detail must point at support, received: %s", res.RawBody)
	require.Empty(t, ctrlClient.AddCustomDomainCalls)

	// The workspace id is internal detail, not part of the caller's fix.
	require.NotContains(t, res.RawBody, env.workspaceID)
}

// TestCreateDomainRejectionsMatchAcrossLayers pins the reason the checks are
// shared: ctrl re-checks authoritatively, and a rejection that only it can see
// (state changed between the handler's read and ctrl's) must reach the caller as
// the same status and wording the handler would have produced.
func TestCreateDomainRejectionsMatchAcrossLayers(t *testing.T) {
	h := testutil.NewHarness(t)

	// What ctrl sends for an allowance rejection: gatefault carries exactly the
	// gate's public message.
	raced := &testutil.MockCustomDomainClient{
		AddCustomDomainFunc: func(_ context.Context, _ *ctrlv1.AddCustomDomainRequest) (*ctrlv1.AddCustomDomainResponse, error) {
			return nil, connect.NewError(
				connect.CodeResourceExhausted,
				errors.New(fault.UserFacingMessage(domaingate.CheckAllowance(1, 1))),
			)
		},
	}
	racedRoute := &handler.Handler{DB: h.DB, CtrlClient: raced}
	h.Register(racedRoute)

	env := seedEnvironment(t, h)
	rootKey := h.CreateRootKey(env.workspaceID, "environment.*.create_domain")

	// The handler's own allowance check passes, so the rejection comes from ctrl.
	fromCtrl := testutil.CallRoute[handler.Request, openapi.ForbiddenErrorResponse](h, racedRoute, authHeaders(rootKey), makeRequest(env, randomDomain()))
	require.Equal(t, http.StatusForbidden, fromCtrl.Status, "expected 403, received: %s", fromCtrl.RawBody)
	require.Len(t, raced.AddCustomDomainCalls, 1)

	// Now provoke the same rejection locally and compare.
	local := &testutil.MockCustomDomainClient{}
	localRoute := &handler.Handler{DB: h.DB, CtrlClient: local}
	setCustomDomainAllowance(t, h, env.workspaceID, 0)

	fromHandler := testutil.CallRoute[handler.Request, openapi.ForbiddenErrorResponse](h, localRoute, authHeaders(rootKey), makeRequest(env, randomDomain()))
	require.Equal(t, fromCtrl.Status, fromHandler.Status)
	require.Equal(t, fromCtrl.Body.Error.Type, fromHandler.Body.Error.Type)
	require.Equal(t, fromCtrl.Body.Error.Detail, fromHandler.Body.Error.Detail)
}

// TestCreateDomainExistenceNotLeaked asserts that a zero-permission root key
// targeting a real environment and a nonexistent one receives responses that are
// indistinguishable apart from the request id. Without this, permission
// rejections would act as an existence oracle over every environment in the
// workspace. Do not weaken this by returning 403 for the real environment.
func TestCreateDomainExistenceNotLeaked(t *testing.T) {
	h := testutil.NewHarness(t)

	route := &handler.Handler{DB: h.DB, CtrlClient: &testutil.MockCustomDomainClient{}}
	h.Register(route)

	env := seedEnvironment(t, h)
	rootKey := h.CreateRootKey(env.workspaceID)
	headers := authHeaders(rootKey)

	realRes := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, headers, makeRequest(env, randomDomain()))
	missingRes := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, headers, handler.Request{
		Project:     env.projectID,
		App:         env.appID,
		Environment: uid.New(uid.EnvironmentPrefix),
		Domain:      randomDomain(),
	})

	require.Equal(t, http.StatusNotFound, realRes.Status, "expected 404, received: %s", realRes.RawBody)
	require.Equal(t, http.StatusNotFound, missingRes.Status, "expected 404, received: %s", missingRes.RawBody)
	require.NotContains(t, realRes.RawBody, env.environmentID)
	require.Equal(t, missingRes.Body.Error.Detail, realRes.Body.Error.Detail)
	require.Equal(t, missingRes.Body.Error.Type, realRes.Body.Error.Type)
	require.Equal(t, missingRes.Body.Error.Status, realRes.Body.Error.Status)
	require.Equal(t, missingRes.Body.Error.Title, realRes.Body.Error.Title)
}
