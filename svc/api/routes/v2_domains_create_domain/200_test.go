package handler_test

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
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

	// A subdomain routes through a CNAME and proves ownership through the TXT
	// record. Both are returned, because whether the CNAME can be read back is
	// only knowable after it is published.
	require.Len(t, res.Body.Data.DnsRecords, 2, "expected routing + ownership records, received: %s", res.RawBody)

	routing := res.Body.Data.DnsRecords[0]
	require.Equal(t, openapi.CNAME, routing.Type)
	require.Equal(t, domain, routing.Name)
	require.Equal(t, "a1b2c3d4e5f6g7h8.cname.unkey.com", routing.Value)
	require.NotNil(t, routing.Note)

	txt := res.Body.Data.DnsRecords[1]
	require.Equal(t, openapi.TXT, txt.Type)
	require.Equal(t, "_unkey."+domain, txt.Name)
	require.Equal(t, "unkey-domain-verify=3ZQ8xK1mP7vT5nR2wY6bJ4hL", txt.Value)
	require.NotNil(t, txt.Note)

	// Domain Connect discovery found nothing, so both fields stay absent rather
	// than being serialized as empty strings.
	require.Nil(t, res.Body.Data.DomainConnectProvider)
	require.Nil(t, res.Body.Data.DomainConnectUrl)

	// Domain creation and its audit log are delegated to the control plane.
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

// TestCreateDomainApexRecords pins the apex contract: an apex domain cannot hold
// a CNAME, so routing must be an apex-compatible alias. Returning CNAME here
// would send the caller to a dead end, since the record they were told to create
// cannot exist.
func TestCreateDomainApexRecords(t *testing.T) {
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
	rootKey := h.CreateRootKey(env.workspaceID, "environment.*.create_domain")

	// No subdomain label, so this is a zone apex.
	apex := strings.ToLower(strings.ReplaceAll(uid.New("test"), "_", "")) + ".com"
	res := testutil.CallRoute[handler.Request, handler.Response](h, route, authHeaders(rootKey), makeRequest(env, apex))
	require.Equal(t, 200, res.Status, "expected 200, received: %s", res.RawBody)
	require.Len(t, res.Body.Data.DnsRecords, 2, "received: %s", res.RawBody)

	routing := res.Body.Data.DnsRecords[0]
	require.Equal(t, openapi.ALIAS, routing.Type, "apex cannot hold a CNAME, received: %s", res.RawBody)
	require.Equal(t, apex, routing.Name)
	require.Equal(t, "a1b2c3d4e5f6g7h8.cname.unkey.com", routing.Value)

	txt := res.Body.Data.DnsRecords[1]
	require.Equal(t, openapi.TXT, txt.Type)
	require.Equal(t, "_unkey."+apex, txt.Name)
	require.NotNil(t, txt.Note)
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
