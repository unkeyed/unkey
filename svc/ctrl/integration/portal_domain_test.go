//go:build integration

package integration

import (
	"context"
	"database/sql"
	"net"
	"sync"
	"testing"
	"time"

	"connectrpc.com/connect"
	restate "github.com/restatedev/sdk-go"
	restatetest "github.com/restatedev/sdk-go/testing"
	"github.com/stretchr/testify/require"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	mysqltype "github.com/unkeyed/unkey/pkg/mysql/types"
	restateadmin "github.com/unkeyed/unkey/pkg/restate/admin"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/ctrl/integration/seed"
	"github.com/unkeyed/unkey/svc/ctrl/internal/auditlogs"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
	portaldomainapi "github.com/unkeyed/unkey/svc/ctrl/services/portaldomain"
	"github.com/unkeyed/unkey/svc/ctrl/worker/domainverify"
	"github.com/unkeyed/unkey/svc/ctrl/worker/environment"
	"github.com/unkeyed/unkey/svc/ctrl/worker/portaldomain"
	"github.com/unkeyed/unkey/svc/ctrl/worker/routing"
)

// tenantDNS answers as if every registered tenant pointed its hostname at the
// CNAME target ctrl issued it. The target is read from the tenant's row because
// AddPortalDomain generates it after the hostname is registered here.
type tenantDNS struct {
	db     db.Database
	mu     sync.Mutex
	owners map[string]string
}

var _ domainverify.Resolver = (*tenantDNS)(nil)

func (r *tenantDNS) point(hostname, workspaceID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.owners[hostname] = workspaceID
}

func (r *tenantDNS) LookupCNAME(ctx context.Context, name string) (string, error) {
	r.mu.Lock()
	workspaceID, ok := r.owners[name]
	r.mu.Unlock()
	if !ok {
		return "", dnsNotFound(name)
	}
	// AddPortalDomain starts the workflow before its insert commits, so a
	// missing row reads as DNS not configured yet.
	claim, err := r.db.FindPortalDomainByWorkspaceAndDomain(ctx, db.FindPortalDomainByWorkspaceAndDomainParams{
		WorkspaceID: workspaceID,
		Domain:      name,
	})
	if err != nil {
		return "", dnsNotFound(name)
	}
	row, err := r.db.FindPortalDomainById(ctx, claim.ID)
	if err != nil {
		return "", dnsNotFound(name)
	}
	return row.TargetCname, nil
}

func (r *tenantDNS) LookupTXT(_ context.Context, name string) ([]string, error) {
	return nil, dnsNotFound(name)
}

func (r *tenantDNS) LookupHost(_ context.Context, name string) ([]string, error) {
	return nil, dnsNotFound(name)
}

func dnsNotFound(name string) error {
	return &net.DNSError{Err: "no such host", Name: name, Server: "", IsTimeout: false, IsTemporary: false, IsNotFound: true, UnwrapErr: nil}
}

// stubCertificates accepts the issuance the workflow sends after routing;
// certificate issuance is outside these tests.
type stubCertificates struct {
	hydrav1.UnimplementedCertificateServiceServer
}

func (stubCertificates) ProcessChallenge(restate.ObjectContext, *hydrav1.ProcessChallengeRequest) (*hydrav1.ProcessChallengeResponse, error) {
	return &hydrav1.ProcessChallengeResponse{}, nil
}

// stubDeployments accepts the standby bookkeeping promotion sends to the
// promoted and demoted deployments.
type stubDeployments struct {
	hydrav1.UnimplementedDeploymentServiceServer
}

func (stubDeployments) ClearScheduledStateChanges(restate.ObjectContext, *hydrav1.ClearScheduledStateChangesRequest) (*hydrav1.ClearScheduledStateChangesResponse, error) {
	return &hydrav1.ClearScheduledStateChangesResponse{}, nil
}

func (stubDeployments) ScheduleDesiredStateChange(restate.ObjectContext, *hydrav1.ScheduleDesiredStateChangeRequest) (*hydrav1.ScheduleDesiredStateChangeResponse, error) {
	return &hydrav1.ScheduleDesiredStateChangeResponse{}, nil
}

