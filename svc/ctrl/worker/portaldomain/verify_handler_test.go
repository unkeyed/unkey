package portaldomain

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

const (
	testTargetCname = "abc123.portal.unkey.local"
	portalEnvID     = "env_portal"
	portalProjectID = "proj_portal"
	portalAppID     = "app_portal"
)

// fakeResolver answers from fixed records; an empty answer is NXDOMAIN.
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
// embedded interface is nil, so an unexpected query panics. Every write is
// appended to events so tests can assert order.
type verifyDB struct {
	db.Database

	// row is nil when the portal domain row does not exist.
	row       *db.PortalDomain
	ownership *db.UpdatePortalDomainOwnershipParams
	failed    *db.UpdatePortalDomainFailedParams

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

func newVerifyDB(row db.PortalDomain) *verifyDB {
	return &verifyDB{
		Database:  nil,
		row:       &row,
		ownership: nil,
		failed:    nil,
		claims:    nil,
		routes:    map[string]db.FrontlineRoute{},
		events:    nil,
	}
}

func (f *verifyDB) FindPortalDomainById(_ context.Context, _ string) (db.PortalDomain, error) {
	if f.row == nil {
		return db.PortalDomain{}, sql.ErrNoRows //nolint:exhaustruct
	}
	return *f.row, nil
}

func (f *verifyDB) UpdatePortalDomainVerificationStatus(_ context.Context, arg db.UpdatePortalDomainVerificationStatusParams) error {
	f.row.VerificationStatus = arg.VerificationStatus
	f.events = append(f.events, "status "+string(arg.VerificationStatus))
	return nil
}

func (f *verifyDB) UpdatePortalDomainCheckAttempt(_ context.Context, _ db.UpdatePortalDomainCheckAttemptParams) error {
	return nil
}

func (f *verifyDB) UpdatePortalDomainOwnership(_ context.Context, arg db.UpdatePortalDomainOwnershipParams) error {
	f.ownership = &arg
	return nil
}

func (f *verifyDB) UpdatePortalDomainFailed(_ context.Context, arg db.UpdatePortalDomainFailedParams) error {
	if f.row != nil && arg.ID == f.row.ID {
		f.failed = &arg
		f.row.VerificationStatus = arg.VerificationStatus
	}
	f.events = append(f.events, "portal failed "+arg.ID+": "+arg.VerificationError.String)
	return nil
}

func (f *verifyDB) UpdateCustomDomainFailed(_ context.Context, arg db.UpdateCustomDomainFailedParams) error {
	f.events = append(f.events, "custom failed "+arg.ID+": "+arg.VerificationError.String)
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

func (f *verifyDB) InsertAcmeChallenge(_ context.Context, arg db.InsertAcmeChallengeParams) error {
	f.events = append(f.events, "insert acme "+arg.DomainID+" "+arg.WorkspaceID)
	return nil
}

func (f *verifyDB) DeleteAcmeChallengeByDomainID(_ context.Context, domainID string) error {
	f.events = append(f.events, "delete acme "+domainID)
	return nil
}

func (f *verifyDB) DeleteFrontlineRouteByFQDN(_ context.Context, fqdn string) error {
	delete(f.routes, fqdn)
	f.events = append(f.events, "delete route "+fqdn)
	return nil
}

func (f *verifyDB) FindEnvironmentById(_ context.Context, id string) (db.Environment, error) {
	if id != portalEnvID {
		return db.Environment{}, sql.ErrNoRows //nolint:exhaustruct
	}
	return db.Environment{ID: portalEnvID, WorkspaceID: "ws_unkey", ProjectID: portalProjectID, AppID: portalAppID}, nil //nolint:exhaustruct
}

func (f *verifyDB) FindAppById(_ context.Context, id string) (db.App, error) {
	return db.App{ID: id, CurrentDeploymentID: sql.NullString{Valid: true, String: "d_portal_live"}}, nil //nolint:exhaustruct
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
		Sticky:                   arg.Sticky,
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

func testDomainRow(domain string) db.PortalDomain {
	return db.PortalDomain{ //nolint:exhaustruct
		ID:                 uid.New(uid.PortalDomainPrefix),
		WorkspaceID:        "ws_tenant",
		PortalID:           "portal_tenant",
		Domain:             domain,
		VerificationStatus: db.PortalDomainsVerificationStatusPending,
		VerificationToken:  "tok_tenant",
		TargetCname:        testTargetCname,
	}
}

func newService(f *verifyDB, resolver fakeResolver) *Service {
	return New(Config{DB: f, Resolver: resolver, EnvironmentID: portalEnvID})
}

// testRunContext is the plain context a journaled step receives.
type testRunContext struct{ context.Context }

func (testRunContext) Log() *slog.Logger         { return slog.Default() }
func (testRunContext) Request() *restate.Request { return nil }

func mockStartedAt(mockCtx *mocks.MockContext, startedAt time.Time) {
	mockCtx.EXPECT().
		Run(mock.Anything, mock.AnythingOfType("*time.Time"), mock.Anything).
		Call.
		Run(func(args mock.Arguments) {
			*args.Get(1).(*time.Time) = startedAt
		}).
		Return(nil)
}

// newVerifyContext mocks the Restate surface VerifyDomain touches and executes
// every Void step for real so the fake database observes the writes. A step
// that fails with a retryable error is surfaced as one, the way Restate would
// retry the invocation.
func newVerifyContext(t *testing.T, domainID string, startedAt time.Time) *mocks.MockContext {
	t.Helper()

	mockCtx := mocks.NewMockContext(t)
	mockCtx.EXPECT().Key().Return(domainID)
	mockStartedAt(mockCtx, startedAt)
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
			if value != nil {
				reflect.ValueOf(out).Elem().Set(reflect.ValueOf(value))
			}
			return nil
		}).
		Maybe()

	return mockCtx
}

// expectProcessChallenge records the certificate send into events and returns
// the request it carried.
func expectProcessChallenge(t *testing.T, mockCtx *mocks.MockContext, f *verifyDB, domain string) *hydrav1.ProcessChallengeRequest {
	t.Helper()

	sent := &hydrav1.ProcessChallengeRequest{} //nolint:exhaustruct
	client := mocks.NewMockClient(t)
	mockCtx.EXPECT().Object("hydra.v1.CertificateService", domain, "ProcessChallenge", mock.Anything).Return(client).Once()
	client.EXPECT().Send(mock.Anything).Run(func(input any, _ ...restate.SendOption) {
		req, ok := input.(*hydrav1.ProcessChallengeRequest)
		require.True(t, ok, "unexpected ProcessChallenge input %T", input)
		sent.WorkspaceId = req.GetWorkspaceId()
		sent.Domain = req.GetDomain()
		f.events = append(f.events, "send ProcessChallenge")
	}).Return(mocks.NewMockInvocation(t)).Once()
	return sent
}

func verify(t *testing.T, svc *Service, mockCtx *mocks.MockContext) error {
	t.Helper()
	_, err := svc.VerifyDomain(restate.WithMockContext(mockCtx), &hydrav1.VerifyPortalDomainRequest{})
	return err
}

// AddPortalDomain submits the workflow inside the insert transaction, so the
// first attempts can run before the commit lands. Inside the grace window a
// missing row must stay retryable; past it the row was deleted and the
// workflow must stop.
func TestVerifyPortalDomainMissingRow(t *testing.T) {
	cases := map[string]struct {
		age      time.Duration
		terminal bool
	}{
		"inside the grace window": {age: 0, terminal: false},
		"past the grace window":   {age: rowVisibilityGrace + time.Second, terminal: true},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			f := newVerifyDB(testDomainRow("portal.example.com"))
			f.row = nil
			svc := newService(f, fakeResolver{cname: "", txt: nil, hosts: nil})

			mockCtx := mocks.NewMockContext(t)
			mockCtx.EXPECT().Key().Return("pdom_missing")
			mockStartedAt(mockCtx, time.Now().Add(-tc.age))

			err := verify(t, svc, mockCtx)
			require.Error(t, err)
			require.Equal(t, tc.terminal, restate.IsTerminalError(err), "got: %v", err)
		})
	}
}

