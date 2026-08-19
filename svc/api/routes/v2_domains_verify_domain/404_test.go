package handler_test

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_domains_verify_domain"
)

func TestVerifyDomainNotFound(t *testing.T) {
	h := testutil.NewHarness(t)
	ctrlClient := &testutil.MockCustomDomainClient{}
	route := &handler.Handler{DB: h.DB, CtrlClient: ctrlClient}
	h.Register(route)

	seeded := seedDomain(t, h, nil)
	rootKey := h.CreateRootKey(seeded.workspaceID, "environment.*.verify_domain")
	headers := authHeaders(rootKey)

	testCases := []struct {
		name       string
		identifier string
	}{
		{name: "nonexistent domain name", identifier: randomDomain()},
		{name: "nonexistent domain id", identifier: uid.New(uid.DomainPrefix)},
		{name: "an environment id is not a domain id", identifier: seeded.environmentID},
		// Identifiers that are neither a parseable name nor a stored id. The schema
		// carries only length bounds, so these reach the handler and miss the lookup.
		{name: "leading dot", identifier: ".acme.com"},
		{name: "trailing dot", identifier: "api.acme.com."},
		{name: "consecutive dots", identifier: "kebap..acme.com"},
		{name: "label starts with hyphen", identifier: "-api.acme.com"},
		{name: "scheme included", identifier: "https://api.acme.com"},
		{name: "path included", identifier: "api.acme.com/v1"},
		{name: "port included", identifier: "api.acme.com:8080"},
		{name: "whitespace", identifier: "kebap acme.com"},
		{name: "wildcard", identifier: "*.acme.com"},
		{name: "traversal", identifier: "../api.acme.com"},
		{name: "public suffix", identifier: "co.uk"},
		{name: "id with a hyphen is neither form", identifier: "dom-1234abcd"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, headers, handler.Request{
				Domain: tc.identifier,
			})
			require.Equal(t, http.StatusNotFound, res.Status, "expected 404, received: %s", res.RawBody)
			require.Equal(t, "https://unkey.com/docs/errors/unkey/data/domain_not_found", res.Body.Error.Type)
		})
	}

	require.Empty(t, ctrlClient.RetryVerificationCalls,
		"an identifier that misses the lookup must never reach ctrl")
}

// TestVerifyDomainCrossWorkspace pins that the lookup is workspace-scoped. Domains are
// unique per workspace rather than globally, so the same name can exist in two
// workspaces and a root key must only ever retry its own. The other workspace's row is
// verified, so this also pins that the 404 wins before the 412 could probe its state.
func TestVerifyDomainCrossWorkspace(t *testing.T) {
	h := testutil.NewHarness(t)
	ctrlClient := &testutil.MockCustomDomainClient{}
	route := &handler.Handler{DB: h.DB, CtrlClient: ctrlClient}
	h.Register(route)

	seeded := seedDomain(t, h, nil)
	rootKey := h.CreateRootKey(seeded.workspaceID, "environment.*.verify_domain")
	headers := authHeaders(rootKey)

	otherWorkspace := h.CreateWorkspace()
	otherProject := h.CreateProject(seed.CreateProjectRequest{
		ID:          uid.New(uid.ProjectPrefix),
		WorkspaceID: otherWorkspace.ID,
		Name:        "Other Workspace Project",
		Slug:        randomSlug(),
	})
	otherApp := h.CreateApp(seed.CreateAppRequest{
		ID:            uid.New(uid.AppPrefix),
		WorkspaceID:   otherWorkspace.ID,
		ProjectID:     otherProject.ID,
		Name:          "Other Workspace App",
		Slug:          randomSlug(),
		DefaultBranch: "main",
	})
	otherEnv := h.CreateEnvironment(seed.CreateEnvironmentRequest{
		ID:          uid.New(uid.EnvironmentPrefix),
		WorkspaceID: otherWorkspace.ID,
		ProjectID:   otherProject.ID,
		AppID:       otherApp.ID,
		Slug:        "production",
		Description: "Production environment",
	})

	// The same name in another workspace, which the unique index permits.
	sharedName := randomDomain()
	otherDomain := h.CreateCustomDomain(seed.CreateCustomDomainRequest{
		ID:                 uid.New(uid.DomainPrefix),
		WorkspaceID:        otherWorkspace.ID,
		ProjectID:          otherProject.ID,
		AppID:              otherApp.ID,
		EnvironmentID:      otherEnv.ID,
		Domain:             sharedName,
		VerificationStatus: db.CustomDomainsVerificationStatusVerified,
		VerificationToken:  "",
		TargetCname:        "",
		OwnershipVerified:  true,
		CnameVerified:      true,
		VerificationError:  "",
		LastCheckedAt:      0,
	})

	testCases := []struct {
		name       string
		identifier string
	}{
		{name: "by name", identifier: sharedName},
		{name: "by id", identifier: otherDomain.ID},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, headers, handler.Request{
				Domain: tc.identifier,
			})
			require.Equal(t, http.StatusNotFound, res.Status, "expected 404, received: %s", res.RawBody)
			require.NotContains(t, res.RawBody, otherDomain.ID, "cross-workspace lookup leaked the domain id: %s", res.RawBody)
			require.NotContains(t, res.RawBody, otherEnv.ID, "cross-workspace lookup leaked the environment id: %s", res.RawBody)
		})
	}

	require.Empty(t, ctrlClient.RetryVerificationCalls,
		"another workspace's domain must never reach ctrl")
}

// TestVerifyDomainCtrlNotFoundIsMasked covers the race the advisory read leaves open:
// the row existed on the replica but was gone when ctrl resolved it on the primary,
// which is what a concurrent delete looks like. The caller gets the same 404 as for a
// domain that never existed, with the family wording rather than ctrl's internal
// message.
func TestVerifyDomainCtrlNotFoundIsMasked(t *testing.T) {
	h := testutil.NewHarness(t)
	ctrlClient := &testutil.MockCustomDomainClient{
		RetryVerificationFunc: func(_ context.Context, req *ctrlv1.RetryVerificationRequest) (*ctrlv1.RetryVerificationResponse, error) {
			return nil, connect.NewError(connect.CodeNotFound, errors.New("domain not found: "+req.GetDomain()))
		},
	}
	route := &handler.Handler{DB: h.DB, CtrlClient: ctrlClient}
	h.Register(route)

	seeded := seedDomain(t, h, nil)
	rootKey := h.CreateRootKey(seeded.workspaceID, "environment.*.verify_domain")

	res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, authHeaders(rootKey), handler.Request{
		Domain: seeded.domain,
	})
	require.Equal(t, http.StatusNotFound, res.Status, "expected 404, received: %s", res.RawBody)
	require.Equal(t, "The requested domain does not exist.", res.Body.Error.Detail,
		"ctrl's internal wording must not leak, received: %s", res.RawBody)
	require.Equal(t, "https://unkey.com/docs/errors/unkey/data/domain_not_found", res.Body.Error.Type)
	require.Len(t, ctrlClient.RetryVerificationCalls, 1)
}