// portalFixture is the portal app (its own workspace, production environment,
// and live deployment) plus the ctrl API and worker services that verify,
// route, delete, and promote, running against real MySQL and Restate.
type portalFixture struct {
	h        *Harness
	tEnv     *restatetest.TestEnvironment
	api      *portaldomainapi.Service
	dns      *tenantDNS
	project  db.Project
	app      db.App
	env      db.Environment
	live     db.Deployment
	portalID string
}

func newPortalFixture(t *testing.T) *portalFixture {
	t.Helper()

	h := New(t)
	ctx := h.Context()

	portalWorkspace := h.Seed.CreateWorkspace(ctx)
	project := h.Seed.CreateProject(ctx, seed.CreateProjectRequest{
		ID:               uid.New("prj"),
		WorkspaceID:      portalWorkspace.ID,
		Name:             "portal",
		Slug:             uid.New("slug"),
		DeleteProtection: false,
	})
	app := h.Seed.CreateApp(ctx, seed.CreateAppRequest{
		ID:          uid.New("app"),
		WorkspaceID: portalWorkspace.ID,
		ProjectID:   project.ID,
		Name:        "portal",
		Slug:        "default",
	})
	env := h.Seed.CreateEnvironment(ctx, seed.CreateEnvironmentRequest{
		ID:               uid.New("env"),
		WorkspaceID:      portalWorkspace.ID,
		ProjectID:        project.ID,
		AppID:            app.ID,
		Slug:             "production",
		Kind:             mysqltype.EnvironmentKindProduction,
		Description:      "",
		SentinelConfig:   []byte("{}"),
		DeleteProtection: false,
	})
	live := h.Seed.CreateDeployment(ctx, seed.CreateDeploymentRequest{
		WorkspaceID:   portalWorkspace.ID,
		ProjectID:     project.ID,
		AppID:         app.ID,
		EnvironmentID: env.ID,
		Status:        mysqltype.DeploymentsStatusReady,
	})
	require.NoError(t, h.DB.UpdateAppDeployments(ctx, db.UpdateAppDeploymentsParams{
		AppID:               app.ID,
		CurrentDeploymentID: sql.NullString{Valid: true, String: live.ID},
		IsRolledBack:        false,
		UpdatedAt:           sql.NullInt64{Valid: true, Int64: h.Now()},
	}))

	auditSvc, err := auditlogs.New(auditlogs.Config{DB: h.DB})
	require.NoError(t, err)

	// Promotion never calls the admin API; the environment service only
	// requires a client to exist.
	environmentSvc, err := environment.New(environment.Config{
		DB:        h.DB,
		Admin:     restateadmin.New(restateadmin.Config{BaseURL: "http://127.0.0.1:1", APIKey: ""}),
		Auditlogs: auditSvc,
	})
	require.NoError(t, err)

	dns := &tenantDNS{db: h.DB, mu: sync.Mutex{}, owners: map[string]string{}}

	// Production retries every minute; the first attempt can run before
	// AddPortalDomain's insert commits, so retry fast here.
	tEnv := restatetest.Start(t,
		hydrav1.NewPortalDomainServiceServer(
			portaldomain.New(portaldomain.Config{DB: h.DB, Resolver: dns, EnvironmentID: env.ID}),
			restate.WithInvocationRetryPolicy(
				restate.WithInitialRetryInterval(100*time.Millisecond),
				restate.WithRetryIntervalFactor(1.0),
				restate.WithMaxRetryInterval(100*time.Millisecond),
			),
		),
		hydrav1.NewRoutingServiceServer(routing.New(routing.Config{DB: h.DB, DefaultDomain: "unkey.test"})),
		hydrav1.NewEnvironmentServiceServer(environmentSvc),
		hydrav1.NewCertificateServiceServer(stubCertificates{UnimplementedCertificateServiceServer: hydrav1.UnimplementedCertificateServiceServer{}}),
		hydrav1.NewDeploymentServiceServer(stubDeployments{UnimplementedDeploymentServiceServer: hydrav1.UnimplementedDeploymentServiceServer{}}),
	)

	api := portaldomainapi.New(portaldomainapi.Config{
		Database:      h.DB,
		Restate:       tEnv.Ingress(),
		RestateAdmin:  nil,
		Auditlogs:     auditSvc,
		CnameDomain:   "portal.unkey.test",
		EnvironmentID: env.ID,
		Bearer:        testBearer,
	})

	return &portalFixture{
		h:        h,
		tEnv:     tEnv,
		api:      api,
		dns:      dns,
		project:  project,
		app:      app,
		env:      env,
		live:     live,
		portalID: uid.New("portal"),
	}
}

