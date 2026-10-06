package customdomain

import (
	"context"
	"database/sql"
	"log/slog"
	"net"
	"reflect"
	"slices"
	"testing"
	"time"

	"github.com/go-sql-driver/mysql"
	restate "github.com/restatedev/sdk-go"
	"github.com/restatedev/sdk-go/x/mocks"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

const testTargetCname = "abc123.cname.unkey.local"

// fakeResolver answers from fixed records; an empty answer is NXDOMAIN, which
// is how an unconfigured record looks to the real resolver.
type fakeResolver struct {
	cname string
	txt   []string
	hosts []string
}

func notFound(name string) error {
	return &net.DNSError{Err: "no such host", Name: name, Server: "", IsTimeout: false, IsTemporary: false, IsNotFound: true, UnwrapErr: nil}
}

func (r fakeResolver) LookupTXT(_ context.Context, name string) ([]string, error) {
	if len(r.txt) == 0 {
		return nil, notFound(name)
	}
	return r.txt, nil
}

func (r fakeResolver) LookupCNAME(_ context.Context, name string) (string, error) {
	if r.cname == "" {
		return "", notFound(name)
	}
	return r.cname, nil
}

func (r fakeResolver) LookupHost(_ context.Context, name string) ([]string, error) {
	if len(r.hosts) == 0 {
		return nil, notFound(name)
	}
	return r.hosts, nil
}

// verifyDB is an in-memory stand-in for the queries VerifyDomain makes. The
// embedded interface is nil, so a query the handler is not expected to make
// panics. Every write is appended to events so tests can assert order.
type verifyDB struct {
	db.Database

	row       db.CustomDomain
	app       db.App
	ownership *db.UpdateCustomDomainOwnershipParams
	failed    *db.UpdateCustomDomainFailedParams

	// claims are verified rows other workspaces hold for the same hostname.
	claims []otherClaim
	// routes is frontline_routes keyed by FQDN, enforcing its unique index.
	routes map[string]db.FrontlineRoute

	events []string
}

type otherClaim struct {
	source      string
	id          string
	workspaceID string
}

func newVerifyDB(row db.CustomDomain) *verifyDB {
	return &verifyDB{
		Database:  nil,
		row:       row,
		app:       db.App{ID: row.AppID, CurrentDeploymentID: sql.NullString{Valid: true, String: "d_live"}}, //nolint:exhaustruct
		ownership: nil,
		failed:    nil,
		claims:    nil,
		routes:    map[string]db.FrontlineRoute{},
		events:    nil,
	}
}

func (f *verifyDB) FindCustomDomainById(_ context.Context, _ string) (db.CustomDomain, error) {
	return f.row, nil
}

func (f *verifyDB) UpdateCustomDomainVerificationStatus(_ context.Context, arg db.UpdateCustomDomainVerificationStatusParams) error {
	f.row.VerificationStatus = arg.VerificationStatus
	f.events = append(f.events, "status "+string(arg.VerificationStatus))
	return nil
}

func (f *verifyDB) UpdateCustomDomainCheckAttempt(_ context.Context, _ db.UpdateCustomDomainCheckAttemptParams) error {
	return nil
}

func (f *verifyDB) UpdateCustomDomainOwnership(_ context.Context, arg db.UpdateCustomDomainOwnershipParams) error {
	f.ownership = &arg
	return nil
}

func (f *verifyDB) FindVerifiedDomainClaimExcludingWorkspace(_ context.Context, arg db.FindVerifiedDomainClaimExcludingWorkspaceParams) (db.FindVerifiedDomainClaimExcludingWorkspaceRow, error) {
	for _, c := range f.claims {
		if c.workspaceID != arg.WorkspaceID {
			return db.FindVerifiedDomainClaimExcludingWorkspaceRow{Source: c.source, ID: c.id, WorkspaceID: c.workspaceID}, nil
		}
	}
	return db.FindVerifiedDomainClaimExcludingWorkspaceRow{}, sql.ErrNoRows //nolint:exhaustruct
}

func (f *verifyDB) UpdatePortalDomainFailed(_ context.Context, arg db.UpdatePortalDomainFailedParams) error {
	f.events = append(f.events, "portal failed "+arg.ID+": "+arg.VerificationError.String)
	return nil
}

func (f *verifyDB) InsertAcmeChallenge(_ context.Context, arg db.InsertAcmeChallengeParams) error {
	f.events = append(f.events, "insert acme "+arg.DomainID)
	return nil
}

func (f *verifyDB) UpdateCustomDomainFailed(_ context.Context, arg db.UpdateCustomDomainFailedParams) error {
	if arg.ID == f.row.ID {
		f.failed = &arg
	}
	f.events = append(f.events, "custom failed "+arg.ID+": "+arg.VerificationError.String)
	return nil
}

func (f *verifyDB) DeleteFrontlineRouteByFQDN(_ context.Context, fqdn string) error {
	delete(f.routes, fqdn)
	f.events = append(f.events, "delete route "+fqdn)
	return nil
}

func (f *verifyDB) DeleteAcmeChallengeByDomainID(_ context.Context, domainID string) error {
	f.events = append(f.events, "delete acme "+domainID)
	return nil
}

func (f *verifyDB) FindAppById(_ context.Context, _ string) (db.App, error) {
	return f.app, nil
}

func (f *verifyDB) InsertFrontlineRoute(_ context.Context, arg db.InsertFrontlineRouteParams) error {
	if _, taken := f.routes[arg.FullyQualifiedDomainName]; taken {
		return &mysql.MySQLError{Number: 1062, SQLState: [5]byte{'2', '3', '0', '0', '0'}, Message: "Duplicate entry for key 'fully_qualified_domain_name'"}
	}
	f.routes[arg.FullyQualifiedDomainName] = db.FrontlineRoute{ //nolint:exhaustruct
		ID:                       arg.ID,
		ProjectID:                arg.ProjectID,
		AppID:                    arg.AppID,
		DeploymentID:             arg.DeploymentID,
		EnvironmentID:            arg.EnvironmentID,
		FullyQualifiedDomainName: arg.FullyQualifiedDomainName,
	}
	f.events = append(f.events, "insert route "+arg.FullyQualifiedDomainName)
	return nil
}

func (f *verifyDB) FindFrontlineRouteByFQDN(_ context.Context, fqdn string) (db.FrontlineRoute, error) {
	route, ok := f.routes[fqdn]
	if !ok {
		return db.FrontlineRoute{}, sql.ErrNoRows //nolint:exhaustruct
	}
	return route, nil
}

func testDomainRow(domain string) db.CustomDomain {
	return db.CustomDomain{ //nolint:exhaustruct
		ID:                 uid.New(uid.DomainPrefix),
		WorkspaceID:        "ws_self",
		ProjectID:          "proj_self",
		AppID:              "app_self",
		EnvironmentID:      "env_self",
		Domain:             domain,
		VerificationStatus: db.CustomDomainsVerificationStatusPending,
		VerificationToken:  "tok_self",
		TargetCname:        testTargetCname,
	}
}

// testRunContext is the plain context a journaled step receives.
type testRunContext struct{ context.Context }

func (testRunContext) Log() *slog.Logger         { return slog.Default() }
func (testRunContext) Request() *restate.Request { return nil }

// newVerifyContext mocks the Restate surface VerifyDomain touches: the key, the
// journaled start time, and every Void step, which it executes for real so the
// fake database observes the writes. The context.Context methods are allowed
// because DNS lookups derive timeouts from the handler context.
func newVerifyContext(t *testing.T, domainID string) *mocks.MockContext {
	t.Helper()

	mockCtx := mocks.NewMockContext(t)
	mockCtx.EXPECT().Key().Return(domainID)
	mockStartedAt(mockCtx, time.Now())
	mockCtx.EXPECT().Deadline().Return(time.Time{}, false).Maybe()
	mockCtx.EXPECT().Done().Return(nil).Maybe()
	mockCtx.EXPECT().Err().Return(nil).Maybe()
	mockCtx.EXPECT().Value(mock.Anything).Return(nil).Maybe()

	mockCtx.EXPECT().
		Run(mock.Anything, mock.AnythingOfType("*encoding.Void"), mock.Anything).
		RunAndReturn(func(fn func(restate.RunContext) (any, error), out any, _ ...restate.RunOption) restate.TerminalError {
			value, err := fn(testRunContext{Context: context.Background()})
			if err != nil {
				terminal := restate.AsTerminalError(err)
				require.NotNil(t, terminal, "a step failed with a retryable error: %v", err)
				return terminal
			}
			// RunVoid steps return no value to store.
			if value != nil {
				reflect.ValueOf(out).Elem().Set(reflect.ValueOf(value))
			}
			return nil
		}).
		Maybe()

	return mockCtx
}

// expectProcessChallenge records the certificate send into events, so tests can
// place it relative to the database writes.
func expectProcessChallenge(t *testing.T, mockCtx *mocks.MockContext, f *verifyDB, domain string) {
	t.Helper()

	client := mocks.NewMockClient(t)
	mockCtx.EXPECT().Object("hydra.v1.CertificateService", domain, "ProcessChallenge", mock.Anything).Return(client).Once()
	client.EXPECT().Send(mock.Anything).Run(func(_ any, _ ...restate.SendOption) {
		f.events = append(f.events, "send ProcessChallenge")
	}).Return(mocks.NewMockInvocation(t)).Once()
}

// Characterization: a subdomain whose CNAME points at its target verifies with
// no TXT record when nobody else holds the hostname.
func TestVerifyDomainCNAMEMatchWithoutContentionVerifies(t *testing.T) {
	row := testDomainRow("api.example.com")
	f := newVerifyDB(row)
	svc := New(Config{DB: f, Resolver: fakeResolver{cname: testTargetCname, txt: nil, hosts: nil}, CnameDomain: "cname.unkey.local"})

	mockCtx := newVerifyContext(t, row.ID)
	expectProcessChallenge(t, mockCtx, f, row.Domain)

	_, err := svc.VerifyDomain(restate.WithMockContext(mockCtx), &hydrav1.VerifyDomainRequest{})
	require.NoError(t, err)
	require.Equal(t, db.CustomDomainsVerificationStatusVerified, f.row.VerificationStatus)
	require.NotNil(t, f.ownership)
	require.True(t, f.ownership.CnameVerified)
	require.Contains(t, f.events, "insert acme "+row.ID)
	require.Contains(t, f.events, "insert route "+row.Domain)
	require.Contains(t, f.events, "send ProcessChallenge")
}

// Characterization: a visible CNAME is not enough when another workspace holds
// the hostname verified; without TXT proof the check stays retryable.
func TestVerifyDomainCNAMEMatchWithContentionRequiresTXT(t *testing.T) {
	row := testDomainRow("api.example.com")
	f := newVerifyDB(row)
	f.claims = []otherClaim{{source: "custom", id: "dom_other", workspaceID: "ws_other"}}
	svc := New(Config{DB: f, Resolver: fakeResolver{cname: testTargetCname, txt: nil, hosts: nil}, CnameDomain: "cname.unkey.local"})

	mockCtx := newVerifyContext(t, row.ID)

	_, err := svc.VerifyDomain(restate.WithMockContext(mockCtx), &hydrav1.VerifyDomainRequest{})
	require.ErrorIs(t, err, errNotVerified)
	require.NotNil(t, f.ownership)
	require.True(t, f.ownership.CnameVerified)
	require.False(t, f.ownership.OwnershipVerified)
	require.NotEqual(t, db.CustomDomainsVerificationStatusVerified, f.row.VerificationStatus)
	require.NotContains(t, f.events, "custom failed dom_other: domain claimed by another workspace")
}

// Characterization: an apex domain proves ownership with TXT but must also
// resolve, or routing was never configured.
func TestVerifyDomainApexWithTXTButNoAddressStaysUnverified(t *testing.T) {
	row := testDomainRow("example.com")
	f := newVerifyDB(row)
	svc := New(Config{DB: f, Resolver: fakeResolver{cname: "", txt: []string{"unkey-domain-verify=" + row.VerificationToken}, hosts: nil}, CnameDomain: "cname.unkey.local"})

	mockCtx := newVerifyContext(t, row.ID)

	_, err := svc.VerifyDomain(restate.WithMockContext(mockCtx), &hydrav1.VerifyDomainRequest{})
	require.ErrorIs(t, err, errNotVerified)
	require.NotNil(t, f.ownership)
	require.True(t, f.ownership.OwnershipVerified)
	require.NotEqual(t, db.CustomDomainsVerificationStatusVerified, f.row.VerificationStatus)
}

// A verified portal domain in another workspace is contention for a deploy
// domain: the CNAME alone no longer verifies, TXT proof does, and the win
// revokes the portal row and the route serving it.
func TestVerifyDomainPortalClaimInAnotherWorkspaceIsContention(t *testing.T) {
	row := testDomainRow("api.example.com")
	claims := []otherClaim{{source: "portal", id: "pdom_other", workspaceID: "ws_other"}}

	t.Run("CNAME alone requires TXT", func(t *testing.T) {
		f := newVerifyDB(row)
		f.claims = claims
		svc := New(Config{DB: f, Resolver: fakeResolver{cname: testTargetCname, txt: nil, hosts: nil}, CnameDomain: "cname.unkey.local"})

		_, err := svc.VerifyDomain(restate.WithMockContext(newVerifyContext(t, row.ID)), &hydrav1.VerifyDomainRequest{})
		require.ErrorIs(t, err, errNotVerified)
		require.False(t, f.ownership.OwnershipVerified)
	})

	t.Run("TXT proof claims it and revokes the portal row", func(t *testing.T) {
		f := newVerifyDB(row)
		f.claims = claims
		f.routes[row.Domain] = db.FrontlineRoute{ID: "fr_other", ProjectID: "", EnvironmentID: "", FullyQualifiedDomainName: row.Domain} //nolint:exhaustruct
		svc := New(Config{DB: f, Resolver: fakeResolver{cname: testTargetCname, txt: []string{"unkey-domain-verify=" + row.VerificationToken}, hosts: nil}, CnameDomain: "cname.unkey.local"})

		mockCtx := newVerifyContext(t, row.ID)
		expectProcessChallenge(t, mockCtx, f, row.Domain)

		_, err := svc.VerifyDomain(restate.WithMockContext(mockCtx), &hydrav1.VerifyDomainRequest{})
		require.NoError(t, err)
		require.Contains(t, f.events, "portal failed pdom_other: domain claimed by another workspace")
		require.Contains(t, f.events, "delete acme pdom_other")
		require.Equal(t, "proj_self", f.routes[row.Domain].ProjectID, "the portal route must be replaced by this domain's")
	})
}

// The same workspace holding the hostname in the other table is not contention:
// isolation between a tenant's own portal and deploy rows is enforced when the
// row is created, not by demanding TXT here.
func TestVerifyDomainSameWorkspaceClaimIsNotContention(t *testing.T) {
	row := testDomainRow("api.example.com")
	f := newVerifyDB(row)
	f.claims = []otherClaim{{source: "portal", id: "pdom_mine", workspaceID: row.WorkspaceID}}
	svc := New(Config{DB: f, Resolver: fakeResolver{cname: testTargetCname, txt: nil, hosts: nil}, CnameDomain: "cname.unkey.local"})

	mockCtx := newVerifyContext(t, row.ID)
	expectProcessChallenge(t, mockCtx, f, row.Domain)

	_, err := svc.VerifyDomain(restate.WithMockContext(mockCtx), &hydrav1.VerifyDomainRequest{})
	require.NoError(t, err)
	for _, event := range f.events {
		require.NotContains(t, event, "pdom_mine")
	}
}

// A route for the hostname that belongs to some other target can never be
// inserted, so retrying until the 24-hour timeout only delays the failure. The
// row is failed and its pending ACME challenge removed so no certificate is
// issued for a hostname this domain does not route.
func TestVerifyDomainRouteOwnedByAnotherTargetFailsTerminally(t *testing.T) {
	row := testDomainRow("api.example.com")
	f := newVerifyDB(row)
	f.routes[row.Domain] = db.FrontlineRoute{ID: "fr_other", ProjectID: "proj_other", EnvironmentID: "env_other", FullyQualifiedDomainName: row.Domain} //nolint:exhaustruct
	svc := New(Config{DB: f, Resolver: fakeResolver{cname: testTargetCname, txt: nil, hosts: nil}, CnameDomain: "cname.unkey.local"})

	mockCtx := newVerifyContext(t, row.ID)
	// No ProcessChallenge expectation: issuance must not start for a domain that
	// ends up failed, and the mock rejects the unexpected send.

	_, err := svc.VerifyDomain(restate.WithMockContext(mockCtx), &hydrav1.VerifyDomainRequest{})
	require.Error(t, err)
	require.True(t, restate.IsTerminalError(err), "a route conflict must stop retries, got: %v", err)
	require.NotNil(t, f.failed, "the row must be marked failed")
	require.Equal(t, db.CustomDomainsVerificationStatusFailed, f.failed.VerificationStatus)
	require.Contains(t, f.events, "delete acme "+row.ID)
	require.Equal(t, "proj_other", f.routes[row.Domain].ProjectID, "the other target's route must be left alone")
}

// The route insert generates a fresh id each attempt, so a replay of an insert
// that already committed hits this domain's own route. That is success.
func TestVerifyDomainReplayedRouteInsertSucceeds(t *testing.T) {
	row := testDomainRow("api.example.com")
	f := newVerifyDB(row)
	f.routes[row.Domain] = db.FrontlineRoute{ID: "fr_committed", ProjectID: row.ProjectID, EnvironmentID: row.EnvironmentID, FullyQualifiedDomainName: row.Domain} //nolint:exhaustruct
	svc := New(Config{DB: f, Resolver: fakeResolver{cname: testTargetCname, txt: nil, hosts: nil}, CnameDomain: "cname.unkey.local"})

	mockCtx := newVerifyContext(t, row.ID)
	expectProcessChallenge(t, mockCtx, f, row.Domain)

	_, err := svc.VerifyDomain(restate.WithMockContext(mockCtx), &hydrav1.VerifyDomainRequest{})
	require.NoError(t, err)
	require.Nil(t, f.failed)
	require.Equal(t, "fr_committed", f.routes[row.Domain].ID)
}

// Issuance resolves the hostname to whichever row is verified, so it must not
// start until the previous owner's claim is revoked; otherwise it can see two.
func TestVerifyDomainRevokesContestedClaimBeforeIssuance(t *testing.T) {
	row := testDomainRow("api.example.com")
	f := newVerifyDB(row)
	f.claims = []otherClaim{{source: "custom", id: "dom_other", workspaceID: "ws_other"}}
	svc := New(Config{DB: f, Resolver: fakeResolver{cname: testTargetCname, txt: []string{"unkey-domain-verify=" + row.VerificationToken}, hosts: nil}, CnameDomain: "cname.unkey.local"})

	mockCtx := newVerifyContext(t, row.ID)
	expectProcessChallenge(t, mockCtx, f, row.Domain)

	_, err := svc.VerifyDomain(restate.WithMockContext(mockCtx), &hydrav1.VerifyDomainRequest{})
	require.NoError(t, err)

	revoked := slices.Index(f.events, "custom failed dom_other: domain claimed by another workspace")
	sent := slices.Index(f.events, "send ProcessChallenge")
	require.NotEqual(t, -1, revoked)
	require.NotEqual(t, -1, sent)
	require.Less(t, revoked, sent, "events: %v", f.events)
}
