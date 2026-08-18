package handler_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_domains_delete_domain"
)

// The handler resolves the identifier to a stored row and hands the canonical name
// to ctrl, which owns the deletion and its audit entry. The assertions here are on
// that hand-off: one call, canonical name, owning project, attributed actor.
func TestDeleteDomainByName(t *testing.T) {
	h := testutil.NewHarness(t)
	ctrlClient := &testutil.MockCustomDomainClient{}
	route := &handler.Handler{DB: h.DB, CtrlClient: ctrlClient}
	h.Register(route)

	seeded := seedDomain(t, h, nil)
	rootKey := h.CreateRootKey(seeded.workspaceID, "environment.*.delete_domain")

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, authHeaders(rootKey), handler.Request{
		Domain: seeded.domain,
	})
	require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)
	require.NotEmpty(t, res.Body.Meta.RequestId)

	require.Len(t, ctrlClient.DeleteCustomDomainCalls, 1)
	call := ctrlClient.DeleteCustomDomainCalls[0]
	require.Equal(t, seeded.workspaceID, call.GetWorkspaceId())
	require.Equal(t, seeded.projectID, call.GetProjectId())
	require.Equal(t, seeded.domain, call.GetDomain())
	require.Equal(t, ctrlv1.ActorType_ACTOR_TYPE_ROOT_KEY, call.GetActor().GetType(),
		"the deletion must be attributed to the calling root key")
	require.NotEmpty(t, call.GetActor().GetId())
}

// TestDeleteDomainById pins that the id form reaches the same row and that ctrl still
// receives the stored name, since ctrl resolves by (workspace, domain).
func TestDeleteDomainById(t *testing.T) {
	h := testutil.NewHarness(t)
	ctrlClient := &testutil.MockCustomDomainClient{}
	route := &handler.Handler{DB: h.DB, CtrlClient: ctrlClient}
	h.Register(route)

	seeded := seedDomain(t, h, nil)
	rootKey := h.CreateRootKey(seeded.workspaceID, "environment.*.delete_domain")

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, authHeaders(rootKey), handler.Request{
		Domain: seeded.domainID,
	})
	require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)

	require.Len(t, ctrlClient.DeleteCustomDomainCalls, 1)
	require.Equal(t, seeded.domain, ctrlClient.DeleteCustomDomainCalls[0].GetDomain(),
		"ctrl must receive the stored name, not the id")
}

// TestDeleteDomainByUnicodeName pins that the Unicode form addresses a domain stored
// in its canonical Punycode form. Without the canonicalization a customer could create
// 'münchen.example.com' and never delete it by the name they typed.
func TestDeleteDomainByUnicodeName(t *testing.T) {
	h := testutil.NewHarness(t)
	ctrlClient := &testutil.MockCustomDomainClient{}
	route := &handler.Handler{DB: h.DB, CtrlClient: ctrlClient}
	h.Register(route)

	// Unique parent keeps parallel runs from colliding on the shared database.
	parent := randomDomain()
	canonical := "xn--mnchen-3ya." + parent

	seeded := seedDomain(t, h, func(req *seed.CreateCustomDomainRequest) {
		req.Domain = canonical
	})
	rootKey := h.CreateRootKey(seeded.workspaceID, "environment.*.delete_domain")

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, authHeaders(rootKey), handler.Request{
		Domain: "münchen." + parent,
	})
	require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)

	require.Len(t, ctrlClient.DeleteCustomDomainCalls, 1)
	require.Equal(t, canonical, ctrlClient.DeleteCustomDomainCalls[0].GetDomain(),
		"ctrl must receive the canonical stored name")
}

func TestDeleteDomainWithSpecificEnvironmentPermission(t *testing.T) {
	h := testutil.NewHarness(t)
	ctrlClient := &testutil.MockCustomDomainClient{}
	route := &handler.Handler{DB: h.DB, CtrlClient: ctrlClient}
	h.Register(route)

	seeded := seedDomain(t, h, nil)
	rootKey := h.CreateRootKey(seeded.workspaceID, "environment."+seeded.environmentID+".delete_domain")

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, authHeaders(rootKey), handler.Request{
		Domain: seeded.domain,
	})
	require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)
}