// uniqueHostname avoids frontline_routes collisions with earlier runs against
// the shared MySQL container.
func uniqueHostname() string {
	return uid.DNS1035() + ".portal-it.example.com"
}

// addVerified attaches hostname to workspaceID's portal through the ctrl API
// with DNS pointed correctly, and waits for the workflow to verify it.
func (f *portalFixture) addVerified(t *testing.T, workspaceID, hostname string) string {
	t.Helper()
	ctx := f.h.Context()

	f.dns.point(hostname, workspaceID)

	req := connect.NewRequest(&ctrlv1.AddPortalDomainRequest{
		WorkspaceId: workspaceID,
		PortalId:    f.portalID,
		Domain:      hostname,
		Actor:       nil,
	})
	req.Header().Set("Authorization", "Bearer "+testBearer)
	resp, err := f.api.AddPortalDomain(ctx, req)
	require.NoError(t, err)
	domainID := resp.Msg.GetDomainId()

	require.Eventually(t, func() bool {
		row, findErr := f.h.DB.FindPortalDomainById(ctx, domainID)
		require.NoError(t, findErr)
		require.NotEqual(t, db.PortalDomainsVerificationStatusFailed, row.VerificationStatus, row.VerificationError.String)
		return row.VerificationStatus == db.PortalDomainsVerificationStatusVerified
	}, time.Minute, 200*time.Millisecond)

	// The route is created after the row is marked verified.
	require.Eventually(t, func() bool {
		_, findErr := f.h.DB.FindFrontlineRouteByFQDN(ctx, hostname)
		return findErr == nil
	}, 30*time.Second, 200*time.Millisecond)

	return domainID
}

func (f *portalFixture) delete(t *testing.T, workspaceID, domainID string) {
	t.Helper()

	req := connect.NewRequest(&ctrlv1.DeletePortalDomainRequest{
		WorkspaceId: workspaceID,
		PortalId:    f.portalID,
		DomainId:    domainID,
		Actor:       nil,
	})
	req.Header().Set("Authorization", "Bearer "+testBearer)
	_, err := f.api.DeletePortalDomain(f.h.Context(), req)
	require.NoError(t, err)
}

// insertDeployRoute gives workspaceID a deploy project with a live route on
// hostname, the shape a verified deploy custom domain leaves behind.
func (f *portalFixture) insertDeployRoute(t *testing.T, workspaceID, hostname string) db.FrontlineRoute {
	t.Helper()
	ctx := f.h.Context()
	now := f.h.Now()

	project := f.h.Seed.CreateProject(ctx, seed.CreateProjectRequest{
		ID:               uid.New("prj"),
		WorkspaceID:      workspaceID,
		Name:             "deploy",
		Slug:             uid.New("slug"),
		DeleteProtection: false,
	})
	app := f.h.Seed.CreateApp(ctx, seed.CreateAppRequest{
		ID:          uid.New("app"),
		WorkspaceID: workspaceID,
		ProjectID:   project.ID,
		Name:        "default",
		Slug:        "default",
	})
	env := f.h.Seed.CreateEnvironment(ctx, seed.CreateEnvironmentRequest{
		ID:               uid.New("env"),
		WorkspaceID:      workspaceID,
		ProjectID:        project.ID,
		AppID:            app.ID,
		Slug:             "production",
		Kind:             mysqltype.EnvironmentKindProduction,
		Description:      "",
		SentinelConfig:   []byte("{}"),
		DeleteProtection: false,
	})

	require.NoError(t, f.h.DB.InsertFrontlineRoute(ctx, db.InsertFrontlineRouteParams{
		ID:                       uid.New(uid.FrontlineRoutePrefix),
		ProjectID:                project.ID,
		AppID:                    app.ID,
		DeploymentID:             "",
		EnvironmentID:            env.ID,
		FullyQualifiedDomainName: hostname,
		Sticky:                   db.FrontlineRoutesStickyLive,
		CreatedAt:                now,
		UpdatedAt:                sql.NullInt64{Valid: true, Int64: now},
	}))

	route, err := f.h.DB.FindFrontlineRouteByFQDN(ctx, hostname)
	require.NoError(t, err)
	return route
}

