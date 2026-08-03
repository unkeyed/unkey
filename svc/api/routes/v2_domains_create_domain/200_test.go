package handler_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_domains_create_domain"
)

func TestCreateDomainSuccessfully(t *testing.T) {
	h := testutil.NewHarness(t)

	domainID := uid.New(uid.DomainPrefix)
	ctrlClient := &testutil.MockCustomDomainClient{
		AddCustomDomainFunc: func(_ context.Context, _ *ctrlv1.AddCustomDomainRequest) (*ctrlv1.AddCustomDomainResponse, error) {
			return &ctrlv1.AddCustomDomainResponse{
				DomainId:          domainID,
				TargetCname:       "a1b2c3d4e5f6g7h8.cname.unkey.com",
				Status:            ctrlv1.CustomDomainStatus_CUSTOM_DOMAIN_STATUS_PENDING,
				VerificationToken: "3ZQ8xK1mP7vT5nR2wY6bJ4hL",
			}, nil
		},
	}
	route := &handler.Handler{DB: h.DB, CtrlClient: ctrlClient}
	h.Register(route)

	env := seedEnvironment(t, h)
	rootKey := h.CreateRootKey(env.workspaceID, "environment.*.create_domain")

	domain := randomDomain()
	res := testutil.CallRoute[handler.Request, handler.Response](h, route, authHeaders(rootKey), makeRequest(env, domain))
	require.Equal(t, 200, res.Status, "expected 200, received: %s", res.RawBody)
	require.NotEmpty(t, res.Body.Meta.RequestId)
	require.Equal(t, domainID, res.Body.Data.DomainId)
	require.Equal(t, "a1b2c3d4e5f6g7h8.cname.unkey.com", res.Body.Data.TargetCname)
	require.Equal(t, "3ZQ8xK1mP7vT5nR2wY6bJ4hL", res.Body.Data.VerificationToken)

	// Domain Connect discovery found nothing, so both fields stay absent rather
	// than being serialized as empty strings.
	require.Nil(t, res.Body.Data.DomainConnectProvider)
	require.Nil(t, res.Body.Data.DomainConnectUrl)

	// Domain creation and its audit log are delegated to the control plane.
	// Assert the RPC carried the resolved ownership chain and the actor.
	require.Len(t, ctrlClient.AddCustomDomainCalls, 1)
	call := ctrlClient.AddCustomDomainCalls[0]
	require.Equal(t, env.workspaceID, call.GetWorkspaceId())
	require.Equal(t, env.projectID, call.GetProjectId())
	require.Equal(t, env.appID, call.GetAppId())
	require.Equal(t, env.environmentID, call.GetEnvironmentId())
	require.Equal(t, domain, call.GetDomain())
	require.Equal(t, ctrlv1.ActorType_ACTOR_TYPE_ROOT_KEY, call.GetActor().GetType())
}

func TestCreateDomainBySlugs(t *testing.T) {
	h := testutil.NewHarness(t)

	domainID := uid.New(uid.DomainPrefix)
	ctrlClient := &testutil.MockCustomDomainClient{
		AddCustomDomainFunc: func(_ context.Context, _ *ctrlv1.AddCustomDomainRequest) (*ctrlv1.AddCustomDomainResponse, error) {
			return &ctrlv1.AddCustomDomainResponse{
				DomainId:          domainID,
				TargetCname:       "a1b2c3d4e5f6g7h8.cname.unkey.com",
				VerificationToken: "3ZQ8xK1mP7vT5nR2wY6bJ4hL",
			}, nil
		},
	}
	route := &handler.Handler{DB: h.DB, CtrlClient: ctrlClient}
	h.Register(route)

	env := seedEnvironment(t, h)
	rootKey := h.CreateRootKey(env.workspaceID, "environment.*.create_domain")

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, authHeaders(rootKey), handler.Request{
		Project:     env.projectSlug,
		App:         env.appSlug,
		Environment: "production",
		Domain:      randomDomain(),
	})
	require.Equal(t, 200, res.Status, "expected 200, received: %s", res.RawBody)
	require.Equal(t, domainID, res.Body.Data.DomainId)

	// Slugs must resolve to the same internal ids the id-addressed form produces.
	require.Len(t, ctrlClient.AddCustomDomainCalls, 1)
	call := ctrlClient.AddCustomDomainCalls[0]
	require.Equal(t, env.projectID, call.GetProjectId())
	require.Equal(t, env.appID, call.GetAppId())
	require.Equal(t, env.environmentID, call.GetEnvironmentId())
}

func TestCreateDomainWithDomainConnect(t *testing.T) {
	h := testutil.NewHarness(t)

	ctrlClient := &testutil.MockCustomDomainClient{
		AddCustomDomainFunc: func(_ context.Context, _ *ctrlv1.AddCustomDomainRequest) (*ctrlv1.AddCustomDomainResponse, error) {
			return &ctrlv1.AddCustomDomainResponse{
				DomainId:              uid.New(uid.DomainPrefix),
				TargetCname:           "a1b2c3d4e5f6g7h8.cname.unkey.com",
				VerificationToken:     "3ZQ8xK1mP7vT5nR2wY6bJ4hL",
				DomainConnectProvider: "Cloudflare",
				DomainConnectUrl:      "https://dash.cloudflare.com/domainconnect/v2/domaintemplates/apply?domain=example.com",
			}, nil
		},
	}
	route := &handler.Handler{DB: h.DB, CtrlClient: ctrlClient}
	h.Register(route)

	env := seedEnvironment(t, h)
	rootKey := h.CreateRootKey(env.workspaceID, "environment.*.create_domain")

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, authHeaders(rootKey), makeRequest(env, randomDomain()))
	require.Equal(t, 200, res.Status, "expected 200, received: %s", res.RawBody)
	require.NotNil(t, res.Body.Data.DomainConnectProvider, "expected provider, received: %s", res.RawBody)
	require.Equal(t, "Cloudflare", *res.Body.Data.DomainConnectProvider)
	require.NotNil(t, res.Body.Data.DomainConnectUrl, "expected url, received: %s", res.RawBody)
	require.Equal(t, "https://dash.cloudflare.com/domainconnect/v2/domaintemplates/apply?domain=example.com", *res.Body.Data.DomainConnectUrl)
}

func TestCreateDomainWithSpecificEnvironmentPermission(t *testing.T) {
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
	rootKey := h.CreateRootKey(env.workspaceID, "environment."+env.environmentID+".create_domain")

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, authHeaders(rootKey), makeRequest(env, randomDomain()))
	require.Equal(t, 200, res.Status, "expected 200, received: %s", res.RawBody)
}