// Covers AE1: a tenant that never sets DNS has its row failed once the
// invocation is past the 24-hour window.
func TestVerifyPortalDomainPastWindowMarksFailed(t *testing.T) {
	row := testDomainRow("portal.example.com")
	f := newVerifyDB(row)
	svc := newService(f, fakeResolver{cname: "", txt: nil, hosts: nil})

	mockCtx := newVerifyContext(t, row.ID, time.Now().Add(-maxVerificationDuration-time.Second))

	err := verify(t, svc, mockCtx)
	require.True(t, restate.IsTerminalError(err), "got: %v", err)
	require.NotNil(t, f.failed)
	require.Equal(t, db.PortalDomainsVerificationStatusFailed, f.failed.VerificationStatus)
	require.Contains(t, f.failed.VerificationError.String, "timed out")
}

func TestVerifyPortalDomainSuccessRoutesToPortalApp(t *testing.T) {
	row := testDomainRow("portal.example.com")
	f := newVerifyDB(row)
	svc := newService(f, fakeResolver{cname: testTargetCname, txt: nil, hosts: nil})

	mockCtx := newVerifyContext(t, row.ID, time.Now())
	sent := expectProcessChallenge(t, mockCtx, f, row.Domain)

	require.NoError(t, verify(t, svc, mockCtx))
	require.Equal(t, db.PortalDomainsVerificationStatusVerified, f.row.VerificationStatus)

	route, ok := f.routes[row.Domain]
	require.True(t, ok, "a verified portal domain must get a route")
	require.Equal(t, portalProjectID, route.ProjectID)
	require.Equal(t, portalAppID, route.AppID)
	require.Equal(t, portalEnvID, route.EnvironmentID)
	require.Equal(t, "d_portal_live", route.DeploymentID)
	require.Equal(t, db.FrontlineRoutesStickyLive, route.Sticky)

	// The certificate is the tenant's, so issuance must run under the tenant
	// workspace's keyring, never the portal app's workspace.
	require.Equal(t, row.WorkspaceID, sent.GetWorkspaceId())
	require.Equal(t, row.Domain, sent.GetDomain())
	require.Contains(t, f.events, "insert acme "+row.ID+" "+row.WorkspaceID)
	require.Equal(t, "send ProcessChallenge", f.events[len(f.events)-1], "events: %v", f.events)
}

