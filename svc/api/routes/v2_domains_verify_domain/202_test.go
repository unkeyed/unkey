package handler_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_domains_verify_domain"
)

// The handler resolves the identifier to a stored row and hands the canonical name
// to ctrl, which owns the restart and its audit entry. The assertions here are on
// that hand-off: one call, canonical name, owning project, attributed actor.
func TestVerifyDomainByName(t *testing.T) {
	h := testutil.NewHarness(t)
	ctrlClient := &testutil.MockCustomDomainClient{}
	route := &handler.Handler{DB: h.DB, CtrlClient: ctrlClient}
	h.Register(route)

	seeded := seedDomain(t, h, nil)
	rootKey := h.CreateRootKey(seeded.workspaceID, "environment.*.verify_domain")

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, authHeaders(rootKey), handler.Request{
		Domain: seeded.domain,
	})
	require.Equal(t, http.StatusAccepted, res.Status, "expected 202, received: %s", res.RawBody)
	require.NotEmpty(t, res.Body.Meta.RequestId)

	require.Len(t, ctrlClient.RetryVerificationCalls, 1)
	call := ctrlClient.RetryVerificationCalls[0]
	require.Equal(t, seeded.workspaceID, call.GetWorkspaceId())
	require.Equal(t, seeded.projectID, call.GetProjectId())
	require.Equal(t, seeded.domain, call.GetDomain())
	require.Equal(t, ctrlv1.ActorType_ACTOR_TYPE_ROOT_KEY, call.GetActor().GetType(),
		"the retry must be attributed to the calling root key")
	require.NotEmpty(t, call.GetActor().GetId())
}

// TestVerifyDomainById pins that the id form reaches the same row and that ctrl still
// receives the stored name, since ctrl resolves by (workspace, domain).
func TestVerifyDomainById(t *testing.T) {
	h := testutil.NewHarness(t)
	ctrlClient := &testutil.MockCustomDomainClient{}
	route := &handler.Handler{DB: h.DB, CtrlClient: ctrlClient}
	h.Register(route)

	seeded := seedDomain(t, h, nil)
	rootKey := h.CreateRootKey(seeded.workspaceID, "environment.*.verify_domain")

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, authHeaders(rootKey), handler.Request{
		Domain: seeded.domainID,
	})
	require.Equal(t, http.StatusAccepted, res.Status, "expected 202, received: %s", res.RawBody)

	require.Len(t, ctrlClient.RetryVerificationCalls, 1)
	require.Equal(t, seeded.domain, ctrlClient.RetryVerificationCalls[0].GetDomain(),
		"ctrl must receive the stored name, not the id")
}

// TestVerifyDomainByUnicodeName pins that the Unicode form addresses a domain stored
// in its canonical Punycode form. Without the canonicalization a customer could create
// 'münchen.example.com' and never retry it by the name they typed.
func TestVerifyDomainByUnicodeName(t *testing.T) {
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
	rootKey := h.CreateRootKey(seeded.workspaceID, "environment.*.verify_domain")

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, authHeaders(rootKey), handler.Request{
		Domain: "münchen." + parent,
	})
	require.Equal(t, http.StatusAccepted, res.Status, "expected 202, received: %s", res.RawBody)

	require.Len(t, ctrlClient.RetryVerificationCalls, 1)
	require.Equal(t, canonical, ctrlClient.RetryVerificationCalls[0].GetDomain(),
		"ctrl must receive the canonical stored name")
}

// TestVerifyDomainPendingIsRetryable pins that the gate only stops verified domains.
// Retrying a pending domain is how a caller grants it a fresh verification window
// after fixing DNS records mid-flight.
func TestVerifyDomainPendingIsRetryable(t *testing.T) {
	h := testutil.NewHarness(t)
	ctrlClient := &testutil.MockCustomDomainClient{}
	route := &handler.Handler{DB: h.DB, CtrlClient: ctrlClient}
	h.Register(route)

	seeded := seedDomain(t, h, func(req *seed.CreateCustomDomainRequest) {
		req.VerificationStatus = db.CustomDomainsVerificationStatusPending
		req.VerificationError = ""
	})
	rootKey := h.CreateRootKey(seeded.workspaceID, "environment.*.verify_domain")

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, authHeaders(rootKey), handler.Request{
		Domain: seeded.domain,
	})
	require.Equal(t, http.StatusAccepted, res.Status, "expected 202, received: %s", res.RawBody)
	require.Len(t, ctrlClient.RetryVerificationCalls, 1)
}

func TestVerifyDomainWithSpecificEnvironmentPermission(t *testing.T) {
	h := testutil.NewHarness(t)
	ctrlClient := &testutil.MockCustomDomainClient{}
	route := &handler.Handler{DB: h.DB, CtrlClient: ctrlClient}
	h.Register(route)

	seeded := seedDomain(t, h, nil)
	rootKey := h.CreateRootKey(seeded.workspaceID, "environment."+seeded.environmentID+".verify_domain")

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, authHeaders(rootKey), handler.Request{
		Domain: seeded.domain,
	})
	require.Equal(t, http.StatusAccepted, res.Status, "expected 202, received: %s", res.RawBody)
}