// insertPortalDomain writes a portal_domains row directly, for states the
// workflow is not driven through.
func (f *portalFixture) insertPortalDomain(t *testing.T, workspaceID, hostname string, status db.PortalDomainsVerificationStatus) string {
	t.Helper()

	domainID := uid.New(uid.PortalDomainPrefix)
	require.NoError(t, f.h.DB.InsertPortalDomain(f.h.Context(), db.InsertPortalDomainParams{
		ID:                    domainID,
		WorkspaceID:           workspaceID,
		PortalID:              f.portalID,
		Domain:                hostname,
		VerificationStatus:    status,
		VerificationToken:     uid.Secure(24),
		TargetCname:           uid.DNS1035(16) + ".portal.unkey.test",
		DomainConnectProvider: sql.NullString{Valid: false, String: ""},
		DomainConnectUrl:      sql.NullString{Valid: false, String: ""},
		InvocationID:          sql.NullString{Valid: false, String: ""},
		CreatedAt:             f.h.Now(),
	}))
	return domainID
}

func TestPortalDomain_VerifiedDomainRoutesToPortalEnvironment(t *testing.T) {
	f := newPortalFixture(t)
	tenant := f.h.Resources().UserWorkspace.ID
	hostname := uniqueHostname()

	f.addVerified(t, tenant, hostname)

	route, err := f.h.DB.FindFrontlineRouteByFQDN(f.h.Context(), hostname)
	require.NoError(t, err)
	require.Equal(t, f.project.ID, route.ProjectID)
	require.Equal(t, f.app.ID, route.AppID)
	require.Equal(t, f.env.ID, route.EnvironmentID)
	require.Equal(t, f.live.ID, route.DeploymentID)
	require.Equal(t, db.FrontlineRoutesStickyLive, route.Sticky)
}

// Every tenant's route shares the portal project, so deletion must remove only
// the deleted hostname's route and leave other tenants and deploy routes alone.
func TestPortalDomain_DeleteRemovesOnlyItsRoute(t *testing.T) {
	f := newPortalFixture(t)
	ctx := f.h.Context()
	tenantA := f.h.Resources().UserWorkspace.ID
	tenantB := f.h.Resources().RootWorkspace.ID

	deleted := uniqueHostname()
	otherTenant := uniqueHostname()
	deployHostname := uniqueHostname()

	deletedID := f.addVerified(t, tenantA, deleted)
	f.addVerified(t, tenantB, otherTenant)
	deployRoute := f.insertDeployRoute(t, tenantB, deployHostname)

	f.delete(t, tenantA, deletedID)

	_, err := f.h.DB.FindFrontlineRouteByFQDN(ctx, deleted)
	require.True(t, db.IsNotFound(err), "the deleted portal domain's route must be gone: %v", err)

	otherRoute, err := f.h.DB.FindFrontlineRouteByFQDN(ctx, otherTenant)
	require.NoError(t, err)
	require.Equal(t, f.env.ID, otherRoute.EnvironmentID)

	gotDeployRoute, err := f.h.DB.FindFrontlineRouteByFQDN(ctx, deployHostname)
	require.NoError(t, err)
	require.Equal(t, deployRoute.ID, gotDeployRoute.ID)
}