// A Restate replay of a committed route insert hits this row's own route.
func TestVerifyPortalDomainReplayedRouteInsertStaysVerified(t *testing.T) {
	row := testDomainRow("portal.example.com")
	f := newVerifyDB(row)
	f.routes[row.Domain] = db.FrontlineRoute{ID: "fr_committed", ProjectID: portalProjectID, EnvironmentID: portalEnvID, FullyQualifiedDomainName: row.Domain} //nolint:exhaustruct
	svc := newService(f, fakeResolver{cname: testTargetCname, txt: nil, hosts: nil})

	mockCtx := newVerifyContext(t, row.ID, time.Now())
	expectProcessChallenge(t, mockCtx, f, row.Domain)

	require.NoError(t, verify(t, svc, mockCtx))
	require.Nil(t, f.failed)
	require.Equal(t, db.PortalDomainsVerificationStatusVerified, f.row.VerificationStatus)
	require.Equal(t, "fr_committed", f.routes[row.Domain].ID)
}

func TestVerifyPortalDomainRouteOwnedByAnotherTargetFailsTerminally(t *testing.T) {
	row := testDomainRow("portal.example.com")
	f := newVerifyDB(row)
	f.routes[row.Domain] = db.FrontlineRoute{ID: "fr_other", ProjectID: "proj_other", EnvironmentID: "env_other", FullyQualifiedDomainName: row.Domain} //nolint:exhaustruct
	svc := newService(f, fakeResolver{cname: testTargetCname, txt: nil, hosts: nil})

	// No ProcessChallenge expectation: the mock rejects an unexpected send.
	mockCtx := newVerifyContext(t, row.ID, time.Now())

	err := verify(t, svc, mockCtx)
	require.True(t, restate.IsTerminalError(err), "got: %v", err)
	require.NotNil(t, f.failed)
	require.Contains(t, f.events, "delete acme "+row.ID)
	require.Equal(t, "proj_other", f.routes[row.Domain].ProjectID)
}

// A verified claim in another workspace, in either table, makes the CNAME
// insufficient; TXT proof wins the hostname and revokes the claim before
// issuance starts.
func TestVerifyPortalDomainContention(t *testing.T) {
	for _, source := range []string{"custom", "portal"} {
		claim := otherClaim{source: source, id: source + "_other", workspaceID: "ws_other"}
		revoked := source + " failed " + claim.id + ": domain claimed by another workspace"

		t.Run(source+" claim requires TXT", func(t *testing.T) {
			row := testDomainRow("portal.example.com")
			f := newVerifyDB(row)
			f.claims = []otherClaim{claim}
			svc := newService(f, fakeResolver{cname: testTargetCname, txt: nil, hosts: nil})

			err := verify(t, svc, newVerifyContext(t, row.ID, time.Now()))
			require.ErrorIs(t, err, errNotVerified)
			require.False(t, f.ownership.OwnershipVerified)
			require.NotContains(t, f.events, revoked)
		})

		t.Run(source+" claim revoked before issuance", func(t *testing.T) {
			row := testDomainRow("portal.example.com")
			f := newVerifyDB(row)
			f.claims = []otherClaim{claim}
			f.routes[row.Domain] = db.FrontlineRoute{ID: "fr_other", ProjectID: "proj_other", EnvironmentID: "env_other", FullyQualifiedDomainName: row.Domain} //nolint:exhaustruct
			svc := newService(f, fakeResolver{cname: testTargetCname, txt: []string{"unkey-domain-verify=" + row.VerificationToken}, hosts: nil})

			mockCtx := newVerifyContext(t, row.ID, time.Now())
			expectProcessChallenge(t, mockCtx, f, row.Domain)

			require.NoError(t, verify(t, svc, mockCtx))
			revokedAt := slices.Index(f.events, revoked)
			sentAt := slices.Index(f.events, "send ProcessChallenge")
			require.NotEqual(t, -1, revokedAt, "events: %v", f.events)
			require.Less(t, revokedAt, sentAt, "events: %v", f.events)
			require.Equal(t, portalProjectID, f.routes[row.Domain].ProjectID, "the old route must be replaced by the portal's")
		})
	}
}

// A worker missing the portal environment cannot route the hostname, so it
// must not mark the row verified; the step stays retryable until the config
// is fixed.
func TestVerifyPortalDomainWithoutEnvironmentStaysUnverified(t *testing.T) {
	row := testDomainRow("portal.example.com")
	f := newVerifyDB(row)
	svc := New(Config{DB: f, Resolver: fakeResolver{cname: testTargetCname, txt: nil, hosts: nil}, EnvironmentID: ""})

	err := verify(t, svc, newVerifyContext(t, row.ID, time.Now()))
	require.Error(t, err)
	require.False(t, restate.IsTerminalError(err), "got: %v", err)
	require.NotEqual(t, db.PortalDomainsVerificationStatusVerified, f.row.VerificationStatus)
	require.Empty(t, f.routes)
}