// Deletion is scoped to the portal project, so a verified portal row whose
// hostname is routed to another workspace's deploy project cannot remove that
// route.
func TestPortalDomain_DeleteLeavesSameHostnameDeployRoute(t *testing.T) {
	f := newPortalFixture(t)
	ctx := f.h.Context()
	tenant := f.h.Resources().UserWorkspace.ID
	hostname := uniqueHostname()

	deployRoute := f.insertDeployRoute(t, f.h.Resources().RootWorkspace.ID, hostname)
	domainID := f.insertPortalDomain(t, tenant, hostname, db.PortalDomainsVerificationStatusVerified)

	f.delete(t, tenant, domainID)

	got, err := f.h.DB.FindFrontlineRouteByFQDN(ctx, hostname)
	require.NoError(t, err)
	require.Equal(t, deployRoute.ID, got.ID)
	require.Equal(t, deployRoute.ProjectID, got.ProjectID)

	_, err = f.h.DB.FindPortalDomainById(ctx, domainID)
	require.True(t, db.IsNotFound(err), "the portal domain row must be gone: %v", err)
}

// A workspace may hold a pending claim on a hostname another workspace's portal
// has verified. Both routes would live in the portal project, so deleting the
// pending claim must not remove the verified tenant's route.
func TestPortalDomain_DeletePendingClaimKeepsVerifiedPortalRoute(t *testing.T) {
	f := newPortalFixture(t)
	ctx := f.h.Context()
	owner := f.h.Resources().UserWorkspace.ID
	claimant := f.h.Resources().RootWorkspace.ID
	hostname := uniqueHostname()

	f.addVerified(t, owner, hostname)
	ownerRoute, err := f.h.DB.FindFrontlineRouteByFQDN(ctx, hostname)
	require.NoError(t, err)

	pendingID := f.insertPortalDomain(t, claimant, hostname, db.PortalDomainsVerificationStatusPending)
	f.delete(t, claimant, pendingID)

	got, err := f.h.DB.FindFrontlineRouteByFQDN(ctx, hostname)
	require.NoError(t, err)
	require.Equal(t, ownerRoute.ID, got.ID)

	_, err = f.h.DB.FindPortalDomainById(ctx, pendingID)
	require.True(t, db.IsNotFound(err), "the pending claim must be gone: %v", err)
}

// Covers AE4: promoting a new portal deployment through the environment
// service moves every verified portal hostname, because they are sticky=live
// routes on the portal environment.
func TestPortalDomain_RouteFollowsPortalPromotion(t *testing.T) {
	f := newPortalFixture(t)
	ctx := f.h.Context()
	hostname := uniqueHostname()

	f.addVerified(t, f.h.Resources().UserWorkspace.ID, hostname)

	route, err := f.h.DB.FindFrontlineRouteByFQDN(ctx, hostname)
	require.NoError(t, err)
	require.Equal(t, f.live.ID, route.DeploymentID)

	next := f.h.Seed.CreateDeployment(ctx, seed.CreateDeploymentRequest{
		WorkspaceID:   f.env.WorkspaceID,
		ProjectID:     f.project.ID,
		AppID:         f.app.ID,
		EnvironmentID: f.env.ID,
		Status:        mysqltype.DeploymentsStatusReady,
	})

	_, err = hydrav1.NewEnvironmentServiceIngressClient(f.tEnv.Ingress(), f.env.ID).
		PromoteDeployment().
		Request(ctx, &hydrav1.PromoteDeploymentRequest{DeploymentId: next.ID, Actor: nil, CorrelationId: ""})
	require.NoError(t, err)

	route, err = f.h.DB.FindFrontlineRouteByFQDN(ctx, hostname)
	require.NoError(t, err)
	require.Equal(t, next.ID, route.DeploymentID)
	require.Equal(t, db.FrontlineRoutesStickyLive, route.Sticky)
}
